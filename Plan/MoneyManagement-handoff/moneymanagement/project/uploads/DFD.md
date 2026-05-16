# DFD — Data Flow Diagram
## KasKu: Personal Finance Tracker

**Versi:** 1.0.0  
**Tanggal:** 2026-04-22  

---

## Level 0 — Context Diagram

Diagram konteks menunjukkan KasKu sebagai sistem tunggal dengan entitas eksternal yang berinteraksi dengannya.

```mermaid
flowchart LR
    USER(["👤 Pengguna"])
    COINGECKO(["🌐 CoinGecko API"])
    METALS(["🌐 metals.live API"])

    subgraph KASKU ["🏦 Sistem KasKu"]
        CORE["KasKu Core"]
    end

    USER -- "Login, CRUD Keuangan\nCRUD Investasi\nCRUD Transaksi" --> KASKU
    KASKU -- "Dashboard, Laporan\nStatus Sync" --> USER
    KASKU -- "GET harga kripto" --> COINGECKO
    COINGECKO -- "Harga IDR real-time" --> KASKU
    KASKU -- "GET harga emas" --> METALS
    METALS -- "Harga USD emas" --> KASKU
```

---

## Level 1 — System Overview

Menunjukkan proses-proses utama di dalam sistem KasKu.

```mermaid
flowchart TB
    USER(["👤 Pengguna"])

    subgraph BROWSER ["🌐 Browser (SvelteKit PWA)"]
        P1["P1\nAutentikasi"]
        P2["P2\nManajemen\nAkun Keuangan"]
        P3["P3\nManajemen\nInvestasi"]
        P4["P4\nPencatatan\nTransaksi"]
        P5["P5\nOffline\nSync Engine"]
        IDB[("IndexedDB\nLocal Store")]
    end

    subgraph SERVER ["🖥️ Server On-Premise (Go)"]
        P6["P6\nAuth Service"]
        P7["P7\nAccount Service"]
        P8["P8\nInvestment Service"]
        P9["P9\nTransaction Service"]
        P10["P10\nPrice Cache\nService"]
        P11["P11\nSync Handler"]
        DB[("PostgreSQL")]
    end

    EXT1(["CoinGecko API"])
    EXT2(["metals.live API"])

    USER --> P1 --> P6
    USER --> P2 --> P7
    USER --> P3 --> P8
    USER --> P4 --> P9
    P5 <--> IDB
    P5 <-->|"sync batch"| P11

    P6 <--> DB
    P7 <--> DB
    P8 <--> DB
    P9 <--> DB
    P11 <--> DB
    P10 <--> DB

    P8 --> P10
    P10 --> EXT1
    P10 --> EXT2
```

---

## Level 2 — P1: Proses Autentikasi

```mermaid
flowchart LR
    USER(["👤 Pengguna"])
    D1[("users\ntable")]
    D2[("refresh_tokens\ntable")]

    P1_1["P1.1\nValidasi Kredensial"]
    P1_2["P1.2\nGenerate JWT\n+ Refresh Token"]
    P1_3["P1.3\nRefresh Access Token"]
    P1_4["P1.4\nLogout &\nInvalidasi Token"]

    USER -- "username + password" --> P1_1
    P1_1 -- "query user" --> D1
    D1 -- "user record" --> P1_1
    P1_1 -- "verified" --> P1_2
    P1_2 -- "simpan refresh token" --> D2
    P1_2 -- "JWT + HttpOnly Cookie" --> USER

    USER -- "refresh token (cookie)" --> P1_3
    P1_3 -- "validasi token" --> D2
    P1_3 -- "new JWT" --> USER

    USER -- "logout request" --> P1_4
    P1_4 -- "invalidate token" --> D2
```

---

## Level 2 — P2: Proses Manajemen Akun Keuangan

```mermaid
flowchart LR
    USER(["👤 Pengguna"])
    D3[("financial_accounts\ntable")]
    D4[("balance_history\ntable")]

    P2_1["P2.1\nTambah Akun"]
    P2_2["P2.2\nUpdate Saldo"]
    P2_3["P2.3\nHapus Akun\n(soft delete)"]
    P2_4["P2.4\nList & Agregat\nSaldo"]

    USER -- "nama, tipe, saldo awal" --> P2_1
    P2_1 -- "insert" --> D3

    USER -- "saldo baru" --> P2_2
    P2_2 -- "update saldo" --> D3
    P2_2 -- "catat snapshot" --> D4

    USER -- "hapus akun" --> P2_3
    P2_3 -- "is_deleted=true" --> D3

    USER -- "minta daftar" --> P2_4
    D3 -- "daftar akun + saldo" --> P2_4
    P2_4 -- "list + total agregat" --> USER
```

