#!/usr/bin/env bash
# =============================================================================
# generate_secrets.sh — Generator secrets untuk GitHub Actions
#
# Jalankan SEKALI di mesin lokal:
#   bash infra/generate_secrets.sh
#
# Output: tabel siap copy-paste ke GitHub Secrets
# =============================================================================
set -euo pipefail

# Warna
RED='\033[0;31m'; YELLOW='\033[1;33m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'
BOLD='\033[1m'; RESET='\033[0m'

hr() { printf '%s\n' "$(printf '─%.0s' {1..72})"; }

# Pastikan openssl tersedia
if ! command -v openssl &>/dev/null; then
  echo -e "${RED}ERROR: openssl tidak ditemukan. Install dulu: apt install openssl${RESET}"
  exit 1
fi

echo -e "${BOLD}${CYAN}"
hr
echo "  KasKu — GitHub Secrets Generator"
hr
echo -e "${RESET}"

# ─────────────────────────────────────────────────────────────────────────────
# 1. Password acak (32 char hex)
# ─────────────────────────────────────────────────────────────────────────────
gen_pass() { openssl rand -hex 24; }

POSTGRES_SUPERUSER_PASS=$(gen_pass)
KASKU_AUTH_DB_PASS=$(gen_pass)
KASKU_BILLING_DB_PASS=$(gen_pass)
KASKU_FINANCE_DB_PASS=$(gen_pass)
KASKU_TRANSACTION_DB_PASS=$(gen_pass)
KASKU_INVESTMENT_DB_PASS=$(gen_pass)
KASKU_SYNC_DB_PASS=$(gen_pass)
KASKU_USER_DB_PASS=$(gen_pass)
KASKU_PRICE_DB_PASS=$(gen_pass)
KASKU_NOTIFICATION_DB_PASS=$(gen_pass)
KASKU_ADMIN_DB_PASS=$(gen_pass)
RABBITMQ_PASS=$(gen_pass)
REDIS_PASSWORD=$(gen_pass)
ADMIN_BOOTSTRAP_PASSWORD=$(gen_pass)
INTERNAL_GRPC_SECRET=$(openssl rand -hex 32)

# ─────────────────────────────────────────────────────────────────────────────
# 2. ADMIN_JWT_SECRET — HS256, harus >= 64 char
# ─────────────────────────────────────────────────────────────────────────────
ADMIN_JWT_SECRET=$(openssl rand -hex 48)

# ─────────────────────────────────────────────────────────────────────────────
# 3. JWT RSA-4096 — base64 single-line (karena .env tidak support multiline)
# ─────────────────────────────────────────────────────────────────────────────
echo -e "${YELLOW}⏳ Generating RSA-4096 key pair (butuh 3-5 detik)...${RESET}"

TMPDIR_JWT=$(mktemp -d)
openssl genrsa -out "${TMPDIR_JWT}/private.pem" 4096 2>/dev/null
openssl rsa -in "${TMPDIR_JWT}/private.pem" -pubout -out "${TMPDIR_JWT}/public.pem" 2>/dev/null

# Base64 single-line (no newline wrapping) — format yang dipakai deploy.yml
JWT_PRIVATE_KEY=$(base64 -w 0 < "${TMPDIR_JWT}/private.pem")
JWT_PUBLIC_KEY=$(base64 -w 0 < "${TMPDIR_JWT}/public.pem")

rm -rf "${TMPDIR_JWT}"

# ─────────────────────────────────────────────────────────────────────────────
# Simpan ke file lokal (untuk backup — JANGAN dicommit!)
# ─────────────────────────────────────────────────────────────────────────────
OUTFILE="infra/.secrets_generated.env"
cat > "${OUTFILE}" << EOF
# Auto-generated — JANGAN COMMIT file ini!
# Simpan di tempat aman (password manager / vault)
# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

DEPLOY_PATH=/home/tubsamy/kasku-deploy
APP_DOMAIN=app.kasku.id
KASKU_API_DOMAIN=api.kasku.id
KASKU_PAYMENT_CALLBACK_BASE_URL=https://api.kasku.id

