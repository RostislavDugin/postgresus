# Before run

> Copy .env.example to .env in the repo root (single source for backend, frontend, and docker compose)

# Run

Commands run from the repository root, with the tools from `mise.toml` installed.

To run the backend together with the frontend and the dev database:

> just run

To run tests:

> just test

Before commit:

> just lint

# Migrations

The backend applies pending migrations on startup. To create a migration:

> goose -dir backend/migrations create MIGRATION_NAME sql

To apply or roll back migrations on the dev database:

> goose -dir backend/migrations postgres "$GOOSE_DBSTRING" up

> goose -dir backend/migrations postgres "$GOOSE_DBSTRING" down

# Swagger

`just run` generates the swagger docs before the backend starts. To regenerate them by hand, from `backend/`:

> swag init -d . -g cmd/main.go -o swagger

Swagger URL is:

> http://localhost:4005/api/v1/docs/swagger/index.html#/
