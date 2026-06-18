# KasKu Architecture Decision Records

Dokumen ini berisi keputusan arsitektur dan desain yang diambil selama pengembangan KasKu.
Claude Code membaca file ini untuk memahami konteks keputusan yang sudah final — sehingga opsi yang sudah ditolak tidak disarankan ulang.

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
