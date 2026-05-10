.PHONY: run build test lint tidy docker-build

SERVICE_NAME = investment-service

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/$(SERVICE_NAME) ./cmd/server/main.go

test:
	go test ./... -race -cover

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-build:
	docker build -t kasku-$(SERVICE_NAME):latest .
