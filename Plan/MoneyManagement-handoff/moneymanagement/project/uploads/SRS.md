# SRS — Software Requirements Specification
## KasKu: Personal Finance Tracker
### IEEE 830 / ISO 29148 Compliant

**Versi:** 1.0.0  
**Tanggal:** 2026-04-22  
**Author:** TubsAMY  
**Status:** DRAFT

---

## 1. Introduction

### 1.1 Purpose
Dokumen ini mendefinisikan spesifikasi kebutuhan perangkat lunak untuk sistem **KasKu**, sebuah aplikasi web pencatatan keuangan pribadi berbasis teknologi Go (backend) dan SvelteKit (frontend), yang berjalan secara offline-first dengan sinkronisasi ke server on-premise.

### 1.2 Scope
Sistem mencakup:
- Manajemen akun keuangan (bank, e-wallet)
- Manajemen instrumen investasi (emas, kripto)
- Pencatatan transaksi pemasukan dan pengeluaran
- Offline capability dengan sync otomatis
- Integrasi harga real-time dari API publik gratis

### 1.3 Definitions

| Istilah | Definisi |
|---|---|
| Saldo Keuangan | Total saldo likuid dari rekening bank atau e-wallet |
| Saldo Investasi | Total nilai aset investasi berdasarkan unit × harga pasar |
| Instrumen | Jenis aset investasi (Emas, Bitcoin, Ethereum, dsb.) |
| Transaksi | Catatan pemasukan atau pengeluaran dengan nominal, kategori, tanggal |
| Sync Queue | Antrian operasi yang dilakukan saat offline, menunggu sinkronisasi |
| Conflict | Kondisi di mana data lokal dan server berbeda untuk record yang sama |

### 1.4 References
- IEEE 830-1998 Software Requirements Specification
- ISO/IEC 29148:2018 Requirements Engineering
- CoinGecko API v3 Documentation
- metals.live API Documentation
- OWASP Top 10 2021

---

## 2. Overall Description

### 2.1 Product Perspective
KasKu adalah sistem standalone yang di-deploy di server on-premise pengguna. Tidak bergantung pada layanan cloud pihak ketiga. Satu-satunya koneksi eksternal adalah ke API harga real-time (CoinGecko, metals.live).

### 2.2 Product Functions

```
KasKu
├── Autentikasi & Sesi
├── Manajemen Akun Keuangan
├── Manajemen Instrumen Investasi
│   └── Integrasi Harga Real-time
├── Pencatatan Transaksi
├── Manajemen Kategori
├── Offline Engine
│   ├── IndexedDB Local Store
│   └── Background Sync
└── Dashboard & Ringkasan
```

### 2.3 User Characteristics
Pengguna tunggal dengan kemampuan teknis untuk men-deploy aplikasi sendiri (Docker / binary). Familiar dengan konsep keuangan pribadi dan investasi.

### 2.4 Constraints
- Harus berjalan di server on-premise (tidak ada ketergantungan AWS/GCP/Azure)
- Database: PostgreSQL ≥ 14
- Go ≥ 1.22
- SvelteKit ≥ 2.0 dengan Svelte 5
- Browser target: Chromium-based terbaru, Firefox terbaru
- Service Worker diperlukan untuk offline capability

### 2.5 Assumptions and Dependencies
- Pengguna memiliki server yang berjalan dan dapat diakses dari browser
- Server memiliki koneksi internet untuk fetch harga real-time
- Browser mendukung Service Worker dan IndexedDB

---

## 3. Functional Requirements

### 3.1 Autentikasi (AUTH)

**AUTH-FR-001: Login**
- Sistem menerima username dan password
- Password di-hash menggunakan Argon2id sebelum disimpan
- Gagal login > 5 kali dalam 15 menit → lockout 15 menit
- Response: JWT access token (15 menit) + refresh token (7 hari) via HttpOnly cookie

**AUTH-FR-002: Refresh Token**
- Refresh token rotation: setiap penggunaan refresh token menghasilkan token baru
- Token lama di-invalidate setelah digunakan (single-use)

**AUTH-FR-003: Logout**
- Invalidasi refresh token di database
- Hapus cookie di browser

**AUTH-FR-004: Proteksi Endpoint**
- Semua endpoint non-publik wajib JWT valid
- Unauthorized → HTTP 401

---

### 3.2 Manajemen Akun Keuangan (ACCOUNT)

**ACC-FR-001: Tambah Akun**
- Input: nama akun, tipe (bank/e-wallet/cash), saldo awal, mata uang (default IDR), warna/ikon opsional
- Nama akun unik per pengguna

**ACC-FR-002: Edit Akun**
- Semua field dapat diubah kecuali ID
- Perubahan saldo dicatat di tabel `balance_history`

