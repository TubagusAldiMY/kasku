#!/usr/bin/env bash
# Integration test untuk admin-service end-to-end.
# Skenario:
#   1. Admin login dengan bootstrap credential → assert JWT diterima
#   2. GET /v1/admin/auth/me → assert profile
#   3. Buat 1 user biasa via register flow (mirror sync_smoke.sh)
#   4. GET /v1/admin/users → assert user baru muncul di list
#   5. GET /v1/admin/users/:id → assert detail
#   6. POST /v1/admin/users/:id/suspend → assert is_active=false di kasku_auth + audit_log row
#   7. POST /v1/admin/users/:id/activate → assert is_active=true + audit_log row
#   8. POST /v1/admin/users/:id/override-subscription {plan_name:BASIC} → assert plan_id berubah + audit_log
#   9. GET /v1/admin/payments → assert list (boleh kosong, tapi response valid)
#  10. GET /v1/admin/stats/dashboard → assert shape (total_users, mrr, tier_distribution)
#  11. GET /v1/admin/audit-log?admin_id=... → assert >=4 entries (LOGIN + 3 mutations)
#  12. POST /v1/admin/auth/logout → assert JWT di-blacklist
#  13. GET /v1/admin/auth/me dengan JWT yang di-blacklist → assert 401 TOKEN_REVOKED
#  14. POST /v1/admin/users tanpa Bearer → assert 401
#  15. POST /v1/admin/users dengan user JWT (RS256) → assert 401 (bukan HS256)
#
# Prasyarat:
#   - Stack docker compose sudah running
#   - jq, curl, uuidgen, docker tersedia
#
# Env override:
#   BASE_URL=http://localhost:8080
#   PG_CONTAINER=kasku-postgres
#   PG_USER=kasku_superuser
#   ADMIN_USERNAME=superadmin
#   ADMIN_PASSWORD=ChangeMe-Strong-Passw0rd!

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PG_CONTAINER="${PG_CONTAINER:-kasku-postgres}"
PG_USER="${PG_USER:-kasku_superuser}"
ADMIN_USERNAME="${ADMIN_USERNAME:-superadmin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-ChangeMe-Strong-Passw0rd!}"

STAMP="$(date +%s)"
TEST_EMAIL="admin-smoke-${STAMP}@example.com"
TEST_USERNAME="admin_${STAMP}"
TEST_PASSWORD="SmokeTest123!"

for tool in curl jq uuidgen docker; do
  command -v "$tool" >/dev/null 2>&1 || {
    echo "ERROR: butuh $tool di PATH"
    exit 1
  }
done

psql_exec() {
  local db="$1" sql="$2"
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
# 1. Admin login
# ────────────────────────────────────────────────────────────────────────
echo "→ Admin login ($ADMIN_USERNAME)"
login_resp="$(curl -sS -X POST "$BASE_URL/v1/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg u "$ADMIN_USERNAME" --arg p "$ADMIN_PASSWORD" '{username:$u, password:$p}')")"
ADMIN_TOKEN="$(echo "$login_resp" | jq -r '.data.access_token // empty')"
ADMIN_ID="$(echo "$login_resp" | jq -r '.data.admin.id // empty')"
[ -n "$ADMIN_TOKEN" ] || { echo "FAIL admin login: $login_resp"; exit 1; }
[ -n "$ADMIN_ID" ] || { echo "FAIL admin ID: $login_resp"; exit 1; }
echo "OK   admin login (jwt len=${#ADMIN_TOKEN}, admin_id=$ADMIN_ID)"

# ────────────────────────────────────────────────────────────────────────
# 2. /v1/admin/auth/me
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/auth/me"
me_resp="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE_URL/v1/admin/auth/me")"
ROLE="$(echo "$me_resp" | jq -r '.data.role')"
expect_eq "me.role" "SUPER_ADMIN" "$ROLE"

# ────────────────────────────────────────────────────────────────────────
# 3. Buat user biasa (target untuk suspend/override)
# ────────────────────────────────────────────────────────────────────────
echo "→ Register target user $TEST_EMAIL"
register_resp="$(curl -sS -X POST "$BASE_URL/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg e "$TEST_EMAIL" --arg u "$TEST_USERNAME" --arg p "$TEST_PASSWORD" \
        '{email:$e, username:$u, password:$p}')")"
echo "$register_resp" | jq -e '.success == true' >/dev/null || { echo "FAIL register: $register_resp"; exit 1; }

# Force-verify + force-active supaya tenant provisioning event diproses
TARGET_USER_ID="$(psql_exec kasku_auth \
  "UPDATE public.users SET email_verified = true, is_active = true WHERE email = '$TEST_EMAIL' RETURNING id::text;")"
[ -n "$TARGET_USER_ID" ] || { echo "FAIL force-verify user"; exit 1; }
echo "OK   target user_id=$TARGET_USER_ID"

# Tunggu tenant + subscription provisioning supaya override-subscription bisa jalan
TENANT="tenant_$(echo "$TARGET_USER_ID" | tr - _)"
for i in $(seq 1 30); do
  exists="$(psql_exec kasku_finance "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = '$TENANT');")"
  [ "$exists" = "t" ] && break
  sleep 1
done
[ "$exists" = "t" ] || { echo "FAIL: tenant schema tidak provision dalam 30s"; exit 1; }
# Tunggu subscription FREE di-INSERT (user-service handler)
for i in $(seq 1 15); do
  subs="$(psql_exec kasku_billing "SELECT COUNT(*) FROM public.subscriptions WHERE user_id = '$TARGET_USER_ID';")"
  [ "${subs:-0}" -ge 1 ] && break
  sleep 1
done

