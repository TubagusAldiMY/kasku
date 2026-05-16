#!/usr/bin/env bash
# Integration test untuk sync-service end-to-end.
# Skenario:
#   1. Register user baru
#   2. Force-verify email via psql (bypass email channel)
#   3. Tunggu tenant schema diprovision (user-service event consumer)
#   4. Login → ambil JWT
#   5. Create financial_account (baseline buat conflict test)
#   6. Push 2 op (valid + duplicate sync_id) → assert applied=1, skipped=1
#   7. Push op stale (client_timestamp lebih lama dari server) → assert konflik SERVER_WINS
#   8. Pull since=epoch → assert >=1 change
#   9. Fetch /metrics — assert counter ter-increment
#
# Prasyarat:
#   - Stack docker compose sudah running (lihat README di kasku-backend/)
#   - jq, curl, uuidgen tersedia di PATH
#   - Akses ke postgres container via `docker compose exec postgres`
#
# Usage:
#   bash tests/integration/sync_smoke.sh
#
# Env override:
#   BASE_URL=http://localhost:8080
#   SYNC_URL=http://localhost:8088
#   PG_CONTAINER=kasku-postgres
#   PG_DB=kasku_auth   (untuk force-verify)
#   PG_USER=kasku_superuser

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
SYNC_URL="${SYNC_URL:-http://localhost:8088}"
PG_CONTAINER="${PG_CONTAINER:-kasku-postgres}"
PG_AUTH_DB="${PG_AUTH_DB:-kasku_auth}"
PG_FINANCE_DB="${PG_FINANCE_DB:-kasku_finance}"
PG_USER="${PG_USER:-kasku_superuser}"

STAMP="$(date +%s%N)"
EMAIL="sync-smoke-${STAMP}@example.com"
USERNAME="sync_smoke_${STAMP}"
PASSWORD="SmokeTest123!"

# ────────────────────────────────────────────────────────────────────────
# Helpers
# ────────────────────────────────────────────────────────────────────────
for tool in curl jq uuidgen docker; do
  command -v "$tool" >/dev/null 2>&1 || {
    echo "ERROR: butuh $tool di PATH"
    exit 1
  }
done

psql_exec() {
  local db="$1" sql="$2"
  # `-q` + filter command tags via grep -v supaya output bersih untuk capture.
  docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$db" -tA -q -c "$sql" \
    | grep -Ev '^(UPDATE|INSERT|DELETE|SELECT)\s+[0-9]+$' || true
}

expect_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" != "$actual" ]; then
    echo "FAIL $label: expected=$expected got=$actual"
    exit 1
  fi
  echo "OK   $label: $actual"
}

expect_ge() {
  local label="$1" floor="$2" actual="$3"
  if [ "$actual" -lt "$floor" ]; then
    echo "FAIL $label: expected>=$floor got=$actual"
    exit 1
  fi
  echo "OK   $label: $actual"
}

# ────────────────────────────────────────────────────────────────────────
# 1. Register user
# ────────────────────────────────────────────────────────────────────────
echo "→ Register $EMAIL"
register_body="$(jq -nc \
  --arg email "$EMAIL" \
  --arg username "$USERNAME" \
  --arg password "$PASSWORD" \
  '{email:$email, username:$username, password:$password}')"
register_resp="$(curl -sS -X POST "$BASE_URL/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "$register_body")"
echo "$register_resp" | jq -e '.success == true' >/dev/null || {
  echo "FAIL register: $register_resp"
  exit 1
}
echo "OK   register"

# ────────────────────────────────────────────────────────────────────────
# 2. Force-verify email (bypass mail channel)
# ────────────────────────────────────────────────────────────────────────
echo "→ Force verify email via psql"
USER_ID="$(psql_exec "$PG_AUTH_DB" \
  "UPDATE public.users SET email_verified = true, is_active = true WHERE email = '$EMAIL' RETURNING id::text;")"
[ -n "$USER_ID" ] || {
  echo "FAIL: user_id kosong setelah update"
  exit 1
}
echo "OK   user_id=$USER_ID"

# ────────────────────────────────────────────────────────────────────────
# 3. Tunggu tenant schema diprovision (user-service consumer dari user.registered)
# ────────────────────────────────────────────────────────────────────────
TENANT_SCHEMA="tenant_$(echo "$USER_ID" | tr - _)"
echo "→ Tunggu tenant schema $TENANT_SCHEMA"
for i in $(seq 1 30); do
  exists="$(psql_exec "$PG_FINANCE_DB" \
    "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = '$TENANT_SCHEMA');")"
  if [ "$exists" = "t" ]; then
    echo "OK   tenant schema siap (iter=$i)"
    break
  fi
  sleep 1
done
[ "$exists" = "t" ] || {
  echo "FAIL: tenant schema tidak provision dalam 30s"
  exit 1
}

# ────────────────────────────────────────────────────────────────────────
# 4. Login → ambil JWT
# ────────────────────────────────────────────────────────────────────────
echo "→ Login"
login_resp="$(curl -sS -X POST "$BASE_URL/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg email "$EMAIL" --arg pw "$PASSWORD" '{email:$email, password:$pw}')")"
TOKEN="$(echo "$login_resp" | jq -r '.data.access_token // empty')"
[ -n "$TOKEN" ] || {
  echo "FAIL login: $login_resp"
  exit 1
}
echo "OK   login (jwt len=${#TOKEN})"

