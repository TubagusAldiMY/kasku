# KasKu

KasKu adalah platform SaaS manajemen keuangan berbasis web/PWA. Repo ini
berbentuk monorepo yang berisi frontend SvelteKit, backend microservices
Go/Rust, konfigurasi Docker Compose, observability, backup, CI, dan deployment.

Fitur utama yang sudah terlihat dari codebase:

- Autentikasi user, verifikasi email, reset password, refresh token, dan JWT.
- Dashboard finansial, akun, transaksi, kategori, budget, investasi, laporan,
  billing, dan profil.
- Offline-first PWA dengan IndexedDB dan sync engine.
- Subscription/payment flow via billing-service dan payment orchestrator.
- Admin dashboard untuk user management, payment view, subscription override,
  dan audit log.
- Observability dengan Prometheus, Grafana, Loki, Alertmanager, Jaeger, dan
  OpenTelemetry Collector.

## Tech Stack

| Area | Teknologi |
| --- | --- |
| Frontend | SvelteKit, Svelte 5, TypeScript, Vite, Tailwind CSS, IndexedDB |
| Backend Go | Gin, pgx, Redis, RabbitMQ, zerolog, gRPC/protobuf-style internal RPC |
| Backend Rust | Axum, sqlx, tonic, Tokio |
| Data/Infra | PostgreSQL 16, Redis 7, RabbitMQ 3.13, Docker Compose, Traefik |
| Observability | Prometheus, Grafana, Loki, Promtail, Alertmanager, Jaeger, OTEL Collector |
| CI/CD | GitHub Actions, GHCR, self-hosted deploy runner |

## Struktur Repo

```text
.
|-- kasku-frontend/              # SvelteKit PWA
|-- kasku-backend/
|   |-- api-gateway/             # Public API gateway, auth middleware, rate limit
|   |-- auth-service/            # Register, login, token, email verification
|   |-- user-service/            # User profile dan tenant provisioning
|   |-- billing-service/         # Subscription dan payment
|   |-- finance-service/         # Financial account dan tenant schema owner
|   |-- transaction-service/     # Transaksi, kategori, budget
|   |-- investment-service/      # Asset investasi
|   |-- price-service/           # Rust service untuk price cache eksternal
|   |-- sync-service/            # Rust offline sync engine
|   |-- notification-service/    # Email notification dan event consumer
|   |-- admin-service/           # Admin API dan audit log
|   |-- observability-go/        # Shared Go metrics/tracing helper
|   |-- infra/                   # Postgres init, Grafana, Prometheus, backup, dll
|   |-- tests/                   # Smoke, integration, dan load test
|   |-- docker-compose.yml
|   `-- docker-compose.override.yml
|-- .github/workflows/           # CI dan deploy workflow
|-- LICENSE
`-- README.md
```

## Arsitektur Singkat

Frontend memanggil `api-gateway` melalui base URL `/v1`. Gateway menangani CORS,
rate limiting, JWT validation, dan proxy request ke service internal.

Data utama berada di PostgreSQL dengan database/service user terpisah. Beberapa
domain memakai boundary khusus:

- `finance-service` memiliki database `kasku_finance` dan tenant schema.
- `transaction-service`, `investment-service`, dan `sync-service` bekerja di
  atas tenant schema finance sesuai kontrak internal.
- `billing-service` memiliki subscription dan payment.
- `admin-service` terisolasi di network `kasku-admin` dan memakai JWT HS256
  terpisah dari JWT user.
- RabbitMQ dipakai untuk async event bus, termasuk provisioning dan notifikasi.
- Redis dipakai untuk rate limit counter dan token blacklist.

## Prasyarat

- Docker dan Docker Compose plugin.
- Node.js 22 dan npm.
- Go 1.25.
- Rust stable.
- OpenSSL untuk membuat key JWT lokal.
- Opsional: `curl`, `jq`, `uuidgen`, `k6`, `golangci-lint`, `govulncheck`,
  `cargo-audit`.

## Quick Start Lokal

### 1. Siapkan environment backend

```bash
cd kasku-backend
cp .env.example .env
```

Buat RSA key pair untuk JWT user:

```bash
openssl genrsa -out private.pem 4096
openssl rsa -in private.pem -pubout -out public.pem

base64 -w0 private.pem
base64 -w0 public.pem

rm private.pem public.pem
```

Masukkan output base64 ke `JWT_PRIVATE_KEY` dan `JWT_PUBLIC_KEY` di
`kasku-backend/.env`. Untuk development, nilai placeholder lain bisa dipakai
sebagai awal, tetapi password dan secret tetap sebaiknya diganti. Jangan commit
file `.env`.

### 2. Jalankan backend stack

```bash
docker compose up -d --build
docker compose ps
```

Compose override lokal otomatis membuka port penting berikut:

