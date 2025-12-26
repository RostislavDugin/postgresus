run:
	go run cmd/main.go

test:
	go test -p=1 -count=1 -failfast -timeout 10m .\internal\...

lint:
	golangci-lint fmt && golangci-lint run

migration-create:
	goose create $(name) sql

migration-up:
	goose up

migration-down:
	goose down

swagger:
	@if ! command -v swag > /dev/null 2>&1; then \
		echo "swag not found, installing..."; \
		go install github.com/swaggo/swag/cmd/swag@v1.16.4; \
		export PATH=$$PATH:$$(go env GOPATH)/bin; \
	fi
	@PATH=$$PATH:$$(go env GOPATH)/bin swag init -g ./cmd/main.go -o swagger
