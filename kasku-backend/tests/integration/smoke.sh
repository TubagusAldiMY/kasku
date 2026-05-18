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
check_optional "auth" "${AUTH_URL:-http://localhost:8081}/metrics"
check_optional "user" "${USER_URL:-http://localhost:8082}/metrics"
check_optional "finance" "${FINANCE_URL:-http://localhost:8084}/metrics"
check_optional "transaction" "${TRANSACTION_URL:-http://localhost:8085}/metrics"
check_optional "investment" "${INVESTMENT_URL:-http://localhost:8086}/metrics"
check_optional "price" "${PRICE_URL:-http://localhost:8087}/metrics"
check_optional "sync" "${SYNC_URL:-http://localhost:8088}/metrics"
check_optional "notification" "${NOTIFICATION_URL:-http://localhost:8089}/metrics"

# Observability stack (hanya aktif kalau --profile observability).
check_optional "prometheus" "${PROMETHEUS_URL:-http://localhost:9090}/-/ready"
check_optional "alertmanager" "${ALERTMANAGER_URL:-http://localhost:9093}/-/ready"
check_optional "grafana" "${GRAFANA_URL:-http://localhost:3000}/api/health"
check_optional "loki" "${LOKI_URL:-http://localhost:3100}/ready"
