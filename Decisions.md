# KasKu Architecture Decision Records

Dokumen ini berisi keputusan arsitektur dan desain yang diambil selama pengembangan KasKu.
Claude Code membaca file ini untuk memahami konteks keputusan yang sudah final — sehingga opsi yang sudah ditolak tidak disarankan ulang.

---

# Port Registry KasKu

Blok port yang direservasi untuk KasKu local dev. Gunakan tabel ini sebagai referensi tunggal saat ada perubahan port.

## Prinsip Penamaan

| Jenis | Pola | Contoh |
|-------|------|--------|
| App services HTTP | `1808x` | api-gateway → `18080` |
| App services gRPC (exposed) | `1818x` | auth gRPC → `18181` |
| Infra database/cache | `1{port-standar}` | postgres 5432 → `15432`, redis 6379 → `16379` |
| Observability (100xx+) | `1{port-standar}` | prometheus 9090 → `19090`, grafana 3000 → `13000` |

## App Services (docker-compose.override.yml)

| Service | HTTP host | gRPC host |
|---------|-----------|-----------|
| api-gateway | 18080 | — |
| auth-service | 18081 | 18181 |
| user-service | 18082 | 18182 |
| billing-service | 18083 | 18183 |
| finance-service | 18084 | 18184 |
| transaction-service | 18085 | 18185 |
| investment-service | 18086 | 18186 |
| price-service | 18087 | 18187 |
| sync-service | 18088 | — |
| notification-service | 18089 | — |
| admin-service | 18090 | — |
| Frontend dev (vite) | 18173 | — |
| Frontend preview | 18300 | — |

## Infra (docker-compose.override.yml)

| Komponen | Host port | Container port | Mnemonic |
|----------|-----------|---------------|---------|
| PostgreSQL | **15432** | 5432 | 1 + 5432 |
| Redis | **16379** | 6379 | 1 + 6379 |
| RabbitMQ AMQP | 18672 | 5672 | suffix 672 |
| RabbitMQ UI | 18673 | 15672 | 18672 + 1 |

## Observability (docker-compose.yml, profile observability)

| Komponen | Host port | Container port | Mnemonic |
|----------|-----------|---------------|---------|
| Grafana | **13000** | 3000 | 1 + 3000 |
| Prometheus | **19090** | 9090 | 1 + 9090 |
| Alertmanager | **19093** | 9093 | 1 + 9093 |
| Loki | **13100** | 3100 | 1 + 3100 |
| Jaeger UI | 18686 | 16686 | suffix 686 |
| OTEL Collector gRPC | 18417 | 4317 | suffix 417 |
| OTEL Collector HTTP | 18418 | 4318 | suffix 418 |

---

# ADR-001: gRPC Internal Tanpa TLS (insecure.NewCredentials)

**Date:** 2026-06-18
**Status:** `Accepted`
**Deciders:** TubsAMY
**Project:** KasKu SaaS
**Tags:** #security #grpc #networking

---

## Context

`api-gateway` berkomunikasi dengan `billing-service` via gRPC pada port `:9083` untuk mengambil tier limits user (`GetUserTierLimits`). Koneksi dibuat di `api-gateway/internal/infrastructure/grpc/billing_client.go` menggunakan `insecure.NewCredentials()`, yang berarti tidak ada TLS/mTLS pada transport layer.

---

## Decision Drivers

- Semua service berjalan dalam Docker network yang terisolasi (`kasku-internal`) — tidak ada traffic yang melintas jaringan publik
- Overhead TLS untuk komunikasi intra-container menambah latency tanpa manfaat keamanan yang signifikan dalam setup ini
- Mengelola sertifikat internal (PKI) menambah kompleksitas operasional yang tidak sepadan untuk tim kecil
- gRPC call ini adalah hot path (dijalankan per-request) dengan timeout 300ms — overhead TLS handshake perlu diminimalkan

---

## Options Considered

### Option A: `insecure.NewCredentials()` ✅ DIPILIH

Tidak ada TLS pada koneksi gRPC internal. Traffic hanya beredar dalam network `kasku-internal` yang ter-isolasi oleh Docker.