**ACC-FR-003: Hapus Akun**
- Soft delete (is_deleted flag)
- Transaksi terkait tetap tersimpan (referential integrity)

**ACC-FR-004: Update Saldo**
- Saldo dapat diupdate manual
- Setiap perubahan saldo mencatat snapshot ke `balance_history`

**ACC-FR-005: List Akun**
- Tampilkan semua akun aktif dengan saldo terkini
- Tampilkan total agregat di atas daftar

---

### 3.3 Manajemen Investasi (INVESTMENT)

**INV-FR-001: Tambah Instrumen**
- Input: nama, tipe (GOLD / CRYPTO / STOCK / OTHER), simbol (XAU, BTC, ETH), jumlah unit, satuan (gram, coin, lot)
- Tipe GOLD → fetch dari metals.live
- Tipe CRYPTO → fetch dari CoinGecko berdasarkan coin_id

**INV-FR-002: Edit Instrumen**
- Semua field dapat diubah
- Perubahan unit dicatat di `unit_history`

**INV-FR-003: Hapus Instrumen**
- Soft delete

**INV-FR-004: Harga Real-time**
- Server meng-cache harga per instrumen selama 15 menit
- Jika API gagal atau timeout → tampilkan harga terakhir yang tersimpan + label "Harga tidak terkini"
- Jika tidak ada data harga sama sekali → tampilkan unit saja tanpa nilai IDR

**INV-FR-005: Tampilan Nilai**
- Nilai IDR = jumlah_unit × harga_per_unit_IDR
- Total agregat investasi = sum(nilai_IDR semua instrumen aktif)

---

### 3.4 Transaksi (TRANSACTION)

**TRX-FR-001: Tambah Transaksi**
- Input wajib: nominal (> 0), tipe (INCOME/EXPENSE), kategori_id, tanggal
- Input opsional: catatan (max 500 karakter), akun_keuangan_id
- Nominal minimum: Rp 1 (mendukung transaksi terkecil apapun)

**TRX-FR-002: Edit Transaksi**
- Semua field dapat diubah
- Audit: updated_at diperbarui

**TRX-FR-003: Hapus Transaksi**
- Soft delete

**TRX-FR-004: List & Filter Transaksi**
- Filter: tanggal dari-sampai, kategori, tipe (INCOME/EXPENSE), akun
- Default: 30 hari terakhir
- Pagination: 50 record per halaman

**TRX-FR-005: Ringkasan**
- Total pemasukan, total pengeluaran, net per periode yang dipilih

---

### 3.5 Kategori (CATEGORY)

**CAT-FR-001:** Tambah kategori (nama, tipe: INCOME/EXPENSE/BOTH, ikon opsional)  
**CAT-FR-002:** Edit kategori  
**CAT-FR-003:** Hapus kategori — hanya jika tidak ada transaksi aktif yang menggunakan kategori ini  
**CAT-FR-004:** Seed kategori default saat setup pertama (Makan, Transport, Belanja, Gaji, dll.)

---

### 3.6 Offline & Sinkronisasi (SYNC)

**SYNC-FR-001: Offline Operation**
- Seluruh operasi CRUD (transaksi, update saldo, update unit investasi) harus dapat dilakukan tanpa koneksi internet
- Data disimpan di IndexedDB dengan struktur mirror database server

**SYNC-FR-002: Sync Queue**
- Setiap operasi offline menghasilkan entry di `sync_queue` (IndexedDB)
- Entry berisi: operation_type (CREATE/UPDATE/DELETE), entity_type, payload, created_at, status (pending/synced/failed)

**SYNC-FR-003: Auto-Sync**
- Background sync dijalankan saat browser mendeteksi koneksi tersedia
- Urutan sync: sesuai urutan waktu operasi (FIFO)
- Setiap item di queue dikirim ke endpoint `POST /api/v1/sync/batch`

**SYNC-FR-004: Conflict Resolution**
- Strategy: **Server Wins** — jika timestamp server > timestamp lokal, data server menang
- Konflik yang terdeteksi dicatat di log, tidak melempar error ke pengguna
- Pengguna dapat melihat status sync (ikon di header: ✓ synced / ⟳ syncing / ✗ failed)

**SYNC-FR-005: Initial Load**
- Saat pertama kali online setelah offline, sistem melakukan full pull dari server ke IndexedDB

---

## 4. Non-Functional Requirements

### 4.1 Performance

| ID | Requirement | Target |
|---|---|---|
| NFR-PERF-01 | Waktu respons API (P95) | < 200ms (local network) |
| NFR-PERF-02 | Waktu load halaman (cached, offline) | < 1 detik |
| NFR-PERF-03 | Sync batch (100 operasi) | < 3 detik |
| NFR-PERF-04 | Cache harga investasi | 15 menit TTL |
| NFR-PERF-05 | Timeout fetch API eksternal | 5 detik |

