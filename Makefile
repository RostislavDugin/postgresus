COMPOSE_TEST = docker compose -f docker-compose.test.yml

.PHONY: test test-backend test-frontend test-logs test-clean

test:
	$(MAKE) -j2 test-backend test-frontend

test-backend:
	$(COMPOSE_TEST) run --rm --build backend-tests

test-frontend:
	$(COMPOSE_TEST) run --rm --build frontend-tests

test-logs:
	$(COMPOSE_TEST) logs

test-clean:
	$(COMPOSE_TEST) down -v --remove-orphans

COMPOSE_DEV = docker compose -f docker-compose.dev.yml

.PHONY: run migration-up migration-down migration-create lint-backend swagger

run:
	$(COMPOSE_DEV) up

migration-up:
	$(COMPOSE_DEV) run --rm --entrypoint goose -w /app/backend/migrations app up

migration-down:
	$(COMPOSE_DEV) run --rm --entrypoint goose -w /app/backend/migrations app down

migration-create:
	$(COMPOSE_DEV) run --rm --entrypoint goose -w /app/backend/migrations app create $(name) sql

lint-backend:
	cd backend && golangci-lint fmt && golangci-lint run

swagger:
	cd backend && swag init -g ./cmd/main.go -o swagger
