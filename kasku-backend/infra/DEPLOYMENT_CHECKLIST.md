# KasKu Production Deployment Checklist

## 1. Server Prerequisites

```bash
# Di server produksi, jalankan satu kali:
sudo apt-get update && sudo apt-get install -y docker.io docker-compose-plugin

# Clone repo ke DEPLOY_PATH
git clone https://github.com/TubagusAldiMY/kasku.git /opt/kasku
# DEPLOY_PATH = /opt/kasku/kasku-backend

# Login GHCR sekali di server (gunakan PAT dengan scope read:packages)
echo "<PAT>" | docker login ghcr.io -u TubagusAldiMY --password-stdin
```

## 2. DNS Records

| Record | Type | Target |
|--------|------|--------|
| `app.kasku.id` | A | IP VPS |
| `api.kasku.id` | A | IP VPS |

## 3. GitHub Repository Secrets

Masuk ke **Settings → Secrets and variables → Actions → New repository secret**.

### Wajib (deploy gagal tanpa ini)

| Secret Name | Deskripsi | Cara Generate |
|-------------|-----------|---------------|
| `DEPLOY_HOST` | IP atau hostname VPS | — |
| `DEPLOY_USER` | SSH username di VPS | — |
| `DEPLOY_SSH_KEY` | Private key SSH (PEM format) | `ssh-keygen -t ed25519 -C "github-actions"` |
| `DEPLOY_PATH` | Path ke direktori kasku-backend di server | contoh: `/opt/kasku/kasku-backend` |
| `POSTGRES_SUPERUSER_PASS` | Password PostgreSQL superuser | `openssl rand -hex 32` |
| `KASKU_AUTH_DB_PASS` | Password DB auth-service | `openssl rand -hex 24` |
| `KASKU_BILLING_DB_PASS` | Password DB billing-service | `openssl rand -hex 24` |
| `KASKU_FINANCE_DB_PASS` | Password DB finance-service | `openssl rand -hex 24` |
| `KASKU_TRANSACTION_DB_PASS` | Password DB transaction-service | `openssl rand -hex 24` |
| `KASKU_INVESTMENT_DB_PASS` | Password DB investment-service | `openssl rand -hex 24` |
| `KASKU_SYNC_DB_PASS` | Password DB sync-service | `openssl rand -hex 24` |
| `KASKU_USER_DB_PASS` | Password DB user-service | `openssl rand -hex 24` |
| `KASKU_PRICE_DB_PASS` | Password DB price-service | `openssl rand -hex 24` |
| `KASKU_NOTIFICATION_DB_PASS` | Password DB notification-service | `openssl rand -hex 24` |
| `KASKU_ADMIN_DB_PASS` | Password DB admin-service | `openssl rand -hex 24` |
| `REDIS_PASSWORD` | Password Redis | `openssl rand -hex 32` |
| `RABBITMQ_PASS` | Password RabbitMQ | `openssl rand -hex 32` |
| `JWT_PRIVATE_KEY` | RSA-4096 private key (base64) | Lihat langkah di bawah |
| `JWT_PUBLIC_KEY` | RSA-4096 public key (base64) | Lihat langkah di bawah |
| `ADMIN_JWT_SECRET` | HS256 secret admin-service | `openssl rand -hex 32` |
| `ADMIN_BOOTSTRAP_PASSWORD` | Password admin pertama | Pilih password kuat |
| `KASKU_PAYMENT_ORCHESTRATOR_API_KEY` | API key dari portal payment orchestrator | Dari https://api-payment.roemahprogram.com |
| `KASKU_PAYMENT_WEBHOOK_SECRET` | Webhook secret dari portal orchestrator | Dari portal → Regenerate Secret |
| `KASKU_PAYMENT_CALLBACK_BASE_URL` | URL publik API backend | `https://api.kasku.id` |
| `ACME_EMAIL` | Email untuk Let's Encrypt | `admin@tubsamy.tech` |
| `APP_DOMAIN` | Domain frontend | `app.kasku.id` |

### Opsional

| Secret Name | Deskripsi | Default jika kosong |
|-------------|-----------|---------------------|
| `KASKU_API_DOMAIN` | Domain api-gateway | `api.{APP_DOMAIN}` |
| `INTERNAL_GRPC_SECRET` | Secret gRPC internal | kosong (warning di log) |
| `SMTP_HOST` | SMTP host | `smtp.gmail.com` |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USER` | SMTP username/email | kosong (email nonaktif) |
| `SMTP_PASS` | SMTP password / App Password | kosong |
| `COINGECKO_API_KEY` | CoinGecko API key | kosong (free tier tanpa key) |
| `GRAFANA_ADMIN_PASSWORD` | Password Grafana | sama dengan ADMIN_BOOTSTRAP_PASSWORD |
| `GHCR_TOKEN` | PAT GitHub dengan `read:packages` | skip docker login (pakai cached creds) |

## 4. Generate JWT RSA-4096 Key Pair

```bash
# Generate key pair
openssl genrsa -out private.pem 4096
openssl rsa -in private.pem -pubout -out public.pem

# Encode ke base64 single-line (copy output ke GitHub Secret)
base64 -w0 private.pem   # → JWT_PRIVATE_KEY
base64 -w0 public.pem    # → JWT_PUBLIC_KEY

# Hapus file setelah di-copy ke GitHub Secrets
rm private.pem public.pem
```

## 5. Konfigurasi Payment Orchestrator

1. Login ke https://api-payment.roemahprogram.com/portal
2. Buat partner account atau gunakan yang sudah ada
3. Copy **API Key** (sk_live_...) → set sebagai `KASKU_PAYMENT_ORCHESTRATOR_API_KEY`
4. Pergi ke **Webhook Settings** → klik **Regenerate Secret**
5. Copy webhook secret → set sebagai `KASKU_PAYMENT_WEBHOOK_SECRET`
6. Set **Callback URL**: `https://api.kasku.id/v1/billing/webhook/payment`

## 6. First Deploy

```bash
# Di server — inisialisasi database (hanya sekali)
cd /opt/kasku/kasku-backend
docker compose -f docker-compose.yml up -d postgres
# Tunggu postgres healthy, lalu:
docker compose -f docker-compose.yml up -d --remove-orphans
```

GitHub Actions akan otomatis deploy setiap push ke `main` yang lulus CI.

## 7. Verifikasi Post-Deploy

```bash
# Health check semua service
curl https://api.kasku.id/health

# Smoke test lengkap (dari server)
bash tests/integration/smoke.sh

# Cek log service tertentu
docker compose logs -f billing-service
```
