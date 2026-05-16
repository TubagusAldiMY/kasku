.PHONY: build run test test-unit test-integration test-all coverage-html lint generate migrate-up migrate-down tidy docker-build audit

SERVICE_NAME    := auth-service
BIN_DIR         := bin
MOCKGEN         := $(shell go env GOPATH)/bin/mockgen
# Coverage diukur dari paket business logic — exclude cmd/ (DI wiring), proto/
# (boilerplate), grpc/wire_*.go (manual proto encoding), cleanup/ (butuh DB,
# di-cover di tests/integration/).
COVERPKG        := ./internal/usecase/...,./internal/delivery/http/handler/...,./internal/delivery/http/middleware/...,./internal/delivery/http/response/...,./internal/infrastructure/ratelimit/...
COVER_THRESHOLD := 85.0

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(SERVICE_NAME) ./cmd/server

run:
	go run ./cmd/server

# Generate mock dari semua interface yang punya //go:generate directive.
generate:
	@command -v $(MOCKGEN) >/dev/null 2>&1 || go install go.uber.org/mock/mockgen@latest
	go generate ./...

# Unit test: tanpa infra (mocks only) — fast feedback dev loop.
test-unit:
	go test ./internal/... -race -count=1 -coverprofile=coverage-unit.out -coverpkg=$(COVERPKG)
	@go tool cover -func=coverage-unit.out | tail -1

# Integration test: testcontainers (postgres, redis, rabbitmq) — perlu Docker.
test-integration:
	go test ./tests/integration/... -race -count=1 -timeout 5m

# Full test + coverage gating ($(COVER_THRESHOLD)%).
test:
	go test ./internal/... -race -count=1 -coverprofile=coverage.out -coverpkg=$(COVERPKG)
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -func=coverage.out | tail -1 | awk -v threshold=$(COVER_THRESHOLD) \
		'{cov=substr($$3, 1, length($$3)-1)+0; if (cov < threshold) { printf "FAIL: coverage %.1f%% < %.1f%%\n", cov, threshold; exit 1 } else { printf "OK: coverage %.1f%% (gate %.1f%%)\n", cov, threshold }}'

# HTML coverage report untuk inspeksi manual.
coverage-html:
	go test ./internal/... -race -count=1 -coverprofile=coverage.out -coverpkg=$(COVERPKG)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in browser"

# Test + lint + security audit — pre-commit gate.
test-all: test lint audit
	@echo "All checks passed"

lint:
	golangci-lint run ./...

# Security audit (CVE scanner).
audit:
	govulncheck ./...

migrate-up:
	@echo "Running migrations up via service startup..."
	go run ./cmd/server

migrate-down:
	migrate -path migrations -database "$$POSTGRES_DSN" down

tidy:
	go mod tidy

docker-build:
	docker build -t kasku/$(SERVICE_NAME):latest .