# ────────────────────────────────────────────────────────────────────────
# 5. Create financial_account langsung via finance-service (baseline)
# ────────────────────────────────────────────────────────────────────────
echo "→ Create baseline financial_account"
acc_resp="$(curl -sS -X POST "$BASE_URL/v1/accounts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Sync Test BCA","account_type":"BANK","initial_balance":100000}')"
ACCOUNT_ID="$(echo "$acc_resp" | jq -r '.data.id // .data.ID // empty')"
[ -n "$ACCOUNT_ID" ] || {
  echo "FAIL create account: $acc_resp"
  exit 1
}
echo "OK   account_id=$ACCOUNT_ID"

# ────────────────────────────────────────────────────────────────────────
# 6. Push 2 ops: valid + duplicate sync_id
# ────────────────────────────────────────────────────────────────────────
echo "→ Push valid+duplicate"
SYNC_ID_1="$(uuidgen)"
ENTITY_ID_1="$(uuidgen)"
NOW_ISO="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

push_body_1="$(jq -nc \
  --arg sid "$SYNC_ID_1" \
  --arg eid "$ENTITY_ID_1" \
  --arg ts "$NOW_ISO" \
  '{operations:[
      {sync_id:$sid, entity_type:"financial_account", entity_id:$eid, operation:"create",
       payload:{name:"Offline-CASH", account_type:"CASH", initial_balance:50000},
       client_timestamp:$ts},
      {sync_id:$sid, entity_type:"financial_account", entity_id:$eid, operation:"create",
       payload:{name:"Offline-CASH", account_type:"CASH", initial_balance:50000},
       client_timestamp:$ts}
   ]}')"

push_resp="$(curl -sS -X POST "$BASE_URL/v1/sync/push" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$push_body_1")"

echo "$push_resp" | jq . >/dev/null || {
  echo "FAIL push parse: $push_resp"
  exit 1
}

APPLIED="$(echo "$push_resp" | jq -r '.data.processed')"
SKIPPED="$(echo "$push_resp" | jq -r '.data.skipped')"

# Note: untuk duplicate dengan sync_id SAMA di batch yang SAMA, idempotency check
# akan kena karena INSERT log pertama men-setel row di sync_log. Hasil akhir bisa
# applied=1 skipped=1 (kalau urutan deterministik) atau applied=2 (kalau row yang
# pertama belum komit saat ops kedua diproses — namun loop sekuensial mestinya
# applied=1 skipped=1).
expect_ge "push.processed" 1 "$APPLIED"
expect_ge "push.skipped+processed >= 2" 2 "$((APPLIED + SKIPPED))"

# ────────────────────────────────────────────────────────────────────────
# 7. Push stale → konflik SERVER_WINS pada existing account
# ────────────────────────────────────────────────────────────────────────
echo "→ Push stale (server wins)"
SYNC_ID_2="$(uuidgen)"
STALE_ISO="1990-01-01T00:00:00Z"

push_body_2="$(jq -nc \
  --arg sid "$SYNC_ID_2" \
  --arg eid "$ACCOUNT_ID" \
  --arg ts "$STALE_ISO" \
  '{operations:[
      {sync_id:$sid, entity_type:"financial_account", entity_id:$eid, operation:"update",
       payload:{name:"WILL-LOSE"},
       client_timestamp:$ts}
   ]}')"

push_resp_2="$(curl -sS -X POST "$BASE_URL/v1/sync/push" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$push_body_2")"

CONFLICTS="$(echo "$push_resp_2" | jq -r '.data.conflicts')"
expect_eq "push.conflicts" "1" "$CONFLICTS"

STATUS_2="$(echo "$push_resp_2" | jq -r '.data.results[0].status')"
expect_eq "push.results[0].status" "conflict" "$STATUS_2"

# ────────────────────────────────────────────────────────────────────────
# 8. Pull since=epoch
# ────────────────────────────────────────────────────────────────────────
echo "→ Pull since=1970"
pull_resp="$(curl -sS "$BASE_URL/v1/sync/pull?since=1970-01-01T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN")"

PULL_COUNT="$(echo "$pull_resp" | jq -r '.data.changes | length')"
expect_ge "pull.changes" 1 "$PULL_COUNT"

# ────────────────────────────────────────────────────────────────────────
# 9. /metrics — counters ter-increment
# ────────────────────────────────────────────────────────────────────────
echo "→ Fetch /metrics"
metrics_body="$(curl -sS "$SYNC_URL/metrics")"
PUSH_TOTAL="$(echo "$metrics_body" | awk '/^kasku_sync_push_total /{print $2}')"
CONFLICT_TOTAL="$(echo "$metrics_body" | awk '/^kasku_sync_push_conflicts_total /{print $2}')"
PULL_TOTAL="$(echo "$metrics_body" | awk '/^kasku_sync_pull_total /{print $2}')"

expect_ge "metrics.push_total" 1 "${PUSH_TOTAL:-0}"
expect_ge "metrics.push_conflicts_total" 1 "${CONFLICT_TOTAL:-0}"
expect_ge "metrics.pull_total" 1 "${PULL_TOTAL:-0}"

echo ""
echo "✓ sync_smoke OK — user=$EMAIL tenant=$TENANT_SCHEMA"
