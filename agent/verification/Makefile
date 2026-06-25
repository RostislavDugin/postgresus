.PHONY: run build test lint e2e e2e-clean

-include .env
export

CLEAN_ARTIFACTS := rm -rf e2e/artifacts
DOWN_QUIET := cd e2e && docker compose down -v --remove-orphans 2>/dev/null
DOWN_CLEAN_QUIET := cd e2e && docker compose down -v --rmi local 2>/dev/null

run:
	go run ./cmd run \
		--databasus-host $(DATABASUS_URL) \
		--agent-id $(AGENT_ID) \
		--token $(AGENT_TOKEN) \
		--max-cpu 4 \
		--max-ram-mb 4096 \
		--max-disk-gb 1000 \
		--max-concurrent-jobs 2 \
		--allow-insecure-http \
		--skip-update

build:
	CGO_ENABLED=0 go build -ldflags "-X main.Version=$(VERSION)" -o databasus-verification-agent ./cmd

test:
	go test -race -count=1 -failfast ./internal/...

lint:
	golangci-lint fmt ./cmd/... ./internal/... ./e2e/... && golangci-lint run ./cmd/... ./internal/... ./e2e/...

e2e:
	-$(DOWN_QUIET)
	-$(CLEAN_ARTIFACTS)
	cd e2e && docker compose build --no-cache e2e-mock-server
	cd e2e && docker compose build
	cd e2e && docker compose run --rm e2e-agent-builder
	cd e2e && docker compose run --rm e2e-fixture-gen
	cd e2e && docker compose up -d e2e-mock-server
	cd e2e && docker compose run --rm e2e-agent-runner
	cd e2e && docker compose down -v

e2e-clean:
	-$(DOWN_CLEAN_QUIET)
	-$(CLEAN_ARTIFACTS)
