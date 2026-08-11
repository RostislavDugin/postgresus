COMPOSE_TEST = docker compose -f docker-compose.test.yml

.PHONY: test test-backend test-frontend test-logs test-clean

test: test-backend

test-backend:
	$(COMPOSE_TEST) run --rm backend-tests

test-logs:
	$(COMPOSE_TEST) logs

test-clean:
	$(COMPOSE_TEST) down -v --remove-orphans
