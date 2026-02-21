# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Databasus is a self-hosted backup manager for PostgreSQL, MySQL, MariaDB, and MongoDB. It supports multiple storage backends (S3, Cloudflare R2, Google Drive, SFTP, NAS, Rclone), scheduling with retention policies, notifications (email, Telegram, Slack, Discord, webhooks), AES-256-GCM encryption, and multi-tenant workspaces with RBAC.

The repository is a monorepo: a Go backend that embeds the compiled React frontend at build time.

## Commands

### Backend (`/backend`)

```bash
make run              # Start development server
make test             # Run all tests (sequential, fail-fast, 15m timeout)
make lint             # golangci-lint format + lint
make swagger          # Regenerate Swagger docs from annotations
make migration-create name=MIGRATION_NAME
make migration-up
make migration-down
```

Run a single test package:
```bash
go test -p=1 -count=1 -failfast ./internal/features/backups/...
```

### Frontend (`/frontend`)

```bash
npm run dev           # Vite dev server
npm run build         # TypeScript check + Vite production build
npm run lint          # ESLint
npm run format        # Prettier (run after significant changes)
npm test              # Vitest (single run)
npm run test:watch    # Vitest watch mode
```

### After significant changes

- Backend: `make lint` (from `/backend`)
- Frontend: `npm run format` (from `/frontend`)

## Architecture

### Backend (`/backend/internal/`)

Feature-based modular structure — each domain is a self-contained package under `internal/features/`. Every feature follows a consistent file layout:

```
features/<feature>/
  controller.go   # HTTP handlers (Gin), Swagger annotations, route registration
  service.go      # Business logic
  repository.go   # Database queries (GORM)
  model.go        # GORM models
  dto.go          # Request/response types
  di.go           # Dependency wiring + SetupDependencies()
  testing.go      # Shared test helpers for this feature
  controller_test.go / service_test.go
```

**20 feature domains:** `audit_logs`, `backups`, `databases`, `disk`, `email`, `encryption`, `healthcheck`, `intervals`, `notifiers`, `plan`, `restores`, `storages`, `system`, `tasks`, `tests`, `users`, `workspaces`, and more.

Key shared infrastructure under `internal/`:
- `storage/` — GORM DB instance (`storage.GetDb()`)
- `util/logger/` — slog-based logger (`logger.GetLogger()`)
- `util/testing/` — HTTP test helpers (`MakeGetRequest`, `MakePostRequestAndUnmarshal`, etc.)

### Frontend (`/frontend/src/`)

Feature-Sliced Design-inspired structure:

```
entity/       # TypeScript domain models (types only)
features/     # Feature modules (forms, actions, domain logic)
pages/        # Page-level components
widgets/      # Reusable composite components
shared/
  api/        # Axios-based API client
  hooks/      # Shared React hooks
  lib/        # Pure utilities
  theme/      # Ant Design theme config
  time/       # Date/time helpers
  toast/      # Notification wrappers
  ui/         # Shared UI primitives
storage/      # localStorage utilities
```

UI stack: React 19 + React Router 7 + Ant Design 5 + Tailwind CSS 4 + Recharts.

### Build & Deployment

Multi-stage Dockerfile: Node 24 builds the React app → Go 1.24 compiles the binary with the frontend embedded via `go:embed` → Debian Slim runtime image. The backend serves the SPA directly; no separate frontend server.

Migrations use Goose (SQL files in `/backend/migrations/`). Swagger docs are generated into `/backend/swagger/` before building.

## Backend Coding Conventions

### File structure order in every Go file

1. Type definitions and constants
2. Public methods/functions (uppercase)
3. Private methods/functions (lowercase) — **always at the bottom**

### Boolean naming

Always prefix booleans: `IsActive`, `HasAccess`, `WasCompleted`, `ShouldRetry`, `CanDelete`.

### Spacing

Add blank lines between logical blocks — before `return`, after error handling, between distinct operations.

### Dependency injection

Use implicit field order (positional) in struct literals for controllers/services/repositories — this forces updating all call sites when dependencies change:

```go
// Correct
var controller = &BackupController{
    backupService,
    workspaceService,
}

// Wrong — easy to miss new fields
var controller = &BackupController{
    backupService:   backupService,
    workspaceService: workspaceService,
}
```

### `SetupDependencies()` pattern

All `SetupDependencies()` functions use `sync.Once` + `atomic.Bool` to be idempotent (safe to call multiple times in tests, warns on re-call).

### Background services

All background service `Run()` methods **panic** if called more than once — prevents duplicate goroutines and resource corruption.

### Time

Always use `time.Now().UTC()`, never `time.Now()`.

### Migrations

- PostgreSQL only
- Primary keys: `UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- Timestamps: `TIMESTAMPTZ`
- Separate `CREATE TABLE`, `ALTER TABLE` (constraints), and index statements

### Controllers

- All routes registered via `RegisterRoutes(router *gin.RouterGroup)`
- Every HTTP handler requires Swagger annotations (`@Summary`, `@Tags`, `@Security BearerAuth`, `@Success`, `@Failure`, `@Router`)
- Authenticated user extracted as: `user, isOk := ctx.MustGet("user").(*user_models.User)`

### Testing

- **Prefer controller tests over unit tests** — test through HTTP endpoints
- Test naming: `Test_WhatWeDo_WhatWeExpect` or `Test_WhatWeDo_WhichConditions_WhatWeExpect`
- Test files: `controller_test.go` or `service_test.go` (not `integration_test.go`)
- Extract shared helpers into `testing.go` within each feature package
- Clean up created resources in tests using `defer` + API DELETE endpoints
- Run `make test` after writing tests to verify they pass

## Frontend Coding Conventions

### React component structure order

```typescript
interface Props {
    someValue: SomeValue;
}

// Helper functions outside the component (pure utilities)
const helperFn = () => { ... };

export const MyComponent = ({ someValue }: Props): JSX.Element => {
    // 1. States
    const [state, setState] = useState<...>(...);

    // 2. Functions (handlers, async loaders)
    const loadData = async () => { ... };

    // 3. Hooks
    useEffect(() => { loadData(); }, []);

    // 4. Calculated/derived values
    const computed = someValue.calculate();

    return <div>...</div>;
};
```

### Comments

Comments explain **why**, not **what**. Avoid obvious comments. Swagger annotations are required on all endpoints. Do not add "Summary" sections to `.md` files unless explicitly requested.
