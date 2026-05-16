# API Documentation
## KasKu: Personal Finance Tracker

**Versi:** 1.0.0  
**Base URL:** `https://{your-server}/api/v1`  
**Format:** JSON  
**Tanggal:** 2026-04-22  

---

## Konvensi Umum

### Authentication
Semua endpoint (kecuali `/auth/login`) memerlukan header:
```
Authorization: Bearer {access_token}
```
Access token diperoleh dari response login. Refresh token disimpan di HttpOnly cookie (`refresh_token`).

### Standar Response

**Success:**
```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 123
  }
}
```

**Error:**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Deskripsi error yang dapat dibaca",
    "details": [
      { "field": "amount", "message": "harus lebih besar dari 0" }
    ]
  }
}
```

### Error Codes

| Code | HTTP Status | Keterangan |
|---|---|---|
| `VALIDATION_ERROR` | 400 | Input tidak valid |
| `UNAUTHORIZED` | 401 | Token tidak ada / expired |
| `FORBIDDEN` | 403 | Tidak punya akses ke resource |
| `NOT_FOUND` | 404 | Resource tidak ditemukan |
| `CONFLICT` | 409 | Duplikasi data |
| `RATE_LIMIT_EXCEEDED` | 429 | Terlalu banyak request |
| `INTERNAL_ERROR` | 500 | Error server |
| `ACCOUNT_LOCKED` | 403 | Akun terkunci (brute force) |

### Pagination
Query params: `?page=1&limit=50`  
Default: page=1, limit=50  
Max limit: 100

### Date Format
Semua tanggal menggunakan ISO 8601: `YYYY-MM-DD` untuk date, `YYYY-MM-DDTHH:mm:ssZ` untuk datetime.

---

## 1. Authentication

### POST /auth/login
Login dan dapatkan access + refresh token.

**Request:**
```json
{
  "username": "string (required)",
  "password": "string (required)"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiJ9...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```
> Refresh token dikirim via `Set-Cookie: refresh_token=...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth`

**Response 401:**
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Username atau password salah"
  }
}
```

**Response 403 (lockout):**
```json
{
  "success": false,
  "error": {
    "code": "ACCOUNT_LOCKED",
    "message": "Akun terkunci. Coba lagi setelah 2026-04-22T10:30:00Z"
  }
}
```

---

### POST /auth/refresh
Dapatkan access token baru menggunakan refresh token (dari cookie).

**Request:** Tidak ada body. Refresh token dibaca dari cookie.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiJ9...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**Response 401:**
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Refresh token tidak valid atau sudah kedaluwarsa"
  }
}
```

---

### POST /auth/logout
Invalidasi refresh token aktif.

**Request:** Tidak ada body.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "message": "Logout berhasil"
  }
}
```

---

## 2. Financial Accounts

### GET /accounts
Daftar semua akun keuangan aktif beserta saldo dan total agregat.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "accounts": [
      {
        "id": "uuid",
        "name": "BCA Tabungan",
        "account_type": "BANK",
        "balance": 5000000.00,
        "currency": "IDR",
        "color": "#2563EB",
        "icon": "bank",
        "created_at": "2026-01-01T00:00:00Z",
        "updated_at": "2026-04-22T08:00:00Z"
      },
      {
        "id": "uuid",
        "name": "Seabank",
        "account_type": "BANK",
        "balance": 2500000.00,
        "currency": "IDR",
        "color": "#16A34A",
        "icon": "mobile-banking",
        "created_at": "2026-01-15T00:00:00Z",
        "updated_at": "2026-04-20T10:00:00Z"
      }
    ],
    "total_balance_idr": 7500000.00
  }
}
```

---

### POST /accounts
Tambah akun keuangan baru.

**Request:**
```json
{
  "name": "string (required, max 100)",
  "account_type": "BANK | EWALLET | CASH (required)",
  "balance": 0.00,
  "currency": "IDR",
  "color": "#2563EB",
  "icon": "bank"
}
```

**Response 201:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "BCA Tabungan",
    "account_type": "BANK",
    "balance": 5000000.00,
    "currency": "IDR",
    "color": "#2563EB",
    "icon": "bank",
    "created_at": "2026-04-22T09:00:00Z",
    "updated_at": "2026-04-22T09:00:00Z"
  }
}
```

