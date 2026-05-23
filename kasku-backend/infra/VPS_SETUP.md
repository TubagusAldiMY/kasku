# VPS Setup — Cloudflare Tunnel + Self-Hosted Runner

VPS tanpa public IP. Cloudflare Tunnel handle web traffic (TLS di edge), self-hosted GitHub Actions runner handle CI/CD (koneksi keluar dari VPS ke GitHub, tidak perlu port masuk).

---

## Arsitektur

```
Internet
  └── Cloudflare Edge (TLS termination)
        └── cloudflared daemon (berjalan di VPS)
              └── Traefik :80 (HTTP only, bind 127.0.0.1)
                    ├── api.kasku.id  → kasku-api-gateway :8080
                    └── app.kasku.id  → kasku-frontend :3000

GitHub Actions
  └── GitHub Runner Service (berjalan di VPS, koneksi keluar)
        └── docker compose up
```

---

## 1. Prasyarat VPS

```bash
# Docker Engine
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker

# Verifikasi
docker --version
docker compose version
```

---

## 2. Install cloudflared

```bash
# Download dan install
curl -L --output cloudflared.deb \
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb
cloudflared --version
```

---

## 3. Buat Cloudflare Tunnel

Jalankan dari VPS (butuh login browser sekali saja):

```bash
# Login — akan buka URL, buka di browser lokal kamu
cloudflared tunnel login

# Buat tunnel baru (nama bebas, mis. kasku-prod)
cloudflared tunnel create kasku-prod

# Catat TUNNEL_ID dari output di atas
# Contoh: Created tunnel kasku-prod with id abc12345-...
TUNNEL_ID="abc12345-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

---

## 4. Konfigurasi Tunnel

```bash
mkdir -p ~/.cloudflared

cat > ~/.cloudflared/config.yml << EOF
tunnel: ${TUNNEL_ID}
credentials-file: /home/${USER}/.cloudflared/${TUNNEL_ID}.json

ingress:
  - hostname: api.kasku.id
    service: http://127.0.0.1:80
    originRequest:
      httpHostHeader: api.kasku.id
  - hostname: app.kasku.id
    service: http://127.0.0.1:80
    originRequest:
      httpHostHeader: app.kasku.id
  - service: http_status:404
EOF
```

---

## 5. DNS Records di Cloudflare Dashboard

Masuk ke Cloudflare Dashboard → kasku.id → DNS:

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| CNAME | api | `${TUNNEL_ID}.cfargotunnel.com` | Proxied (orange) |
| CNAME | app | `${TUNNEL_ID}.cfargotunnel.com` | Proxied (orange) |

Atau pakai CLI:
```bash
cloudflared tunnel route dns kasku-prod api.kasku.id
cloudflared tunnel route dns kasku-prod app.kasku.id
```

---

## 6. Jalankan cloudflared sebagai Systemd Service

```bash
sudo cloudflared service install
sudo systemctl enable cloudflared
sudo systemctl start cloudflared

# Cek status
sudo systemctl status cloudflared
```

Jika perlu path config eksplisit:
```bash
sudo cloudflared --config /home/${USER}/.cloudflared/config.yml service install
```

---

## 7. Install Self-Hosted GitHub Actions Runner

Di GitHub repo → Settings → Actions → Runners → **New self-hosted runner** → pilih Linux x64 → ikuti perintah yang ditampilkan. Contoh:

```bash
mkdir ~/actions-runner && cd ~/actions-runner

# Download (ganti versi sesuai yang ditampilkan GitHub)
curl -o actions-runner-linux-x64.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.320.0/actions-runner-linux-x64-2.320.0.tar.gz
tar xzf ./actions-runner-linux-x64.tar.gz

# Configure — token dari halaman GitHub (hanya berlaku 1 jam)
./config.sh --url https://github.com/TubagusAldiMY/kasku \
  --token <TOKEN_DARI_GITHUB> \
  --name kasku-vps \
  --labels self-hosted,linux \
  --work /home/${USER}/runner-work \
  --unattended

# Install sebagai service systemd
sudo ./svc.sh install
sudo ./svc.sh start

# Cek status
sudo ./svc.sh status
```

---

## 8. GitHub Secrets yang Wajib Diset

Masuk ke GitHub → Settings → Secrets and variables → Actions → **New repository secret**:

### Secrets wajib (BARU / BERUBAH)

| Secret | Nilai |
|--------|-------|
| `DEPLOY_PATH` | `/home/tubsamy/kasku-deploy` |
| `APP_DOMAIN` | `app.kasku.id` |
| `KASKU_API_DOMAIN` | `api.kasku.id` |
| `KASKU_PAYMENT_CALLBACK_BASE_URL` | `https://api.kasku.id` |

> `ACME_EMAIL`, `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY` **tidak perlu lagi**.

### Secrets yang tetap wajib

```
POSTGRES_SUPERUSER_PASS
KASKU_AUTH_DB_PASS
KASKU_BILLING_DB_PASS
KASKU_FINANCE_DB_PASS
KASKU_TRANSACTION_DB_PASS
KASKU_INVESTMENT_DB_PASS
KASKU_SYNC_DB_PASS
KASKU_USER_DB_PASS
KASKU_PRICE_DB_PASS
KASKU_NOTIFICATION_DB_PASS
KASKU_ADMIN_DB_PASS
JWT_PRIVATE_KEY
JWT_PUBLIC_KEY
INTERNAL_GRPC_SECRET
RABBITMQ_PASS
REDIS_PASSWORD
ADMIN_JWT_SECRET
ADMIN_BOOTSTRAP_PASSWORD
KASKU_PAYMENT_ORCHESTRATOR_API_KEY
KASKU_PAYMENT_WEBHOOK_SECRET
GHCR_TOKEN  (Personal Access Token dengan scope packages:read)
```

### Opsional (ada fallback default)
```
SMTP_HOST / SMTP_PORT / SMTP_USER / SMTP_PASS
COINGECKO_API_KEY
GRAFANA_ADMIN_PASSWORD
KASKU_API_DOMAIN  (default: api.APP_DOMAIN)
```

---

## 9. Test Tunnel (sebelum push kode)

```bash
# Jalankan stack secara manual dari VPS untuk verifikasi tunnel
cd ~/kasku-deploy   # atau path mana saja yang ada docker-compose.yml
cp /path/to/.env .
docker compose -f docker-compose.yml up -d

# Cek Traefik berjalan di port 80
curl -H "Host: api.kasku.id" http://127.0.0.1:80/health

# Cek tunnel dari luar (di browser atau curl dari mesin lain)
curl https://api.kasku.id/health
```

---

## 10. Verifikasi Akhir

```bash
# Runner terhubung ke GitHub
sudo ./svc.sh status   # dari ~/actions-runner

# Tunnel aktif
sudo systemctl status cloudflared

# Docker berjalan
docker compose -f ~/kasku-deploy/docker-compose.yml ps
```

Push ke `main` → CI lulus → deploy otomatis berjalan di runner VPS → semua service healthy.
