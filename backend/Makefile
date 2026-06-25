-include ../.env

PORT ?= 4005
TEST_PARALLEL_WORKERS ?= 8

run: kill-stale
	go run ./cmd

test: clean-testcontainers pull-testcontainers
	TEST_PARALLEL_WORKERS=$(TEST_PARALLEL_WORKERS) go run ./cmd/cleanup_test_db
	@for i in $$(seq 0 $$(( $(TEST_PARALLEL_WORKERS) - 1 ))); do \
		dbstring=$$(echo "$(GOOSE_TEST_DBSTRING)" | sed -E "s#/([^/?]+)\?#/\1_w$$i?#"); \
		echo "migrating slot $$i"; \
		goose -dir ./migrations postgres "$$dbstring" up || exit 1; \
	done
	TESTCONTAINERS_RYUK_DISABLED=true TEST_PARALLEL_WORKERS=$(TEST_PARALLEL_WORKERS) go test -p=$(TEST_PARALLEL_WORKERS) -count=1 -failfast -timeout 15m ./internal/...

clean-testcontainers:
	@ids=$$(docker ps -aq --filter "label=org.testcontainers"); \
	if [ -n "$$ids" ]; then docker rm -f $$ids; else echo "no leftover testcontainers"; fi

pull-testcontainers:
	# Tag must start with a digit so a quoted "postgres:postgres" chown arg in physical
	# tests isn't mistaken for an image tag; every real image tag here is version-like.
	@grep -rhoE '"(mysql|mariadb|mongo|postgres):[0-9][a-zA-Z0-9._-]*"' ./internal \
		| tr -d '"' | sort -u \
		| xargs -P 4 -I {} sh -c 'for n in 1 2 3; do docker pull {} && exit 0; echo "retry {} ($$n)"; sleep 3; done; echo "FAILED to pull {} after 3 attempts"; exit 1'

lint:
	golangci-lint fmt ./cmd/... ./internal/... && golangci-lint run ./cmd/... ./internal/...

migration-create:
	goose -dir ./migrations create $(name) sql

migration-up:
	goose -dir ./migrations postgres "$(GOOSE_DBSTRING)" up

migration-up-test:
	goose -dir ./migrations postgres "$(GOOSE_TEST_DBSTRING)" up

migration-down:
	goose -dir ./migrations postgres "$(GOOSE_DBSTRING)" down

swagger:
	swag init -g ./cmd/main.go -o swagger

# A stale `make run` keeps holding $(PORT) and answers with the previously built
# code; a second run that fails to bind only logs the error and serves nothing,
# so requests silently hit the old binary. Free the port first — killing the
# listener also makes its parent `go run` exit. The dev container has no
# lsof/fuser, so resolve the listener through /proc.
kill-stale:
	@port_hex=$$(printf ':%04X' $(PORT)); \
	inode=$$(awk -v p="$$port_hex" '$$4 == "0A" && $$2 ~ p"$$" {print $$10}' /proc/net/tcp /proc/net/tcp6 | head -1); \
	[ -n "$$inode" ] || exit 0; \
	for pid in $$(ls /proc | grep -E '^[0-9]+$$'); do \
		if ls -l /proc/$$pid/fd 2>/dev/null | grep -q "socket:\[$$inode\]"; then \
			echo "freeing port $(PORT): killing stale backend pid $$pid"; \
			kill -9 $$pid 2>/dev/null || true; \
		fi; \
	done