**Response 409 (nama duplikat):**
```json
{
  "success": false,
  "error": {
    "code": "CONFLICT",
    "message": "Nama akun sudah digunakan"
  }
}
```

---

### GET /accounts/:id
Detail satu akun.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "BCA Tabungan",
    "account_type": "BANK",
    "balance": 5000000.00,
    "currency": "IDR",
    "color": "#2563EB",
    "icon": "bank",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-04-22T08:00:00Z"
  }
}
```

---

### PUT /accounts/:id
Update akun (semua field).

**Request:**
```json
{
  "name": "string (required, max 100)",
  "account_type": "BANK | EWALLET | CASH (required)",
  "balance": 5000000.00,
  "currency": "IDR",
  "color": "#2563EB",
  "icon": "bank"
}
```

**Response 200:** Sama dengan GET /accounts/:id

---

### PATCH /accounts/:id/balance
Update saldo saja (tanpa mengubah field lain).

**Request:**
```json
{
  "balance": 6000000.00,
  "reason": "Gaji masuk bulan April"
}
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "balance_before": 5000000.00,
    "balance_after": 6000000.00,
    "updated_at": "2026-04-22T09:30:00Z"
  }
}
```

---

### DELETE /accounts/:id
Soft delete akun.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "message": "Akun berhasil dihapus"
  }
}
```

---

### GET /accounts/:id/history
Riwayat perubahan saldo akun.

**Query Params:** `?page=1&limit=20`

**Response 200:**
```json
{
  "success": true,
  "data": {
    "history": [
      {
        "id": "uuid",
        "balance_before": 5000000.00,
        "balance_after": 6000000.00,
        "reason": "Gaji masuk bulan April",
        "recorded_at": "2026-04-22T09:30:00Z"
      }
    ]
  },
  "meta": { "page": 1, "limit": 20, "total": 5 }
}
```

---

## 3. Investment Assets

### GET /investments
Daftar semua instrumen investasi dengan harga real-time dan nilai estimasi.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "assets": [
      {
        "id": "uuid",
        "name": "Emas Antam",
        "asset_type": "GOLD",
        "symbol": "XAU",
        "quantity": 50.5,
        "unit_label": "gram",
        "price": {
          "price_idr": 1850000.00,
          "price_usd": 113.5,
          "source": "metals.live",
          "fetched_at": "2026-04-22T09:00:00Z",
          "is_fresh": true
        },
        "estimated_value_idr": 93425000.00
      },
      {
        "id": "uuid",
        "name": "Bitcoin",
        "asset_type": "CRYPTO",
        "symbol": "BTC",
        "coin_id": "bitcoin",
        "quantity": 0.0025,
        "unit_label": "BTC",
        "price": {
          "price_idr": 1650000000.00,
          "price_usd": 101000.0,
          "source": "coingecko",
          "fetched_at": "2026-04-22T09:00:00Z",
          "is_fresh": true
        },
        "estimated_value_idr": 4125000.00
      }
    ],
    "total_estimated_value_idr": 97550000.00,
    "price_last_updated": "2026-04-22T09:00:00Z"
  }
}
```

> `is_fresh: false` jika harga lebih dari 15 menit, `null` jika tidak ada harga sama sekali (hanya tampilkan quantity).

---

### POST /investments
Tambah instrumen investasi.

**Request:**
```json
{
  "name": "string (required, max 100)",
  "asset_type": "GOLD | CRYPTO | STOCK | OTHER (required)",
  "symbol": "string (required, max 20)",
  "coin_id": "string (optional, wajib jika asset_type=CRYPTO)",
  "quantity": 50.5,
  "unit_label": "gram"
}
```

**Response 201:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Emas Antam",
    "asset_type": "GOLD",
    "symbol": "XAU",
    "quantity": 50.5,
    "unit_label": "gram",
    "created_at": "2026-04-22T09:00:00Z",
    "updated_at": "2026-04-22T09:00:00Z"
  }
}
```

---

### PUT /investments/:id
Update instrumen investasi.

