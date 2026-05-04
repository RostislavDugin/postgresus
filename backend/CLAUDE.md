# Backend guidelines (Go + Gin + GORM)

Coding standards for the Databasus backend (Go + Gin + GORM + PostgreSQL).
For project-wide engineering philosophy, naming, and lint/format commands, see the root `CLAUDE.md`.

---

## Table of Contents

- [Spacing between logical statements](#spacing-between-logical-statements)
- [Comments](#comments)
- [Controllers](#controllers)
- [Dependency injection (DI)](#dependency-injection-di)
- [Background services](#background-services)
- [Migrations](#migrations)
- [Testing](#testing)
- [Time handling](#time-handling)
- [Logging](#logging)
- [CRUD examples](#crud-examples)
- [Modern Go](#modern-go)

---

## Spacing between logical statements

Add blank lines between logical blocks so the flow is visible at a glance:

- before the final `return`
- after variable declarations, before they're used
- between error handling and subsequent logic
- between distinct logical operations

Bad:

```go
func (r *Repository) FindById(id uuid.UUID) (*models.Task, error) {
	var task models.Task
	result := storage.GetDb().Where("id = ?", id).First(&task)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}
		return nil, result.Error
	}
	return &task, nil
}
```

Good:

```go
func (r *Repository) FindById(id uuid.UUID) (*models.Task, error) {
	var task models.Task

	result := storage.GetDb().Where("id = ?", id).First(&task)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}

		return nil, result.Error
	}

	return &task, nil
}
```

---

## Comments

- **No obvious comments** — don't restate what the code already shows.
- **Explain *why*, not *what*** — code shows what happens; comments explain business rules, hidden constraints, or non-obvious optimizations.
- **Prefer refactoring over commenting** — if code needs explaining, consider clearer names or smaller functions first.
- **Swagger comments are mandatory** for every HTTP endpoint.
- **Complex algorithms deserve comments** — formulas, business rules, non-obvious optimizations.
- **No "Summary" / "Conclusion" sections in `.md` files** unless explicitly requested.

Bad (comments restate the function name):

```go
// Create test project
project := CreateTestProject(projectName, user, router)

// CreateValidLogItems creates valid log items for testing
func CreateValidLogItems(count int, uniqueID string) []logs_receiving.LogItemRequestDTO {
```

---

## Controllers

- One controller per feature, combining all routes for that feature.
- Method names express **what we do** (`GetAvailableTasks`), not the suffix `Handler`.
- All routes use Gin and take `*gin.Context`.
- Every route is documented with Swagger annotations — see [CRUD examples → controller.go](#controllergo) for the canonical annotation format.

```go
type AuditLogController struct {
    auditLogService *AuditLogService
}

func (c *AuditLogController) RegisterRoutes(router *gin.RouterGroup) {
    auditRoutes := router.Group("/audit-logs")

    auditRoutes.GET("/global", c.GetGlobalAuditLogs)
    auditRoutes.GET("/users/:userId", c.GetUserAuditLogs)
}
```

---

## Dependency injection (DI)

For DI files (controllers, services, repositories, use cases — not plain data structs), use **implicit positional field declaration**. Adding a new dependency must force every callsite to update; named-field syntax silently accepts a missing field as zero.

```go
// good — positional, fails to compile if a field is added or reordered
var orderController = &OrderController{
    orderService,
    bot_users.GetBotUserService(),
    bots.GetBotService(),
    users.GetUserService(),
}

// bad — named fields, silently accept new fields with zero values
var orderController = &OrderController{
    orderService:   orderService,
    botUserService: bot_users.GetBotUserService(),
    botService:     bots.GetBotService(),
    userService:    users.GetUserService(),
}
```

Apply this anywhere you see the `var fooController = &FooController{...}` + `GetFooController()` getter shape.

### `SetupDependencies()` pattern

`SetupDependencies()` runs cross-feature wiring (e.g. registering audit-log writers on user services) and **must be idempotent** — tests call it many times. Use `sync.OnceFunc`:

```go
var SetupDependencies = sync.OnceFunc(func() {
    users_services.GetUserService().SetAuditLogWriter(auditLogService)
    users_services.GetSettingsService().SetAuditLogWriter(auditLogService)
})
```

`sync.OnceFunc` (Go 1.21+) is concise and thread-safe — don't reinvent it with `sync.Once` + `atomic.Bool`.

### Never inject another feature's repository

A feature's repositories are its **private implementation detail**. When feature A needs data that lives under feature B, extend B's *service* with a public method and inject **that service** — not B's repository.

The seam must live at the service layer so authorization, logging, and invariants stay auditable in one place, and so schema changes don't ripple across features. A feature injecting its **own** repository is unchanged.

```go
// good — cross-feature data access goes through the owning service
var backupController = &BackupController{
    backupService,
    databases.GetDatabaseService(),  // service, not repository
}

// bad — reaches into another feature's private layer
var backupController = &BackupController{
    backupService,
    databases.GetDatabaseRepository(), // bypasses authz / logging / invariants
}
```

---

## Background services

A background service runs an infinite loop in a goroutine. Calling `Run()` twice on the same instance is always a bug — duplicate goroutines leak resources and corrupt state. **Always panic; never just log a warning.**

```go
type BackgroundService struct {
    // ...
    hasRun atomic.Bool
}

func (s *BackgroundService) Run(ctx context.Context) {
    if s.hasRun.Swap(true) {
        panic(fmt.Sprintf("%T.Run() called multiple times", s))
    }

    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.doWork()
        }
    }
}
```

`atomic.Bool.Swap(true)` does the check-and-set atomically — no `sync.Once` needed. Applies to schedulers, registries, worker nodes, and cleanup services.

---

## Migrations

- PostgreSQL only.
- Primary UUID keys → `gen_random_uuid()`.
- Time columns → `TIMESTAMPTZ` (timestamp with zone), never `TIMESTAMP`.
- Tables, constraints, and indexes go in **separate** declarations (table first, then each constraint / index on its own statement).
- Format SQL: align column types, put each constraint clause on its own line.

```sql
CREATE TABLE marketplace_info (
    bot_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title             TEXT NOT NULL,
    description       TEXT NOT NULL,
    short_description TEXT NOT NULL,
    tutorial_url      TEXT,
    info_order        BIGINT NOT NULL DEFAULT 0,
    is_published      BOOLEAN NOT NULL DEFAULT FALSE
);

ALTER TABLE marketplace_info_images
    ADD CONSTRAINT fk_marketplace_info_images_bot_id
    FOREIGN KEY (bot_id)
    REFERENCES marketplace_info (bot_id);
```

---

## Testing

**Always run tests after writing them and verify they pass.**

### Naming

- `Test_WhatWeDo_WhatWeExpect`
- `Test_WhatWeDo_WhichConditions_WhatWeExpect`

Examples: `Test_CreateApiKey_WhenUserIsProjectOwner_ApiKeyCreated`, `Test_DeleteApiKey_WhenUserIsProjectMember_ReturnsForbidden`, `Test_GetProjectAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly`, `Test_ProjectLifecycleE2E_CompletesSuccessfully`.

### Prefer controller tests over unit tests

- Test through HTTP endpoints whenever possible — that's the contract real callers see.
- Avoid testing repositories or services in isolation; cover them via the API.
- Use unit tests only for complex model logic with no API surface.
- File names: `controller_test.go` or `service_test.go` — never `integration_test.go`.

### Shared testing utilities

Each feature creates a `testing.go` (or `testing/testing.go`) with router builders, model creation helpers, and request helpers used by other features' tests. Build creation helpers via the API (`POST /...`) — not direct DB inserts — so the helpers double as a sanity check on the create endpoint.

### Refactor tests as you touch them

When editing existing tests, look for: repetitive setup that should become a helper, oversized tests that should be split, inline test data that should reuse a helper, and similar patterns across files that should be consolidated.

### Clean up test data

If the feature has a DELETE endpoint or cleanup method, **use it** in the test (`defer` or `t.Cleanup`). Prefer API cleanup over direct DB delete. Skip cleanup only when the test runs in an auto-rollback transaction, the cleanup endpoint doesn't exist yet, or the test explicitly validates a failure path where cleanup isn't possible.

### Cloud-mode tests

Toggle `config.GetEnv().IsCloud` via a helper that auto-restores in `t.Cleanup`:

```go
func enableCloud(t *testing.T) {
    t.Helper()
    config.GetEnv().IsCloud = true
    t.Cleanup(func() { config.GetEnv().IsCloud = false })
}
```

### Canonical controller test

```go
func Test_CreateApiKey_WhenUserIsProjectOwner_ApiKeyCreated(t *testing.T) {
    router := CreateApiKeyTestRouter(GetProjectController(), GetMembershipController())
    owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
    project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

    request := CreateApiKeyRequestDTO{Name: "Test API Key"}
    var response ApiKey
    test_utils.MakePostRequestAndUnmarshal(t, router,
        "/api/v1/projects/api-keys/"+project.ID.String(),
        "Bearer "+owner.Token, request, http.StatusOK, &response)

    assert.Equal(t, "Test API Key", response.Name)
    assert.NotEmpty(t, response.Token)
}
```

The same shape extends to:
- **Permission-failure tests** — assert `http.StatusForbidden` plus the error message body.
- **Cross-tenant isolation tests** — create a second user/project, attempt to access the first via the second's endpoint, assert `http.StatusBadRequest`.
- **E2E lifecycle tests** — chain create → mutate → transfer → verify, all via API.

---

## Time handling

Always use `time.Now().UTC()` instead of `time.Now()` to keep timezones consistent across the application.

---

## Logging

We use `log/slog`. Three rules.

### 1. Scope IDs early via `logger.With(...)`

Attach `database_id`, `backup_id`, `subscription_id`, etc. as soon as you know them so every downstream line carries them automatically.

```go
func (s *BillingService) CreateSubscription(logger *slog.Logger, databaseID uuid.UUID) {
    logger = logger.With("database_id", databaseID)

    logger.Debug("creating subscription")
    // every subsequent log automatically carries database_id
}
```

For background services, also scope `task_name` per subtask in `Run()`. Inside loops, scope further with the loop's identifying ID.

### 2. Values in message, IDs and errors as kv pairs

Sizes, counts, and status transitions go into the message via `fmt.Sprintf`. IDs and errors stay as structured kv pairs so they're searchable in log aggregation tools.

```go
// good
logger.Info(fmt.Sprintf("subscription renewed: %s -> %s, %d GB", oldStatus, newStatus, sub.StorageGB))
logger.Info("deleted old backup", "backup_id", backup.ID)
logger.Error("failed to save subscription", "error", err)

// bad — ID buried in the message string, error formatted instead of attached
logger.Info(fmt.Sprintf("deleted old backup %s", backup.ID))
logger.Error(fmt.Sprintf("failed to save subscription: %v", err))
```

### 3. Style and level

- All keys `snake_case` (`database_id`, `total_size_mb`) — never camelCase.
- Messages start lowercase, no trailing period.
- **Debug**: routine ops, function entry, query result counts.
- **Info**: significant state changes, completed actions.
- **Warn**: degraded but recoverable.
- **Error**: failures that need attention.

### Required contextual fields

Every log line must carry enough to reconstruct one entity's history:

- **`request_id`** — auto-generated per HTTP request by Gin middleware and echoed back in the response. Attached to every log inside an HTTP request; do not add it manually.
- **`user_id`, `database_id`, `backup_id`, `subscription_id`, etc.** — attach via `logger.With(...)` at the boundary where the entity becomes known so downstream logs inherit it.
- **Background jobs / schedulers** (no HTTP request, no `request_id`): pass these inline on every log call:
  - **`job_id`** — fresh UUID per execution; the correlation ID for one run.
  - **`job_name`** — stable snake_case identifier of which job this is (e.g. `"backup_retention_cleanup"`, `"audit_log_cleanup"`). Define it as a `const` next to the service — never the struct's type name (renames break log queries). Lets you query "all runs of job X" or "all errors from job Y" across history.

```go
const jobName = "backup_retention_cleanup"

func (c *BackupCleaner) Run(ctx context.Context) {
    jobID := uuid.New()
    logger := c.logger.With("job_id", jobID, "job_name", jobName)

    logger.Info("job started")
    // ...
}
```

### What never goes into logs

- **Passwords, tokens, API keys, full `Authorization` / `Cookie` headers** — centralize redaction at the logger config; don't redact ad-hoc at call sites.
- **PII (email, phone)** — mask before logging (`r***@example.com`, `+7***1234`).
- **Full request / response bodies** — log only the fields you actually need, never whole payloads.

---

## CRUD examples

A complete feature has these files. The controller carries the canonical Swagger annotation format — that's the part to memorize.

### controller.go

```go
package audit_logs

type AuditLogController struct {
    auditLogService *AuditLogService
}

func (c *AuditLogController) RegisterRoutes(router *gin.RouterGroup) {
    auditRoutes := router.Group("/audit-logs")

    auditRoutes.GET("/global", c.GetGlobalAuditLogs)
    auditRoutes.GET("/users/:userId", c.GetUserAuditLogs)
}

// GetGlobalAuditLogs
// @Summary Get global audit logs (ADMIN only)
// @Description Retrieve all audit logs across the system
// @Tags audit-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} GetAuditLogsResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit-logs/global [get]
func (c *AuditLogController) GetGlobalAuditLogs(ctx *gin.Context) {
    user, isOk := ctx.MustGet("user").(*user_models.User)
    if !isOk {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
        return
    }

    request := &GetAuditLogsRequest{}
    if err := ctx.ShouldBindQuery(request); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
        return
    }

    response, err := c.auditLogService.GetGlobalAuditLogs(user, request)
    if err != nil {
        if err.Error() == "only administrators can view global audit logs" {
            ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
        return
    }

    ctx.JSON(http.StatusOK, response)
}
```

### di.go

```go
var auditLogRepository = &AuditLogRepository{}
var auditLogService = &AuditLogService{
    auditLogRepository,
    logger.GetLogger(),
}
var auditLogController = &AuditLogController{auditLogService}

func GetAuditLogService() *AuditLogService       { return auditLogService }
func GetAuditLogController() *AuditLogController { return auditLogController }

var SetupDependencies = sync.OnceFunc(func() {
    users_services.GetUserService().SetAuditLogWriter(auditLogService)
    users_services.GetSettingsService().SetAuditLogWriter(auditLogService)
})
```

### dto.go

```go
type GetAuditLogsRequest struct {
    Limit      int        `form:"limit"      json:"limit"`
    Offset     int        `form:"offset"     json:"offset"`
    BeforeDate *time.Time `form:"beforeDate" json:"beforeDate"`
}

type GetAuditLogsResponse struct {
    AuditLogs []*AuditLog `json:"auditLogs"`
    Total     int64       `json:"total"`
    Limit     int         `json:"limit"`
    Offset    int         `json:"offset"`
}
```

### model.go

```go
type AuditLog struct {
    ID        uuid.UUID  `json:"id"        gorm:"column:id"`
    UserID    *uuid.UUID `json:"userId"    gorm:"column:user_id"`
    ProjectID *uuid.UUID `json:"projectId" gorm:"column:project_id"`
    Message   string     `json:"message"   gorm:"column:message"`
    CreatedAt time.Time  `json:"createdAt" gorm:"column:created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
```

### repository.go

```go
type AuditLogRepository struct{}

func (r *AuditLogRepository) Create(auditLog *AuditLog) error {
    if auditLog.ID == uuid.Nil {
        auditLog.ID = uuid.New()
    }

    return storage.GetDb().Create(auditLog).Error
}

func (r *AuditLogRepository) GetGlobal(limit, offset int, beforeDate *time.Time) ([]*AuditLog, error) {
    var auditLogs []*AuditLog

    query := storage.GetDb().Order("created_at DESC")
    if beforeDate != nil {
        query = query.Where("created_at < ?", *beforeDate)
    }

    err := query.Limit(limit).Offset(offset).Find(&auditLogs).Error

    return auditLogs, err
}
```

### service.go

```go
type AuditLogService struct {
    auditLogRepository *AuditLogRepository
    logger             *slog.Logger
}

func (s *AuditLogService) WriteAuditLog(message string, userID, projectID *uuid.UUID) {
    auditLog := &AuditLog{
        UserID:    userID,
        ProjectID: projectID,
        Message:   message,
        CreatedAt: time.Now().UTC(),
    }

    if err := s.auditLogRepository.Create(auditLog); err != nil {
        s.logger.Error("failed to create audit log", "error", err)
    }
}

func (s *AuditLogService) GetGlobalAuditLogs(
    user *user_models.User,
    request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
    if user.Role != user_enums.UserRoleAdmin {
        return nil, errors.New("only administrators can view global audit logs")
    }

    limit := request.Limit
    if limit <= 0 || limit > 1000 {
        limit = 100
    }

    offset := max(request.Offset, 0)

    auditLogs, err := s.auditLogRepository.GetGlobal(limit, offset, request.BeforeDate)
    if err != nil {
        return nil, err
    }

    total, err := s.auditLogRepository.CountGlobal(request.BeforeDate)
    if err != nil {
        return nil, err
    }

    return &GetAuditLogsResponse{
        AuditLogs: auditLogs,
        Total:     total,
        Limit:     limit,
        Offset:    offset,
    }, nil
}
```

For `controller_test.go`, see [Testing → Canonical controller test](#canonical-controller-test) above.

---

## Modern Go

Prefer modern stdlib idioms over manual equivalents.

### `slices` package — avoid manual loops

```go
slices.Contains(items, x)
slices.Index(items, x)                                         // returns index or -1
slices.IndexFunc(items, func(item T) bool { return item.ID == id })
slices.SortFunc(items, func(a, b T) int { return cmp.Compare(a.X, b.X) })
slices.Sort(items)                                             // ordered types
slices.Max(items) / slices.Min(items)
slices.Reverse(items)                                          // in-place
slices.Compact(items)                                          // remove consecutive duplicates
slices.Clone(s)                                                // shallow copy
slices.Clip(s)                                                 // trim unused capacity
```

### Quick wins

- `any` instead of `interface{}`.
- `for i := range len(items)` instead of `for i := 0; i < len(items); i++`.
- `sync.OnceFunc(fn)` / `sync.OnceValue(fn)` instead of `sync.Once` + wrapper.
- `t.Context()` in tests instead of `context.WithCancel(context.Background())` + `defer cancel()` — auto-cancels at test end.
- `wg.Go(fn)` instead of `wg.Add(1)` + `go func() { defer wg.Done(); fn() }()`.

### `context` helpers

```go
stop := context.AfterFunc(ctx, cleanup)                                  // run cleanup on cancellation
ctx, cancel := context.WithTimeoutCause(parent, d, ErrTimeout)           // timeout with cause
ctx, cancel := context.WithDeadlineCause(parent, deadline, ErrDeadline)  // deadline with cause
```

### `omitzero` instead of `omitempty` for non-nullable types

`omitempty` is broken for `time.Duration`, `time.Time`, structs, slices, and maps — it doesn't omit a zero value. Use `omitzero`:

```go
// good
type Config struct {
    Timeout   time.Duration `json:"timeout,omitzero"`
    CreatedAt time.Time     `json:"createdAt,omitzero"`
}

// bad
type Config struct {
    Timeout   time.Duration `json:"timeout,omitempty"` // broken for Duration!
    CreatedAt time.Time     `json:"createdAt,omitempty"`
}
```

### `new(val)` for pointer literals (Go 1.26+)

`new()` accepts expressions, eliminating the temp-variable pattern:

```go
// good
cfg := Config{Timeout: new(30), Debug: new(true)}

// bad
timeout := 30
debug := true
cfg := Config{Timeout: &timeout, Debug: &debug}
```
