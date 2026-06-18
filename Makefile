# =============================================================================
# KasKu — Root Makefile
# Jalankan dari direktori ini (kasku/)
# =============================================================================

BACKEND  := kasku-backend
FRONTEND := kasku-frontend
MONOLITH := kasku-backend-monolith

GO_SERVICES := \
	api-gateway          \
	auth-service         \
	user-service         \
	billing-service      \
	finance-service      \
	transaction-service  \
	notification-service \
	investment-service   \
	admin-service

RUST_SERVICES := price-service sync-service

# Variabel opsional — override via CLI: make logs SERVICE=auth-service
SERVICE ?= api-gateway
TARGET  ?= test

.PHONY: \
	up up-obs down restart ps logs \
	build test lint tidy svc \
	fe-dev fe-build fe-test fe-lint fe-check fe-format \
	mono-build mono-release mono-run mono-test mono-lint \
	mono-migrate mono-migrate-revert \
	smoke secrets backup reset-db \
	ci audit help

# =============================================================================
# Docker Compose Stack
# =============================================================================

up: ## Jalankan full dev stack (auto-include override.yml)
	cd $(BACKEND) && docker compose up -d

up-obs: ## Jalankan stack + observability (Prometheus, Grafana, Jaeger, dll.)
	cd $(BACKEND) && docker compose --profile observability up -d

down: ## Stop semua container
	cd $(BACKEND) && docker compose down

restart: down up ## Stop lalu start ulang semua container

ps: ## Status semua container
	cd $(BACKEND) && docker compose ps

logs: ## Log service tertentu. Contoh: make logs SERVICE=auth-service
	cd $(BACKEND) && docker compose logs -f $(SERVICE)

# =============================================================================
# Backend — semua microservices (Go + Rust)
# =============================================================================

build: ## Build semua Go + Rust microservices
	@for svc in $(GO_SERVICES); do \
		printf '\n\033[1;34m──── build: %s ────\033[0m\n' "$$svc"; \
		$(MAKE) -C $(BACKEND)/$$svc build; \
	done
	@for svc in $(RUST_SERVICES); do \
		printf '\n\033[1;34m──── build: %s ────\033[0m\n' "$$svc"; \
		$(MAKE) -C $(BACKEND)/$$svc build; \
	done

test: ## Test semua Go + Rust microservices
	@for svc in $(GO_SERVICES); do \
		printf '\n\033[1;34m──── test: %s ────\033[0m\n' "$$svc"; \
		$(MAKE) -C $(BACKEND)/$$svc test; \
	done
	@for svc in $(RUST_SERVICES); do \
		printf '\n\033[1;34m──── test: %s ────\033[0m\n' "$$svc"; \
		$(MAKE) -C $(BACKEND)/$$svc test; \
	done

lint: ## Lint semua service (golangci-lint + cargo clippy)
	@for svc in $(GO_SERVICES); do \
		printf '\n\033[1;34m──── lint: %s ────\033[0m\n' "$$svc"; \
		$(MAKE) -C $(BACKEND)/$$svc lint; \
	done
	@for svc in $(RUST_SERVICES); do \
		printf '\n\033[1;34m──── lint: %s ────\033[0m\n' "$$svc"; \
		$(MAKE) -C $(BACKEND)/$$svc lint; \
	done

tidy: ## go mod tidy semua service Go
	@for svc in $(GO_SERVICES); do \
		printf '\n\033[1;34m──── tidy: %s ────\033[0m\n' "$$svc"; \
		$(MAKE) -C $(BACKEND)/$$svc tidy; \
	done

svc: ## Target di satu service. Contoh: make svc SERVICE=auth-service TARGET=lint
	$(MAKE) -C $(BACKEND)/$(SERVICE) $(TARGET)

# =============================================================================
# Frontend (kasku-frontend/)
# =============================================================================

fe-dev: ## Dev server frontend → http://localhost:5173
	cd $(FRONTEND) && npm run dev

fe-build: ## Build frontend production
	cd $(FRONTEND) && npm run build

fe-test: ## Unit + e2e test frontend (vitest + playwright)
	cd $(FRONTEND) && npm test

fe-lint: ## Lint frontend (prettier check + eslint)
	cd $(FRONTEND) && npm run lint

fe-check: ## TypeScript check frontend (svelte-check)
	cd $(FRONTEND) && npm run check

fe-format: ## Format frontend (prettier --write)
	cd $(FRONTEND) && npm run format

# =============================================================================
# Monolith (kasku-backend-monolith/)
# =============================================================================

mono-build: ## Build monolith (offline — tanpa koneksi DB)
	cd $(MONOLITH) && SQLX_OFFLINE=true cargo build

mono-release: ## Build monolith release binary → target/release/kasku
	cd $(MONOLITH) && cargo build --release

mono-run: ## Jalankan monolith (DATABASE_URL, REDIS_URL, dll. wajib di-set)
	cd $(MONOLITH) && cargo run

mono-test: ## Test monolith
	cd $(MONOLITH) && cargo test

mono-lint: ## Lint monolith (cargo clippy -D warnings)
	cd $(MONOLITH) && cargo clippy -- -D warnings

mono-migrate: ## Jalankan semua migration monolith ke DB `kasku`
	cd $(MONOLITH) && sqlx migrate run

mono-migrate-revert: ## Rollback satu step migration monolith
	cd $(MONOLITH) && sqlx migrate revert

# =============================================================================
# Infrastructure & Operasional
# =============================================================================

smoke: ## Smoke test semua /health endpoint (stack harus sudah up)
	cd $(BACKEND) && bash tests/integration/smoke.sh

secrets: ## Generate semua secrets (JWT RSA-4096, DB passwords, dll.)
	cd $(BACKEND) && bash infra/generate_secrets.sh

backup: ## Backup DB ke S3 (butuh BACKUP_BUCKET + AWS_* di .env)
	cd $(BACKEND) && docker compose --profile backup run --rm backup

reset-db: ## [DESTRUCTIVE] Hapus semua data DB, reinit dari nol
	@printf '\033[1;31mHATI-HATI: Volume kasku-pgdata akan DIHAPUS — semua data hilang.\033[0m\n'
	@printf 'Tekan Ctrl+C dalam 5 detik untuk batal...\n'
	@sleep 5
	cd $(BACKEND) && docker compose down
	docker volume rm kasku-pgdata 2>/dev/null || true
	cd $(BACKEND) && docker compose up -d

# =============================================================================
# CI / Security Audit
# =============================================================================

ci: lint test fe-lint fe-check ## Lint + test semua (simulasi CI pipeline lokal)

audit: ## Security audit: govulncheck (Go) + cargo audit (Rust + monolith)
	@for svc in $(GO_SERVICES); do \
		printf '\n\033[1;34m──── govulncheck: %s ────\033[0m\n' "$$svc"; \
		(cd $(BACKEND)/$$svc && govulncheck ./...); \
	done
	@for svc in $(RUST_SERVICES); do \
		printf '\n\033[1;34m──── cargo audit: %s ────\033[0m\n' "$$svc"; \
		(cd $(BACKEND)/$$svc && cargo audit); \
	done
	@printf '\n\033[1;34m──── cargo audit: monolith ────\033[0m\n'
	cd $(MONOLITH) && cargo audit

# =============================================================================
# Help
# =============================================================================

help: ## Tampilkan semua target yang tersedia
	@printf '\n\033[1mKasKu — available targets:\033[0m\n\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@printf '\n\033[2mVariabel: SERVICE=<nama> (default: api-gateway), TARGET=<nama> (default: test)\033[0m\n\n'

.DEFAULT_GOAL := help
