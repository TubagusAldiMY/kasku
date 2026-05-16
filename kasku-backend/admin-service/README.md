# admin-service

Dashboard backend untuk operator KasKu — manajemen user, override subscription,
audit log, dan statistik platform. Diisolasi dari user-facing services di network
`kasku-admin`.

- Tech: Go 1.25 + Gin + pgx + zerolog
- Port: HTTP **8090** (tidak menjalankan gRPC server)
- Database utama: `kasku_admin` (R/W)
- Cross-DB credential: `kasku_auth_svc` (R/W terbatas) + `kasku_billing_svc` (R/W terbatas)
- Auth: **HS256 JWT** (terpisah dari user RS256 JWT)

---

## Endpoint Surface

| Method | Path | Auth | PRD ID |
|--------|------|------|--------|
| GET | `/health` | — | — |
| GET | `/metrics` | — | — |
| POST | `/v1/admin/auth/login` | — | F-ADM-01 |
| POST | `/v1/admin/auth/logout` | Admin JWT | — |
| GET | `/v1/admin/auth/me` | Admin JWT | — |
| GET | `/v1/admin/stats/dashboard` | Admin JWT | F-ADM-02 |
| GET | `/v1/admin/users` | Admin JWT | F-ADM-03 |
| GET | `/v1/admin/users/:id` | Admin JWT | F-ADM-04 |
| POST | `/v1/admin/users/:id/suspend` | Admin JWT | F-ADM-05 |
| POST | `/v1/admin/users/:id/activate` | Admin JWT | F-ADM-06 |
| POST | `/v1/admin/users/:id/override-subscription` | Admin JWT | F-ADM-07 |
| GET | `/v1/admin/payments` | Admin JWT | F-ADM-08 |
| GET | `/v1/admin/audit-log` | Admin JWT | — |

Semua endpoint protected wajib `Authorization: Bearer <admin_jwt>` (HS256). Token
diperoleh dari `POST /v1/admin/auth/login`. JWT user (RS256) tidak akan diterima.

---

## Network Isolation

```
[admin browser]
    │ HTTPS
    ▼
[traefik] ─► [api-gateway :8080] ─► /v1/admin/** ─► [admin-service :8090]
                  │  RateLimit only (NO user AuthMiddleware)
                  │
                  ├─ kasku-public
                  ├─ kasku-internal       ← admin-service TIDAK di network ini
                  ├─ kasku-admin           ← admin-service join di sini
                  └─ kasku-data            ← admin-service join di sini (perlu untuk Postgres + Redis)
```

api-gateway memproksi `/v1/admin/**` ke admin-service **tanpa** memasang user
AuthMiddleware. admin-service memverifikasi HS256 JWT sendiri — pesan
keseluruhan admin auth tidak melewati infrastructure user JWT.

`admin-service` tidak join `kasku-internal` agar tidak memiliki akses
langsung ke service user-facing (mencegah lateral movement bila akun admin
terkompromi).

---

## Cross-DB Credential Trade-off

admin-service punya R/W terbatas ke 3 database:

| DB | Tujuan | User yang dipakai |
|----|--------|-------------------|
| `kasku_admin` | admin_users, admin_audit_log | `kasku_admin_svc` (owner) |
| `kasku_auth` | UPDATE `users.is_active` (suspend/activate) + SELECT detail | `kasku_auth_svc` (membonceng) |
| `kasku_billing` | UPDATE `subscriptions.plan_id` (override) + SELECT payments | `kasku_billing_svc` (membonceng) |

**Trade-off**: pendekatan paling bersih adalah role `kasku_admin_writer` dengan
GRANT minimal (cuma kolom `is_active` di users, kolom `plan_id` di
subscriptions). Untuk MVP kita pakai credential service pemilik — risiko: kalau
admin-service tertembus, attacker dapat akses penuh ke `kasku_auth_svc` dan
`kasku_billing_svc`. Mitigasi: code path admin-service strict (cuma update kolom
yang diperlukan, tidak ada SQL bebas, tidak ada query parameterized dengan
bind dinamis). **Action item produksi**: tambah role `kasku_admin_writer` di
`infra/postgres/00-init-databases.sh` dengan grant scoped.

---

## Audit Log

Setiap aksi mutation **wajib** menghasilkan satu row di `kasku_admin.admin_audit_log`:

| Kolom | Nilai |
|-------|-------|
| `admin_id` | UUID admin yang melakukan aksi |
| `action` | `LOGIN` \| `LOGOUT` \| `SUSPEND_USER` \| `ACTIVATE_USER` \| `OVERRIDE_SUBSCRIPTION` |
| `target_user_id` | UUID user terdampak (nullable untuk LOGIN/LOGOUT) |
| `target_entity` | `user` \| `subscription` |
| `metadata` | JSONB — reason, old/new plan, ip, jti, dll |
| `success` | `false` bila mutation gagal (audit log tetap dibuat agar attempt terlihat) |
| `created_at` | Timestamp UTC |

