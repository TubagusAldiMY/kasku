# SRS — Software Requirements Specification
## KasKu: Personal Finance SaaS Platform
### IEEE 830 / ISO 29148 Compliant

**Versi:** 2.0.0
**Tanggal:** 2026-04-27
**Author:** TubsAMY (admin@tubsamy.tech)
**Status:** DRAFT
**Changelog:** v2.0.0 — Pivot dari single-user on-premise monolith ke multi-tenant SaaS dengan 11 microservices, subscription billing, RabbitMQ event-driven architecture, dan compliance UU PDP No. 27/2022.

---

## Daftar Isi

1. [Introduction](#1-introduction)
2. [Overall Description](#2-overall-description)
3. [Functional Requirements](#3-functional-requirements)
4. [Non-Functional Requirements](#4-non-functional-requirements)
5. [System Interfaces](#5-system-interfaces)
6. [Data Requirements](#6-data-requirements)
7. [Quality Attributes Summary](#7-quality-attributes-summary)

---

## 1. Introduction

### 1.1 Purpose

Dokumen ini merupakan Software Requirements Specification (SRS) untuk **KasKu v2.0**, sebuah platform SaaS manajemen keuangan pribadi multi-tenant. SRS ini mendefinisikan secara lengkap seluruh functional requirements dan non-functional requirements untuk sistem yang terdiri dari 11 microservices, subscription billing via Midtrans, notifikasi email event-driven, admin portal, dan frontend PWA offline-first.

Dokumen ini menjadi kontrak teknis antara product owner (TubsAMY) dan tim engineering. Semua implementasi harus mengacu pada spesifikasi di dokumen ini.

### 1.2 Scope

KasKu SaaS Platform mencakup komponen-komponen berikut:

**Backend (11 microservices):**
- `api-gateway` — Routing, JWT verification, rate limiting, tier header injection
- `auth-service` — Registrasi, login, JWT RS256, refresh token rotation, email verification
- `user-service` — Profile management, tenant provisioning/deprovisioning
- `billing-service` — Subscription plans, Midtrans payment, tier limit enforcement (gRPC)
- `finance-service` — CRUD akun keuangan, balance history
- `transaction-service` — CRUD transaksi, kategori, export CSV
- `investment-service` — CRUD instrumen investasi, unit history, price aggregation
- `price-service` (Rust) — Real-time price cache dari CoinGecko dan metals.live
- `sync-service` (Rust) — Offline sync engine, conflict resolution
- `notification-service` — Email transaksional event-driven via RabbitMQ
- `admin-service` — Platform dashboard, user management (internal network only)

**Frontend:**
- SvelteKit 2.0 + Svelte 5 PWA offline-first

**Infrastructure:**
- PostgreSQL ≥ 14 (5 database dengan schema-per-tenant di kasku_finance)
- RabbitMQ ≥ 3.12 (async event bus)
- Redis ≥ 7 (rate limiting, JWT blacklist)
- Traefik v3 (TLS, reverse proxy)

**Out of scope:** Mobile native app, open banking integration, laporan pajak, multi-user per akun (family sharing).

### 1.3 Definitions

| Term | Definisi |
|------|----------|
| Tenant | Satu akun pengguna terdaftar. Setiap tenant memiliki schema PostgreSQL terisolasi di database kasku_finance dengan format `tenant_{user_uuid_sanitized}` |
| Subscription Plan | Paket layanan yang menentukan fitur dan batas penggunaan yang tersedia untuk seorang tenant. Terdapat 3 plan: FREE, BASIC, PRO |
| Tier Limit | Batas kuantitatif fitur yang berlaku per plan: jumlah maksimal transaksi/bulan, akun keuangan, instrumen investasi, dan durasi retensi history |
| Tenant Schema | Schema PostgreSQL milik satu tenant di database kasku_finance. Format nama: `tenant_{user_uuid_sanitized}` (karakter `-` diganti `_`) |
| Tenant Provisioning | Proses pembuatan schema, tabel, dan seed data kategori untuk tenant baru, dieksekusi oleh PostgreSQL function `provision_tenant(UUID)` |
| RabbitMQ Event | Pesan asinkron yang dipublikasikan ke exchange `kasku.events` (topic type) saat terjadi domain event tertentu |
| gRPC | Google Remote Procedure Call — protokol inter-service synchronous menggunakan Protobuf untuk komunikasi type-safe berkinerja tinggi |
| Webhook | HTTP callback yang dikirim oleh Midtrans ke endpoint KasKu saat status pembayaran berubah |
| MRR | Monthly Recurring Revenue — total pendapatan berulang bulanan dari subscription aktif |
| Churn Rate | Persentase pengguna berbayar yang membatalkan atau tidak memperbarui subscription dalam periode satu bulan |
| Offline-first | Paradigma desain di mana aplikasi berfungsi penuh tanpa koneksi internet, menggunakan IndexedDB sebagai local data store |
| Schema-per-tenant | Strategi multi-tenancy di mana setiap tenant memiliki set tabel terisolasi dalam schema PostgreSQL yang terpisah |
| Soft Delete | Penandaan data sebagai dihapus via kolom `is_deleted = true` + `deleted_at` tanpa menghapus baris dari database |
| JWT RS256 | JSON Web Token dengan algoritma RSASSA-PKCS1-v1_5 menggunakan SHA-256. Private key di auth-service, public key di api-gateway |
| Argon2id | Algoritma hashing password yang direkomendasikan OWASP. Parameter: memory=64MB, iterations=3, parallelism=2 |
| HTTP 402 | HTTP status Payment Required — digunakan saat tier limit terlampaui |
| DLQ | Dead Letter Queue — antrian RabbitMQ untuk menyimpan event yang gagal diproses setelah retry maksimal |
| Server Wins | Strategi conflict resolution offline sync di mana data server selalu menang atas perubahan lokal yang berkonflik |

### 1.4 References

| Dokumen / Standar | Keterangan |
|-------------------|------------|
| IEEE 830-1998 | Recommended Practice for Software Requirements Specifications |
| ISO/IEC/IEEE 29148:2018 | Systems and software engineering — Requirements engineering |
| `Plan/Arsitektur.md` (v2.0) | ADR-001 s/d ADR-012, system diagram, service matrix, security architecture, deployment topology |
| `Plan/databaseScheme.md` (v2.0) | Full DDL 5 database, `provision_tenant()` function, migration files, index strategy |
| `Plan/ApiSpecOpenAPI.yaml` (v2.0) | OpenAPI 3.0.3 spec — 37 paths, ~60 schemas |
| `Plan/PRD.md` (v2.0) | Product Requirements Document |
| OWASP Top 10 2021 | Panduan keamanan aplikasi web |
| UU PDP No. 27 Tahun 2022 | Undang-Undang Perlindungan Data Pribadi Indonesia |
| Midtrans API Documentation | https://docs.midtrans.com — Snap API reference |
| RabbitMQ Documentation | https://www.rabbitmq.com/documentation.html |
| gRPC Documentation | https://grpc.io/docs/ |
| CoinGecko API v3 | https://www.coingecko.com/api/documentation |

---

## 2. Overall Description

### 2.1 Product Perspective

KasKu SaaS Platform adalah sistem mandiri (standalone) yang dioperasikan oleh TubsAMY di VPS dengan Docker Compose. Platform tidak bergantung pada cloud provider besar untuk data storage pengguna. Semua data keuangan pengguna tersimpan di PostgreSQL yang berjalan di VPS yang sama.

**Topology deployment:**
```
Internet → Traefik (TLS/443) → api-gateway (:8080) → [11 microservices]
Admin → VPN/internal → admin-service (:8090)
Midtrans → api-gateway (:8080) → billing-service (:8083) [webhook]
```

Platform berinteraksi dengan sistem eksternal berikut:
- **Midtrans** — payment gateway untuk subscription billing (outbound + webhook inbound)
- **CoinGecko API** — sumber harga kripto (outbound dari price-service)
- **metals.live API** — sumber harga emas (outbound dari price-service)
- **SMTP Server / SendGrid** — pengiriman email transaksional (outbound dari notification-service)

### 2.2 Product Functions

```
KasKu SaaS Platform
├── Platform Layer
│   ├── api-gateway          — Routing, JWT verify, rate limit, tier header inject
│   ├── auth-service         — Register, verify email, login, JWT RS256, refresh, logout, forgot/reset password
│   ├── user-service         — Profile, tenant provisioning/deprovisioning
│   └── admin-service        — Platform dashboard, user management (internal only)
│
├── Billing Layer
│   └── billing-service      — Plan listing, Midtrans Snap, webhook, subscription lifecycle, tier enforcement (gRPC)
│
├── Finance Layer
│   ├── finance-service      — CRUD akun keuangan, balance history
│   ├── transaction-service  — CRUD transaksi, kategori, export CSV
│   └── investment-service   — CRUD instrumen investasi, unit history, price aggregation
│
├── Infrastructure Layer
│   ├── price-service (Rust) — Real-time price cache CoinGecko + metals.live, gRPC endpoint
│   ├── sync-service (Rust)  — Offline sync batch handler, conflict resolution Server Wins
│   └── notification-service — Email transaksional event-driven via RabbitMQ
│
└── Client Layer
    └── SvelteKit PWA        — Offline-first (IndexedDB + Dexie.js + Service Worker)
```

### 2.3 User Characteristics

| Tipe User | Karakteristik | Akses |
|-----------|--------------|-------|
| Regular User — Free | Entry-level, pencatatan dasar, budget terbatas | Dashboard, 3 akun, 2 instrumen, 50 tx/bulan |
| Regular User — Basic | Investor pemula, butuh lebih banyak akun dan riwayat 1 tahun | Dashboard, 10 akun, 10 instrumen, 500 tx/bulan |
| Regular User — Pro | Investor aktif, butuh unlimited, export CSV | Dashboard, unlimited semua, export CSV |
| Platform Admin | Operator TubsAMY, akses internal | Admin portal — platform stats, user management |

### 2.4 Constraints

| Constraint | Detail |
|------------|--------|
| Runtime | VPS Linux x86_64, minimum 2 vCPU dan 4 GB RAM |
| Database | PostgreSQL ≥ 14, satu instance dengan 5 database terpisah |
| Tenancy model | Schema-per-tenant di kasku_finance (bukan shared tables, bukan DB-per-tenant) |
| Language | Go ≥ 1.22 untuk 9 service, Rust stable untuk price-service dan sync-service |
| Frontend | SvelteKit ≥ 2.0 + Svelte 5, target browser Chrome/Firefox ≥ 100, Safari ≥ 15 |
| Orchestration | Docker Compose v2 (bukan Kubernetes — resource VPS terbatas, ADR-009) |
| Broker | RabbitMQ ≥ 3.12 (bukan Kafka — overhead Kafka tidak proporsional untuk skala VPS, ADR-004) |
| Payment | Midtrans saja (bukan Stripe/Xendit — penetrasi pasar Indonesia, ADR-007) |
| Admin service | Tidak terekspos ke public internet — internal network only |

### 2.5 Assumptions and Dependencies

| Asumsi / Dependency | Detail |
|--------------------|--------|
| VPS tersedia | VPS Linux dengan minimum 2 vCPU + 4 GB RAM dan IP publik statis |
| Domain dan TLS | Domain dikonfigurasi mengarah ke VPS; Traefik mengurus TLS via Let's Encrypt |
| Koneksi internet VPS | Diperlukan untuk Midtrans webhook, CoinGecko, metals.live, dan SMTP |
| Midtrans sandbox | Tersedia untuk testing sebelum production |
| CoinGecko free tier | 60 request/menit — mencukupi untuk skala awal |
| SMTP tersedia | Operator mengkonfigurasi SMTP credentials via env vars |
| PostgreSQL single instance | Semua 5 database berjalan di satu PostgreSQL instance |

---

## 3. Functional Requirements

### Konvensi Penomoran

Format: `{SERVICE_CODE}-FR-{NNN}`

Service codes: AUTH, USER, BILLING, FINANCE, TRX, INV, PRICE, SYNC, NOTIF, ADMIN, DASH

### 3.1 Auth Service

**AUTH-FR-001 — Register**
Sistem HARUS menerima permintaan registrasi dengan field: `email` (format RFC 5322, maksimal 254 karakter), `username` (3–50 karakter, alfanumerik + underscore + hyphen), `password` (minimal 8 karakter, minimal 1 huruf besar, 1 huruf kecil, 1 angka). Sistem HARUS memverifikasi uniqueness email dan username. Sistem HARUS membuat user dengan status `email_verified = false`, meng-hash password menggunakan Argon2id (memory=64MB, iterations=3, parallelism=2), dan mempublikasikan event `user.registered` ke RabbitMQ.

**AUTH-FR-002 — Verify Email**
Sistem HARUS memvalidasi token verifikasi dari parameter query URL. Token harus: belum pernah digunakan, belum kadaluwarsa (TTL 24 jam sejak dibuat), dan cocok dengan token di tabel `email_verifications`. Setelah validasi berhasil, sistem HARUS mengupdate `email_verified = true` dan menandai token sebagai `used_at = NOW()`.

**AUTH-FR-003 — Resend Verification Email**
Sistem HARUS memungkinkan pengguna yang belum terverifikasi untuk meminta ulang email verifikasi. Sistem HARUS membuat token baru dan mempublikasikan event yang memicu notification-service mengirim ulang email. Token lama yang belum digunakan HARUS diinvalidasi.

**AUTH-FR-004 — Login**
Sistem HARUS menerima `identifier` (email atau username) dan `password`. Sebelum verifikasi password, sistem HARUS memeriksa: (1) apakah akun dalam status lockout, (2) apakah email sudah terverifikasi. Jika email belum terverifikasi, return HTTP 403 dengan pesan spesifik. Sistem HARUS memverifikasi password menggunakan Argon2id. Jika berhasil, sistem HARUS: reset counter gagal login, buat JWT access token RS256 (15 menit) beserta claims `user_id`, `email`, `subscription_tier`, dan buat refresh token (random 256-bit, hash SHA-256 untuk disimpan di DB, token asli di HttpOnly Secure SameSite=Strict cookie, TTL 7 hari).

**AUTH-FR-005 — Brute Force Protection**
Sistem HARUS mencatat setiap percobaan login gagal per akun. Setelah 5 percobaan gagal berturut-turut, sistem HARUS mengunci akun selama 15 menit. Selama lockout, semua percobaan login HARUS ditolak dengan HTTP 429. Counter HARUS direset setelah login berhasil.

**AUTH-FR-006 — Refresh Token**
Sistem HARUS memvalidasi refresh token dari HttpOnly cookie: cocok dengan hash SHA-256 yang tersimpan di tabel `refresh_tokens`, belum kadaluwarsa, belum direvoke. Jika valid, sistem HARUS: menerbitkan access token JWT baru, menerbitkan refresh token baru (rotation), merevoke token lama di DB. Deteksi reuse token lama (token sudah direvoke digunakan lagi) HARUS memicu revokasi semua refresh token milik user tersebut.

**AUTH-FR-007 — Logout**
Sistem HARUS merevoke refresh token yang aktif di database. Sistem HARUS menambahkan access token saat ini ke JWT blacklist di Redis dengan TTL sama dengan sisa waktu valid token.

**AUTH-FR-008 — Forgot Password**
Sistem HARUS menerima `email`. Jika email tidak terdaftar, sistem TETAP HARUS mengembalikan HTTP 200 (untuk mencegah email enumeration). Sistem HARUS membuat password reset token (random 256-bit, hash SHA-256 di DB, TTL 1 jam) dan mempublikasikan event yang memicu notification-service mengirim reset link.

**AUTH-FR-009 — Reset Password**
Sistem HARUS memvalidasi reset token: belum digunakan, belum kadaluwarsa. Sistem HARUS meng-hash password baru dengan Argon2id. Sistem HARUS merevoke semua refresh token aktif milik user. Sistem HARUS menandai reset token sebagai `used_at = NOW()`.

### 3.2 User Service

**USER-FR-001 — Get Profile**
Sistem HARUS mengembalikan data profil user yang terautentikasi: `user_id`, `email`, `username`, `display_name`, `created_at`. Password hash TIDAK BOLEH dikembalikan di response apapun.

**USER-FR-002 — Update Profile**
Sistem HARUS memungkinkan user mengupdate `username` dan `display_name`. Uniqueness username HARUS divalidasi sebelum update. Email tidak dapat diubah.

**USER-FR-003 — Tenant Provisioning**
Sistem HARUS mengonsumsi event `user.registered` dari RabbitMQ. Setelah menerima event, sistem HARUS memanggil PostgreSQL function `provision_tenant(user_id::UUID)` di kasku_finance untuk: membuat schema `tenant_{user_uuid_sanitized}`, membuat semua tabel (financial_accounts, balance_history, transactions, categories, investment_assets, unit_history, sync_log), dan meng-insert seed data 10 kategori default (Makan, Transportasi, Belanja, Tagihan, Hiburan, Kesehatan, Pendidikan, Gaji, Investasi, Lainnya).

**USER-FR-004 — Tenant Deprovisioning**
Sistem HARUS mengonsumsi event `user.deletion_requested` dari RabbitMQ. Sistem HARUS memanggil `deprovision_tenant(user_id::UUID)` untuk DROP SCHEMA tenant secara permanen (ini adalah operasi irreversible yang hanya boleh dipicu setelah konfirmasi eksplisit dari user).

**USER-FR-005 — Delete Account**
Sistem HARUS menerima permintaan DELETE /me dari user yang terautentikasi. Sistem HARUS melakukan soft delete pada user (is_deleted = true, deleted_at = NOW()) dan mempublikasikan event `user.deletion_requested`.

### 3.3 Billing Service

**BILLING-FR-001 — List Plans**
Sistem HARUS mengembalikan daftar semua subscription plan yang aktif dengan field: `plan_id`, `name`, `price_idr`, `billing_period`, dan batas tier: `max_transactions_per_month`, `max_financial_accounts`, `max_investment_instruments`, `history_retention_months` (NULL = unlimited). Data bersumber dari tabel `subscription_plans` di kasku_billing.

**BILLING-FR-002 — Get Current Subscription**
Sistem HARUS mengembalikan subscription aktif user yang terautentikasi: plan saat ini, tanggal mulai, tanggal berakhir, dan status.

**BILLING-FR-003 — Subscribe / Upgrade**
Sistem HARUS membuat Midtrans Snap transaction dengan `order_id` format `KASKU-{first_8_chars_user_id}-{unix_timestamp}`. Sistem HARUS menyimpan payment record dengan status PENDING di tabel `payments` sebelum menghubungi Midtrans. Sistem HARUS mengembalikan `snap_redirect_url` ke frontend.

**BILLING-FR-004 — Midtrans Webhook Handler**
Sistem HARUS memverifikasi Midtrans webhook signature sebelum memproses apapun. Verifikasi: SHA-512 hash dari `order_id + status_code + gross_amount + server_key` harus cocok dengan `signature_key` di payload. Sistem HARUS melakukan deduplicate berdasarkan `midtrans_order_id` — jika sudah diproses sebelumnya, abaikan dan kembalikan HTTP 200. Untuk status `settlement` atau `capture`, sistem HARUS mengupdate payment status ke PAID dan mempublikasikan event `payment.succeeded`. Untuk status `deny`, `cancel`, `expire`, sistem HARUS mempublikasikan event `payment.failed`.

**BILLING-FR-005 — Activate Subscription**
Sistem HARUS mengonsumsi event `payment.succeeded` dan mengupdate atau membuat subscription record dengan status ACTIVE, `started_at = NOW()`, `expires_at = NOW() + 30 hari`.

**BILLING-FR-006 — Cancel Subscription**
Sistem HARUS mengupdate subscription status ke CANCELLED. Akses fitur berbayar HARUS tetap berlaku hingga `expires_at`. Sistem HARUS mempublikasikan event `subscription.cancelled`.

**BILLING-FR-007 — Tier Limit Check (gRPC)**
Sistem HARUS mengekspos gRPC endpoint `CheckTierLimit(user_id, resource_type, current_count) → (allowed bool, limit int32, http_status int32)`. Resource types: `financial_accounts`, `investment_instruments`, `monthly_transactions`. Logika: ambil tier user dari tabel subscriptions, bandingkan current_count dengan limit plan. Jika melebihi, return allowed=false dan http_status=402.

**BILLING-FR-008 — Subscription Expiry Scheduler**
Sistem HARUS memiliki scheduler yang berjalan setiap hari untuk: (1) mempublikasikan event `subscription.expiring` untuk subscription yang akan berakhir dalam 3 hari, (2) mempublikasikan event `subscription.expired` dan mengupdate status ke EXPIRED untuk subscription yang sudah melewati `expires_at`.

**BILLING-FR-009 — Auto-downgrade**
Sistem HARUS mengonsumsi event `subscription.expired` dan memastikan user dikembalikan ke plan FREE.

**BILLING-FR-010 — Invoice List**
Sistem HARUS mengembalikan daftar payment records milik user yang terautentikasi, diurutkan dari yang terbaru, dengan pagination.

### 3.4 Finance Service

Semua query di finance-service WAJIB menggunakan tenant schema prefix berdasarkan `user_id` dari JWT claim.

**FINANCE-FR-001 — Create Financial Account**
Sistem HARUS memanggil billing-service gRPC `CheckTierLimit` untuk resource_type `financial_accounts` sebelum INSERT. Jika tidak diizinkan, return HTTP 402. Field wajib: `name` (1–100 karakter), `account_type` (checking/savings/cash/credit/investment/other), `currency` (default IDR), `initial_balance`. Sistem HARUS menyimpan record di `tenant_{id}.financial_accounts` dan mencatat entri awal di `balance_history`.

**FINANCE-FR-002 — List Financial Accounts**
Sistem HARUS mengembalikan semua akun keuangan dengan `is_deleted = false` milik tenant yang terautentikasi, beserta `current_balance` terkini.

**FINANCE-FR-003 — Update Financial Account**
Sistem HARUS memungkinkan update `name` dan `account_type` saja. Saldo tidak dapat diubah langsung — harus melalui transaksi.

**FINANCE-FR-004 — Archive Financial Account**
Sistem HARUS melakukan soft delete: set `is_deleted = true`, `deleted_at = NOW()`. Data historis saldo tetap dipertahankan di `balance_history`.

**FINANCE-FR-005 — Balance History**
Sistem HARUS mengembalikan riwayat perubahan saldo akun. Data yang dikembalikan HARUS difilter berdasarkan batas retensi tier: Free = 3 bulan, Basic = 12 bulan, Pro = semua. Data di luar batas retensi tetap ada di database, hanya tidak dikembalikan di response.

### 3.5 Transaction Service

Semua query di transaction-service WAJIB menggunakan tenant schema prefix.

**TRX-FR-001 — Create Transaction**
Sistem HARUS memeriksa quota transaksi bulanan via billing-service gRPC `CheckTierLimit` untuk resource_type `monthly_transactions` dengan current_count = jumlah transaksi bulan kalender berjalan (berdasarkan `date` field, bukan `created_at`). Jika melebihi, return HTTP 402. Field wajib: `account_id`, `amount` (desimal positif), `type` (income/expense/transfer), `category_id`, `date`, `notes` (opsional, maks 500 karakter).

**TRX-FR-002 — List Transactions**
Sistem HARUS mendukung filter: `account_id`, `category_id`, `type`, `date_from`, `date_to`. Sistem HARUS mengimplementasikan cursor-based atau offset pagination. Data yang dikembalikan HARUS difilter berdasarkan batas retensi tier.

**TRX-FR-003 — Update Transaction**
Sistem HARUS memungkinkan update semua field kecuali `transaction_id` dan `created_at`.

**TRX-FR-004 — Delete Transaction**
Sistem HARUS melakukan soft delete: `is_deleted = true`, `deleted_at = NOW()`. Setelah soft delete, jumlah transaksi dalam quota bulanan HARUS didecremen.

**TRX-FR-005 — Manage Categories**
Sistem HARUS mendukung CRUD untuk kategori milik tenant. Default kategori di-seed saat tenant provisioning. Soft delete untuk kategori yang sudah memiliki transaksi terkait.

**TRX-FR-006 — Export CSV (Pro Only)**
Sistem HARUS memverifikasi tier Pro dari JWT claim sebelum mengeksekusi. Sistem HARUS mengeksport semua transaksi dalam rentang tanggal yang diminta ke format CSV dengan header kolom: `date`, `type`, `category`, `account`, `amount`, `notes`.

### 3.6 Investment Service

Semua query di investment-service WAJIB menggunakan tenant schema prefix.

**INV-FR-001 — Create Investment Instrument**
Sistem HARUS memanggil billing-service gRPC `CheckTierLimit` untuk resource_type `investment_instruments` sebelum INSERT. Jika tidak diizinkan, return HTTP 402. Field wajib: `name` (1–100 karakter), `instrument_type` (crypto/gold/mutual_fund/stock/other), `units` (desimal positif), `buy_price_idr` (harga rata-rata beli), `coin_id` (opsional — untuk mapping ke CoinGecko), `is_manual_price` (boolean).

**INV-FR-002 — List Investment Instruments**
Sistem HARUS mengembalikan daftar instrumen aktif (`is_deleted = false`) beserta nilai pasar terkini. Untuk instrumen dengan `is_manual_price = false` dan memiliki `coin_id`, sistem HARUS memanggil price-service via gRPC `GetPrice(coin_id)` untuk mendapat harga terkini.

**INV-FR-003 — Update Investment Instrument**
Sistem HARUS mendukung update: `name`, `units` (beli atau jual parsial), `buy_price_idr`, `manual_price_idr` (jika `is_manual_price = true`), `coin_id`. Setiap perubahan `units` HARUS mencatat entri di `unit_history`.

**INV-FR-004 — Archive Investment Instrument**
Sistem HARUS melakukan soft delete. Data di `unit_history` tetap dipertahankan.

**INV-FR-005 — Get Current Price**
Sistem HARUS mengembalikan harga terkini instrumen dari price-service via gRPC. Response HARUS menyertakan field `is_fresh` yang mengindikasikan apakah harga masih dalam cache TTL 15 menit atau sudah stale.

**INV-FR-006 — Unit History**
Sistem HARUS mengembalikan riwayat perubahan unit instrumen, difilter berdasarkan batas retensi tier.

### 3.7 Price Service (Rust)

**PRICE-FR-001 — Fetch CoinGecko Price**
Sistem HARUS mengambil harga dari CoinGecko API untuk coin_id yang diminta. Sistem HARUS menggunakan parameterized request dan timeout 5 detik. Harga HARUS disimpan ke tabel `price_cache` dengan `expires_at = NOW() + 900 seconds` menggunakan UPSERT (UPDATE jika coin_id sudah ada).

**PRICE-FR-002 — Fetch metals.live Price**
Sistem HARUS mengambil harga emas (XAU/USD) dari metals.live API. Sistem HARUS mengkonversi harga USD ke IDR menggunakan kurs yang dikonfigurasi via environment variable `GOLD_USD_IDR_RATE`. Disimpan ke `price_cache` dengan key khusus emas.

**PRICE-FR-003 — Cache Validation**
Sebelum mengambil data dari API eksternal, sistem HARUS memeriksa apakah cache masih valid (`expires_at > NOW()`). Jika valid, return data dari cache tanpa memanggil API eksternal.

**PRICE-FR-004 — Graceful Fallback**
Jika API eksternal mengembalikan error atau timeout, sistem HARUS mengembalikan harga terakhir yang tersimpan di cache dengan flag `is_fresh: false`. Sistem tidak boleh mengembalikan error kepada pemanggil hanya karena API eksternal tidak tersedia.

**PRICE-FR-005 — gRPC GetPrice Endpoint**
Sistem HARUS mengekspos gRPC endpoint `GetPrice(coin_id string) → (price_idr float64, price_usd float64, is_fresh bool, updated_at timestamp)` yang dipanggil oleh investment-service.

### 3.8 Sync Service (Rust)

Semua operasi sync WAJIB diverifikasi bahwa `user_id` dari JWT claim cocok dengan tenant schema yang diakses.

**SYNC-FR-001 — Push Sync**
Sistem HARUS menerima batch operasi offline dalam format `[{operation_id, entity_type, entity_id, operation: create|update|delete, payload, client_timestamp}]`. Sistem HARUS memproses setiap operasi secara berurutan berdasarkan `client_timestamp`.

**SYNC-FR-002 — Pull Sync**
Sistem HARUS mengembalikan semua perubahan yang terjadi di server setelah `last_sync_timestamp` yang dikirim oleh client.

**SYNC-FR-003 — Conflict Resolution (Server Wins)**
Jika terjadi konflik antara operasi client dan data server (data di server dimodifikasi setelah `client_timestamp`), sistem HARUS memilih data server sebagai kebenaran akhir. Sistem HARUS mengembalikan data server yang menang kepada client untuk update IndexedDB.

**SYNC-FR-004 — Idempotency**
Setiap operasi sync HARUS menyertakan `sync_id` unik. Sistem HARUS mengabaikan operasi yang memiliki `sync_id` yang sudah pernah diproses sebelumnya (check via partial unique index di `sync_log`).

**SYNC-FR-005 — Sync Log**
Sistem HARUS mencatat setiap operasi sync yang berhasil ke tabel `tenant_{id}.sync_log` untuk audit trail.

**SYNC-FR-006 — Tenant Isolation**
Sistem HARUS memverifikasi bahwa `user_id` dari JWT claim identik dengan `user_id` yang diekstrak dari nama tenant schema yang diakses. Jika tidak cocok, sistem HARUS mengembalikan HTTP 403 dan mencatat security event.

### 3.9 Notification Service

**NOTIF-FR-001 — Welcome Email with Verification Link**
Sistem HARUS mengonsumsi event `user.registered` dari RabbitMQ. Sistem HARUS mengirim email welcome kepada alamat email user yang baru terdaftar. Email HARUS menyertakan link verifikasi dengan format `https://{domain}/verify-email?token={raw_verification_token}`.

**NOTIF-FR-002 — Password Reset Email**
Sistem HARUS mengirim email reset password saat dipicu. Email HARUS menyertakan link dengan format `https://{domain}/reset-password?token={raw_reset_token}`. Token dalam URL adalah token asli (bukan hash).

**NOTIF-FR-003 — Payment Receipt Email**
Sistem HARUS mengonsumsi event `payment.succeeded`. Email HARUS menyertakan: nama plan, jumlah pembayaran dalam IDR, metode pembayaran, tanggal pembayaran, dan periode aktif subscription.

**NOTIF-FR-004 — Payment Failed Alert**
Sistem HARUS mengonsumsi event `payment.failed`. Email HARUS memberikan informasi tentang kegagalan dan instruksi untuk mencoba ulang melalui billing portal.

**NOTIF-FR-005 — Subscription Expiry Warning**
Sistem HARUS mengonsumsi event `subscription.expiring`. Email HARUS memberitahu pengguna bahwa subscription akan berakhir dalam 3 hari beserta link untuk memperpanjang.

**NOTIF-FR-006 — Subscription Cancelled Confirmation**
Sistem HARUS mengonsumsi event `subscription.cancelled`. Email HARUS mengkonfirmasi pembatalan dan memberitahu kapan akses fitur berbayar berakhir.

**NOTIF-FR-007 — Subscription Expired Notification**
Sistem HARUS mengonsumsi event `subscription.expired`. Email HARUS memberitahu bahwa akun telah di-downgrade ke tier Free beserta penjelasan fitur apa yang tidak lagi tersedia.

**NOTIF-FR-008 — Retry dan DLQ**
Sistem HARUS mengimplementasikan retry dengan exponential backoff (maks 3 kali) jika pengiriman email gagal. Setelah retry maksimal, pesan HARUS dipindahkan ke Dead Letter Queue `kasku.events.dlq` untuk investigasi manual.

### 3.10 Admin Service

Admin-service TIDAK BOLEH dapat diakses dari internet publik. Semua akses harus melalui jaringan internal atau VPN.

**ADMIN-FR-001 — Admin Login**
Sistem HARUS mengautentikasi admin menggunakan kredensial dari tabel `admin_users` di kasku_admin DB. Autentikasi HARUS terpisah sepenuhnya dari user auth. Sistem HARUS membuat admin JWT dengan claim `admin_id` dan `role`. Argon2id HARUS digunakan untuk password hashing admin.

**ADMIN-FR-002 — Platform Statistics Dashboard**
Sistem HARUS mengembalikan agregat platform untuk bulan berjalan: total user terdaftar, total user aktif (login dalam 30 hari), distribusi user per tier (FREE/BASIC/PRO), MRR bulan berjalan (sum payment amounts dengan status PAID dalam 30 hari), dan churn rate (persentase subscription CANCELLED atau EXPIRED terhadap total subscription aktif bulan lalu).

**ADMIN-FR-003 — List Users**
Sistem HARUS mengembalikan daftar user dengan pagination (cursor-based). Filter yang didukung: `subscription_tier`, `is_active`, `email_verified`, `created_after`, `created_before`. Sistem HARUS memquery kasku_auth DB untuk data user dan kasku_billing DB untuk data subscription.

**ADMIN-FR-004 — User Detail**
Sistem HARUS mengembalikan detail lengkap satu user: profil (dari kasku_auth), subscription aktif (dari kasku_billing), usage stats (jumlah transaksi bulan ini, jumlah akun keuangan, jumlah instrumen investasi — dari kasku_finance tenant schema).

**ADMIN-FR-005 — Suspend User**
Sistem HARUS mengupdate kolom `is_active = false` di tabel `users` (kasku_auth). Sistem HARUS mencatat aksi ini ke tabel `admin_audit_log` dengan `admin_id`, `action`, `target_user_id`, dan `timestamp`. User yang disuspend HARUS mendapat HTTP 403 pada semua request.

**ADMIN-FR-006 — Activate User**
Sistem HARUS mengupdate `is_active = true`. Sistem HARUS mencatat ke `admin_audit_log`.

**ADMIN-FR-007 — Override Subscription**
Sistem HARUS memungkinkan admin mengubah subscription plan user tanpa payment. Sistem HARUS mencatat ke `admin_audit_log` dengan menyertakan `reason` yang diisi oleh admin.

**ADMIN-FR-008 — List All Payments**
Sistem HARUS mengembalikan semua payment records dari kasku_billing DB dengan pagination. Filter: `status`, `plan_id`, `date_from`, `date_to`, `user_id`.

### 3.11 API Gateway

**GATEWAY-FR-001 — JWT Verification**
api-gateway HARUS memvalidasi JWT access token dari header `Authorization: Bearer {token}` pada setiap request ke protected endpoint. Validasi: signature dengan public key RSA auth-service, `exp` claim belum kadaluwarsa, token tidak ada di Redis blacklist. Jika tidak valid, return HTTP 401.

**GATEWAY-FR-002 — Rate Limiting**
api-gateway HARUS menerapkan rate limiting via Redis: auth endpoints (register, login, forgot-password) = 10 request/menit per IP, semua endpoint lain = 200 request/menit per user_id. Jika limit terlampaui, return HTTP 429 dengan header `Retry-After`.

**GATEWAY-FR-003 — Tier Header Injection**
Setelah JWT validation berhasil, api-gateway HARUS menginjeksikan headers ke downstream services: `X-User-ID`, `X-User-Email`, `X-Subscription-Tier`, `X-Correlation-ID` (UUID v4 per request). Downstream services HARUS menggunakan header ini, bukan melakukan JWT decode sendiri.

**GATEWAY-FR-004 — CORS**
api-gateway HARUS menerapkan CORS dengan `Access-Control-Allow-Origin` yang hanya mengizinkan origin yang dikonfigurasi via environment variable `ALLOWED_ORIGINS`.

**GATEWAY-FR-005 — Routing**
api-gateway HARUS merutekan request ke service yang sesuai berdasarkan path prefix: `/api/v1/auth/*` → auth-service, `/api/v1/users/*` → user-service, `/api/v1/billing/*` → billing-service, `/api/v1/accounts/*` → finance-service, `/api/v1/transactions/*` → transaction-service, `/api/v1/investments/*` → investment-service, `/api/v1/sync/*` → sync-service.

---

## 4. Non-Functional Requirements

### 4.1 Performance Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-PERF-01 | API Gateway routing overhead per request | < 10ms |
| NFR-PERF-02 | Endpoint response time P95 (aplikasi) | < 300ms |
| NFR-PERF-03 | Endpoint response time P99 (aplikasi) | < 800ms |
| NFR-PERF-04 | Waktu load halaman pertama (cached, offline) | < 1 detik |
| NFR-PERF-05 | Sync batch 100 operasi | < 3 detik |
| NFR-PERF-06 | Price cache TTL | 15 menit (900 detik) |
| NFR-PERF-07 | External API timeout (CoinGecko, metals.live) | 5 detik |
| NFR-PERF-08 | Midtrans webhook response time | < 2 detik |
| NFR-PERF-09 | RabbitMQ message processing time P95 | < 500ms |
| NFR-PERF-10 | gRPC inter-service latency P95 | < 20ms |

### 4.2 Security Requirements

| ID | Requirement |
|----|-------------|
| NFR-SEC-01 | Password hashing: Argon2id dengan memory=64MB, iterations=3, parallelism=2 |
| NFR-SEC-02 | JWT: RS256 (RSA 4096-bit). Access token TTL 15 menit. Refresh token TTL 7 hari |
| NFR-SEC-03 | Refresh token disimpan di HttpOnly + Secure + SameSite=Strict cookie |
| NFR-SEC-04 | HTTPS wajib di seluruh komunikasi (TLS 1.2+, dikelola oleh Traefik) |
| NFR-SEC-05 | Rate limiting: auth endpoints 10 req/mnt per IP, API endpoint 200 req/mnt per user |
| NFR-SEC-06 | Input validation wajib di setiap service boundary sebelum diproses |
| NFR-SEC-07 | CORS hanya mengizinkan origin yang dikonfigurasi via environment variable |
| NFR-SEC-08 | Semua query database menggunakan parameterized query — string interpolation dilarang |
| NFR-SEC-09 | Semua secret dan credential dari environment variables, tidak pernah hardcoded |
| NFR-SEC-10 | Content-Security-Policy header diterapkan di api-gateway |
| NFR-SEC-11 | Midtrans webhook: signature key verification wajib sebelum memproses payload |
| NFR-SEC-12 | Tenant isolation: setiap query di kasku_finance wajib menggunakan tenant schema yang diverifikasi dari JWT claim |
| NFR-SEC-13 | Admin-service tidak terekspos ke public internet — internal network only |
| NFR-SEC-14 | gRPC inter-service menggunakan mTLS di production (opsional di development) |
| NFR-SEC-15 | JWT blacklist di Redis untuk token invalidation saat logout |
| NFR-SEC-16 | Refresh token disimpan sebagai SHA-256 hash di database, bukan plaintext |
| NFR-SEC-17 | Email enumeration prevention: forgot password selalu return HTTP 200 tanpa mengungkap apakah email terdaftar |
| NFR-SEC-18 | Password reset link menggunakan token sekali pakai (one-time use) dengan TTL 1 jam |

### 4.3 Reliability Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-REL-01 | Platform uptime SLA | ≥ 99.5% per bulan |
| NFR-REL-02 | RabbitMQ Dead Letter Queue untuk event yang gagal diproses setelah retry maksimal | Wajib |
| NFR-REL-03 | Sync retry dengan exponential backoff | Maksimal 5 retry, base delay 1 detik |
| NFR-REL-04 | Graceful degradation jika price API tidak tersedia | App tetap berjalan penuh, harga ditampilkan sebagai stale |
| NFR-REL-05 | Graceful shutdown semua service | SIGTERM handler wajib, drain in-flight requests maksimal 30 detik |
| NFR-REL-06 | Database connection pool dengan health check | Wajib di semua service |
| NFR-REL-07 | Idempotency webhook Midtrans | Deduplicate via midtrans_order_id sebelum memproses |
| NFR-REL-08 | Notification email retry | 3 kali dengan exponential backoff sebelum masuk DLQ |

### 4.4 Scalability Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-SCALE-01 | Semua service harus stateless (tidak menyimpan session di memory) untuk horizontal scaling | Wajib |
| NFR-SCALE-02 | Target beban awal di satu VPS (2 vCPU, 4 GB RAM) | 1.000 concurrent users |
| NFR-SCALE-03 | Schema-per-tenant memungkinkan migrasi ke DB-per-tenant tanpa perubahan application code jika diperlukan | Wajib dipertimbangkan dalam desain |
| NFR-SCALE-04 | Tidak ada shared mutable state antar service (stateless design) | Wajib |

### 4.5 Maintainability Requirements

| ID | Requirement |
|----|-------------|
| NFR-MAIN-01 | Clean Architecture (domain → usecase → infrastructure → delivery) diterapkan per service |
| NFR-MAIN-02 | Structured JSON logging menggunakan zerolog (Go) dengan field: `level`, `time`, `service`, `correlation_id`, `message`. PII tidak boleh muncul di log |
| NFR-MAIN-03 | Semua konfigurasi dari environment variables (12-factor app) |
| NFR-MAIN-04 | Database migration menggunakan migration-as-code: golang-migrate (Go services), sqlx-migrate atau Refinery (Rust services) |
| NFR-MAIN-05 | Semua API public endpoint menggunakan prefix `/api/v1/` |
| NFR-MAIN-06 | Health check endpoint `GET /health` wajib di setiap service, mengembalikan status DB dan dependency |
| NFR-MAIN-07 | Dependency injection diterapkan di semua layer untuk testability |
| NFR-MAIN-08 | Unit test untuk semua use case, integration test untuk semua repository |

### 4.6 Compliance Requirements

| ID | Requirement |
|----|-------------|
| NFR-COMP-01 | UU PDP No. 27/2022: data pribadi (email, display_name) dienkripsi at-rest di PostgreSQL |
| NFR-COMP-02 | Right to erasure: `DELETE /me` memicu soft delete user + event `user.deletion_requested` → `deprovision_tenant()` |
| NFR-COMP-03 | PII (email, IP address) tidak boleh muncul di application log. Security audit log terpisah |
| NFR-COMP-04 | Audit log wajib untuk semua aksi admin yang mengakses atau memodifikasi data user |
| NFR-COMP-05 | Zero telemetri — tidak ada analytics pihak ketiga, tidak ada tracking, tidak ada iklan |
| NFR-COMP-06 | Tidak ada penjualan atau sharing data pengguna ke pihak ketiga manapun |

### 4.7 Portability Requirements

| ID | Requirement |
|----|-------------|
| NFR-PORT-01 | Docker image untuk setiap service, mendukung platform Linux/amd64 dan Linux/arm64 |
| NFR-PORT-02 | Target browser: Chrome ≥ 100, Firefox ≥ 100, Safari ≥ 15 |
| NFR-PORT-03 | PWA harus dapat diinstall di Android dan iOS via browser |
| NFR-PORT-04 | Seluruh environment dapat direproduksi dari `docker-compose.yml` + `.env` file |

---

## 5. System Interfaces

### 5.1 User Interface

Frontend dibangun sebagai SvelteKit PWA yang responsif dan dapat diinstall. Requirement UI:

| Requirement | Detail |
|-------------|--------|
| Responsive design | Berfungsi penuh pada lebar layar ≥ 375px (smartphone) hingga desktop |
| PWA installable | Manifest valid, Service Worker terdaftar, meet PWA installability criteria |
| Dark / light mode | Mengikuti preferensi sistem (prefers-color-scheme) |
| Offline indicator | Banner atau badge yang jelas saat pengguna sedang offline |
| Sync status | Indikator teks/ikon yang menampilkan status: synced / X pending / sync error |
| Aksesibilitas | Semantic HTML, ARIA labels pada form dan navigasi |

### 5.2 External System Interfaces

| Sistem | Tipe | Endpoint/Protocol | Service Owner |
|--------|------|-------------------|---------------|
| CoinGecko API v3 | REST/HTTPS outbound | `https://api.coingecko.com/api/v3/simple/price` | price-service |
| metals.live API | REST/HTTPS outbound | metals.live endpoint (dikonfigurasi via env) | price-service |
| Midtrans Snap API | REST/HTTPS outbound | `https://app.midtrans.com/snap/v1/transactions` | billing-service |
| Midtrans Webhook | REST/HTTPS inbound | `POST /api/v1/billing/webhook/midtrans` | billing-service via api-gateway |
| SMTP Server / SendGrid | SMTP/HTTPS outbound | Dikonfigurasi via `SMTP_HOST:SMTP_PORT` | notification-service |

### 5.3 Software Interfaces

| Service | Bahasa | HTTP Port | gRPC Port | Framework / Library Utama | Database |
|---------|--------|-----------|-----------|---------------------------|----------|
| api-gateway | Go ≥1.22 | :8080 | — | Gin, grpc-go, go-redis/v9, golang-jwt | Redis |
| auth-service | Go ≥1.22 | :8081 | :9081 | Gin, grpc-go, pgx/v5, golang-migrate, golang-jwt, crypto/argon2 | kasku_auth |
| user-service | Go ≥1.22 | :8082 | :9082 | Gin, grpc-go, pgx/v5, amqp091-go | kasku_finance (DDL) |
| billing-service | Go ≥1.22 | :8083 | :9083 | Gin, grpc-go, pgx/v5, amqp091-go, midtrans-go | kasku_billing |
| finance-service | Go ≥1.22 | :8084 | :9084 | Gin, grpc-go, pgx/v5 | kasku_finance |
| transaction-service | Go ≥1.22 | :8085 | :9085 | Gin, grpc-go, pgx/v5, encoding/csv | kasku_finance |
| investment-service | Go ≥1.22 | :8086 | :9086 | Gin, grpc-go, pgx/v5 | kasku_finance |
| price-service | Rust stable | :8087 | :9087 | Axum, Tonic, SQLx, tokio, reqwest | kasku_price |
| sync-service | Rust stable | :8088 | — | Axum, Tonic, SQLx, tokio | kasku_finance |
| notification-service | Go ≥1.22 | :8089 | — | amqp091-go, gomail/v2 | — |
| admin-service | Go ≥1.22 | :8090 | — | Gin, pgx/v5 | kasku_admin, kasku_auth (R), kasku_billing (R) |
| Frontend | TypeScript | — | — | SvelteKit ≥2.0, Svelte 5, @vite-pwa/sveltekit, Dexie.js ≥4.0, Workbox | IndexedDB |
| PostgreSQL | — | :5432 | — | PostgreSQL ≥14 | 5 database |
| RabbitMQ | — | :5672 | — | RabbitMQ ≥3.12 (AMQP 0-9-1) | — |
| Redis | — | :6379 | — | Redis ≥7 | — |
| Traefik | — | :80/:443 | — | Traefik v3, Let's Encrypt | — |

### 5.4 Komunikasi Inter-Service

| Pengirim | Penerima | Protokol | Trigger |
|----------|---------|----------|---------|
| api-gateway | billing-service | gRPC | Setiap request CREATE ke finance/transaction/investment service |
| api-gateway | auth-service | gRPC | Verifikasi JWT (atau via shared public key) |
| investment-service | price-service | gRPC | GET harga terkini instrumen |
| auth-service | RabbitMQ | AMQP publish | `user.registered` |
| billing-service | RabbitMQ | AMQP publish | `payment.succeeded`, `payment.failed`, `subscription.expiring`, `subscription.expired`, `subscription.cancelled` |
| user-service | RabbitMQ | AMQP consume | `user.registered`, `user.deletion_requested` |
| billing-service | RabbitMQ | AMQP consume | `payment.succeeded`, `subscription.expired` |
| notification-service | RabbitMQ | AMQP consume | Semua events |

---

## 6. Data Requirements

### 6.1 Data Retention per Tier

| Data Type | Free Tier | Basic Tier | Pro Tier |
|-----------|-----------|------------|---------|
| Riwayat saldo (balance_history) | 3 bulan terakhir | 12 bulan terakhir | Unlimited |
| Daftar transaksi | 3 bulan terakhir | 12 bulan terakhir | Unlimited |
| Riwayat unit investasi (unit_history) | 3 bulan terakhir | 12 bulan terakhir | Unlimited |
| Sync log | 3 bulan terakhir | 12 bulan terakhir | Unlimited |

**Penting:** Data di luar batas retensi tetap tersimpan di database dan tidak dihapus secara otomatis. Filter retensi diterapkan di application layer saat query. Jika user upgrade tier, seluruh data historis kembali terlihat.

### 6.2 Data Privacy dan Compliance

| Prinsip | Implementasi |
|---------|-------------|
| Tenant isolation | Setiap tenant memiliki schema PostgreSQL terpisah di kasku_finance. Tidak ada query yang dapat mengakses data lintas tenant |
| PII di-rest | Email dan display_name dienkripsi at-rest di PostgreSQL (kolom bertipe pgcrypto atau encryption layer) sesuai NFR-COMP-01 |
| PII di-transit | Semua komunikasi menggunakan TLS 1.2+ (Traefik) |
| Right to erasure | `DELETE /me` → soft delete user → event → `deprovision_tenant()` → DROP SCHEMA. Data benar-benar dihapus dari storage |
| No telemetry | Tidak ada analytics tracker, pixel tracking, atau reporting ke pihak ketiga |
| Log hygiene | Email, IP, dan data pribadi lainnya tidak muncul di application log |
| Audit trail | Semua aksi admin ke data user dicatat di `admin_audit_log` dengan timestamp, admin_id, dan action type |

### 6.3 Backup dan Recovery

Backup adalah tanggung jawab operator VPS. Rekomendasi:
- `pg_dump` harian untuk semua 5 database, disimpan di lokasi terpisah dari VPS
- Retensi backup minimal 30 hari
- RTO (Recovery Time Objective) yang disarankan: < 4 jam
- RPO (Recovery Point Objective) yang disarankan: < 24 jam

### 6.4 Data Volume Estimation

| Entity | Estimasi per Tenant per Tahun | Total (1.000 tenant) |
|--------|------------------------------|----------------------|
| Transaksi | ~1.200 rows (100/bulan rata-rata) | ~1,2 juta rows |
| Balance history | ~365 rows | ~365.000 rows |
| Unit history | ~100 rows | ~100.000 rows |
| Sync log | ~2.400 rows | ~2,4 juta rows |
| **Total kasku_finance** | ~4.000 rows | ~4 juta rows |

Estimasi storage kasku_finance untuk 1.000 tenant pada tahun pertama: < 2 GB.

---

## 7. Quality Attributes Summary

| Atribut | Requirement | Metric |
|---------|-------------|--------|
| Availability | Uptime SLA ≥ 99.5% | Downtime maksimal 3,65 jam/bulan |
| Performance | P95 latency < 300ms | Diukur dari api-gateway hingga response |
| Security | OWASP Top 10 compliance | Zero critical/high vulnerability |
| Security | Tenant isolation | Zero data leakage lintas tenant (integration test) |
| Reliability | Graceful degradation | App berfungsi saat price API down atau SMTP down |
| Reliability | Event delivery | DLQ untuk semua failed events setelah 3 retry |
| Scalability | Concurrent users | 1.000 concurrent users pada 1 VPS (2 vCPU, 4 GB RAM) |
| Maintainability | Clean Architecture | Setiap service memiliki layer terpisah: domain, usecase, infrastructure, delivery |
| Compliance | UU PDP No. 27/2022 | PII dienkripsi, right to erasure, no telemetry, audit log |
| Offline capability | Offline-first | 100% CRUD operasi berhasil saat IndexedDB tersedia dan browser online |
| Portability | Multi-platform | Docker image amd64 + arm64, PWA di Chrome/Firefox/Safari |

---

*Dokumen SRS ini bersifat living document. Perubahan requirement harus melalui review arsitektur dan diupdate di dokumen ini sebelum implementasi.*
