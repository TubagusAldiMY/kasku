.PHONY: build run test lint migrate-up migrate-down tidy docker-build

SERVICE_NAME := finance-service
BIN_DIR      := bin

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(SERVICE_NAME) ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

migrate-up:
	@echo "Running migrations up via service startup..."
	go run ./cmd/server

migrate-down:
	migrate -path migrations -database "$$POSTGRES_DSN" down

tidy:
	go mod tidy

docker-build:
	docker build -t kasku/$(SERVICE_NAME):latest .
