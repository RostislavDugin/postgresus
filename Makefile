COMPOSE_TEST = docker compose -f docker-compose.test.yml

.PHONY: test test-backend test-frontend test-logs test-clean

test:
	$(MAKE) -j2 test-backend test-frontend

test-backend:
	$(COMPOSE_TEST) run --rm backend-tests

test-frontend:
	$(COMPOSE_TEST) run --rm --build frontend-tests

test-logs:
	$(COMPOSE_TEST) logs

test-clean:
	$(COMPOSE_TEST) down -v --remove-orphans