---

## Level 2 — P3: Proses Manajemen Investasi & Harga

```mermaid
flowchart TB
    USER(["👤 Pengguna"])
    D5[("investment_assets\ntable")]
    D6[("price_cache\ntable")]
    EXT1(["CoinGecko API"])
    EXT2(["metals.live API"])

    P3_1["P3.1\nTambah/Edit\nInstrumen"]
    P3_2["P3.2\nFetch Harga\nReal-time"]
    P3_3["P3.3\nKalkulasi\nNilai IDR"]
    P3_4["P3.4\nTampilkan\nPortofolio"]

    USER -- "instrumen, unit" --> P3_1
    P3_1 --> D5

    P3_2 -- "GET /simple/price" --> EXT1
    P3_2 -- "GET /api/v1/spot" --> EXT2
    EXT1 -- "harga IDR" --> P3_2
    EXT2 -- "harga USD" --> P3_2
    P3_2 -- "cache 15 menit" --> D6
    D6 -- "harga cached" --> P3_2

    D5 -- "unit per instrumen" --> P3_3
    D6 -- "harga per unit" --> P3_3
    P3_3 -- "nilai IDR = unit × harga" --> P3_4
    P3_4 -- "portofolio + total" --> USER
```

---

## Level 2 — P4: Proses Pencatatan Transaksi

```mermaid
flowchart LR
    USER(["👤 Pengguna"])
    D7[("transactions\ntable")]
    D8[("categories\ntable")]

    P4_1["P4.1\nValidasi\nInput Transaksi"]
    P4_2["P4.2\nSimpan\nTransaksi"]
    P4_3["P4.3\nFilter & List\nTransaksi"]
    P4_4["P4.4\nHitung\nRingkasan"]

    USER -- "nominal, tipe,\nkategori, tanggal" --> P4_1
    D8 -- "validasi kategori" --> P4_1
    P4_1 -- "valid" --> P4_2
    P4_2 --> D7

    USER -- "filter params" --> P4_3
    D7 -- "filtered records" --> P4_3
    P4_3 --> P4_4
    P4_4 -- "list + total in/out/net" --> USER
```

---

## Level 2 — P5: Proses Offline Sync

```mermaid
flowchart TB
    USER(["👤 Pengguna"])
    IDB[("IndexedDB\nBrowser")]
    SERVER["Server\nSync Handler"]

    P5_1["P5.1\nDeteksi\nStatus Koneksi"]
    P5_2["P5.2\nEnqueue\nOperasi Offline"]
    P5_3["P5.3\nBackground\nSync Worker"]
    P5_4["P5.4\nConflict\nResolution"]
    P5_5["P5.5\nUpdate\nLocal Mirror"]

    USER -- "CRUD saat offline" --> P5_2
    P5_2 -- "simpan ke queue" --> IDB
    IDB -- "queue + local data" --> USER

    P5_1 -- "online detected" --> P5_3
    P5_3 -- "baca queue" --> IDB
    P5_3 -- "POST /sync/batch" --> SERVER
    SERVER -- "response + server state" --> P5_4
    P5_4 -- "server wins resolution" --> P5_5
    P5_5 -- "update mirror data" --> IDB
    P5_5 -- "hapus queue item" --> IDB
    P5_5 -- "status: synced ✓" --> USER
```

---

## Data Store Summary

| Data Store | Lokasi | Teknologi | Deskripsi |
|---|---|---|---|
| D1: users | Server | PostgreSQL | Kredensial pengguna |
| D2: refresh_tokens | Server | PostgreSQL | Token refresh aktif |
| D3: financial_accounts | Server | PostgreSQL | Akun keuangan |
| D4: balance_history | Server | PostgreSQL | Riwayat perubahan saldo |
| D5: investment_assets | Server | PostgreSQL | Instrumen dan unit investasi |
| D6: price_cache | Server | PostgreSQL / Memory | Cache harga real-time |
| D7: transactions | Server | PostgreSQL | Semua transaksi |
| D8: categories | Server | PostgreSQL | Kategori transaksi |
| IDB: IndexedDB | Browser | Dexie.js | Mirror data + sync queue |