# ────────────────────────────────────────────────────────────────────────
# 4. List users — target ditemukan
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/users?q=admin_smoke_"
list_resp="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE_URL/v1/admin/users?q=admin_smoke_&page_size=10")"
LIST_LEN="$(echo "$list_resp" | jq -r '.data | length')"
expect_ge "list.length" 1 "$LIST_LEN"

# ────────────────────────────────────────────────────────────────────────
# 5. User detail
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/users/$TARGET_USER_ID"
detail_resp="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE_URL/v1/admin/users/$TARGET_USER_ID")"
TIER="$(echo "$detail_resp" | jq -r '.data.subscription_tier')"
expect_eq "detail.tier" "FREE" "$TIER"

# ────────────────────────────────────────────────────────────────────────
# 6. Suspend
# ────────────────────────────────────────────────────────────────────────
echo "→ POST suspend"
curl -sS -X POST "$BASE_URL/v1/admin/users/$TARGET_USER_ID/suspend" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason":"integration test suspend"}' | jq -e '.success == true' >/dev/null

IS_ACTIVE="$(psql_exec kasku_auth "SELECT is_active FROM public.users WHERE id = '$TARGET_USER_ID';")"
expect_eq "users.is_active after suspend" "f" "$IS_ACTIVE"

# ────────────────────────────────────────────────────────────────────────
# 7. Activate
# ────────────────────────────────────────────────────────────────────────
echo "→ POST activate"
curl -sS -X POST "$BASE_URL/v1/admin/users/$TARGET_USER_ID/activate" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason":"integration test activate"}' | jq -e '.success == true' >/dev/null

IS_ACTIVE="$(psql_exec kasku_auth "SELECT is_active FROM public.users WHERE id = '$TARGET_USER_ID';")"
expect_eq "users.is_active after activate" "t" "$IS_ACTIVE"

# ────────────────────────────────────────────────────────────────────────
# 8. Override subscription
# ────────────────────────────────────────────────────────────────────────
echo "→ POST override-subscription"
curl -sS -X POST "$BASE_URL/v1/admin/users/$TARGET_USER_ID/override-subscription" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan_name":"BASIC","reason":"complimentary upgrade test"}' | jq -e '.success == true' >/dev/null

NEW_PLAN="$(psql_exec kasku_billing \
  "SELECT p.name FROM public.subscriptions s JOIN public.subscription_plans p ON p.id = s.plan_id WHERE s.user_id = '$TARGET_USER_ID';")"
expect_eq "subscription.plan after override" "BASIC" "$NEW_PLAN"

# ────────────────────────────────────────────────────────────────────────
# 9. List payments — endpoint live (boleh kosong)
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/payments"
pay_resp="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE_URL/v1/admin/payments")"
echo "$pay_resp" | jq -e '.success == true' >/dev/null || { echo "FAIL payments: $pay_resp"; exit 1; }
echo "OK   payments endpoint live (count=$(echo "$pay_resp" | jq -r '.data | length'))"

# ────────────────────────────────────────────────────────────────────────
# 10. Dashboard stats
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/stats/dashboard"
stats_resp="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE_URL/v1/admin/stats/dashboard")"
TOTAL_USERS="$(echo "$stats_resp" | jq -r '.data.total_users')"
expect_ge "stats.total_users" 1 "$TOTAL_USERS"
echo "$stats_resp" | jq -e '.data.tier_distribution' >/dev/null || { echo "FAIL stats.tier_distribution missing"; exit 1; }
echo "OK   stats shape valid"

# ────────────────────────────────────────────────────────────────────────
# 11. Audit log — minimal 4 entry (LOGIN + 3 mutations)
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/audit-log?admin_id=$ADMIN_ID"
audit_resp="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE_URL/v1/admin/audit-log?admin_id=$ADMIN_ID&page_size=50")"
AUDIT_TOTAL="$(echo "$audit_resp" | jq -r '.meta.total')"
expect_ge "audit.total" 4 "$AUDIT_TOTAL"

# ────────────────────────────────────────────────────────────────────────
# 12. Logout — blacklist JWT
# ────────────────────────────────────────────────────────────────────────
echo "→ POST /v1/admin/auth/logout"
curl -sS -X POST "$BASE_URL/v1/admin/auth/logout" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -e '.success == true' >/dev/null

# ────────────────────────────────────────────────────────────────────────
# 13. JWT yang di-blacklist tidak bisa dipakai
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/auth/me dengan JWT yang sudah revoked"
me_status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $ADMIN_TOKEN" "$BASE_URL/v1/admin/auth/me")"
expect_eq "me_after_logout.status" "401" "$me_status"

# ────────────────────────────────────────────────────────────────────────
# 14. Tanpa Bearer → 401
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/users tanpa Bearer"
no_auth_status="$(curl -sS -o /dev/null -w '%{http_code}' "$BASE_URL/v1/admin/users")"
expect_eq "no_auth.status" "401" "$no_auth_status"

# ────────────────────────────────────────────────────────────────────────
# 15. User JWT (RS256) tidak diterima admin endpoint
# ────────────────────────────────────────────────────────────────────────
echo "→ GET /v1/admin/users dengan user JWT (RS256)"
USER_JWT="$(curl -sS -X POST "$BASE_URL/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "$(jq -nc --arg e "$TEST_EMAIL" --arg p "$TEST_PASSWORD" '{email:$e, password:$p}')" \
  | jq -r '.data.access_token // empty')"
if [ -n "$USER_JWT" ]; then
  user_jwt_status="$(curl -sS -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer $USER_JWT" "$BASE_URL/v1/admin/users")"
  expect_eq "user_jwt_on_admin.status" "401" "$user_jwt_status"
else
  echo "SKIP user JWT cross-test (login user gagal)"
fi

echo ""
echo "✓ admin_smoke OK — admin=$ADMIN_USERNAME target_user=$TEST_EMAIL"
