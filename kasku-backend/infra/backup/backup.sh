#!/usr/bin/env bash
# KasKu PostgreSQL backup — pg_dump + GPG AES256 + S3 upload
# Dipanggil via: docker compose --profile backup run --rm backup
# Cron production: 0 2 * * * docker compose --profile backup run --rm backup
set -euo pipefail

: "${POSTGRES_HOST:=postgres}"
: "${POSTGRES_PORT:=5432}"
: "${POSTGRES_SUPERUSER:=kasku_superuser}"
: "${KASKU_SUPERUSER_PASS:?KASKU_SUPERUSER_PASS wajib diset}"
: "${BACKUP_BUCKET:?BACKUP_BUCKET wajib diset}"
: "${BACKUP_PASSPHRASE:?BACKUP_PASSPHRASE wajib diset}"
: "${AWS_ACCESS_KEY_ID:?AWS_ACCESS_KEY_ID wajib diset}"
: "${AWS_SECRET_ACCESS_KEY:?AWS_SECRET_ACCESS_KEY wajib diset}"
: "${AWS_DEFAULT_REGION:=ap-southeast-1}"

DATABASES=(
    kasku_auth
    kasku_billing
    kasku_finance
    kasku_price
    kasku_admin
)

TIMESTAMP=$(date -u +"%Y%m%dT%H%M%SZ")
DATE_PATH=$(date -u +"%Y/%m/%d")
TMP_DIR=$(mktemp -d)
FAILED=0

export PGPASSWORD="${KASKU_SUPERUSER_PASS}"

log() { echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] $*"; }

cleanup() {
    rm -rf "${TMP_DIR}"
    log "Temp files dibersihkan"
}
trap cleanup EXIT

log "=== KasKu backup dimulai: ${TIMESTAMP} ==="

for DB in "${DATABASES[@]}"; do
    log "Backup database: ${DB}"

    DUMP_FILE="${TMP_DIR}/${DB}_${TIMESTAMP}.dump"
    ENCRYPTED_FILE="${DUMP_FILE}.gpg"
    S3_KEY="${DATE_PATH}/${DB}_${TIMESTAMP}.dump.gpg"

    # pg_dump format custom (-Fc) untuk kompresi dan selective restore
    if ! pg_dump \
        -h "${POSTGRES_HOST}" \
        -p "${POSTGRES_PORT}" \
        -U "${POSTGRES_SUPERUSER}" \
        -Fc \
        -f "${DUMP_FILE}" \
        "${DB}"; then
        log "ERROR: Gagal backup ${DB}"
        FAILED=$((FAILED + 1))
        continue
    fi

    DUMP_SIZE=$(du -sh "${DUMP_FILE}" | cut -f1)
    log "${DB}: dump selesai (${DUMP_SIZE})"

    # Enkripsi dengan GPG symmetric AES256
    if ! gpg \
        --batch \
        --yes \
        --symmetric \
        --cipher-algo AES256 \
        --passphrase "${BACKUP_PASSPHRASE}" \
        --output "${ENCRYPTED_FILE}" \
        "${DUMP_FILE}"; then
        log "ERROR: Gagal enkripsi ${DB}"
        FAILED=$((FAILED + 1))
        continue
    fi

    # Upload ke S3
    if ! aws s3 cp \
        "${ENCRYPTED_FILE}" \
        "s3://${BACKUP_BUCKET}/${S3_KEY}" \
        --storage-class STANDARD_IA; then
        log "ERROR: Gagal upload ${DB} ke S3"
        FAILED=$((FAILED + 1))
        continue
    fi

    log "${DB}: berhasil diupload ke s3://${BACKUP_BUCKET}/${S3_KEY}"
    rm -f "${DUMP_FILE}" "${ENCRYPTED_FILE}"
done

log "=== Backup selesai: ${FAILED} gagal dari ${#DATABASES[@]} database ==="

# Exit non-zero jika ada yang gagal (supaya cron job bisa alert)
exit "${FAILED}"
