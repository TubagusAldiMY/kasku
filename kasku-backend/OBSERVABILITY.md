# Observability Stack — KasKu

Stack observability KasKu memakai **Prometheus + Loki + Grafana + Alertmanager + Promtail**. Tidak aktif default — jalankan dengan profile khusus.

## Quick Start (< 5 menit)

```bash
cd kasku-backend
docker compose --profile observability up -d
```

Akses UI:

| UI | URL | Default Creds |
|---|---|---|
| Grafana | http://localhost:3000 | admin / admin (override via `.env`) |
| Prometheus | http://localhost:9090 | — |
| Alertmanager | http://localhost:9093 | — |
| Loki | (via Grafana Explore saja, tidak punya UI sendiri) | — |

Stop:
```bash
docker compose --profile observability down
```

---

## Komponen

### 1. Prometheus (port 9090)
- Scrape interval 15s untuk semua service Go (8 service), Rust (price + sync), admin-service, dan RabbitMQ Prometheus plugin (port 15692).
- Alert rule di `infra/prometheus/alert-rules.yml` — di-evaluate tiap 30–60s.
- Reload config tanpa restart: `curl -X POST http://localhost:9090/-/reload`.

### 2. Loki (port 3100, akses via Grafana)
- TSDB storage di volume `loki-data`, retention **7 hari** (168h).
- Promtail discovery container Docker, extract label:
  - `service` — dari nama container `kasku-<X>`
  - `level`, `correlation_id` — dari field JSON log
- LogQL filter cepat: `{service="auth-service"} | correlation_id="abc-123"`.

### 3. Grafana (port 3000)
- Auto-provision dari `infra/grafana/provisioning/`:
  - **Data sources**: Prometheus (default) + Loki
  - **Folder**: `KasKu` dengan 4 dashboard preset
- 4 Dashboard:
  - **Services Overview** (`kasku-overview`): up/down stat, request rate, 5xx rate, p95 latency per service.
  - **HTTP Drill-Down** (`kasku-http`): variable `$service`, p50/p95/p99 per route, request rate per route, status code distribution.
  - **Sync Operations** (`kasku-sync`): push/pull/conflict/error counter + rate.
  - **RabbitMQ DLQ Watch** (`kasku-dlq`): DLQ count (threshold colored), DLQ growth, all queues backlog.

### 4. Alertmanager (port 9093)
- Route by severity:
  - `critical` → receiver `webhook-critical`
  - `warning` → receiver `webhook-warning`
  - default fallback → `webhook-default`
- Inhibit rule: `critical` suppress `warning` di service yang sama.
- Persist state di volume `alertmanager-data`.

### 5. Alert Rules (3 group)
| Alert | Severity | Trigger |
|---|---|---|
| `ServiceDown` | critical | `up == 0` selama 2m |
| `HighErrorRate` | warning | rate 5xx / total > 5% selama 5m |
| `HighLatencyP95` | warning | p95 > 1s selama 10m |
| `RabbitMQDLQNonEmpty` | warning | DLQ messages > 0 selama 2m |
| `DBPoolNearExhaustion` | warning | acquired/max > 90% selama 5m |
| `SyncConflictSpike` | info | conflict rate > 0.5/s selama 10m |

---

## Konfigurasi Webhook Alertmanager

Alertmanager **tidak** support env-var substitution di config. Edit langsung file ini untuk integrasi Slack/Discord/Telegram:

```bash
# kasku-backend/infra/alertmanager/alertmanager.yml
receivers:
  - name: 'webhook-critical'
    webhook_configs:
      - url: 'https://hooks.slack.com/services/T00/B00/XXX'   # ← ganti
        send_resolved: true
```

Lalu reload Alertmanager:
```bash
curl -X POST http://localhost:9093/-/reload
```

Atau pakai sidecar webhook adapter (mis. [bitnami/alertmanager-webhook-relay](https://hub.docker.com/r/bitnami/alertmanager-webhook-relay)) yang bisa baca dari env var dan forward ke URL final.

---

## LogQL Query Examples

```logql
# Semua error di auth-service hari ini
{service="auth-service", level="error"}

# Trace request lewat correlation_id (cross-service)
{correlation_id="abc-123-def"}

# Filter status >= 500 dari api-gateway
{service="api-gateway"} | json | status >= 500

# Hitung error rate per service (5 menit)
sum by (service) (rate({level="error"}[5m]))
```

---

## Cara Tambah Custom Metric di Service Baru (Go)

1. `go.mod` tambah replace + require ke `../observability-go`.
2. Buat registry di `main.go`:
   ```go
   reg := obsmetrics.NewRegistry("my-service")
   reg.RegisterDBPool(pool)   // kalau pakai pgxpool
   ```
3. Wire ke router: `r.Use(reg.HTTPMetrics())` + `r.GET("/metrics", gin.WrapH(reg.Handler()))`.
4. Business counter:
   ```go
   loginAttempts := reg.Counter(
       "kasku_auth_login_attempts_total",
       "Jumlah upaya login.",
       []string{"result"},
   )
   loginAttempts.WithLabelValues("success").Inc()
   ```

Untuk Rust, ikuti pola `sync-service` (AtomicU64 counter + manual Prometheus text format di `/metrics` handler).

---

## Troubleshooting

### Service tidak muncul di Prometheus targets
- Cek `http://localhost:9090/targets` — apakah `state: UP`?
- Kalau `state: DOWN`, cek `docker compose logs <service>` — apakah service expose /metrics di port yang benar?
- Cek network — Prometheus harus bisa reach service (network `kasku-internal` + `kasku-data`).

### Dashboard kosong
- Datasource kosong: cek `http://localhost:3000/datasources` — harus ada "Prometheus" dan "Loki".
- Tidak ada metric: trigger sample request (`curl http://localhost:8081/health` 5x), tunggu 15s, refresh.

### Alert firing tapi webhook tidak sampai
- Cek `http://localhost:9093/#/status` — receiver config benar?
- Webhook URL placeholder default 9999 — pasti gagal. Edit `infra/alertmanager/alertmanager.yml`.
- Cek `docker compose logs alertmanager` untuk error pengiriman.

### Loki tidak ingest log
- Cek `docker compose logs promtail` — apakah ada error parse JSON?
- Cek positions file `/tmp/positions.yaml` di container Promtail.
- Pastikan service log dalam format JSON (zerolog/tracing JSON), bukan plain text.

---

## PII Logging Policy

Per `Plan/Arsitektur.md` seksi 9 dan kebijakan KasKu:

- **Email** wajib di-mask sebelum di-log: `u***@domain.com` (helper `maskEmail` di notification-service).
- **Password / token / refresh token / JWT secret** tidak pernah di-log dalam bentuk apa pun.
- **PII finansial** (nomor rekening, saldo) tidak di-log di production path.
- Audit log untuk admin action di-store di `kasku_admin.admin_audit_log` (bukan Loki) untuk retensi permanen.

---

## Out of Scope

- Distributed tracing (OpenTelemetry / Jaeger) — Phase 3.
- APM komersial (Datadog / New Relic).
- SLO formal docs — tetapkan setelah ada baseline 2 minggu data.
- Multi-region observability.
