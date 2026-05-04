# Agent guidelines (Go CLI)

Coding standards for the Databasus agent — a Go CLI tool that runs alongside a managed Postgres instance and talks to the backend over HTTP. It uses `pgx` directly (no GORM), has no Gin HTTP server, and owns no database schema.

For project-wide engineering philosophy, naming, and lint/format commands, see the root `CLAUDE.md`. For the backend (Gin/GORM/Swagger) ruleset, see `backend/CLAUDE.md`. Go conventions in this file overlap with `backend/CLAUDE.md` on purpose — both should stay in sync as conventions evolve.

---

## Table of Contents

- [Spacing between logical statements](#spacing-between-logical-statements)
- [Comments](#comments)
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
// Run pg_dump
runPgDump(request)

// CreateValidLogItems creates valid log items for testing
func CreateValidLogItems(count int, uniqueID string) []LogItemRequestDTO {
```

---

## Background services

The agent ships at least one long-running goroutine (e.g. `BackgroundUpgrader`). Calling `Run()` twice on the same instance is always a bug — duplicate goroutines leak resources and corrupt state. **Always panic; never just log a warning.**

```go
type BackgroundUpgrader struct {
    // ...
    hasRun atomic.Bool
}

func (u *BackgroundUpgrader) Run(ctx context.Context) {
    if u.hasRun.Swap(true) {
        panic(fmt.Sprintf("%T.Run() called multiple times", u))
    }

    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            u.checkAndUpgrade(ctx)
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

Examples: `Test_RunFullBackup_WhenPgDumpSucceeds_UploadsToStorage`, `Test_StreamWAL_OnConnectionDrop_RetriesWithBackoff`, `Test_BackgroundUpgrader_WhenCalledTwice_Panics`.

### Where tests live

- **Unit / package tests**: alongside the code, named `*_test.go` (e.g. `agent/internal/features/full_backup/backuper_test.go`).
- **End-to-end tests**: under `agent/e2e/`. Run via `make e2e` / `make e2e-clean`.

The agent has no HTTP API of its own, so there are no controller tests — test exported functions of services, the CLI commands, and the streaming/IO logic directly.

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

Attach `database_id`, `backup_id`, `wal_segment`, `upgrade_target`, etc. as soon as you know them so every downstream line carries them automatically.

```go
func (b *Backuper) RunFullBackup(logger *slog.Logger, databaseID, backupID uuid.UUID) {
    logger = logger.With("database_id", databaseID, "backup_id", backupID)

    logger.Debug("starting full backup")
    // every subsequent log carries both IDs
}
```

For background services, also scope `task_name` per subtask in `Run()`. Inside loops, scope further with the loop's identifying ID.

### 2. Values in message, IDs and errors as kv pairs

Sizes, counts, and status transitions go into the message via `fmt.Sprintf`. IDs and errors stay as structured kv pairs so they're searchable in log aggregation tools.

```go
// good
logger.Info(fmt.Sprintf("uploaded backup chunk: %d / %d MB", uploadedMB, totalMB))
logger.Info("uploaded backup", "backup_id", backupID)
logger.Error("failed to upload backup", "error", err)

// bad — ID buried in the message string, error formatted instead of attached
logger.Info(fmt.Sprintf("uploaded backup %s", backupID))
logger.Error(fmt.Sprintf("failed to upload backup: %v", err))
```

### 3. Style and level

- All keys `snake_case` (`database_id`, `total_size_mb`) — never camelCase.
- Messages start lowercase, no trailing period.
- **Debug**: routine ops, function entry, query result counts.
- **Info**: significant state changes, completed actions (`"backup uploaded"`, `"agent started"`).
- **Warn**: degraded but recoverable (`"retrying after WAL stream drop"`, `"upgrade skipped: same version"`).
- **Error**: failures that need attention (`"failed to upload backup"`, `"failed to spawn pg_dump"`).

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
