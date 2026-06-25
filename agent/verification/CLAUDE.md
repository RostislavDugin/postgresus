# Verification agent guidelines (Go CLI)

Coding standards for the Databasus verification agent — a Go CLI worker that runs on a cloud-managed Linux VM, reports its capacity to the backend over HTTP, and (once the restore phase lands) claims and verifies backups. It has no Gin HTTP server and owns no database schema. The long-running goroutine in this phase is the **capacity heartbeat loop**; the claim/report runner arrives with the restore phase. There is no Windows daemon — the agent runs in the foreground under systemd or as a container.

For project-wide engineering philosophy, naming, and lint/format commands, see the root `CLAUDE.md`. For the backend (Gin/GORM/Swagger) ruleset, see `backend/CLAUDE.md`.

---

## Table of Contents

- [Spacing between logical statements](#spacing-between-logical-statements)
- [Comments](#comments)
- [File organization](#file-organization)
- [Background services](#background-services)
- [Testing](#testing)
- [Time handling](#time-handling)
- [Logging](#logging)
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
func encodeMessages(messages []Message) (string, error) {
	if len(messages) > 0 {
		messagesBytes, err := json.Marshal(messages)
		if err != nil {
			return "", err
		}
		return string(messagesBytes), nil
	}
	return "", nil
}
```

Good:

```go
func encodeMessages(messages []Message) (string, error) {
	if len(messages) > 0 {
		messagesBytes, err := json.Marshal(messages)
		if err != nil {
			return "", err
		}

		return string(messagesBytes), nil
	}

	return "", nil
}
```

---

## Comments

- **No obvious comments** — don't restate what the code already shows.
- **Explain *why*, not *what*** — code shows what happens; comments explain business rules, hidden constraints, or non-obvious optimizations.
- **Prefer refactoring over commenting** — better names or smaller functions usually beat a comment.
- **Complex algorithms deserve comments** — formulas, business rules, non-obvious optimizations.
- **No "Summary" / "Conclusion" sections in `.md` files** unless explicitly requested.

Bad (each comment restates the function name):

```go
// Send heartbeat
sendHeartbeat(request)

// CreateValidLogItems creates valid log items for testing
func CreateValidLogItems(count int, uniqueID string) []LogItemRequestDTO {
```

---

## File organization

One responsibility per file. Don't dump a whole package into one file — split
by role so a reader can find a type by its filename. Conventional names within
a feature package:

- `doc.go` — package doc comment, once the package spans more than one file
- `<feature>.go` — the core type and its methods (the orchestrator/executor)
- `dto.go` — request/response and cross-package data + interface seams
- `errors.go` — sentinel errors (`var Err... = errors.New(...)`)
- `enums.go` — typed-constant groups (`type Status string` + its values)
- `constants.go` — package-level constants that aren't an enum
- background loops, reapers, and pools get their own file (`reaper.go`, `pool.go`)

Only create a file when there is real content for it — an empty `enums.go` or
`constants.go` is noise, not structure. Test files mirror the source split:
`restorer.go` → `restorer_test.go`, `diskexhaustion.go` →
`diskexhaustion_test.go`.

---

## Background services

The agent ships at least one long-running goroutine (the capacity `Heartbeater`; also `BackgroundUpgrader`). Calling `Run()` twice on the same instance is always a bug — duplicate goroutines leak resources and corrupt state. **Always panic; never just log a warning.**

```go
type Heartbeater struct {
    // ...
    hasRun atomic.Bool
}

func (h *Heartbeater) Run(ctx context.Context) {
    if h.hasRun.Swap(true) {
        panic(fmt.Sprintf("%T.Run() called multiple times", h))
    }

    ticker := time.NewTicker(heartbeatInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            h.beat(ctx, logger)
        }
    }
}
```

`atomic.Bool.Swap(true)` does the check-and-set atomically — no `sync.Once` needed.

---

## Testing

**Always run tests after writing them and verify they pass.**

### Naming

- `Test_WhatWeDo_WhatWeExpect`
- `Test_WhatWeDo_WhichConditions_WhatWeExpect`

Examples: `Test_DeriveCapacity_WhenConcurrentJobsExceedCPU_ReturnsError`, `Test_ValidateTransport_WhenHttpAndNotTTYWithoutFlag_FailsFast`, `Test_Heartbeat_WhenCalled_SendsFlatEnvelopeWithBearerAndAgentPath`, `Test_BackgroundUpgrader_WhenRunCalledTwice_Panics`.

### Where tests live

- **Unit / package tests**: alongside the code, named `*_test.go` (e.g. `internal/features/heartbeat/heartbeat_test.go`).
- **End-to-end tests**: under `agent/verification/e2e/`. Run via `make e2e` / `make e2e-clean`.

The agent has no HTTP API of its own, so there are no controller tests — test exported functions of the packages, the CLI commands, and the HTTP-client/IO logic directly.

### Refactor tests as you touch them

When editing existing tests, look for repetitive setup that should become a helper, oversized tests that should be split, and similar patterns across files that should be consolidated. Helpers live in a `testing.go` file inside the package.

### Clean up test data

If a test creates files, processes, or external state, clean it up via `t.Cleanup(...)` or `defer`. Skip cleanup only when the test uses an isolated tempdir the OS reclaims, or when it explicitly validates a failure path where cleanup isn't possible.

---

## Time handling

Always use `time.Now().UTC()` instead of `time.Now()` to keep timezones consistent across the application.

---

## Logging

We use `log/slog`. Three rules.

### 1. Scope IDs early via `logger.With(...)`

Attach `agent_id`, `verification_id`, `upgrade_target`, etc. as soon as you know them so every downstream line carries them automatically. For background services, also scope `job_id` (fresh UUID per execution) and `job_name` (stable snake_case const) in `Run()`.

```go
const jobName = "verification_heartbeat"

func (h *Heartbeater) Run(ctx context.Context) {
    logger := h.log.With("job_id", uuid.New(), "job_name", jobName)

    logger.Info("heartbeat loop started")
    // every subsequent log carries job_id + job_name
}
```

### 2. Values in message, IDs and errors as kv pairs

Sizes, counts, and status transitions go into the message via `fmt.Sprintf`. IDs and errors stay as structured kv pairs so they're searchable in log aggregation tools.

```go
// good
logger.Info(fmt.Sprintf("heartbeat ok: last_seen_at=%s", seenAt))
logger.Info("agent registered", "agent_id", agentID)
logger.Error("heartbeat failed", "error", err)

// bad — ID buried in the message string, error formatted instead of attached
logger.Info(fmt.Sprintf("agent registered %s", agentID))
logger.Error(fmt.Sprintf("heartbeat failed: %v", err))
```

### 3. Style and level

- All keys `snake_case` (`agent_id`, `verification_id`) — never camelCase.
- Messages start lowercase, no trailing period.
- **Debug**: routine ops, function entry, query result counts.
- **Info**: significant state changes, completed actions (`"agent started"`, `"heartbeat loop stopped"`).
- **Warn**: degraded but recoverable (`"heartbeat failed, will retry next tick"`, `"upgrade skipped: same version"`).
- **Error**: failures that need attention (`"auto-update failed"`, `"failed to re-exec after upgrade"`).

### What never goes into logs

- **Passwords, tokens, API keys, full `Authorization` headers** — centralize redaction; don't redact ad-hoc at call sites (config logging masks the token via `maskSensitive`).
- **Full request / response bodies** — log only the fields you actually need.

---

## Modern Go

Prefer modern stdlib idioms over manual equivalents.

### `slices` package — avoid manual loops

```go
slices.Contains(items, x)
slices.Index(items, x)                                         // returns index or -1
slices.IndexFunc(items, func(item T) bool { return item.ID == id })
slices.SortFunc(items, func(a, b T) int { return cmp.Compare(a.X, b.X) })
slices.Sort(items)
slices.Max(items) / slices.Min(items)
slices.Reverse(items)                                          // in-place
slices.Compact(items)                                          // remove consecutive duplicates
slices.Clone(s)
slices.Clip(s)
```

### Quick wins

- `any` instead of `interface{}`.
- `for i := range len(items)` instead of `for i := 0; i < len(items); i++`.
- `sync.OnceFunc(fn)` / `sync.OnceValue(fn)` instead of `sync.Once` + wrapper.
- `t.Context()` in tests instead of `context.WithCancel(context.Background())` + `defer cancel()` — auto-cancels at test end.
- `wg.Go(fn)` instead of `wg.Add(1)` + `go func() { defer wg.Done(); fn() }()`.
- `slog.DiscardHandler` for a no-op logger in tests.

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
