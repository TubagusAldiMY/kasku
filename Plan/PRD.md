# PRD — Product Requirements Document
## KasKu: Personal Finance SaaS Platform

**Versi:** 2.0.0
**Tanggal:** 2026-04-27
**Author:** TubsAMY (admin@tubsamy.tech)
**Status:** DRAFT
**Changelog:** v2.0.0 — Pivot dari single-user on-premise monolith ke multi-tenant SaaS dengan subscription billing, 11 microservices, dan notifikasi email.

---

## Daftar Isi

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Goals dan Non-Goals](#3-goals-dan-non-goals)
4. [Target User](#4-target-user)
5. [Feature Requirements](#5-feature-requirements)
6. [External API Integration](#6-external-api-integration)
7. [Technical Constraints](#7-technical-constraints)
8. [Success Metrics](#8-success-metrics)
9. [Roadmap](#9-roadmap)

---

## 1. Executive Summary

KasKu adalah platform SaaS manajemen keuangan pribadi multi-tenant yang ditargetkan untuk pengguna Indonesia. Platform ini memungkinkan pengguna mencatat, memantau, dan menganalisis kondisi keuangan mereka — mencakup rekening bank, aset investasi (kripto, emas, reksa dana, saham), dan transaksi harian — dalam satu dashboard terpusat.

**Proposisi nilai utama:**

- **Privasi data nyata**: Data pengguna disimpan di server KasKu yang dikelola langsung oleh TubsAMY, bukan di infrastruktur cloud pihak ketiga yang tidak transparan. Setiap pengguna mendapat schema PostgreSQL yang terisolasi penuh dari pengguna lain.
- **Offline-first**: Aplikasi tetap dapat digunakan sepenuhnya tanpa koneksi internet. Data tersinkronisasi otomatis saat koneksi kembali tersedia, dengan conflict resolution Server Wins yang deterministik.
- **Harga terjangkau**: Model freemium dengan tier Free yang fungsional, Basic Rp 29.000/bulan, dan Pro Rp 59.000/bulan — jauh di bawah rata-rata aplikasi finansial premium Indonesia.
- **Agregat real-time**: Harga aset kripto (via CoinGecko) dan emas (via metals.live) diperbarui otomatis dengan cache 15 menit sehingga nilai portofolio selalu mutakhir.
- **Zero telemetri**: Tidak ada analytics pihak ketiga, tidak ada iklan, tidak ada penjualan data pengguna ke siapapun.

Platform dibangun di atas arsitektur 11 microservices yang berjalan di VPS dengan Docker Compose, menggunakan Go dan Rust sebagai bahasa backend, SvelteKit 2.0 + Svelte 5 sebagai frontend PWA, dan PostgreSQL dengan schema-per-tenant sebagai lapisan data.

---

## 2. Problem Statement

### 2.1 Masalah Utama yang Diselesaikan

**Masalah 1 — Fragmentasi data keuangan**
Pengguna Indonesia umumnya memiliki beberapa rekening bank, dompet digital, reksa dana, saham, dan aset kripto di platform yang berbeda-beda. Tidak ada satu titik pandang yang menunjukkan total kekayaan bersih secara real-time. Pengguna harus membuka 5–10 aplikasi berbeda hanya untuk mendapat gambaran keuangan yang utuh.

**Masalah 2 — Privasi data pada aplikasi keuangan**
Aplikasi pencatatan keuangan populer (Wallet, Money Manager, dsb.) menyimpan data sensitif pengguna di cloud pihak ketiga, seringkali di infrastruktur asing. Pengguna yang peduli privasi tidak memiliki alternatif yang nyaman, terjangkau, dan tidak membutuhkan keahlian server administrator.

**Masalah 3 — Ketergantungan pada koneksi internet**
Mayoritas aplikasi keuangan berbasis web membutuhkan koneksi internet aktif. Di area dengan koneksi tidak stabil, pencatatan transaksi harian menjadi terganggu dan pengguna kehilangan habit pencatatan karena ketidaknyamanan teknis ini.

**Masalah 4 — Biaya aplikasi premium tidak proporsional**
Aplikasi keuangan dengan fitur lengkap (multi-aset, riwayat panjang, export data) umumnya mematok harga Rp 100.000–500.000/bulan, tidak sebanding dengan nilai yang diterima pengguna individu non-bisnis.

### 2.2 Solusi KasKu

KasKu menyelesaikan keempat masalah tersebut melalui:
- Dashboard agregat tunggal yang menggabungkan data dari semua kategori akun keuangan dan investasi
- Infrastruktur yang dioperasikan sendiri oleh TubsAMY dengan isolasi schema PostgreSQL per pengguna, tanpa third-party cloud besar
- PWA offline-first berbasis IndexedDB (Dexie.js) + Service Worker — pengguna tetap dapat mencatat dan membaca data tanpa internet
- Model freemium dengan tier Free yang fungsional untuk pengguna entry-level dan harga tier berbayar yang terjangkau

---

## 3. Goals dan Non-Goals

### 3.1 Goals

| Kategori | Goal |
|----------|------|
| Produk | Menyediakan dashboard keuangan pribadi yang mengagregasi rekening, investasi, dan transaksi dalam satu antarmuka |
| Produk | Mendukung pencatatan offline penuh dengan sinkronisasi otomatis saat koneksi tersedia |
| Produk | Menampilkan harga aset real-time dengan staleness maksimal 15 menit |
| Bisnis | Mengoperasikan model SaaS subscription yang menghasilkan MRR berkelanjutan |
| Bisnis | Mengelola siklus hidup subscription penuh: subscribe, upgrade, downgrade, cancel, expire, downgrade otomatis |
| Teknis | Membangun platform multi-tenant dengan isolasi data ketat menggunakan schema-per-tenant di PostgreSQL |
| Teknis | Menyediakan admin portal untuk monitoring platform dan manajemen pengguna |
| Teknis | Memastikan uptime ≥ 99.5% dengan graceful degradation saat layanan eksternal tidak tersedia |
| Keamanan | Mematuhi UU PDP No. 27/2022 — enkripsi data at-rest, right to erasure, zero telemetri |
| Keamanan | Mengimplementasikan OWASP Top 10 di setiap service boundary |

### 3.2 Non-Goals (Versi 2.0)

Fitur-fitur berikut secara eksplisit berada di luar cakupan KasKu v2.0:

| Non-Goal | Alasan Eksklusi |
|----------|-----------------|
| Multi-user per akun (family sharing) | Satu akun = satu tenant; sharing membutuhkan model permission yang berbeda secara fundamental |
| Open banking / screen scraping otomatis | Regulasi OJK kompleks, ketergantungan tinggi pada pihak ketiga yang berisiko |
| Laporan pajak otomatis (SPT) | Domain pajak membutuhkan keahlian hukum khusus, bukan cakupan platform keuangan personal |
| Mobile native app (iOS/Android) | PWA mencukupi untuk MVP; native app adalah roadmap fase selanjutnya |
| Budgeting rules / envelope budgeting | Ditargetkan Phase 5+ setelah core stabil |
| Notifikasi push (FCM/APNs) | Email notification mencukupi untuk v2; push notification adalah roadmap |
| Laporan keuangan PDF | Export CSV (Pro tier) mencukupi untuk v2 |
| Integrasi dengan broker saham lokal | Harga saham Indonesia membutuhkan lisensi data dari BEI |
| Peer-to-peer payments atau transfer | Bukan domain manajemen keuangan personal |

---

## 4. Target User

### 4.1 Persona A — Profesional Muda (Entry-Level User)

| Atribut | Detail |
|---------|--------|
| Demografis | Usia 22–35 tahun, karyawan atau freelancer, pendapatan Rp 5–15 juta/bulan |
| Kondisi keuangan | Memiliki 1–2 rekening bank, baru mulai berinvestasi di reksa dana atau saham |
| Pain point utama | Tidak tahu kemana uang habis tiap bulan, belum punya habit pencatatan yang konsisten |
| Kebutuhan | Antarmuka sederhana, gratis atau murah, dapat digunakan di smartphone |
| Tier yang relevan | Free (awalnya), Basic setelah terbiasa |
| Tech literacy | Sedang — dapat install PWA, tidak membutuhkan konfigurasi teknis |

### 4.2 Persona B — Investor Aktif (Core User)

| Atribut | Detail |
|---------|--------|
| Demografis | Usia 28–45 tahun, profesional atau wirausaha, pendapatan > Rp 15 juta/bulan |
| Kondisi keuangan | Memiliki 3–5 rekening bank, portofolio reksa dana, saham, kripto, dan/atau emas |
| Pain point utama | Data portofolio tersebar di banyak platform, tidak dapat melihat total net worth secara instan |
| Kebutuhan | Agregat multi-aset, harga real-time, riwayat setidaknya 1 tahun, privasi data |
| Tier yang relevan | Basic atau Pro |
| Tech literacy | Tinggi — nyaman dengan aplikasi keuangan, memahami konsep investasi |

### 4.3 Persona C — Power User (Pro Subscriber)

| Atribut | Detail |
|---------|--------|
| Demografis | Usia 30–50 tahun, investor berpengalaman atau akuntan freelance |
| Kondisi keuangan | Portofolio kompleks, data keuangan historis penting untuk analisis mandiri |
| Pain point utama | Membutuhkan data historis panjang, export untuk analisis di spreadsheet, privasi mutlak |
| Kebutuhan | Export CSV, riwayat unlimited, semua fitur tanpa batasan, transparansi data |
| Tier yang relevan | Pro |
| Tech literacy | Sangat tinggi — memahami konsep enkripsi dan privasi data |

### 4.4 Persona D — Platform Admin (Internal Operator)

| Atribut | Detail |
|---------|--------|
| Peran | Operator platform TubsAMY atau tim support internal |
| Akses | admin-service — hanya dapat diakses dari jaringan internal, tidak terekspos ke internet publik |
| Kebutuhan | Monitoring platform (MRR, total user, churn), manajemen user, override subscription untuk support |
| Auth | Akun admin terpisah di kasku_admin DB, tidak berbagi dengan user auth di kasku_auth |

---

## 5. Feature Requirements

### Konvensi Kolom

**ID** | **Deskripsi Fitur** | **Tier** | **Prioritas** | **Service**

Prioritas: **P0** = blocker MVP (tidak boleh release tanpa ini), **P1** = penting post-MVP, **P2** = nice-to-have

### 5.1 Modul Registrasi Akun

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-REG-01 | Self-service registration via email, username, dan password dengan validasi format dan uniqueness check | Semua | P0 | auth-service |
| F-REG-02 | Verifikasi email — token 24 jam, satu kali pakai. Pengguna harus memverifikasi sebelum dapat login pertama kali | Semua | P0 | auth-service + notification-service |
| F-REG-03 | Resend verification email — pengguna dapat meminta ulang email verifikasi jika token kadaluwarsa | Semua | P0 | auth-service + notification-service |
| F-REG-04 | Forgot password — pengguna memasukkan email, sistem mengirim reset link dengan token 1 jam | Semua | P0 | auth-service + notification-service |
| F-REG-05 | Reset password — validasi token reset, update password hash Argon2id, invalidate semua refresh token aktif milik user tersebut | Semua | P0 | auth-service |
| F-REG-06 | Delete account — soft delete user, publish event `user.deletion_requested` untuk memicu deprovision tenant schema | Semua | P1 | auth-service + user-service |

### 5.2 Modul Autentikasi

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-AUTH-01 | Login dengan email atau username + password | Semua | P0 | auth-service |
| F-AUTH-02 | JWT RS256 (access token 15 menit) + refresh token 7 hari disimpan di HttpOnly Secure SameSite=Strict cookie | Semua | P0 | auth-service + api-gateway |
| F-AUTH-03 | Refresh token rotation — token lama diinvalidasi di database saat token baru diterbitkan | Semua | P0 | auth-service |
| F-AUTH-04 | Logout — invalidasi refresh token aktif di database dan tambahkan access token ke JWT blacklist Redis | Semua | P0 | auth-service + api-gateway |
| F-AUTH-05 | Brute force protection — lockout akun 15 menit setelah 5 kali gagal login berturut-turut | Semua | P0 | auth-service |

### 5.3 Modul Subscription & Billing

Tier limits: **Free** = 50 transaksi/bulan, 3 akun keuangan, 2 instrumen investasi, riwayat 3 bulan | **Basic** = 500 transaksi/bulan, 10 akun, 10 instrumen, riwayat 1 tahun | **Pro** = unlimited semua.

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-BIL-01 | Tampilkan daftar plan (Free / Basic / Pro) beserta limit masing-masing, harga, dan fitur yang tersedia | Semua | P0 | billing-service |
| F-BIL-02 | Subscribe ke plan berbayar — generate Midtrans Snap transaction, return redirect URL ke halaman pembayaran | Basic/Pro | P0 | billing-service |
| F-BIL-03 | Webhook handler Midtrans — verifikasi signature key, deduplicate via midtrans_order_id, update payment status, publish event | Basic/Pro | P0 | billing-service |
| F-BIL-04 | Aktivasi subscription otomatis saat event `payment.succeeded` diterima dari RabbitMQ | Basic/Pro | P0 | billing-service |
| F-BIL-05 | Enforcement tier limits — gRPC endpoint dipanggil api-gateway sebelum operasi CREATE; return HTTP 402 Payment Required jika limit terlampaui | Semua | P0 | billing-service + api-gateway |
| F-BIL-06 | Auto-downgrade ke Free saat subscription expired — dipicu oleh event `subscription.expired` | Semua | P0 | billing-service + user-service |
| F-BIL-07 | Riwayat invoice dan pembayaran — list semua payment records dengan status | Basic/Pro | P1 | billing-service |
| F-BIL-08 | Cancel subscription — status CANCELLED, akses fitur berbayar tetap berlaku sampai akhir periode saat ini | Basic/Pro | P1 | billing-service |

### 5.4 Modul Saldo Keuangan

Batas tier: Free = 3 akun, Basic = 10 akun, Pro = tidak terbatas.

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-KU-01 | Buat akun keuangan baru (nama, tipe rekening, mata uang, saldo awal) — enforcement batas tier sebelum CREATE | Semua | P0 | finance-service |
| F-KU-02 | Tampilkan daftar semua akun keuangan aktif beserta saldo terkini | Semua | P0 | finance-service |
| F-KU-03 | Edit akun keuangan (nama, tipe) — perubahan saldo hanya dilakukan melalui pencatatan transaksi | Semua | P0 | finance-service |
| F-KU-04 | Arsipkan akun keuangan (soft delete) — data historis saldo tetap dipertahankan | Semua | P0 | finance-service |
| F-KU-05 | Riwayat perubahan saldo akun dengan filter periode — tampil sesuai batas retensi tier | Semua | P0 | finance-service |

### 5.5 Modul Saldo Investasi

Batas tier: Free = 2 instrumen, Basic = 10 instrumen, Pro = tidak terbatas.

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-INV-01 | Tambah instrumen investasi baru (nama, tipe: crypto/emas/reksa_dana/saham/lainnya, jumlah unit, harga beli) — enforcement batas tier | Semua | P0 | investment-service |
| F-INV-02 | Tampilkan daftar instrumen investasi aktif beserta nilai terkini (harga pasar × unit) | Semua | P0 | investment-service |
| F-INV-03 | Update jumlah unit instrumen (pembelian atau penjualan parsial) | Semua | P0 | investment-service |
| F-INV-04 | Arsipkan instrumen investasi (soft delete) — data historis unit tetap dipertahankan | Semua | P0 | investment-service |
| F-INV-05 | Tampilkan harga real-time dari price-service: kripto via CoinGecko, emas via metals.live, dengan indikator kesegaran data | Semua | P0 | investment-service + price-service |
| F-INV-06 | Input harga manual untuk instrumen yang tidak memiliki sumber harga otomatis (reksa dana, saham) | Semua | P0 | investment-service |
| F-INV-07 | Riwayat perubahan unit (unit_history) dengan filter periode sesuai batas retensi tier | Semua | P1 | investment-service |

### 5.6 Modul Transaksi

Batas tier: Free = 50 transaksi/bulan kalender, Basic = 500/bulan, Pro = tidak terbatas.

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-TRX-01 | Catat transaksi baru (akun, jumlah, tipe: income/expense/transfer, kategori, tanggal, catatan) — enforcement quota bulanan sebelum CREATE | Semua | P0 | transaction-service |
| F-TRX-02 | Tampilkan daftar transaksi dengan filter (akun, kategori, periode, tipe) dan pagination | Semua | P0 | transaction-service |
| F-TRX-03 | Edit transaksi yang sudah dicatat | Semua | P0 | transaction-service |
| F-TRX-04 | Hapus transaksi (soft delete) — jumlah transaksi dalam quota bulanan berkurang setelah soft delete | Semua | P0 | transaction-service |
| F-TRX-05 | Manajemen kategori (buat, edit, arsipkan) — tenant mendapat seed kategori default saat provisioning | Semua | P0 | transaction-service |
| F-TRX-06 | Export transaksi ke CSV dengan filter periode — hanya tersedia untuk tier Pro | Pro | P1 | transaction-service |

### 5.7 Modul Offline dan Sinkronisasi

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-SYNC-01 | Semua operasi CRUD dapat dilakukan saat offline — data disimpan di IndexedDB via Dexie.js dan di-queue untuk sync | Semua | P0 | Frontend + sync-service |
| F-SYNC-02 | Sinkronisasi otomatis saat koneksi internet terdeteksi kembali | Semua | P0 | sync-service |
| F-SYNC-03 | Sync manual — pengguna dapat memicu sinkronisasi kapan saja via tombol di UI | Semua | P0 | sync-service |
| F-SYNC-04 | Conflict resolution Server Wins — data di server adalah kebenaran akhir; perubahan lokal yang konflikt di-overwrite | Semua | P0 | sync-service |
| F-SYNC-05 | Sync status indicator — UI menampilkan status real-time: synced, pending operations, atau sync error | Semua | P0 | Frontend |

### 5.8 Modul Notifikasi Email

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-NOT-01 | Email selamat datang beserta link verifikasi email — dikirim segera setelah registrasi | Semua | P0 | notification-service |
| F-NOT-02 | Email link reset password — dikirim saat pengguna mengajukan forgot password | Semua | P0 | notification-service |
| F-NOT-03 | Email konfirmasi pembayaran berhasil (receipt) — dikirim saat event `payment.succeeded` | Basic/Pro | P0 | notification-service |
| F-NOT-04 | Email notifikasi pembayaran gagal — dikirim saat event `payment.failed` | Basic/Pro | P0 | notification-service |
| F-NOT-05 | Email peringatan subscription akan berakhir dalam 3 hari — dikirim saat event `subscription.expiring` | Basic/Pro | P1 | notification-service |
| F-NOT-06 | Email konfirmasi pembatalan subscription — dikirim saat event `subscription.cancelled` | Basic/Pro | P1 | notification-service |
| F-NOT-07 | Email notifikasi subscription expired dan akun di-downgrade ke Free — dikirim saat event `subscription.expired` | Basic/Pro | P1 | notification-service |

### 5.9 Modul Dashboard

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-DASH-01 | Total net worth — agregat saldo semua akun keuangan aktif + nilai pasar semua instrumen investasi aktif | Semua | P0 | api-gateway (aggregation) |
| F-DASH-02 | Ringkasan bulanan — total income, total expense, dan net cashflow bulan berjalan | Semua | P0 | api-gateway (aggregation) |
| F-DASH-03 | Grafik alokasi aset — visualisasi proporsi saldo keuangan vs nilai investasi per kategori | Semua | P1 | Frontend |

### 5.10 Modul Admin Platform

Admin-service hanya dapat diakses dari jaringan internal — tidak terekspos ke internet publik.

| ID | Deskripsi Fitur | Tier | Prioritas | Service |
|----|-----------------|------|-----------|---------|
| F-ADM-01 | Login admin — akun terpisah di kasku_admin DB, tidak menggunakan user auth | Admin | P0 | admin-service |
| F-ADM-02 | Dashboard platform — total user terdaftar, MRR bulan berjalan, distribusi subscription per tier, churn rate bulan ini | Admin | P0 | admin-service |
| F-ADM-03 | List dan cari users — pagination, filter berdasarkan tier, status akun, dan tanggal registrasi | Admin | P0 | admin-service |
| F-ADM-04 | Detail user — profile, subscription aktif, usage stats (transaksi bulan ini, jumlah akun keuangan, jumlah instrumen investasi) | Admin | P0 | admin-service |
| F-ADM-05 | Suspend user — menonaktifkan akses pengguna (is_active = false di users table) | Admin | P1 | admin-service |
| F-ADM-06 | Activate user — mengaktifkan kembali akun yang disuspend | Admin | P1 | admin-service |
| F-ADM-07 | Override subscription — admin dapat mengubah plan user tanpa payment, untuk keperluan support atau promo | Admin | P1 | admin-service |
| F-ADM-08 | List semua payments — filter berdasarkan status, tier, periode, dan user tertentu | Admin | P1 | admin-service |

---

## 6. External API Integration

### 6.1 CoinGecko API

| Atribut | Detail |
|---------|--------|
| Tujuan | Harga real-time aset kripto (BTC, ETH, dan altcoin lainnya) dalam IDR dan USD |
| Endpoint | `GET /api/v3/simple/price?ids={coin_id}&vs_currencies=idr,usd` |
| Tier API | Free tier (60 req/menit) — upgrade jika volume pengguna meningkat |
| Cache TTL | 15 menit di price-service (kasku_price DB) |
| Fallback | Kembalikan harga terakhir yang tersimpan beserta flag `is_fresh: false` |
| Service owner | price-service (Rust) |

### 6.2 metals.live API

| Atribut | Detail |
|---------|--------|
| Tujuan | Harga emas dan logam mulia dalam USD (XAU/USD, XAG/USD) |
| Konversi | Harga USD dikonversi ke IDR menggunakan kurs yang dikonfigurasi via environment variable |
| Cache TTL | 15 menit di price-service |
| Fallback | Kembalikan harga terakhir yang tersimpan beserta flag `is_fresh: false` |
| Service owner | price-service (Rust) |

### 6.3 Midtrans Payment Gateway

| Atribut | Detail |
|---------|--------|
| Tujuan | Memproses pembayaran subscription (subscribe, upgrade) |
| Metode pembayaran | QRIS, Virtual Account (BCA, BNI, BRI, Mandiri), GoPay, OVO, ShopeePay |
| Integrasi | Midtrans Snap API — server-side create transaction, frontend redirect ke Snap UI |
| Webhook endpoint | `POST /api/v1/billing/webhook/midtrans` |
| Verifikasi | Signature key verification wajib pada setiap webhook yang masuk sebelum diproses |
| Order ID format | `KASKU-{first_8_chars_user_id}-{unix_timestamp}` |
| Idempotency | Deduplicate via midtrans_order_id di kolom payments table sebelum proses webhook |
| Service owner | billing-service |

### 6.4 SMTP / Email Provider

| Atribut | Detail |
|---------|--------|
| Tujuan | Mengirim email transaksional (verifikasi akun, receipt pembayaran, alert notifikasi) |
| Provider | Configurable via env vars — SMTP standar atau SendGrid API |
| Config env vars | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM_ADDRESS` |
| Service owner | notification-service |

---

## 7. Technical Constraints

| Constraint | Detail |
|------------|--------|
| Architecture | Microservices — 11 service dengan isolated bounded context. Clean Architecture (domain → usecase → infrastructure → delivery) diterapkan per service |
| Backend Go | Go ≥ 1.22, digunakan di 9 service: api-gateway, auth-service, user-service, billing-service, finance-service, transaction-service, investment-service, notification-service, admin-service. Framework: Gin (HTTP) + grpc-go + pgx/v5 |
| Backend Rust | Rust stable channel, digunakan di 2 service: price-service dan sync-service. Framework: Axum (HTTP) + Tonic (gRPC) + SQLx + tokio |
| Frontend | SvelteKit ≥ 2.0 + Svelte 5 (Runes syntax), dibangun sebagai PWA installable. Target browser: Chrome ≥ 100, Firefox ≥ 100, Safari ≥ 15 |
| Database | PostgreSQL ≥ 14 (satu instance, 5 database: kasku_auth, kasku_billing, kasku_finance, kasku_price, kasku_admin). Schema per tenant di kasku_finance |
| Multi-tenancy | 1 akun = 1 tenant. Format schema: `tenant_{user_uuid_sanitized}` (karakter `-` diganti `_`). Provisioning via PostgreSQL function `provision_tenant(UUID)` |
| Message Broker | RabbitMQ ≥ 3.12. Exchange `kasku.events` (topic, durable). Dead Letter Exchange `kasku.events.dlx` untuk failed events |
| Inter-service sync | gRPC + Protobuf. Digunakan untuk: api-gateway → billing-service (tier limit check), investment-service → price-service (get price), auth-service → api-gateway (JWT public key distribution) |
| Inter-service async | RabbitMQ events. Semua event name menggunakan format `{domain}.{action}` lowercase: `user.registered`, `payment.succeeded`, `payment.failed`, `subscription.expiring`, `subscription.expired`, `subscription.cancelled` |
| Cache dan Rate Limit | Redis ≥ 7. Digunakan untuk rate limiting di api-gateway (10 req/menit auth, 200 req/menit per user) dan JWT blacklist |
| Auth | JWT RS256 (RSA 4096-bit). Access token 15 menit, refresh token 7 hari disimpan di HttpOnly Secure SameSite=Strict cookie |
| Password hashing | Argon2id: memory=64MB, iterations=3, parallelism=2 |
| Payment | Midtrans Snap API |
| Reverse proxy | Traefik v3 dengan TLS otomatis via Let's Encrypt |
| Deployment | Docker Compose v2 di VPS Linux. Minimum: 2 vCPU, 4 GB RAM |
| Offline Storage | IndexedDB via Dexie.js ≥ 4.0 di browser + Workbox Service Worker |
| Email | SMTP standar atau SendGrid (configurable via env vars) |
| Database Migration | golang-migrate untuk Go services, sqlx-migrate atau Refinery untuk Rust services |
| Config | 12-factor: semua konfigurasi dari environment variables, tidak ada hardcoded value |
| Logging | Structured JSON (zerolog di Go, tracing di Rust) dengan `correlation_id` per request. PII tidak boleh muncul di log |
| API versioning | Semua endpoint publik menggunakan prefix `/api/v1/` |

---

## 8. Success Metrics

### 8.1 Business Metrics (Target Month 6 Setelah Launch)

| Metrik | Target |
|--------|--------|
| Monthly Recurring Revenue (MRR) | ≥ Rp 5.000.000 |
| Monthly Active Users (MAU) | ≥ 500 |
| Total Registered Users | ≥ 1.000 |
| Free-to-Paid Conversion Rate | ≥ 10% |
| Monthly Churn Rate | ≤ 5% |
| Average Revenue Per User (ARPU) | ≥ Rp 35.000/bulan |

### 8.2 Technical Metrics

| Metrik | Target |
|--------|--------|
| API latency P95 (endpoint aplikasi) | < 300ms |
| API Gateway routing overhead | < 10ms |
| Uptime SLA | ≥ 99.5% |
| Waktu load halaman (cached, offline) | < 1 detik |
| Sinkronisasi setelah reconnect (100 operasi) | < 5 detik |
| Price cache staleness maksimal | 15 menit |
| Payment success rate (Midtrans) | ≥ 95% |
| Midtrans webhook processing time | < 2 detik |
| RabbitMQ message processing P95 | < 500ms |

### 8.3 Quality Metrics

| Metrik | Target |
|--------|--------|
| OWASP Top 10 compliance | 100% — zero critical vulnerability |
| Zero data leakage lintas tenant | 100% — verified via integration test |
| Email delivery rate | ≥ 98% |
| Offline CRUD success rate | 100% saat IndexedDB tersedia |

---

## 9. Roadmap

| Fase | Scope | Estimasi Durasi |
|------|-------|-----------------|
| **Phase 1 — Foundation (MVP)** | auth-service (register, verify email, login, refresh, logout, forgot/reset password, brute force protection), user-service (profile, tenant provisioning via `provision_tenant()`), billing-service (Free tier + plan listing + tier limit gRPC endpoint), api-gateway (routing, JWT verify, rate limit, tier header injection), finance-service (CRUD akun keuangan + balance history), transaction-service (CRUD transaksi + kategori) | 6–8 minggu |
| **Phase 2 — Investment & Sync** | investment-service (CRUD investasi, harga manual), price-service/Rust (CoinGecko + metals.live + cache TTL 15 menit), sync-service/Rust (offline sync engine, conflict resolution Server Wins), PWA offline-first lengkap (IndexedDB + Dexie.js + Workbox Service Worker), Frontend SvelteKit 2.0 + Svelte 5 | 3–4 minggu |
| **Phase 3 — Monetization** | Midtrans Snap integration penuh (create transaction, webhook handler, signature verification, idempotency), tier enforcement HTTP 402 di semua service, notification-service (semua 7 email event), upgrade/downgrade/cancel subscription, auto-downgrade saat expired, riwayat invoice | 2–3 minggu |
| **Phase 4 — Admin & Analytics** | admin-service (dashboard platform, user management, suspend/activate, override subscription), platform stats (MRR, churn rate, distribusi tier), export CSV Pro tier | 2–3 minggu |
| **Phase 5 — Growth** | Grafik historis keuangan dan investasi, laporan periodik bulanan, mobile PWA polish dan install prompt, query optimization dan caching layer, onboarding flow untuk user baru | TBD |

---

*Dokumen ini adalah single source of truth untuk product requirements KasKu v2.0. Setiap perubahan requirement harus diperbarui di dokumen ini sebelum diimplementasikan.*
