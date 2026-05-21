#!/usr/bin/env bash
# KasKu PostgreSQL restore dari S3
# Usage: ./restore.sh <database> <s3-key>
# Contoh: ./restore.sh kasku_auth 2026/05/21/kasku_auth_20260521T020000Z.dump.gpg
set -euo pipefail

: "${1:?Usage: $0 <database> <s3-key>}"
: "${2:?Usage: $0 <database> <s3-key>}"

DB="${1}"
S3_KEY="${2}"

: "${POSTGRES_HOST:=postgres}"
: "${POSTGRES_PORT:=5432}"
: "${POSTGRES_SUPERUSER:=kasku_superuser}"
: "${KASKU_SUPERUSER_PASS:?KASKU_SUPERUSER_PASS wajib diset}"
: "${BACKUP_BUCKET:?BACKUP_BUCKET wajib diset}"
: "${BACKUP_PASSPHRASE:?BACKUP_PASSPHRASE wajib diset}"
: "${AWS_ACCESS_KEY_ID:?AWS_ACCESS_KEY_ID wajib diset}"
: "${AWS_SECRET_ACCESS_KEY:?AWS_SECRET_ACCESS_KEY wajib diset}"
: "${AWS_DEFAULT_REGION:=ap-southeast-1}"

TMP_DIR=$(mktemp -d)
ENCRYPTED_FILE="${TMP_DIR}/restore.dump.gpg"
DUMP_FILE="${TMP_DIR}/restore.dump"

export PGPASSWORD="${KASKU_SUPERUSER_PASS}"

log() { echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] $*"; }

cleanup() {
    rm -rf "${TMP_DIR}"
    log "Temp files dibersihkan"
}
trap cleanup EXIT

log "=== Restore ${DB} dari s3://${BACKUP_BUCKET}/${S3_KEY} ==="

# Download dari S3
log "Download backup dari S3..."
aws s3 cp "s3://${BACKUP_BUCKET}/${S3_KEY}" "${ENCRYPTED_FILE}"

# Dekripsi
log "Dekripsi backup..."
gpg \
    --batch \
    --yes \
    --decrypt \
    --passphrase "${BACKUP_PASSPHRASE}" \
    --output "${DUMP_FILE}" \
    "${ENCRYPTED_FILE}"

# Konfirmasi sebelum restore ke production
if [[ "${POSTGRES_HOST}" != "localhost" && "${POSTGRES_HOST}" != "127.0.0.1" ]]; then
    echo ""
    echo "PERINGATAN: Anda akan merestore ke host '${POSTGRES_HOST}'."
    echo "Database '${DB}' akan DROP dan dibuat ulang."
    read -rp "Ketik nama database untuk konfirmasi: " CONFIRM
    if [[ "${CONFIRM}" != "${DB}" ]]; then
        log "Restore dibatalkan"
        exit 1
    fi
fi

# Restore
log "Restore ke ${DB}..."
pg_restore \
    -h "${POSTGRES_HOST}" \
    -p "${POSTGRES_PORT}" \
    -U "${POSTGRES_SUPERUSER}" \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    -d "${DB}" \
    "${DUMP_FILE}"

log "=== Restore ${DB} selesai ==="
