# PRD — Product Requirements Document
## KasKu: Personal Finance Tracker

**Versi:** 1.0.0  
**Tanggal:** 2026-04-22  
**Author:** TubsAMY  
**Status:** DRAFT

---

## 1. Executive Summary

**KasKu** adalah aplikasi web personal untuk mencatat dan memantau kondisi keuangan secara komprehensif. Aplikasi ini memisahkan dengan tegas antara **Saldo Keuangan** (dana likuid di rekening bank/e-wallet) dan **Saldo Investasi** (aset non-kas seperti emas dan kripto). Dilengkapi dengan pencatatan transaksi pemasukan dan pengeluaran harian, serta kemampuan bekerja secara **offline-first** dengan sinkronisasi otomatis ke server on-premise milik pengguna.

---

## 2. Problem Statement

Pengguna membutuhkan satu titik pantau tunggal untuk:
- Mengetahui total saldo likuid dari berbagai rekening
- Mengetahui total nilai investasi berdasarkan unit/satuan (gram emas, jumlah BTC)
- Mencatat setiap transaksi keuangan sekecil apapun
- Mengakses data bahkan saat tidak ada koneksi internet
- Menjaga privasi penuh karena data hanya tersimpan di server sendiri

---

## 3. Goals & Non-Goals

### Goals ✅
- Menampilkan agregat saldo keuangan dari semua rekening yang terdaftar
- Menampilkan saldo investasi berdasarkan satuan (gram, BTC, dst.) beserta nilai real-time jika API tersedia
- Mencatat transaksi dengan nominal, kategori, dan tanggal
- Berjalan offline, sync otomatis saat online
- Berjalan di web browser, mobile-friendly (responsif)
- Data tersimpan di server on-premise pengguna

### Non-Goals ❌
- Multi-user / fitur kolaborasi
- Import otomatis dari rekening bank (open banking)
- Fitur budgeting / anggaran bulanan (fase berikutnya)
- Laporan pajak otomatis
- Notifikasi push / reminder tagihan

---

## 4. Target User

| Atribut | Detail |
|---|---|
| Pengguna | 1 orang (owner/admin tunggal) |
| Persona | Individu tech-savvy dengan server on-premise sendiri |
| Keahlian Teknis | Mampu deploy aplikasi sendiri |
| Perangkat Utama | Desktop browser, kadang mobile |

---

## 5. Feature Requirements

### 5.1 Modul Saldo Keuangan

| ID | Fitur | Prioritas |
|---|---|---|
| F-KU-01 | Tambah / edit / hapus akun keuangan (bank, e-wallet) | P0 |
| F-KU-02 | Update saldo manual per akun | P0 |
| F-KU-03 | Tampilkan total agregat saldo keuangan | P0 |
| F-KU-04 | Tampilkan breakdown per akun | P0 |
| F-KU-05 | Riwayat perubahan saldo per akun | P1 |

### 5.2 Modul Saldo Investasi

| ID | Fitur | Prioritas |
|---|---|---|
| F-INV-01 | Tambah / edit / hapus instrumen investasi | P0 |
| F-INV-02 | Input jumlah unit (gram emas, jumlah BTC, dsb.) | P0 |
| F-INV-03 | Harga real-time via API publik gratis (CoinGecko untuk kripto, metals.live untuk emas) | P0 |
| F-INV-04 | Fallback graceful jika API tidak tersedia (tampilkan unit saja) | P0 |
| F-INV-05 | Tampilkan nilai estimasi = unit × harga_realtime | P0 |
| F-INV-06 | Total agregat nilai investasi | P0 |
| F-INV-07 | Riwayat perubahan unit per instrumen | P1 |

### 5.3 Modul Transaksi (Buku Kas)

| ID | Fitur | Prioritas |
|---|---|---|
| F-TRX-01 | Catat transaksi: nominal, kategori, tanggal, catatan | P0 |
| F-TRX-02 | Tipe transaksi: Pemasukan / Pengeluaran | P0 |
| F-TRX-03 | Manajemen kategori (tambah, edit, hapus) | P0 |
| F-TRX-04 | Filter transaksi by tanggal, kategori, tipe | P0 |
| F-TRX-05 | Pencarian transaksi | P1 |
| F-TRX-06 | Ringkasan pemasukan vs pengeluaran per periode | P1 |

### 5.4 Offline & Sinkronisasi

| ID | Fitur | Prioritas |
|---|---|---|
| F-SYNC-01 | Aplikasi dapat diakses dan dioperasikan tanpa internet | P0 |
| F-SYNC-02 | Data offline disimpan di IndexedDB | P0 |
| F-SYNC-03 | Auto-sync ke server saat koneksi tersedia | P0 |
| F-SYNC-04 | Indikator status sync (synced / pending / conflict) | P0 |
| F-SYNC-05 | Conflict resolution: server wins (last-write-wins) | P1 |

### 5.5 Autentikasi

| ID | Fitur | Prioritas |
|---|---|---|
| F-AUTH-01 | Login dengan username + password | P0 |
| F-AUTH-02 | JWT RS256 dengan refresh token rotation | P0 |
| F-AUTH-03 | Session timeout configurable | P0 |

---

## 6. External API Integration

| Instrumen | API | Endpoint | Auth | Fallback |
|---|---|---|---|---|
| Bitcoin, Ethereum, dst. | CoinGecko | `GET /api/v3/simple/price?ids=bitcoin&vs_currencies=idr` | Tidak perlu | Tampilkan unit saja |
| Emas (XAU) | metals.live | `GET https://metals.live/api/v1/spot` | Tidak perlu | Tampilkan gram saja |

> **Catatan:** Konversi harga USD ke IDR menggunakan endpoint tambahan CoinGecko atau nilai IDR yang sudah tersedia di response. Rate di-cache 15 menit di server.

---

## 7. Technical Constraints

| Constraint | Detail |
|---|---|
| Backend | Go (Gin framework, Clean Architecture) |
| Frontend | SvelteKit (PWA, offline-first) |
| Database | PostgreSQL |
| Deployment | On-premise server milik pengguna |
| Offline Storage | IndexedDB (browser) |
| Auth | JWT RS256, secret tidak pernah di-hardcode |
| Data Privasi | Tidak ada third-party analytics, tidak ada CDN eksternal untuk data |

---

## 8. Success Metrics

| Metrik | Target |
|---|---|
| Waktu load halaman utama (cached) | < 1 detik |
| Operasi offline (CRUD transaksi) | Berfungsi penuh tanpa internet |
| Sync setelah reconnect | < 5 detik |
| Waktu update harga real-time | Cache 15 menit, refresh otomatis |
| Uptime server (on-premise) | Tanggung jawab pengguna |

---

## 9. Roadmap

| Fase | Scope | Estimasi |
|---|---|---|
| Phase 1 (MVP) | Auth, Saldo Keuangan, Saldo Investasi, Transaksi dasar | 4–6 minggu |
| Phase 2 | Offline sync penuh, real-time price, filter & search | 2–3 minggu |
| Phase 3 | Ringkasan periodik, grafik, riwayat saldo | 2–3 minggu |
| Phase 4 | Budgeting, export CSV/PDF | TBD |