**Request:**
```json
{
  "name": "string",
  "quantity": 55.0,
  "unit_label": "gram",
  "reason": "Beli emas baru 4.5 gram"
}
```

**Response 200:** Detail instrumen terbaru.

---

### DELETE /investments/:id
Soft delete instrumen.

**Response 200:**
```json
{
  "success": true,
  "data": { "message": "Instrumen berhasil dihapus" }
}
```

---

### GET /investments/prices
Fetch harga real-time semua instrumen aktif (trigger manual refresh).

**Response 200:**
```json
{
  "success": true,
  "data": {
    "prices": [
      {
        "symbol": "XAU",
        "price_idr": 1850000.00,
        "price_usd": 113.5,
        "source": "metals.live",
        "fetched_at": "2026-04-22T09:15:00Z"
      },
      {
        "symbol": "BTC",
        "price_idr": 1650000000.00,
        "price_usd": 101000.0,
        "source": "coingecko",
        "fetched_at": "2026-04-22T09:15:00Z"
      }
    ]
  }
}
```

---

## 4. Categories

### GET /categories
Daftar semua kategori aktif.

**Query Params:** `?type=INCOME|EXPENSE|BOTH`

**Response 200:**
```json
{
  "success": true,
  "data": {
    "categories": [
      {
        "id": "uuid",
        "name": "Makan & Minum",
        "category_type": "EXPENSE",
        "icon": "utensils",
        "is_default": true
      },
      {
        "id": "uuid",
        "name": "Gaji",
        "category_type": "INCOME",
        "icon": "briefcase",
        "is_default": true
      }
    ]
  }
}
```

---

### POST /categories
Tambah kategori baru.

**Request:**
```json
{
  "name": "string (required, max 100)",
  "category_type": "INCOME | EXPENSE | BOTH (required)",
  "icon": "string (optional)"
}
```

**Response 201:** Detail kategori baru.

---

### PUT /categories/:id
Update kategori. Kategori default (`is_default: true`) tidak bisa diubah namanya.

---

### DELETE /categories/:id
Hapus kategori. Gagal jika ada transaksi aktif yang menggunakan kategori ini.

**Response 409 (ada transaksi):**
```json
{
  "success": false,
  "error": {
    "code": "CONFLICT",
    "message": "Kategori tidak dapat dihapus karena masih digunakan oleh 15 transaksi"
  }
}
```

---

## 5. Transactions

### GET /transactions
List transaksi dengan filter dan pagination.

**Query Params:**

| Param | Tipe | Default | Keterangan |
|---|---|---|---|
| `from` | date | 30 hari lalu | Filter tanggal mulai |
| `to` | date | hari ini | Filter tanggal akhir |
| `type` | string | - | INCOME / EXPENSE |
| `category_id` | uuid | - | Filter kategori |
| `account_id` | uuid | - | Filter akun |
| `page` | int | 1 | Halaman |
| `limit` | int | 50 | Per halaman (max 100) |

**Response 200:**
```json
{
  "success": true,
  "data": {
    "transactions": [
      {
        "id": "uuid",
        "transaction_type": "EXPENSE",
        "amount": 25000.00,
        "currency": "IDR",
        "transaction_date": "2026-04-22",
        "note": "Makan siang warteg",
        "category": {
          "id": "uuid",
          "name": "Makan & Minum",
          "icon": "utensils"
        },
        "account": {
          "id": "uuid",
          "name": "BCA Tabungan"
        },
        "created_at": "2026-04-22T12:30:00Z",
        "updated_at": "2026-04-22T12:30:00Z"
      }
    ],
    "summary": {
      "total_income": 10000000.00,
      "total_expense": 3500000.00,
      "net": 6500000.00
    }
  },
  "meta": {
    "page": 1,
    "limit": 50,
    "total": 87,
    "from": "2026-03-23",
    "to": "2026-04-22"
  }
}
```

---

### POST /transactions
Catat transaksi baru.

**Request:**
```json
{
  "transaction_type": "INCOME | EXPENSE (required)",
  "amount": 25000.00,
  "currency": "IDR",
  "category_id": "uuid (required)",
  "account_id": "uuid (optional)",
  "transaction_date": "2026-04-22",
  "note": "string (optional, max 500)",
  "sync_id": "uuid (optional, untuk idempotency saat offline sync)"
}
```

