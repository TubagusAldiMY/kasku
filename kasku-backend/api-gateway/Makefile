BINARY_NAME := api-gateway
MODULE      := github.com/TubagusAldiMY/kasku/api-gateway

.PHONY: build run test lint tidy docker-build

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd/server/main.go

run:
	go run ./cmd/server/main.go

test:
	go test ./... -race -cover

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-build:
	docker build -t kasku/$(BINARY_NAME):latest .