### 4.2 Security

| ID | Requirement |
|---|---|
| NFR-SEC-01 | Password hashing: Argon2id (memory=64MB, iterations=3, parallelism=2) |
| NFR-SEC-02 | JWT: RS256, access token TTL 15 menit, refresh token TTL 7 hari |
| NFR-SEC-03 | Refresh token disimpan di database (untuk invalidasi), cookie HttpOnly + Secure + SameSite=Strict |
| NFR-SEC-04 | HTTPS wajib di production (TLS 1.2 minimum) |
| NFR-SEC-05 | Rate limiting: 10 req/menit untuk endpoint auth, 200 req/menit untuk endpoint lain |
| NFR-SEC-06 | Input validation di semua entry point (backend) |
| NFR-SEC-07 | CORS hanya mengizinkan origin yang dikonfigurasi |
| NFR-SEC-08 | SQL query menggunakan parameterized query (no string concatenation) |
| NFR-SEC-09 | Secret dari environment variable, tidak pernah hardcoded |
| NFR-SEC-10 | CSP header dikonfigurasi (Content-Security-Policy) |

### 4.3 Reliability

| ID | Requirement |
|---|---|
| NFR-REL-01 | Operasi offline harus tidak kehilangan data (durable di IndexedDB) |
| NFR-REL-02 | Sync retry otomatis dengan exponential backoff (max 5 retry) |
| NFR-REL-03 | Graceful degradation: jika API harga gagal, aplikasi tetap berjalan |

### 4.4 Maintainability

| ID | Requirement |
|---|---|
| NFR-MAIN-01 | Clean Architecture: domain tidak bergantung pada framework |
| NFR-MAIN-02 | Structured logging (JSON) dengan level dan correlation ID |
| NFR-MAIN-03 | Configuration via environment variables (12-factor app) |
| NFR-MAIN-04 | Database migration menggunakan golang-migrate |
| NFR-MAIN-05 | API versioning (/api/v1/) |

### 4.5 Portability

| ID | Requirement |
|---|---|
| NFR-PORT-01 | Docker image tersedia untuk deployment mudah |
| NFR-PORT-02 | Berjalan di Linux x86_64 dan ARM64 |
| NFR-PORT-03 | Browser support: Chrome 100+, Firefox 100+, Safari 15+ |

---

## 5. System Interfaces

### 5.1 User Interface
- Web application berbasis SvelteKit
- Responsif: mobile (≥ 375px) dan desktop
- PWA: dapat di-install di homescreen, bekerja offline
- Dark/Light mode

### 5.2 External Interfaces

#### CoinGecko API
```
Host: https://api.coingecko.com
Endpoint: GET /api/v3/simple/price
Query: ids={coin_id}&vs_currencies=idr
Rate Limit: 30 call/menit (free tier)
```

#### metals.live API
```
Host: https://metals.live
Endpoint: GET /api/v1/spot
Response: array of {metal, price_usd}
Catatan: harga dalam USD, konversi ke IDR menggunakan rate dari CoinGecko (XAU via /simple/price?ids=xau&vs_currencies=idr atau alternatif)
```

### 5.3 Software Interfaces

| Komponen | Teknologi | Versi |
|---|---|---|
| Backend Runtime | Go | ≥ 1.22 |
| Web Framework | Gin | ≥ 1.9 |
| ORM | GORM | ≥ 1.25 |
| Migration | golang-migrate | ≥ 4.17 |
| Database | PostgreSQL | ≥ 14 |
| Frontend | SvelteKit | ≥ 2.0 |
| Svelte | Svelte | 5.x |
| PWA | @vite-pwa/sveltekit | latest |
| Offline DB | Dexie.js (IndexedDB wrapper) | ≥ 4.0 |

---

## 6. Data Requirements

### 6.1 Data Retention
- Transaksi: tidak ada penghapusan otomatis (soft delete saja)
- Harga cache: 15 menit TTL di memory (Redis opsional, default in-memory)
- Sync queue lokal: dihapus setelah berhasil sync

### 6.2 Data Privacy
- Semua data tersimpan di server on-premise pengguna
- Tidak ada telemetri atau pengiriman data ke pihak ketiga
- Tidak ada logging data sensitif (nominal transaksi tidak di-log di level INFO)

---

## 7. Quality Attributes Summary

| Atribut | Skala | Target |
|---|---|---|
| Availability | Bergantung uptime server pengguna | N/A |
| Performance | API latency P95 | < 200ms |
| Security | OWASP compliance | Top 10 mitigated |
| Usability | Operasi offline penuh | ✓ |
| Maintainability | Clean Architecture | ✓ |
