-include ../.env

run:
	go run cmd/main.go

test:
	go test -p=1 -count=1 -failfast -timeout 15m ./internal/...

lint:
	golangci-lint fmt ./cmd/... ./internal/... && golangci-lint run ./cmd/... ./internal/...

migration-create:
	goose -dir ./migrations create $(name) sql

migration-up:
	goose -dir ./migrations postgres "$(GOOSE_DBSTRING)" up

migration-down:
	goose -dir ./migrations postgres "$(GOOSE_DBSTRING)" down

swagger:
	swag init -g ./cmd/main.go -o swagger
