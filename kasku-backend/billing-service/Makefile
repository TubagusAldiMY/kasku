.PHONY: build run test test-unit test-integration lint migrate-up migrate-down tidy generate docker-build

SERVICE_NAME := billing-service
BIN_DIR      := bin

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(SERVICE_NAME) ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v -race -count=1

test-unit:
	go test ./internal/... -v -race -count=1 -short

test-integration:
	go test ./tests/integration/... -v -race -count=1

lint:
	golangci-lint run ./...

migrate-up:
	@echo "Running migrations up via service startup..."
	go run ./cmd/server

migrate-down:
	migrate -path migrations -database "$$POSTGRES_DSN" down

tidy:
	go mod tidy

generate:
	go generate ./...

docker-build:
	docker build -t kasku/$(SERVICE_NAME):latest .