POSTGRES_SUPERUSER_PASS=${POSTGRES_SUPERUSER_PASS}
KASKU_AUTH_DB_PASS=${KASKU_AUTH_DB_PASS}
KASKU_BILLING_DB_PASS=${KASKU_BILLING_DB_PASS}
KASKU_FINANCE_DB_PASS=${KASKU_FINANCE_DB_PASS}
KASKU_TRANSACTION_DB_PASS=${KASKU_TRANSACTION_DB_PASS}
KASKU_INVESTMENT_DB_PASS=${KASKU_INVESTMENT_DB_PASS}
KASKU_SYNC_DB_PASS=${KASKU_SYNC_DB_PASS}
KASKU_USER_DB_PASS=${KASKU_USER_DB_PASS}
KASKU_PRICE_DB_PASS=${KASKU_PRICE_DB_PASS}
KASKU_NOTIFICATION_DB_PASS=${KASKU_NOTIFICATION_DB_PASS}
KASKU_ADMIN_DB_PASS=${KASKU_ADMIN_DB_PASS}
RABBITMQ_PASS=${RABBITMQ_PASS}
REDIS_PASSWORD=${REDIS_PASSWORD}
ADMIN_JWT_SECRET=${ADMIN_JWT_SECRET}
ADMIN_BOOTSTRAP_PASSWORD=${ADMIN_BOOTSTRAP_PASSWORD}
INTERNAL_GRPC_SECRET=${INTERNAL_GRPC_SECRET}
JWT_PRIVATE_KEY=${JWT_PRIVATE_KEY}
JWT_PUBLIC_KEY=${JWT_PUBLIC_KEY}

# --- ISI MANUAL ---
KASKU_PAYMENT_ORCHESTRATOR_API_KEY=
KASKU_PAYMENT_WEBHOOK_SECRET=
GHCR_TOKEN=
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
EOF

chmod 600 "${OUTFILE}"

# ─────────────────────────────────────────────────────────────────────────────
# Output: tabel lengkap
# ─────────────────────────────────────────────────────────────────────────────
echo -e "${BOLD}${GREEN}"
hr
echo "  BAGIAN A — Nilai tetap (copy-paste apa adanya)"
hr
echo -e "${RESET}"
printf "%-42s  %s\n" "Secret Name" "Value"
printf "%-42s  %s\n" "─────────────────────────────────────────" "──────────────────────────────────────────────"
printf "%-42s  %s\n" "DEPLOY_PATH"                   "/home/tubsamy/kasku-deploy"
printf "%-42s  %s\n" "APP_DOMAIN"                    "app.kasku.id"
printf "%-42s  %s\n" "KASKU_API_DOMAIN"              "api.kasku.id"
printf "%-42s  %s\n" "KASKU_PAYMENT_CALLBACK_BASE_URL" "https://api.kasku.id"