**Cross-DB atomicity**: UPDATE di `kasku_auth.users` + INSERT di
`kasku_admin.admin_audit_log` **bukan satu transaksi** (database berbeda).
Trade-off: kami memilih availability > strict auditability. Jika audit log
INSERT gagal, error di-log via zerolog tetapi mutation tidak di-rollback.

---

## Dev Commands

```bash
make build         # CGO_ENABLED=0 go build -o bin/admin-service ./cmd/server
make run           # go run ./cmd/server
make test          # go test ./... -v -race -count=1
make lint          # golangci-lint run ./...
make docker-build  # docker build -t kasku/admin-service:latest .
```

Local dev butuh:
- `kasku_admin` + `kasku_auth` + `kasku_billing` PostgreSQL DBs (via `docker compose up -d postgres`)
- Redis (untuk JWT blacklist)
- Env vars di `.env.example` di-copy ke `.env`

Bootstrap admin **otomatis di-seed** saat container pertama kali start kalau
tabel `admin_users` masih kosong (dari env `ADMIN_BOOTSTRAP_USERNAME` +
`ADMIN_BOOTSTRAP_PASSWORD`). Restart tidak menghasilkan duplikat.

---

## Environment Variables

| Variable | Default | Wajib | Deskripsi |
|----------|---------|-------|-----------|
| `SERVER_PORT` | `8090` | — | Port HTTP |
| `APP_ENV` | `development` | — | Label log |
| `LOG_LEVEL` | `info` | — | zerolog level |
| `POSTGRES_ADMIN_DSN` | — | ✓ | DSN `kasku_admin` (user `kasku_admin_svc`) |
| `POSTGRES_AUTH_ADMIN_DSN` | — | ✓ | DSN `kasku_auth` (user `kasku_auth_svc`) |
| `POSTGRES_BILLING_ADMIN_DSN` | — | ✓ | DSN `kasku_billing` (user `kasku_billing_svc`) |
| `REDIS_ADDR` | `redis:6379` | — | Untuk JWT blacklist |
| `REDIS_PASSWORD` | — | — | Password Redis |
| `ADMIN_JWT_SECRET` | — | ✓ | HS256 secret, **terpisah dari JWT_PRIVATE_KEY** |
| `ADMIN_JWT_TTL` | `8h` | — | Lifetime admin token; tidak ada refresh rotation |
| `ADMIN_BOOTSTRAP_USERNAME` | — | ⚠ saat first boot | Username admin pertama |
| `ADMIN_BOOTSTRAP_PASSWORD` | — | ⚠ saat first boot | Password admin pertama |
| `ARGON2_TIME` | `3` | — | Argon2id iterations |
| `ARGON2_MEMORY_KB` | `65536` | — | 64MiB |
| `ARGON2_THREADS` | `4` | — | Argon2id parallelism |
| `ARGON2_KEY_LENGTH` | `32` | — | Output hash length (bytes) |

Lihat `.env.example` untuk template lengkap.

---

## Security Notes

- Tenant isolation: **N/A** (admin-service tidak multi-tenant).
- Password Argon2id (sama parameter dengan auth-service); constant-time verify.
- Email/PII tidak di-log. Audit log mengandung `target_user_id` + email
  (kategori PII rendah; perlu untuk operasional). Akses tabel ini terbatas ke
  `kasku_admin_svc` saja.
- Admin JWT secret di-rotate? Belum (MVP). Bila secret bocor, semua token aktif
  invalid setelah rotation. Recommend: tambah `secret_id` claim untuk
  graceful rotation di iterasi berikut.
- Tidak ada refresh token rotation untuk admin (MVP) — admin re-login setelah
  8 jam.
- `/v1/admin/auth/login` tidak di-rate-limit secara spesifik di service ini —
  proteksi datang dari api-gateway RateLimitMiddleware (default 10/menit per IP).
- SQL injection: semua query parameterized (`$1`, `$2`, …). Filter dinamis
  (where clause) dibangun via index `$N` placeholder, **tidak ada** string
  concat dari input user.

---

## Open Items

- [ ] **Role `kasku_admin_writer`** dengan grant kolom-scoped (mengganti
  membonceng credential service pemilik untuk write ke kasku_auth + kasku_billing)
- [ ] **JWT secret rotation** via `kid` (key id) claim
- [ ] **Refresh token rotation** untuk admin session (optional — saat ini 8h fixed)
- [ ] **2FA** untuk admin login (TOTP via authenticator app)
- [ ] **Change password endpoint** `PUT /v1/admin/auth/password` untuk admin
  rotate bootstrap password
- [ ] **Cross-service health aggregation** via Prometheus/Grafana scrape
  (di luar scope service ini)
