#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://localhost:8080}"

check() {
  name="$1"
  url="$2"
  code="$(curl -sS -o /tmp/kasku-smoke-response -w '%{http_code}' "$url")"
  if [ "$code" -lt 200 ] || [ "$code" -ge 300 ]; then
    echo "FAIL $name: HTTP $code"
    cat /tmp/kasku-smoke-response
    exit 1
  fi
  echo "OK $name"
}

check "gateway" "$BASE_URL/health"
check "auth" "${AUTH_URL:-http://localhost:8081}/health"
check "user" "${USER_URL:-http://localhost:8082}/health"
check "billing" "${BILLING_URL:-http://localhost:8083}/health"
check "finance" "${FINANCE_URL:-http://localhost:8084}/health"
check "transaction" "${TRANSACTION_URL:-http://localhost:8085}/health"
check "investment" "${INVESTMENT_URL:-http://localhost:8086}/health"
check "price" "${PRICE_URL:-http://localhost:8087}/health"
check "sync" "${SYNC_URL:-http://localhost:8088}/health"
check "notification" "${NOTIFICATION_URL:-http://localhost:8089}/health"
check "admin" "${ADMIN_URL:-http://localhost:8090}/health"

# Metrics endpoints (Prometheus scrape target untuk service yang sudah implement).
# Tidak fail kalau service belum expose /metrics — pakai check_optional.
check_optional() {
  name="$1"
  url="$2"
  code="$(curl -sS -o /tmp/kasku-smoke-response -w '%{http_code}' "$url" || echo 000)"
  if [ "$code" = "200" ]; then
    echo "OK $name (metrics)"
  else
    echo "SKIP $name (metrics): HTTP $code"
  fi
}

check_optional "admin" "${ADMIN_URL:-http://localhost:8090}/metrics"
check_optional "billing" "${BILLING_URL:-http://localhost:8083}/metrics"