echo ""
echo -e "${BOLD}${GREEN}"
hr
echo "  BAGIAN B — Password yang sudah di-generate"
hr
echo -e "${RESET}"
printf "%-42s  %s\n" "Secret Name" "Value"
printf "%-42s  %s\n" "─────────────────────────────────────────" "──────────────────────────────────────────────"
printf "%-42s  %s\n" "POSTGRES_SUPERUSER_PASS"       "${POSTGRES_SUPERUSER_PASS}"
printf "%-42s  %s\n" "KASKU_AUTH_DB_PASS"            "${KASKU_AUTH_DB_PASS}"
printf "%-42s  %s\n" "KASKU_BILLING_DB_PASS"         "${KASKU_BILLING_DB_PASS}"
printf "%-42s  %s\n" "KASKU_FINANCE_DB_PASS"         "${KASKU_FINANCE_DB_PASS}"
printf "%-42s  %s\n" "KASKU_TRANSACTION_DB_PASS"     "${KASKU_TRANSACTION_DB_PASS}"
printf "%-42s  %s\n" "KASKU_INVESTMENT_DB_PASS"      "${KASKU_INVESTMENT_DB_PASS}"
printf "%-42s  %s\n" "KASKU_SYNC_DB_PASS"            "${KASKU_SYNC_DB_PASS}"
printf "%-42s  %s\n" "KASKU_USER_DB_PASS"            "${KASKU_USER_DB_PASS}"
printf "%-42s  %s\n" "KASKU_PRICE_DB_PASS"           "${KASKU_PRICE_DB_PASS}"
printf "%-42s  %s\n" "KASKU_NOTIFICATION_DB_PASS"    "${KASKU_NOTIFICATION_DB_PASS}"
printf "%-42s  %s\n" "KASKU_ADMIN_DB_PASS"           "${KASKU_ADMIN_DB_PASS}"
printf "%-42s  %s\n" "RABBITMQ_PASS"                 "${RABBITMQ_PASS}"
printf "%-42s  %s\n" "REDIS_PASSWORD"                "${REDIS_PASSWORD}"
printf "%-42s  %s\n" "ADMIN_JWT_SECRET"              "${ADMIN_JWT_SECRET}"
printf "%-42s  %s\n" "ADMIN_BOOTSTRAP_PASSWORD"      "${ADMIN_BOOTSTRAP_PASSWORD}"
printf "%-42s  %s\n" "INTERNAL_GRPC_SECRET"          "${INTERNAL_GRPC_SECRET}"

echo ""
echo -e "${BOLD}${GREEN}"
hr
echo "  BAGIAN C — JWT RSA-4096 Key Pair (panjang, paste seluruhnya)"
hr
echo -e "${RESET}"
echo -e "${BOLD}JWT_PRIVATE_KEY${RESET} (paste seluruh string berikut ke GitHub Secret):"
echo "${JWT_PRIVATE_KEY}"
echo ""
echo -e "${BOLD}JWT_PUBLIC_KEY${RESET} (paste seluruh string berikut ke GitHub Secret):"
echo "${JWT_PUBLIC_KEY}"

echo ""
echo -e "${BOLD}${YELLOW}"
hr
echo "  BAGIAN D — Isi Manual (kamu yang tahu nilainya)"
hr
echo -e "${RESET}"
printf "%-42s  %s\n" "Secret Name" "Cara dapat nilainya"
printf "%-42s  %s\n" "─────────────────────────────────────────" "─────────────────────────────────────────────────────"
printf "%-42s  %s\n" "KASKU_PAYMENT_ORCHESTRATOR_API_KEY" "Dashboard payment orchestrator (api-payment.roemahprogram.com)"
printf "%-42s  %s\n" "KASKU_PAYMENT_WEBHOOK_SECRET"  "Dashboard payment orchestrator — webhook secret"
printf "%-42s  %s\n" "GHCR_TOKEN"                    "GitHub → Settings → Developer settings → PAT → New token → scope: write:packages"
printf "%-42s  %s\n" "SMTP_USER"                     "Email pengirim notifikasi (mis. noreply@kasku.id)"
printf "%-42s  %s\n" "SMTP_PASS"                     "App password Gmail atau SMTP provider"

echo ""
echo -e "${BOLD}${GREEN}"
hr
echo "  BAGIAN E — Cara input ke GitHub"
hr
echo -e "${RESET}"
cat << 'HOWTO'
1. Buka: https://github.com/TubagusAldiMY/kasku/settings/secrets/actions
2. Klik "New repository secret"
3. Name  → isi nama secret (kolom kiri tabel di atas)
4. Secret → isi nilai (kolom kanan)
5. Klik "Add secret"
6. Ulangi untuk semua secret

TOTAL: 26 secrets wajib + 4 opsional (SMTP_* dan COINGECKO_API_KEY)
HOWTO

echo -e "${BOLD}${CYAN}"
hr
echo -e "  ✅  Backup tersimpan di: ${OUTFILE}  (jangan dicommit!)"
hr
echo -e "${RESET}"