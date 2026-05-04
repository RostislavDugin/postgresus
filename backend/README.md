# Before run

> Copy .env.example to .env in the repo root (single source for backend, frontend, and docker compose)
> Copy docker-compose.example.yml to docker-compose.yml in the repo root (for development only)

# Run

To run:

> make run

To run tests:

> make test

Before commit (make sure `golangci-lint` is installed):

> make lint

# Migrations

To create migration:

> make migration-create name=MIGRATION_NAME

To run migrations:

> make migration-up

If latest migration failed:

To rollback on migration:

> make migration-down

# Swagger

To generate swagger docs:

> make swagger

Swagger URL is:

> http://localhost:4005/api/v1/docs/swagger/index.html#/