**Response 201:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "transaction_type": "EXPENSE",
    "amount": 25000.00,
    "currency": "IDR",
    "transaction_date": "2026-04-22",
    "note": "Makan siang warteg",
    "category": { "id": "uuid", "name": "Makan & Minum" },
    "account": { "id": "uuid", "name": "BCA Tabungan" },
    "created_at": "2026-04-22T12:30:00Z",
    "updated_at": "2026-04-22T12:30:00Z"
  }
}
```

> Jika `sync_id` sudah ada di database (duplikat dari sync), server mengembalikan **200** dengan data yang sudah ada (idempotent).

---

### GET /transactions/:id
Detail satu transaksi.

**Response 200:** Sama dengan item dalam array GET /transactions.

---

### PUT /transactions/:id
Update transaksi.

**Request:** Sama dengan POST, semua field dapat diubah.

**Response 200:** Detail transaksi terbaru.

---

### DELETE /transactions/:id
Soft delete transaksi.

**Response 200:**
```json
{
  "success": true,
  "data": { "message": "Transaksi berhasil dihapus" }
}
```

---

## 6. Dashboard

### GET /dashboard
Data ringkasan untuk halaman utama.

**Response 200:**
```json
{
  "success": true,
  "data": {
    "financial": {
      "total_balance_idr": 7500000.00,
      "accounts_count": 3,
      "accounts": [
        { "id": "uuid", "name": "BCA Tabungan", "balance": 5000000.00, "account_type": "BANK" },
        { "id": "uuid", "name": "Seabank", "balance": 2500000.00, "account_type": "BANK" }
      ]
    },
    "investment": {
      "total_estimated_value_idr": 97550000.00,
      "assets_count": 2,
      "assets": [
        {
          "id": "uuid",
          "name": "Emas Antam",
          "symbol": "XAU",
          "quantity": 50.5,
          "unit_label": "gram",
          "estimated_value_idr": 93425000.00,
          "price_is_fresh": true
        }
      ],
      "price_last_updated": "2026-04-22T09:00:00Z"
    },
    "transactions_this_month": {
      "total_income": 10000000.00,
      "total_expense": 3500000.00,
      "net": 6500000.00,
      "period": {
        "from": "2026-04-01",
        "to": "2026-04-22"
      }
    },
    "recent_transactions": [
      {
        "id": "uuid",
        "transaction_type": "EXPENSE",
        "amount": 25000.00,
        "category": { "name": "Makan & Minum", "icon": "utensils" },
        "transaction_date": "2026-04-22",
        "note": "Makan siang warteg"
      }
    ]
  }
}
```

---

## 7. Sync (Offline → Server)

### POST /sync/batch
Kirim batch operasi yang dilakukan saat offline.

**Request:**
```json
{
  "operations": [
    {
      "sync_id": "uuid-v4-client-generated",
      "operation": "CREATE",
      "entity_type": "transaction",
      "payload": {
        "transaction_type": "EXPENSE",
        "amount": 15000.00,
        "currency": "IDR",
        "category_id": "uuid",
        "transaction_date": "2026-04-20",
        "note": "Parkir motor",
        "sync_id": "uuid-v4-client-generated"
      },
      "client_timestamp": "2026-04-20T14:30:00Z"
    },
    {
      "sync_id": "uuid-v4-client-generated-2",
      "operation": "UPDATE",
      "entity_type": "financial_account",
      "entity_id": "server-uuid-of-account",
      "payload": {
        "balance": 4800000.00,
        "reason": "Bayar tagihan"
      },
      "client_timestamp": "2026-04-20T15:00:00Z"
    }
  ]
}
```

**Response 207 (Multi-Status):**
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "sync_id": "uuid-v4-client-generated",
        "status": "SUCCESS",
        "entity_id": "server-uuid-created",
        "server_timestamp": "2026-04-22T09:00:00Z"
      },
      {
        "sync_id": "uuid-v4-client-generated-2",
        "status": "CONFLICT",
        "message": "Data server lebih baru. Server state dikembalikan.",
        "server_data": {
          "balance": 5200000.00,
          "updated_at": "2026-04-21T10:00:00Z"
        },
        "server_timestamp": "2026-04-22T09:00:00Z"
      }
    ],
    "summary": {
      "total": 2,
      "success": 1,
      "conflict": 1,
      "failed": 0
    }
  }
}
```

