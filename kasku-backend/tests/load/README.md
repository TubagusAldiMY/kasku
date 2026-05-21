# Load Testing — KasKu

Baseline load test menggunakan [k6](https://k6.io). Dijalankan terakhir setelah semua infra production-ready.

## Instalasi k6

```bash
# Docker (no install)
docker run --rm -i grafana/k6 run - < tests/load/auth-flow.js

# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" \
  | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6
```

## Menjalankan Test

Pastikan stack `docker compose up -d` sudah berjalan dan semua service healthy.

```bash
cd kasku-backend/tests/load

# Auth flow (tidak butuh user pre-created)
k6 run --env BASE_URL=http://localhost:8080 auth-flow.js

# Transaction flow (butuh user verified + tenant provisioned)
k6 run \
  --env BASE_URL=http://localhost:8080 \
  --env TEST_USER_EMAIL=verified@kasku.test \
  --env TEST_USER_PASSWORD=TestPass1! \
  transaction-flow.js

# Sync flow (butuh user verified + tenant provisioned)
k6 run \
  --env BASE_URL=http://localhost:8080 \
  --env TEST_USER_EMAIL=verified@kasku.test \
  --env TEST_USER_PASSWORD=TestPass1! \
  sync-flow.js
```

## Target Baseline (diukur saat initial run, bukan dikira)

| Scenario | VUs | P50 | P95 | P99 | Error Rate |
|----------|-----|-----|-----|-----|------------|
| Auth flow | 50 | — | < 500ms | — | < 1% |
| Transaction flow | 50 | — | < 300ms | — | < 1% |
| Sync push (10 ops) | 20 | — | < 1s | — | < 1% |

**Catatan**: Isi kolom P50/P95/P99 dengan hasil aktual setelah test pertama dijalankan. Gunakan angka ini sebagai reference untuk regresi pada release berikutnya.

## Output dengan InfluxDB + Grafana (opsional)

```bash
# Start InfluxDB
docker run -d -p 8086:8086 --name influxdb influxdb:1.8

# Run dengan output ke InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 \
  --env BASE_URL=http://localhost:8080 \
  auth-flow.js
```

Import dashboard k6 dari Grafana.com (ID: 2587) untuk visualisasi real-time.

## Interpretasi Hasil

- **P95 > 2x target** → Investigate sebelum production launch. Periksa: connection pool (pgxpool), Redis hit rate, N+1 query.
- **Error rate > 1%** → Periksa logs (`docker compose logs -f api-gateway`) untuk 5xx pattern.
- **Spike di sync flow** → Kemungkinan bottleneck di sync_log write atau gRPC timeout ke finance/transaction/investment service.
