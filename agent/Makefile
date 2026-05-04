.PHONY: run build test lint wal-generator e2e e2e-clean e2e-backup-restore e2e-backup-restore-clean

-include .env
export

run:
	go run cmd/main.go start \
		--databasus-host http://localhost:4005 \
		--db-id $(AGENT_DB_ID) \
		--token $(AGENT_TOKEN) \
		--pg-host 127.0.0.1 \
		--pg-port 7433 \
		--pg-user devuser \
		--pg-password devpassword \
		--pg-type docker \
		--pg-docker-container-name agent-db \
		--pg-wal-dir ../wal-queue \
		--skip-update

wal-generator:
	go run cmd/wal-generator/main.go

build:
	CGO_ENABLED=0 go build -ldflags "-X main.Version=$(VERSION)" -o databasus-agent ./cmd/main.go

test:
	go test -count=1 -failfast ./internal/...

lint:
	golangci-lint fmt ./cmd/... ./internal/... ./e2e/... && golangci-lint run ./cmd/... ./internal/... ./e2e/...

e2e:
	cd e2e && docker compose build --no-cache e2e-mock-server
	cd e2e && docker compose build
	cd e2e && docker compose run --rm e2e-agent-builder
	cd e2e && docker compose up -d e2e-postgres e2e-mock-server
	cd e2e && docker compose run --rm e2e-agent-runner
	cd e2e && docker compose run --rm e2e-agent-docker
	cd e2e && docker compose down -v

e2e-clean:
	cd e2e && docker compose down -v --rmi local
	cd e2e && docker compose -f docker-compose.backup-restore.yml down -v --rmi local 2>/dev/null || true
	rm -rf e2e/artifacts