---

### GET /sync/pull
Pull semua data terbaru dari server untuk inisialisasi atau full sync.

**Query Params:** `?since=2026-04-01T00:00:00Z` (opsional, jika kosong pull semua)

**Response 200:**
```json
{
  "success": true,
  "data": {
    "financial_accounts": [ ... ],
    "investment_assets": [ ... ],
    "categories": [ ... ],
    "transactions": [ ... ],
    "server_timestamp": "2026-04-22T09:00:00Z"
  }
}
```

---

## 8. Health Check

### GET /health
Status server (tidak perlu auth).

**Response 200:**
```json
{
  "status": "ok",
  "version": "1.0.0",
  "timestamp": "2026-04-22T09:00:00Z",
  "database": "ok",
  "uptime_seconds": 86400
}
```

**Response 503 (database down):**
```json
{
  "status": "degraded",
  "database": "error",
  "timestamp": "2026-04-22T09:00:00Z"
}
```

---

## Appendix: Struktur Project Go (Clean Architecture)

```
kasKu-server/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── entity/
│   │   │   ├── user.go
│   │   │   ├── financial_account.go
│   │   │   ├── investment_asset.go
│   │   │   ├── transaction.go
│   │   │   └── category.go
│   │   ├── repository/
│   │   │   ├── user_repository.go
│   │   │   ├── account_repository.go
│   │   │   ├── investment_repository.go
│   │   │   ├── transaction_repository.go
│   │   │   └── category_repository.go
│   │   └── errors/
│   │       └── domain_errors.go
│   ├── usecase/
│   │   ├── auth/
│   │   │   └── auth_usecase.go
│   │   ├── account/
│   │   │   └── account_usecase.go
│   │   ├── investment/
│   │   │   └── investment_usecase.go
│   │   ├── transaction/
│   │   │   └── transaction_usecase.go
│   │   └── sync/
│   │       └── sync_usecase.go
│   ├── infrastructure/
│   │   ├── database/
│   │   │   ├── postgres.go
│   │   │   └── migrations/
│   │   ├── repository/
│   │   │   ├── user_repo_impl.go
│   │   │   ├── account_repo_impl.go
│   │   │   └── ...
│   │   ├── external/
│   │   │   ├── coingecko_client.go
│   │   │   └── metals_client.go
│   │   └── cache/
│   │       └── price_cache.go
│   └── delivery/
│       └── http/
│           ├── router.go
│           ├── middleware/
│           │   ├── auth.go
│           │   ├── rate_limit.go
│           │   └── cors.go
│           └── handler/
│               ├── auth_handler.go
│               ├── account_handler.go
│               ├── investment_handler.go
│               ├── transaction_handler.go
│               └── sync_handler.go
├── pkg/
│   ├── jwt/
│   ├── argon2/
│   └── logger/
├── configs/
│   └── config.go
├── docker-compose.yml
├── Dockerfile
├── .env.example
└── go.mod
```

## Appendix: Environment Variables

```env
# Server
APP_PORT=8080
APP_ENV=production
APP_CORS_ORIGIN=https://your-frontend-domain.com

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=kasku
DB_USER=kasku_user
DB_PASSWORD=your_secure_password
DB_SSL_MODE=require

# JWT (RS256)
JWT_PRIVATE_KEY_PATH=/run/secrets/jwt_private.pem
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public.pem
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7

# Rate Limiting
RATE_LIMIT_AUTH=10
RATE_LIMIT_API=200

# Price Fetch
PRICE_CACHE_TTL_MINUTES=15
PRICE_FETCH_TIMEOUT_SECONDS=5

# Argon2id
ARGON2_MEMORY_KB=65536
ARGON2_ITERATIONS=3
ARGON2_PARALLELISM=2
```