**Pros:**
- Zero overhead handshake TLS
- Tidak perlu mengelola sertifikat internal
- Setup sederhana dan tidak ada rotasi sertifikat

**Cons:**
- Traffic dalam plaintext di dalam Docker network
- Jika Docker network dikompromikan (container escape), traffic gRPC bisa disadap

**Mitigasi risiko:**
- `kasku-internal` network ter-isolasi — hanya container yang secara eksplisit terdaftar di network ini yang bisa berkomunikasi
- `admin-service` dengan sengaja dikeluarkan dari `kasku-internal` (hanya `kasku-admin` + `kasku-data`)
- Semua autentikasi dan otorisasi dilakukan di application layer (JWT RS256), bukan di transport layer

---

### Option B: mTLS dengan sertifikat internal ❌ DITOLAK

Setiap service mendapat sertifikat client + server, dikelola via internal CA (Vault, cert-manager, atau self-signed).

**Alasan ditolak:** Kompleksitas operasional tidak sebanding untuk tim kecil dan infrastruktur single-node. Rotasi sertifikat memerlukan tooling tambahan. Masalah keamanan yang dimitigasi (sadap intra-container) sudah ter-handle oleh Docker network isolation.

---

### Option C: TLS satu arah (server-side only) ❌ DITOLAK

Server gRPC (billing-service) menyajikan sertifikat; client (api-gateway) hanya memverifikasi server.

**Alasan ditolak:** Memberikan perlindungan partial tanpa mitigasi penuh (tidak ada mutual auth). Tetap memerlukan pengelolaan sertifikat server. Trade-off tidak cukup menguntungkan vs. Option A untuk konteks ini.

---

## Consequences

- Seluruh komunikasi gRPC internal (`api-gateway` ↔ `billing-service`) berjalan tanpa enkripsi transport
- **Batas berlaku keputusan ini**: jika arsitektur berubah menjadi multi-node atau traffic gRPC melintas jaringan publik/shared, keputusan ini HARUS ditinjau ulang dan TLS/mTLS WAJIB diimplementasikan
- Jika ada service baru yang ditambahkan ke `kasku-internal` dan berkomunikasi via gRPC, keputusan ini berlaku secara default — dokumentasikan di sini

---

# ADR-002: validator v0.20 — Rencana Integrasi & Regression Test

**Date:** 2026-06-18
**Status:** `Accepted`
**Deciders:** TubsAMY
**Project:** KasKu SaaS (kasku-backend-monolith)
**Tags:** #security #validation #rust #monolith

---

## Context

`kasku-backend-monolith/Cargo.toml` mencantumkan `validator = { version = "0.20", features = ["derive"] }` sebagai dependency, tetapi crate ini **belum digunakan** di production code (`src/**/*.rs`) — tidak ada `#[derive(Validate)]` atau `.validate()` call.

Validator v0.19/v0.20 memperkenalkan breaking change pada API custom validator dibandingkan v0.18:
- v0.18: custom validator signature adalah `FnOnce`-based
- v0.20: custom validator menggunakan trait-based approach

---

## Decision

Saat crate `validator` mulai diintegrasikan ke handler DTOs monolith, wajib:

1. **Definisikan constraint lengkap** untuk semua DTO input (email format, password length min/max, amount range, string length, dll.) — jangan hanya derive `Validate` tanpa constraint eksplisit
2. **Jalankan regression test** untuk setiap field yang divalidasi, termasuk:
   - Input valid (happy path)
   - Input kosong / null
   - Input di batas bawah dan atas constraint
   - Input dengan karakter Unicode, emoji, injection attempts (`<script>`, `' OR '1'='1`)
3. **Custom validator wajib gunakan trait-based API v0.20** — jangan referensikan contoh kode dari dokumentasi v0.18
4. **Prioritas modul yang perlu validasi ketat**: `auth` (register, login, reset password), `billing` (plan selection), `finance` (account creation), `transaction` (amount, description)

## Consequences

Keputusan ini menjadi checklist yang harus diselesaikan sebelum modul pertama yang menggunakan `validator` di-merge ke production.