| Komponen | URL/Port |
| --- | --- |
| API gateway | `http://localhost:8080` |
| PostgreSQL | `localhost:5433` |
| Redis | `localhost:6380` |
| RabbitMQ AMQP | `localhost:5672` |
| RabbitMQ Management | `http://localhost:15672` |
| Sync service direct | `http://localhost:8088` |

Cek health gateway:

```bash
curl http://localhost:8080/health
```

### 3. Jalankan frontend

Di terminal lain:

```bash
cd kasku-frontend
cp .env.example .env
npm ci
npm run dev
```

Frontend development server berjalan di `http://localhost:5173`. Default
`PUBLIC_API_BASE_URL` di `kasku-frontend/.env.example` sudah mengarah ke
`http://localhost:8080/v1`.

## Command Development

Frontend:

```bash
cd kasku-frontend
npm run check
npm run lint
npm run test:unit -- --run
npm run build
```

Go service, contoh `auth-service`:

```bash
cd kasku-backend/auth-service
make build
make test
make lint
make docker-build
```

Rust service, contoh `sync-service`:

```bash
cd kasku-backend/sync-service
make build
make test
make lint
make docker-build
```

Validasi Docker Compose:

```bash
cd kasku-backend
cp .env.example .env
docker compose config --quiet
```

## Testing

CI menjalankan:

- Lint dan test semua Go service dengan race detector.
- `cargo check`, `cargo clippy`, dan `cargo test` untuk `price-service` dan
  `sync-service`.
- `npm run check`, `npm run lint`, dan unit test frontend.
- Validasi `docker compose config`.
- Security scan dengan `govulncheck` dan `cargo audit`.

Smoke test tersedia di:

```bash
cd kasku-backend
bash tests/integration/smoke.sh
bash tests/integration/sync_smoke.sh
bash tests/integration/admin_smoke.sh
```

Catatan: beberapa smoke test membutuhkan stack lengkap yang sudah healthy,
akses Docker/PostgreSQL, serta tool seperti `curl`, `jq`, dan `uuidgen`.
`smoke.sh` default mengecek beberapa service lewat `localhost:8081` sampai
`localhost:8090`; expose port service tersebut atau override env URL jika
setup lokal hanya membuka api-gateway.

Load test k6 tersedia di `kasku-backend/tests/load/`.

## Observability

Stack observability tidak aktif secara default. Jalankan dengan profile:

```bash
cd kasku-backend
docker compose --profile observability up -d
```

UI lokal:

| Tool | URL |
| --- | --- |
| Grafana | `http://localhost:3000` |
| Prometheus | `http://localhost:9090` |
| Alertmanager | `http://localhost:9093` |
| Jaeger | `http://localhost:16686` |

Detail lengkap ada di `kasku-backend/OBSERVABILITY.md`.

## Deployment

Production deployment memakai GitHub Actions:

1. Workflow `CI` berjalan pada push ke `main` dan pull request.
2. Jika CI pada `main` sukses, workflow `Deploy to Production` berjalan di
   self-hosted runner.
3. Workflow build image Go/Rust/frontend, push ke GHCR, membuat `.env`
   production dari GitHub Secrets, lalu deploy dengan Docker Compose.

Checklist produksi ada di:

```text
kasku-backend/infra/DEPLOYMENT_CHECKLIST.md
```

## Environment Penting

Backend template: `kasku-backend/.env.example`.

Variabel yang biasanya wajib diperhatikan:

- `POSTGRES_SUPERUSER_PASS` dan `KASKU_*_DB_PASS`.
- `REDIS_PASSWORD`.
- `RABBITMQ_USER` dan `RABBITMQ_PASS`.
- `JWT_PRIVATE_KEY` dan `JWT_PUBLIC_KEY`.
- `ADMIN_JWT_SECRET`, `ADMIN_BOOTSTRAP_USERNAME`, dan
  `ADMIN_BOOTSTRAP_PASSWORD`.
- `KASKU_PAYMENT_ORCHESTRATOR_API_KEY`,
  `KASKU_PAYMENT_WEBHOOK_SECRET`, dan `KASKU_PAYMENT_CALLBACK_BASE_URL`.
- `SMTP_*` untuk notification-service.
- `APP_DOMAIN`, `KASKU_API_DOMAIN`, dan `CORS_ALLOWED_ORIGINS` untuk production.

Frontend template: `kasku-frontend/.env.example`.

- `PUBLIC_API_BASE_URL` adalah URL api-gateway dengan suffix `/v1`.

## Dokumentasi Tambahan

- `kasku-backend/OBSERVABILITY.md`
- `kasku-backend/infra/DEPLOYMENT_CHECKLIST.md`
- `kasku-backend/admin-service/README.md`
- `kasku-backend/sync-service/README.md`
- `kasku-backend/tests/load/README.md`

## License

Project ini menggunakan Apache License 2.0. Lihat `LICENSE`.
