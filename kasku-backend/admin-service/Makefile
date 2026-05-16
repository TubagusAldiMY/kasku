.PHONY: build run test lint tidy docker-build

SERVICE_NAME := admin-service
BIN_DIR      := bin

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(SERVICE_NAME) ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-build:
	docker build -t kasku/$(SERVICE_NAME):latest .
