# DFD — Data Flow Diagram
## KasKu: Personal Finance SaaS Platform

**Versi:** 2.0.0
**Tanggal:** 2026-04-27
**Author:** TubsAMY (admin@tubsamy.tech)
**Status:** DRAFT
**Changelog:** v2.0.0 — Pivot ke SaaS Microservices. Diagram direkonstruksi penuh dari arsitektur on-premise monolith ke 11 microservices multi-tenant dengan RabbitMQ event bus, gRPC inter-service, schema-per-tenant PostgreSQL, dan Midtrans payment integration.

---

## Daftar Isi

1. [Level 0 — Context Diagram](#1-level-0--context-diagram)
2. [Level 1 — System Overview](#2-level-1--system-overview)
3. [Level 2 — Detail Diagrams](#3-level-2--detail-diagrams)
   - [P1: Registrasi dan Autentikasi](#p1-registrasi-dan-autentikasi)
   - [P2: Subscription dan Payment](#p2-subscription-dan-payment)
   - [P3: Tenant Provisioning](#p3-tenant-provisioning)
   - [P4: Manajemen Akun Keuangan](#p4-manajemen-akun-keuangan)
   - [P5: Manajemen Investasi dan Harga](#p5-manajemen-investasi-dan-harga)
   - [P6: Pencatatan Transaksi](#p6-pencatatan-transaksi)
   - [P7: Offline Sync](#p7-offline-sync)
   - [P8: Notifikasi Email](#p8-notifikasi-email)
   - [P9: Admin Platform](#p9-admin-platform)
4. [Data Store Summary](#4-data-store-summary)
5. [Event Contract Summary](#5-event-contract-summary)

---

## 1. Level 0 — Context Diagram

Level 0 menampilkan KasKu sebagai black box dengan semua entitas eksternal yang berinteraksi dengannya.

```mermaid
flowchart LR
    USER(["Pengguna\nUser Browser"])
    ADMIN(["Admin\nInternal Operator"])
    MIDTRANS(["Midtrans\nPayment Gateway"])
    COINGECKO(["CoinGecko API\nKripto Price"])
    METALS(["metals.live API\nEmas Price"])
    SMTP(["SMTP Server\nEmail Provider"])

    subgraph KASKU ["KasKu SaaS Platform"]
        CORE["KasKu Core\n11 Microservices"]
    end

    USER -- "Register, Login\nCRUD Keuangan/Investasi/Transaksi\nSubscribe, Offline Sync" --> KASKU
    KASKU -- "Dashboard, Laporan\nStatus Sinkronisasi\nPayment Redirect URL" --> USER

    ADMIN -- "Login Admin\nManajemen User\nLihat Statistik Platform" --> KASKU
    KASKU -- "Platform Stats\nData User\nRiwayat Payment" --> ADMIN

    MIDTRANS -- "Payment Notification\nWebhook POST" --> KASKU
    KASKU -- "Create Snap Transaction\nOrder Details" --> MIDTRANS

    KASKU -- "GET harga kripto\ncoin_id + vs_currencies" --> COINGECKO
    COINGECKO -- "Harga IDR dan USD\nreal-time" --> KASKU

    KASKU -- "GET harga emas\nXAU/USD" --> METALS
    METALS -- "Harga emas USD" --> KASKU

    KASKU -- "Kirim email transaksional\nwelcome, receipt, alert" --> SMTP
```

---

## 2. Level 1 — System Overview

Level 1 menampilkan semua 11 microservice, 5 database PostgreSQL, RabbitMQ, Redis, dan IndexedDB browser.

```mermaid
flowchart TB
    USER(["Pengguna"])
    ADMIN(["Admin"])

    subgraph BROWSER ["Browser — SvelteKit PWA"]
        BP1["P1: Auth UI\nregister, login, profile"]
        BP2["P2: Finance UI\nakun keuangan"]
        BP3["P3: Investment UI\ninstrumen investasi"]
        BP4["P4: Transaction UI\ntransaksi dan kategori"]
        BP5["P5: Billing UI\nsubscription dan invoice"]
        BP6["P6: Offline Sync Engine\nIndexedDB queue"]
        IDB[("IndexedDB\nDexie.js\nlocal mirror + sync queue")]
    end

    subgraph GATEWAY ["API Layer"]
        GW["api-gateway\n:8080\nRouting, JWT Verify\nRate Limit, CORS\nTier Header Inject"]
        REDIS_GW[("Redis :6379\nRate Limit\nJWT Blacklist")]
    end

    subgraph AUTH_LAYER ["Auth dan User Layer"]
        AUTH["auth-service\n:8081 HTTP / :9081 gRPC"]
        USR["user-service\n:8082 HTTP / :9082 gRPC"]
    end

    subgraph BILLING_LAYER ["Billing Layer"]
        BILL["billing-service\n:8083 HTTP / :9083 gRPC\nTier Limit gRPC Endpoint"]
    end

    subgraph FINANCE_LAYER ["Finance Layer"]
        FIN["finance-service\n:8084 HTTP / :9084 gRPC\nakun keuangan"]
        TRX["transaction-service\n:8085 HTTP / :9085 gRPC\ntransaksi dan kategori"]
        INV["investment-service\n:8086 HTTP / :9086 gRPC\ninstrumen investasi"]
    end

    subgraph INFRA_LAYER ["Infrastructure Layer"]
        PRICE["price-service Rust\n:8087 HTTP / :9087 gRPC\nreal-time price cache"]
        SYNC["sync-service Rust\n:8088 HTTP\noffline sync handler"]
        NOTIF["notification-service\n:8089 HTTP\nemail transaksional"]
    end

    subgraph ADMIN_LAYER ["Admin Layer — Internal Only"]
        ADM["admin-service\n:8090 HTTP\nplatform dashboard\nuser management"]
    end

    subgraph DATA_LAYER ["Data Layer"]
        DB_AUTH[("kasku_auth\nPostgreSQL\nusers, refresh_tokens\nemail_verifications")]
        DB_BILL[("kasku_billing\nPostgreSQL\nsubscription_plans\nsubscriptions, payments")]
        DB_FIN[("kasku_finance\nPostgreSQL\ntenant schemas\nper-tenant tables")]
        DB_PRICE[("kasku_price\nPostgreSQL\nprice_cache")]
        DB_ADM[("kasku_admin\nPostgreSQL\nadmin_users\nadmin_audit_log")]
        MQ[("RabbitMQ\nkasku.events exchange\ntopic type, durable\n+ DLX/DLQ")]
    end

    EXT_MIDTRANS(["Midtrans"])
    EXT_CG(["CoinGecko API"])
    EXT_ML(["metals.live API"])
    EXT_SMTP(["SMTP Server"])

    USER --> BROWSER
    BP1 & BP2 & BP3 & BP4 & BP5 --> GW
    BP6 <--> IDB
    BP6 <-- "batch push/pull\nsync operations" --> SYNC

    GW <--> REDIS_GW
    GW -- "JWT verify\nrouting" --> AUTH
    GW -- "routing" --> USR
    GW -- "routing" --> BILL
    GW -- "routing" --> FIN
    GW -- "routing" --> TRX
    GW -- "routing" --> INV
    GW -- "gRPC CheckTierLimit\nsebelum CREATE" --> BILL

    AUTH <--> DB_AUTH
    AUTH -- "publish user.registered" --> MQ

    USR <--> DB_AUTH
    USR -- "provision_tenant()\nwrite DDL" --> DB_FIN
    USR -- "consume user.registered\nconsume user.deletion_requested" --> MQ

    BILL <--> DB_BILL
    BILL -- "create transaction\nSnap API" --> EXT_MIDTRANS
    EXT_MIDTRANS -- "webhook\npayment.succeeded\npayment.failed" --> GW
    BILL -- "publish payment.succeeded\npublish payment.failed\npublish subscription events" --> MQ
    BILL -- "consume payment.succeeded\nconsume subscription.expired" --> MQ

    FIN <--> DB_FIN
    TRX <--> DB_FIN
    INV <--> DB_FIN
    INV -- "gRPC GetPrice" --> PRICE

    PRICE <--> DB_PRICE
    PRICE -- "GET kripto price" --> EXT_CG
    PRICE -- "GET emas price" --> EXT_ML

    SYNC <--> DB_FIN

    NOTIF -- "consume all events" --> MQ
    NOTIF -- "send email" --> EXT_SMTP

    ADMIN --> ADM
    ADM <--> DB_ADM
    ADM -- "read user data\nread billing data" --> DB_AUTH
    ADM -- "read payment data\nread subscription data" --> DB_BILL
```

---

## 3. Level 2 — Detail Diagrams

### P1: Registrasi dan Autentikasi

Alur lengkap dari register hingga reset password, termasuk verifikasi email dan brute force protection.

```mermaid
sequenceDiagram
    actor User as Pengguna
    participant FE as SvelteKit PWA
    participant GW as api-gateway
    participant AUTH as auth-service
    participant DB_AUTH as kasku_auth DB
    participant MQ as RabbitMQ
    participant NOTIF as notification-service
    participant SMTP as SMTP Server

    Note over User,SMTP: Alur Register dan Verify Email

    User->>FE: isi form register\n(email, username, password)
    FE->>GW: POST /api/v1/auth/register
    GW->>AUTH: forward request
    AUTH->>DB_AUTH: cek uniqueness email dan username
    AUTH->>DB_AUTH: INSERT users (status: unverified)\nINSERT email_verifications (token hash, expires_at)
    AUTH->>MQ: publish user.registered\n{user_id, email, verification_token}
    MQ->>NOTIF: consume user.registered
    NOTIF->>SMTP: kirim welcome email\n+ link verifikasi
    AUTH-->>FE: HTTP 201 Created

    User->>FE: klik link di email\n/verify-email?token=...
    FE->>GW: GET /api/v1/auth/verify-email?token=...
    GW->>AUTH: forward request
    AUTH->>DB_AUTH: validasi token (not used, not expired)\nUPDATE users SET email_verified=true\nUPDATE email_verifications SET used_at=NOW()
    AUTH-->>FE: HTTP 200 — email verified

    Note over User,SMTP: Alur Login

    User->>FE: isi form login\n(identifier, password)
    FE->>GW: POST /api/v1/auth/login
    GW->>AUTH: forward (rate limit: 10 req/mnt per IP)
    AUTH->>DB_AUTH: cek status lockout
    AUTH->>DB_AUTH: ambil user by email/username
    AUTH->>AUTH: Argon2id verify password
    alt Login gagal
        AUTH->>DB_AUTH: INCREMENT failed_login_count\n(lockout jika >= 5)
        AUTH-->>FE: HTTP 401 Unauthorized
    else Login berhasil
        AUTH->>DB_AUTH: RESET failed_login_count\nINSERT refresh_tokens (hash, expires_at)
        AUTH-->>GW: JWT RS256 access token (15 menit)\nSet-Cookie refresh token (7 hari)
        GW-->>FE: HTTP 200 + access token + cookie
    end

    Note over User,SMTP: Alur Refresh Token

    FE->>GW: POST /api/v1/auth/refresh\n(cookie: refresh_token)
    GW->>AUTH: forward
    AUTH->>DB_AUTH: validasi refresh token hash\n(not expired, not revoked)
    AUTH->>DB_AUTH: REVOKE token lama\nINSERT refresh token baru (rotation)
    AUTH-->>FE: HTTP 200 + access token baru + cookie baru

    Note over User,SMTP: Alur Forgot Password dan Reset

    User->>FE: input email di form forgot password
    FE->>GW: POST /api/v1/auth/forgot-password
    GW->>AUTH: forward
    AUTH->>DB_AUTH: cek apakah email ada\n(response selalu 200 untuk anti-enumeration)
    AUTH->>DB_AUTH: INSERT password_resets (token hash, expires_at 1 jam)
    AUTH->>MQ: publish event password.reset_requested
    MQ->>NOTIF: consume event
    NOTIF->>SMTP: kirim email reset link
    AUTH-->>FE: HTTP 200 (selalu, tanpa mengungkap keberadaan email)

    User->>FE: input password baru di form reset
    FE->>GW: POST /api/v1/auth/reset-password\n{token, new_password}
    GW->>AUTH: forward
    AUTH->>DB_AUTH: validasi token (not used, not expired)\nArgon2id hash password baru\nUPDATE users SET password_hash\nREVOKE semua refresh_tokens user\nMARK token as used
    AUTH-->>FE: HTTP 200 — password updated
```

---

### P2: Subscription dan Payment

Alur subscribe ke plan berbayar via Midtrans Snap, webhook handling, dan aktivasi subscription.

```mermaid
sequenceDiagram
    actor User as Pengguna
    participant FE as SvelteKit PWA
    participant GW as api-gateway
    participant BILL as billing-service
    participant DB_BILL as kasku_billing DB
    participant MIDTRANS as Midtrans Snap API
    participant MQ as RabbitMQ
    participant NOTIF as notification-service
    participant SMTP as SMTP Server

    Note over User,SMTP: Alur Lihat Plan dan Subscribe

    User->>FE: buka halaman subscription
    FE->>GW: GET /api/v1/billing/plans
    GW->>BILL: forward
    BILL->>DB_BILL: SELECT * FROM subscription_plans WHERE is_active=true
    BILL-->>FE: daftar plan FREE/BASIC/PRO beserta limits dan harga

    User->>FE: pilih plan BASIC/PRO dan klik subscribe
    FE->>GW: POST /api/v1/billing/subscribe\n{plan_id}
    GW->>BILL: forward (header X-User-ID dari JWT)
    BILL->>DB_BILL: INSERT payments\n(status: PENDING, order_id: KASKU-{user_id_8}-{timestamp})
    BILL->>MIDTRANS: POST /snap/v1/transactions\n{order_id, gross_amount, customer_details}
    MIDTRANS-->>BILL: snap_redirect_url
    BILL->>DB_BILL: UPDATE payments SET snap_url
    BILL-->>FE: HTTP 200 {snap_redirect_url}

    FE->>User: redirect ke Midtrans Snap payment page
    User->>MIDTRANS: pilih metode pembayaran\n(QRIS/VA/GoPay/OVO)
    User->>MIDTRANS: selesaikan pembayaran

    Note over User,SMTP: Webhook dari Midtrans setelah pembayaran

    MIDTRANS->>GW: POST /api/v1/billing/webhook/midtrans\n{order_id, transaction_status, signature_key, ...}
    GW->>BILL: forward webhook
    BILL->>BILL: verifikasi signature\nSHA512(order_id+status_code+gross_amount+server_key)
    BILL->>DB_BILL: cek apakah order_id sudah diproses\n(idempotency check)

    alt transaction_status = settlement atau capture
        BILL->>DB_BILL: UPDATE payments SET status=PAID
        BILL->>MQ: publish payment.succeeded\n{user_id, plan_id, amount_idr, period_start, period_end}
        MQ->>BILL: consume payment.succeeded
        BILL->>DB_BILL: UPSERT subscriptions\n(status: ACTIVE, expires_at: NOW()+30 hari)
        MQ->>NOTIF: consume payment.succeeded
        NOTIF->>SMTP: kirim email receipt\n(nama plan, jumlah IDR, tanggal, periode)
    else transaction_status = deny/cancel/expire
        BILL->>DB_BILL: UPDATE payments SET status=FAILED
        BILL->>MQ: publish payment.failed\n{user_id, order_id, reason}
        MQ->>NOTIF: consume payment.failed
        NOTIF->>SMTP: kirim email notifikasi gagal\n+ instruksi retry
    end

    BILL-->>GW: HTTP 200 (selalu, agar Midtrans tidak retry)

    Note over User,SMTP: Expiry dan Auto-Downgrade

    BILL->>BILL: daily scheduler cek subscriptions
    BILL->>DB_BILL: query subscription expires_at BETWEEN NOW() AND NOW()+3 hari
    BILL->>MQ: publish subscription.expiring per user
    MQ->>NOTIF: consume subscription.expiring
    NOTIF->>SMTP: kirim peringatan 3 hari sebelum expire

    BILL->>DB_BILL: query subscription expires_at < NOW() AND status=ACTIVE
    BILL->>DB_BILL: UPDATE subscriptions SET status=EXPIRED
    BILL->>MQ: publish subscription.expired per user
    MQ->>BILL: consume subscription.expired
    BILL->>DB_BILL: UPDATE subscriptions — set plan ke FREE
    MQ->>NOTIF: consume subscription.expired
    NOTIF->>SMTP: kirim email downgrade ke Free
```

---

### P3: Tenant Provisioning

Alur event-driven dari registrasi pengguna hingga tenant schema siap digunakan.

```mermaid
sequenceDiagram
    participant AUTH as auth-service
    participant MQ as RabbitMQ
    participant USR as user-service
    participant DB_FIN as kasku_finance DB
    participant BILL as billing-service
    participant DB_BILL as kasku_billing DB

    Note over AUTH,DB_BILL: Dipicu oleh registrasi user baru

    AUTH->>MQ: publish user.registered\n{user_id, email, username, created_at}

    par Consume user.registered
        MQ->>USR: consume user.registered
        USR->>DB_FIN: CALL provision_tenant(user_id::UUID)\n---\nCREATE SCHEMA tenant_{user_uuid_sanitized}\nCREATE TABLE financial_accounts\nCREATE TABLE balance_history\nCREATE TABLE transactions\nCREATE TABLE categories\nCREATE TABLE investment_assets\nCREATE TABLE unit_history\nCREATE TABLE sync_log\nINSERT 10 seed categories\n(Makan, Transportasi, Belanja, Tagihan,\nHiburan, Kesehatan, Pendidikan,\nGaji, Investasi, Lainnya)
        DB_FIN-->>USR: SCHEMA CREATED OK

    and Consume user.registered (parallel)
        MQ->>BILL: consume user.registered
        BILL->>DB_BILL: INSERT subscriptions\n(user_id, plan_id: FREE_PLAN_UUID,\nstatus: ACTIVE,\nstarted_at: NOW(),\nexpires_at: NULL untuk Free)
        DB_BILL-->>BILL: subscription FREE created
    end

    Note over AUTH,DB_BILL: Tenant siap digunakan setelah kedua consumer selesai

    Note over AUTH,DB_BILL: Alur Tenant Deprovisioning (Delete Account)

    AUTH->>MQ: publish user.deletion_requested\n{user_id}
    MQ->>USR: consume user.deletion_requested
    USR->>DB_FIN: CALL deprovision_tenant(user_id::UUID)\n---\nDROP SCHEMA tenant_{user_uuid_sanitized} CASCADE\n(operasi permanen dan irreversible)
    DB_FIN-->>USR: SCHEMA DROPPED
```

---

### P4: Manajemen Akun Keuangan

Alur CRUD akun keuangan dengan enforcement tier limit via billing-service gRPC.

```mermaid
flowchart TD
    USER(["Pengguna"])

    subgraph FE ["SvelteKit PWA"]
        UI_ACC["Finance UI\nakun keuangan"]
    end

    subgraph GW_BOX ["api-gateway"]
        GW["routing\nJWT verify\nX-User-ID inject\nX-Subscription-Tier inject"]
    end

    subgraph FIN_BOX ["finance-service"]
        FIN_UC["Use Case:\nCreateAccount\nListAccounts\nUpdateAccount\nArchiveAccount\nGetBalanceHistory"]
    end

    subgraph BILL_GRPC ["billing-service gRPC"]
        TIER_CHECK["CheckTierLimit\nresource: financial_accounts\nreturn: allowed bool"]
    end

    subgraph DB_FIN_BOX ["kasku_finance — tenant schema"]
        T_ACC[("tenant_{id}.\nfinancial_accounts")]
        T_BAL[("tenant_{id}.\nbalance_history")]
    end

    USER --> UI_ACC
    UI_ACC -- "GET /api/v1/accounts\nPOST /api/v1/accounts\nPUT /api/v1/accounts/:id\nDELETE /api/v1/accounts/:id\nGET /api/v1/accounts/:id/history" --> GW
    GW --> FIN_UC

    FIN_UC -- "hanya untuk POST (CREATE):\ngRPC CheckTierLimit\ncurrent_count = COUNT aktif" --> TIER_CHECK
    TIER_CHECK -- "allowed=false → HTTP 402\nallowed=true → lanjut" --> FIN_UC

    FIN_UC -- "SELECT WHERE tenant schema\ndan is_deleted=false" --> T_ACC
    FIN_UC -- "INSERT on CREATE\nSELECT history" --> T_BAL

    T_ACC --> FIN_UC
    T_BAL --> FIN_UC
    FIN_UC --> GW
    GW --> UI_ACC
    UI_ACC --> USER
```

---

### P5: Manajemen Investasi dan Harga

Alur CRUD instrumen investasi dengan tier limit check dan pengambilan harga real-time via price-service gRPC.

```mermaid
sequenceDiagram
    actor User as Pengguna
    participant FE as SvelteKit PWA
    participant GW as api-gateway
    participant INV as investment-service
    participant BILL as billing-service gRPC
    participant PRICE as price-service gRPC
    participant DB_FIN as kasku_finance tenant schema
    participant DB_PRICE as kasku_price DB
    participant CG as CoinGecko API
    participant ML as metals.live API

    Note over User,ML: Create Instrumen Investasi

    User->>FE: isi form tambah instrumen\n(nama, tipe, units, buy_price)
    FE->>GW: POST /api/v1/investments\n(Authorization header)
    GW->>INV: forward + X-User-ID + X-Subscription-Tier
    INV->>BILL: gRPC CheckTierLimit\n(user_id, investment_instruments, current_count)
    alt limit terlampaui
        BILL-->>INV: allowed=false, http_status=402
        INV-->>FE: HTTP 402 Payment Required\n{message: "Upgrade plan untuk instrumen lebih banyak"}
    else dalam batas tier
        BILL-->>INV: allowed=true
        INV->>DB_FIN: INSERT tenant_{id}.investment_assets\nINSERT tenant_{id}.unit_history (initial)
        INV-->>FE: HTTP 201 Created
    end

    Note over User,ML: List Instrumen dengan Harga Real-time

    User->>FE: buka halaman investasi
    FE->>GW: GET /api/v1/investments
    GW->>INV: forward
    INV->>DB_FIN: SELECT * FROM tenant_{id}.investment_assets\nWHERE is_deleted=false

    loop untuk setiap instrumen dengan coin_id dan is_manual_price=false
        INV->>PRICE: gRPC GetPrice(coin_id)
        PRICE->>DB_PRICE: SELECT FROM price_cache\nWHERE coin_id=? AND expires_at > NOW()
        alt cache valid
            DB_PRICE-->>PRICE: harga dari cache, is_fresh=true
        else cache expired
            PRICE->>CG: GET /api/v3/simple/price?ids={coin_id}&vs_currencies=idr,usd
            alt CoinGecko ok
                CG-->>PRICE: harga terbaru
                PRICE->>DB_PRICE: UPSERT price_cache (expires_at = NOW()+900s)
                PRICE-->>INV: harga terbaru, is_fresh=true
            else CoinGecko timeout/error
                DB_PRICE-->>PRICE: harga terakhir tersimpan
                PRICE-->>INV: harga lama, is_fresh=false
            end
        end
    end

    loop untuk instrumen emas
        INV->>PRICE: gRPC GetPrice(GOLD_XAU)
        PRICE->>DB_PRICE: cek cache emas
        alt cache emas expired
            PRICE->>ML: GET XAU/USD dari metals.live
            PRICE->>PRICE: konversi USD ke IDR\n(GOLD_USD_IDR_RATE dari env)
            PRICE->>DB_PRICE: UPSERT harga emas
        end
        PRICE-->>INV: harga emas IDR, is_fresh flag
    end

    INV-->>FE: list instrumen + harga pasar terkini + is_fresh flag
    FE-->>User: tampilkan dashboard investasi\ndengan indikator kesegaran data
```

---

### P6: Pencatatan Transaksi

Alur CRUD transaksi dengan enforcement quota bulanan dan soft delete.

```mermaid
flowchart TD
    USER(["Pengguna"])

    subgraph FE ["SvelteKit PWA"]
        UI_TRX["Transaction UI\ntransaksi dan kategori"]
    end

    subgraph GW_BOX ["api-gateway"]
        GW["routing + JWT verify\n+ header inject"]
    end

    subgraph TRX_BOX ["transaction-service"]
        TRX_CREATE["Use Case: CreateTransaction\ncek quota bulanan\nvia billing gRPC"]
        TRX_LIST["Use Case: ListTransactions\nfilter + pagination\n+ retensi tier"]
        TRX_UPDATE["Use Case: UpdateTransaction"]
        TRX_DELETE["Use Case: SoftDeleteTransaction\ndecrement quota"]
        TRX_EXPORT["Use Case: ExportCSV\nhanya tier Pro"]
        CAT["Use Case: ManageCategories\nCRUD kategori"]
    end

    subgraph BILL_GRPC ["billing-service gRPC"]
        QUOTA["CheckTierLimit\nresource: monthly_transactions\ncurrent_count = tx bulan ini"]
    end

    subgraph DB_SCHEMA ["kasku_finance — tenant schema"]
        T_TRX[("tenant_{id}.transactions\n+ sync_id partial unique index")]
        T_CAT[("tenant_{id}.categories")]
        T_ACC_REF[("tenant_{id}.financial_accounts\nreferensi saldo")]
    end

    USER --> UI_TRX
    UI_TRX -- "POST /api/v1/transactions" --> GW
    GW --> TRX_CREATE
    TRX_CREATE -- "gRPC CheckTierLimit\nFree: 50/bulan\nBasic: 500/bulan\nPro: unlimited" --> QUOTA
    QUOTA -- "allowed=false → HTTP 402" --> TRX_CREATE
    TRX_CREATE -- "INSERT transaction\nUPDATE balance" --> T_TRX
    TRX_CREATE -- "UPDATE current_balance" --> T_ACC_REF

    UI_TRX -- "GET /api/v1/transactions\n?account_id=&category_id=\n&type=&date_from=&date_to=" --> GW
    GW --> TRX_LIST
    TRX_LIST -- "SELECT dengan filter\ndan retensi tier" --> T_TRX
    TRX_LIST --> T_CAT

    UI_TRX -- "PUT /api/v1/transactions/:id" --> GW
    GW --> TRX_UPDATE
    TRX_UPDATE --> T_TRX

    UI_TRX -- "DELETE /api/v1/transactions/:id" --> GW
    GW --> TRX_DELETE
    TRX_DELETE -- "soft delete\nis_deleted=true" --> T_TRX

    UI_TRX -- "GET /api/v1/transactions/export\n(Pro tier only)" --> GW
    GW --> TRX_EXPORT
    TRX_EXPORT -- "cek X-Subscription-Tier=PRO" --> TRX_EXPORT
    TRX_EXPORT -- "SELECT semua transaksi\ndalam rentang tanggal" --> T_TRX
    TRX_EXPORT -- "stream CSV response" --> UI_TRX

    UI_TRX -- "CRUD /api/v1/categories" --> GW
    GW --> CAT
    CAT --> T_CAT
```

---

### P7: Offline Sync

Alur sinkronisasi data antara IndexedDB browser dan server, termasuk conflict resolution Server Wins.

```mermaid
sequenceDiagram
    actor User as Pengguna
    participant FE as SvelteKit PWA
    participant IDB as IndexedDB\nDexie.js
    participant SW as Service Worker\nWorkbox
    participant GW as api-gateway
    participant SYNC as sync-service Rust
    participant DB_FIN as kasku_finance\ntenant schema

    Note over User,DB_FIN: Operasi Saat Offline

    User->>FE: buat transaksi baru\n(tidak ada internet)
    FE->>IDB: INSERT transaction ke local store\nstatus: pending_sync\nsync_id: UUID-baru
    FE->>IDB: INSERT ke sync_queue\n{operation: create, entity_type: transaction,\nentity_id, payload, client_timestamp}
    IDB-->>FE: OK (data tersimpan lokal)
    FE-->>User: transaksi tersimpan\n(indikator: X pending)

    Note over User,DB_FIN: Koneksi Internet Kembali — Auto Sync

    SW->>SW: deteksi online event
    SW->>IDB: ambil semua pending di sync_queue
    IDB-->>SW: [{sync_id, operation, entity_type,\nentity_id, payload, client_timestamp}]

    SW->>GW: POST /api/v1/sync/push\n(Authorization header)\n[batch operasi]
    GW->>SYNC: forward + X-User-ID (tenant isolation check)

    SYNC->>SYNC: verifikasi X-User-ID\ncocok dengan tenant schema
    loop setiap operasi dalam batch
        SYNC->>DB_FIN: cek sync_log — apakah sync_id sudah diproses?\n(idempotency)
        alt sync_id sudah ada
            SYNC->>SYNC: skip operasi ini (idempotent)
        else sync_id baru
            SYNC->>DB_FIN: cek apakah data di server\nlebih baru dari client_timestamp?
            alt conflict — server data lebih baru (Server Wins)
                SYNC->>DB_FIN: KEEP data server, tidak overwrite
                SYNC->>SYNC: tambahkan ke conflict_resolutions\n{server_data untuk return ke client}
            else no conflict
                SYNC->>DB_FIN: eksekusi operasi\n(INSERT/UPDATE/DELETE)
                SYNC->>DB_FIN: INSERT sync_log\n(sync_id, user_id, entity_type,\noperation, processed_at)
            end
        end
    end

    SYNC-->>GW: HTTP 200\n{processed: N, conflicts: [{entity_id, server_data}]}
    GW-->>SW: response

    SW->>IDB: hapus operasi yang sudah diproses dari sync_queue
    alt ada conflicts
        SW->>IDB: UPDATE local data dengan server_data\n(Server Wins — overwrite local)
    end
    SW-->>FE: sync complete event
    FE-->>User: indikator: synced

    Note over User,DB_FIN: Pull Sync — Ambil Perubahan Server

    FE->>GW: GET /api/v1/sync/pull?since={last_sync_timestamp}
    GW->>SYNC: forward
    SYNC->>DB_FIN: SELECT semua perubahan\nWHERE updated_at > last_sync_timestamp\nDAN tenant schema = tenant user
    DB_FIN-->>SYNC: delta perubahan
    SYNC-->>FE: {changes: [...], server_timestamp: NOW()}
    FE->>IDB: UPDATE local IndexedDB\ndengan perubahan dari server
```

---

### P8: Notifikasi Email

Alur event-driven penuh — semua 7 tipe event RabbitMQ dan email yang dikirim.

```mermaid
flowchart LR
    subgraph PUBLISHERS ["Event Publishers"]
        AUTH_PUB["auth-service"]
        BILL_PUB["billing-service\n(daily scheduler)"]
    end

    subgraph MQ_BOX ["RabbitMQ\nkasku.events exchange (topic)"]
        E1["user.registered"]
        E2["payment.succeeded"]
        E3["payment.failed"]
        E4["subscription.expiring"]
        E5["subscription.expired"]
        E6["subscription.cancelled"]
        DLX["kasku.events.dlx\nDead Letter Exchange"]
        DLQ["kasku.events.dlq\nDead Letter Queue"]
    end

    subgraph NOTIF_BOX ["notification-service"]
        N1["Handler:\nWelcomeEmail\n+ verifikasi link"]
        N2["Handler:\nPaymentReceiptEmail\n(nama plan, amount, periode)"]
        N3["Handler:\nPaymentFailedEmail\n(instruksi retry)"]
        N4["Handler:\nExpiryWarningEmail\n(3 hari sebelum)"]
        N5["Handler:\nExpiredDowngradeEmail\n(info Free tier)"]
        N6["Handler:\nCancelledConfirmEmail\n(tanggal berakhir akses)"]
        RETRY["Retry Logic\nexponential backoff\nmaks 3x"]
    end

    SMTP(["SMTP Server\nEmail Terkirim"])

    AUTH_PUB --> E1
    BILL_PUB --> E2
    BILL_PUB --> E3
    BILL_PUB --> E4
    BILL_PUB --> E5
    BILL_PUB --> E6

    E1 --> N1
    E2 --> N2
    E3 --> N3
    E4 --> N4
    E5 --> N5
    E6 --> N6

    N1 & N2 & N3 & N4 & N5 & N6 --> RETRY
    RETRY -- "kirim email" --> SMTP
    RETRY -- "gagal setelah 3x retry\nnack + requeue=false" --> DLX
    DLX --> DLQ
```

---

### P9: Admin Platform

Alur admin login dan operasi platform management yang hanya dapat diakses dari jaringan internal.

```mermaid
sequenceDiagram
    actor AdminUser as Admin Operator
    participant FE_ADM as Admin Browser
    participant ADM as admin-service :8090\n(internal only)
    participant DB_ADM as kasku_admin DB
    participant DB_AUTH as kasku_auth DB\n(read only)
    participant DB_BILL as kasku_billing DB\n(read only)
    participant DB_FIN as kasku_finance DB\n(read only — tenant schemas)

    Note over AdminUser,DB_FIN: Admin Login (terpisah dari user auth)

    AdminUser->>FE_ADM: input admin credentials
    FE_ADM->>ADM: POST /admin/auth/login\n{email, password}
    ADM->>DB_ADM: SELECT admin_users WHERE email=?\nArgon2id verify password
    alt credentials valid
        ADM->>ADM: buat admin JWT\n{admin_id, role, exp: 8 jam}
        ADM-->>FE_ADM: HTTP 200 + admin JWT
    else invalid
        ADM-->>FE_ADM: HTTP 401
    end

    Note over AdminUser,DB_FIN: Platform Statistics Dashboard

    AdminUser->>FE_ADM: buka dashboard
    FE_ADM->>ADM: GET /admin/stats\n(Authorization: Bearer admin_jwt)
    ADM->>DB_AUTH: COUNT users\nCOUNT users WHERE last_login > NOW()-30 hari
    ADM->>DB_BILL: COUNT subscriptions GROUP BY plan_id\nSUM payments WHERE status=PAID AND created_at bulan ini\n(= MRR)\nCOUNT expired/cancelled subscriptions bulan ini\n(= churn)
    ADM-->>FE_ADM: {total_users, active_users, mrr, tier_distribution, churn_rate}

    Note over AdminUser,DB_FIN: Detail User dan Usage Stats

    AdminUser->>FE_ADM: cari user dan buka detail
    FE_ADM->>ADM: GET /admin/users/{user_id}
    ADM->>DB_AUTH: SELECT user profile
    ADM->>DB_BILL: SELECT subscription aktif user
    ADM->>DB_FIN: SELECT COUNT(transactions) WHERE bulan ini\nfrom tenant_{user_id_sanitized}.transactions\nSELECT COUNT(financial_accounts) WHERE is_deleted=false\nSELECT COUNT(investment_assets) WHERE is_deleted=false
    ADM-->>FE_ADM: {profile, subscription, usage_stats}

    Note over AdminUser,DB_FIN: Override Subscription (tanpa payment)

    AdminUser->>FE_ADM: isi form override\n{plan_id, reason}
    FE_ADM->>ADM: PUT /admin/users/{user_id}/subscription\n{plan_id, reason}
    ADM->>DB_BILL: UPDATE subscriptions\nSET plan_id=?, status=ACTIVE\nWHERE user_id=?
    ADM->>DB_ADM: INSERT admin_audit_log\n{admin_id, action: override_subscription,\ntarget_user_id, reason, timestamp}
    ADM-->>FE_ADM: HTTP 200 — subscription updated

    Note over AdminUser,DB_FIN: Suspend User

    AdminUser->>FE_ADM: klik suspend pada user
    FE_ADM->>ADM: POST /admin/users/{user_id}/suspend
    ADM->>DB_AUTH: UPDATE users SET is_active=false\nWHERE id=?
    ADM->>DB_ADM: INSERT admin_audit_log\n{admin_id, action: suspend_user, target_user_id, timestamp}
    ADM-->>FE_ADM: HTTP 200 — user suspended
```

---

## 4. Data Store Summary

| ID | Nama Store | Service Owner | Database | Teknologi | Deskripsi |
|----|-----------|---------------|----------|-----------|-----------|
| D1 | users | auth-service | kasku_auth | PostgreSQL | Kredensial dan status user (email_verified, is_active, failed_login_count, lockout_until) |
| D2 | refresh_tokens | auth-service | kasku_auth | PostgreSQL | Refresh token aktif (hash SHA-256, expires_at, revoked_at) |
| D3 | email_verifications | auth-service | kasku_auth | PostgreSQL | Token verifikasi email (hash, expires_at, used_at) |
| D4 | password_resets | auth-service | kasku_auth | PostgreSQL | Token reset password (hash, expires_at, used_at) |
| D5 | subscription_plans | billing-service | kasku_billing | PostgreSQL | Definisi plan FREE/BASIC/PRO beserta limits dan harga |
| D6 | subscriptions | billing-service | kasku_billing | PostgreSQL | Subscription aktif per user (plan_id, status, started_at, expires_at) |
| D7 | payments | billing-service | kasku_billing | PostgreSQL | Riwayat payment (midtrans_order_id, status, amount_idr, snap_url) |
| D8 | tenant_{id}.financial_accounts | finance-service | kasku_finance | PostgreSQL (tenant schema) | Akun keuangan per tenant (name, type, currency, is_deleted) |
| D9 | tenant_{id}.balance_history | finance-service | kasku_finance | PostgreSQL (tenant schema) | Riwayat perubahan saldo (append-only audit log) |
| D10 | tenant_{id}.investment_assets | investment-service | kasku_finance | PostgreSQL (tenant schema) | Instrumen investasi per tenant (name, type, units, coin_id, is_deleted) |
| D11 | tenant_{id}.unit_history | investment-service | kasku_finance | PostgreSQL (tenant schema) | Riwayat perubahan unit (append-only audit log) |
| D12 | tenant_{id}.transactions | transaction-service | kasku_finance | PostgreSQL (tenant schema) | Transaksi keuangan per tenant (type, amount, date, sync_id, is_deleted) |
| D13 | tenant_{id}.categories | transaction-service | kasku_finance | PostgreSQL (tenant schema) | Kategori transaksi per tenant (seed + custom, is_deleted) |
| D14 | tenant_{id}.sync_log | sync-service | kasku_finance | PostgreSQL (tenant schema) | Audit log operasi sync (sync_id, operation, entity_type, processed_at) |
| D15 | price_cache | price-service | kasku_price | PostgreSQL | Cache harga aset (coin_id, price_idr, price_usd, expires_at) |
| D16 | admin_users | admin-service | kasku_admin | PostgreSQL | Kredensial admin platform (email, password_hash Argon2id, role) |
| D17 | admin_audit_log | admin-service | kasku_admin | PostgreSQL | Log aksi admin ke data user (admin_id, action, target_user_id, reason, timestamp) |
| D18 | kasku.events (RabbitMQ) | Multiple publishers | RabbitMQ | AMQP 0-9-1 | Async event bus — topic exchange durable |
| D19 | kasku.events.dlq (RabbitMQ) | notification-service | RabbitMQ | AMQP 0-9-1 | Dead Letter Queue untuk failed events setelah retry maksimal |
| D20 | rate_limit (Redis) | api-gateway | Redis | Redis ≥7 | Counter rate limit per IP dan per user_id |
| D21 | jwt_blacklist (Redis) | api-gateway | Redis | Redis ≥7 | Token JWT yang sudah di-logout (key=token hash, TTL=sisa waktu valid) |
| IDB | IndexedDB | Frontend (browser) | Browser | Dexie.js ≥4.0 | Mirror lokal data tenant + sync_queue untuk operasi offline |

---

## 5. Event Contract Summary

Exchange: `kasku.events` (topic type, durable)
Format routing key: `{domain}.{action}` (lowercase)

| Event | Routing Key | Publisher | Consumer(s) | Payload Fields |
|-------|-------------|-----------|-------------|----------------|
| User Registered | `user.registered` | auth-service | user-service, billing-service, notification-service | `user_id`, `email`, `username`, `created_at`, `verification_token` |
| User Deletion Requested | `user.deletion_requested` | user-service | user-service | `user_id`, `requested_at` |
| Payment Succeeded | `payment.succeeded` | billing-service | billing-service, notification-service | `user_id`, `plan_id`, `plan_name`, `amount_idr`, `payment_method`, `period_start`, `period_end` |
| Payment Failed | `payment.failed` | billing-service | notification-service | `user_id`, `order_id`, `reason`, `failed_at` |
| Subscription Expiring | `subscription.expiring` | billing-service (scheduler) | notification-service | `user_id`, `email`, `plan_name`, `expires_at` |
| Subscription Expired | `subscription.expired` | billing-service (scheduler) | billing-service, notification-service | `user_id`, `email`, `previous_plan`, `expired_at` |
| Subscription Cancelled | `subscription.cancelled` | billing-service | notification-service | `user_id`, `email`, `plan_name`, `access_until` |

**Dead Letter Policy:**
- DLX: `kasku.events.dlx`
- DLQ: `kasku.events.dlq`
- Trigger: message di-nack dengan requeue=false setelah 3 kali retry di notification-service
- Action: pesan tersimpan di DLQ untuk investigasi manual oleh operator

---

*Dokumen DFD ini merupakan panduan visual untuk alur data sistem KasKu v2.0. Untuk detail schema database, lihat `databaseScheme.md`. Untuk detail endpoint, lihat `ApiSpecOpenAPI.yaml`. Untuk keputusan arsitektur, lihat `Arsitektur.md`.*
