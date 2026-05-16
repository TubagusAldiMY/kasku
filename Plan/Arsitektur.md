# KasKu — Arsitektur SaaS Microservices
# Architecture Decision Records + System Design

**Versi:** 2.0.0 (Pivot: Single-user On-premise → Multi-tenant SaaS)
**Tanggal:** 2026-04-27
**Owner:** TubsAMY — admin@tubsamy.tech
**Status:** Active

---

## Daftar Isi

1. [Architecture Decision Records (ADR-001 s/d ADR-012)](#adrs)
2. [System Architecture Diagram](#system-architecture-diagram)
3. [Service Responsibility Matrix](#service-responsibility-matrix)
4. [Inter-Service Communication Map](#inter-service-communication-map)
5. [Tenant Provisioning Flow](#tenant-provisioning-flow)
6. [Clean Architecture per Service](#clean-architecture-per-service)
7. [Security Architecture](#security-architecture)
8. [Deployment Architecture](#deployment-architecture)
9. [Observability](#observability)

---

## 1. Architecture Decision Records {#adrs}

### ADR-001: Microservices vs Monolith untuk SaaS

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

KasKu sebelumnya adalah monolith single-user on-premise (satu binary Go). Pivot ke SaaS
multi-tenant memperkenalkan kebutuhan baru: multi-tenancy, subscription billing, payment
gateway integration, email notifications, admin dashboard, dan real-time price fetching.
Setiap domain memiliki karakteristik beban, siklus deployment, dan kebutuhan keamanan yang
berbeda-beda.

#### Pilihan yang Dipertimbangkan

| Pilihan | Deskripsi |
|---------|-----------|
| A — Modular Monolith | Satu binary Go, dipisah secara logis via package |
| B — Microservices (dipilih) | 11 service terpisah, independent deployable |
| C — Mini-services (2-3 service) | Auth+Core dalam satu service, Billing terpisah |

#### Keputusan

**Pilihan B: Fine-grained Microservices (11 service)**

Justifikasi:
- **billing-service** dan **price-service** memiliki SLA berbeda (billing butuh 99.9% uptime,
  price-service boleh degraded gracefully dengan stale data).
- **sync-service** dan **price-service** cocok ditulis Rust untuk performa I/O-intensive.
- **admin-service** harus terisolasi secara network dari service publik (security boundary).
- Independent deployability memungkinkan hotfix billing tanpa restart seluruh aplikasi.
- Schema per tenant berjalan di `kasku_finance` — service yang mengakses data tenant
  (finance, transaction, investment, sync) dapat di-scale independen.

#### Konsekuensi

**Pro:**
- Independent deployability dan scalability per domain.
- Technology heterogeneity (Go + Rust) sesuai karakteristik setiap service.
- Fault isolation — kegagalan price-service tidak mematikan billing-service.
- Admin service terisolasi di network internal, memperkecil attack surface.

**Kontra:**
- Operational overhead lebih tinggi: 11 container, 11 log stream, 11 health check.
- Distributed tracing diperlukan untuk debug cross-service flow.
- Latency inter-service (gRPC call) menambah overhead dibanding in-process call.

**Risiko:**
- Distributed monolith apabila service-service memanggil satu sama lain secara sinkronus
  berlebihan. Mitigasi: batasi gRPC call chain maksimal 2 hop; gunakan event-driven untuk
  operasi non-critical-path.

---

### ADR-002: Schema-per-Tenant vs Shared Tables vs DB-per-Tenant

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Multi-tenancy di `kasku_finance` memerlukan isolasi data antar user. Tiga strategi utama
tersedia. Keputusan ini berdampak pada keamanan data, kemudahan provisioning, dan biaya
operasional di VPS dengan resource terbatas.

#### Pilihan yang Dipertimbangkan

| Pilihan | Deskripsi |
|---------|-----------|
| A — Shared tables dengan `user_id` kolom | Satu tabel untuk semua tenant, difilter by user_id |
| B — Schema per tenant (dipilih) | Satu schema PostgreSQL per user di database `kasku_finance` |
| C — Database per tenant | Satu PostgreSQL database per user |

#### Keputusan

**Pilihan B: Schema per Tenant**

Justifikasi:
- Shared tables (A) membawa risiko data leakage apabila query WHERE user_id terlewat.
  Row-Level Security PostgreSQL dapat memitigasi, namun menambah kompleksitas.
- Database per tenant (C) tidak feasible di VPS dengan resource terbatas — setiap database
  membutuhkan koneksi pool terpisah.
- Schema per tenant memberikan isolasi logis kuat: query ke schema yang salah menghasilkan
  "relation does not exist" bukan data silang.
- PostgreSQL mendukung ribuan schema dalam satu instance tanpa degradasi signifikan.
- Provisioning mudah: `CREATE SCHEMA tenant_{uuid}` + jalankan DDL template.

Format nama schema: `tenant_{user_uuid_dengan_underscore}` — contoh:
`tenant_550e8400_e29b_41d4_a716_446655440000`

#### Konsekuensi

**Pro:**
- Isolasi data kuat — cross-tenant leak membutuhkan bug eksplisit di query prefix.
- Provisioning otomatis via PostgreSQL function `provision_tenant(p_user_id UUID)`.
- Backup dan restore per tenant feasible dengan `pg_dump --schema`.
- Schema migration dapat di-apply ke semua tenant schema via iterasi di application layer.

**Kontra:**
- Semua query ke `kasku_finance` harus menyertakan schema prefix yang berasal dari
  context request (user_id). Application code harus enforce ini di setiap repository.
- Schema migration untuk tenant yang sudah ada harus di-run secara iteratif
  (satu schema per user) — berpotensi lambat jika jumlah user besar.
- `search_path` harus di-set per connection atau per query untuk mencegah fallback ke
  `public` schema.

**Risiko:**
- Developer lupa set schema prefix sehingga query fallback ke public (kosong) atau error.
  Mitigasi: repository base di Go selalu menerima `tenantSchema string` sebagai parameter
  wajib; unit test memverifikasi schema prefix ada di setiap query.

---

### ADR-003: Go + Rust Polyglot vs Go-only

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Dua service memiliki karakteristik berbeda dari service Go lainnya:
- **price-service**: I/O-bound, memanggil external API (CoinGecko, metals.live), perlu
  low-latency response, TTL cache kritis untuk menghindari rate limit external.
- **sync-service**: CPU + I/O intensive, memproses batch push/pull dari offline clients,
  menjalankan conflict resolution untuk ratusan record per request.

#### Pilihan yang Dipertimbangkan

| Pilihan | Deskripsi |
|---------|-----------|
| A — Go untuk semua service | Konsistensi ekosistem, satu bahasa untuk semua |
| B — Go + Rust untuk price dan sync (dipilih) | Rust untuk service performa-kritis |
| C — Go + Node.js untuk service tertentu | Node untuk async I/O-heavy service |

#### Keputusan

**Pilihan B: Go (9 service) + Rust (price-service, sync-service)**

Justifikasi:
- Rust memberikan zero-cost abstraction dan memory safety tanpa GC pause — kritis untuk
  sync-service yang menangani banyak data dalam satu request.
- Axum (Rust HTTP framework) + Tonic (gRPC) + SQLx (async DB) adalah stack matang untuk
  production Rust services.
- Interoperabilitas sempurna via gRPC/Protobuf — bahasa tidak relevan di service boundary.
- Tim (TubsAMY) memiliki keahlian Rust yang memadai.
- Node.js (C) ditolak karena GC dan event loop tidak lebih baik dari Go untuk use case ini.

#### Konsekuensi

**Pro:**
- Rust services memiliki predictable latency tanpa GC pause.
- Memory footprint lebih kecil untuk Rust services.
- Type safety Rust mencegah runtime panic di conflict resolution logic yang kompleks.
- Ekosistem Cargo untuk dependency management yang ketat.

**Kontra:**
- Dua bahasa berarti dua toolchain (Go + Rust), dua CI pipeline, dua Dockerfile pattern.
- Onboarding developer baru memerlukan Rust knowledge untuk dua service tersebut.
- Compile time Rust lebih lambat dari Go sehingga CI pipeline lebih lama untuk price dan sync service.

**Risiko:**
- Ketergantungan pada crate ekosistem Rust (Axum, Tonic, SQLx) — versi mayor update
  dapat breaking. Mitigasi: pin versi minor di `Cargo.toml`, audit dependency setiap kuartal.

---

### ADR-004: RabbitMQ vs Kafka vs NATS untuk Async Messaging

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Async event-driven communication diperlukan untuk: welcome email saat register, notifikasi
payment, downgrade subscription expired, dll. Message broker harus dapat berjalan di VPS
dengan resource terbatas (RAM < 4GB untuk broker itu sendiri).

#### Pilihan yang Dipertimbangkan

| Pilihan | Memory Minimum | Throughput | Persistence | Kompleksitas |
|---------|---------------|-----------|-------------|-------------|
| A — Kafka | 1-2 GB JVM | Sangat Tinggi | Log-based | Tinggi (ZooKeeper/KRaft) |
| B — RabbitMQ (dipilih) | 128-256 MB | Menengah | Queue-based | Menengah |
| C — NATS JetStream | 64-128 MB | Tinggi | Stream-based | Rendah |

#### Keputusan

**Pilihan B: RabbitMQ**

Justifikasi:
- Kafka (A) membutuhkan JVM + ZooKeeper/KRaft yang tidak feasible di VPS terbatas.
  Untuk volume event KasKu (ratusan per hari, bukan jutaan), Kafka adalah over-engineering.
- NATS JetStream (C) lebih ringan, namun ekosistem Go client-nya kurang mature dibanding
  RabbitMQ. Dead Letter Queue dan routing yang fleksibel di RabbitMQ sudah battle-tested.
- RabbitMQ memiliki Management UI bawaan yang memudahkan monitoring di awal operasi.
- Exchange + routing key pattern RabbitMQ cocok untuk event routing yang dibutuhkan:
  satu event dapat di-route ke multiple consumer (notification + billing untuk `payment.succeeded`).
- `amqp091-go` adalah library Go yang mature dan well-maintained.

#### Konsekuensi

**Pro:**
- Resource footprint kecil — ideal untuk VPS.
- Dead Letter Queue (DLQ) bawaan untuk handling failed message processing.
- Flexible routing via exchange types (direct, fanout, topic).
- Management UI memudahkan observability message flow tanpa tooling tambahan.
- Ekosistem client library matang untuk Go dan Rust (lapin crate).

**Kontra:**
- Tidak mendukung message replay seperti Kafka (event tidak bisa di-replay dari offset).
  Jika notification-service down dan pesan sudah expired di queue, pesan hilang.
- Throughput lebih rendah dari Kafka/NATS untuk volume sangat tinggi.
- Clustering RabbitMQ lebih kompleks dibanding NATS jika suatu hari perlu HA setup.

**Risiko:**
- Message loss apabila service consumer down lebih dari TTL queue. Mitigasi: set message
  TTL yang panjang (24 jam) untuk event kritis, aktifkan durable queue dan persistent
  message, DLQ untuk semua queue production.

---

### ADR-005: gRPC vs REST untuk Inter-Service Sync Communication

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Beberapa operasi memerlukan komunikasi sinkronus antar service: JWT verification, tier limit
check, tenant schema lookup. Pilihan protokol mempengaruhi latency, type safety, dan
kemudahan debugging.

#### Pilihan yang Dipertimbangkan

| Pilihan | Format | Type Safety | Latency | Debugging |
|---------|--------|-------------|---------|-----------|
| A — REST/JSON | JSON | Manual schema | Lebih tinggi | Mudah (curl) |
| B — gRPC + Protobuf (dipilih) | Binary | Compile-time | Lebih rendah | Butuh tooling |
| C — GraphQL Federation | JSON | Schema-based | Menengah | Moderate |

#### Keputusan

**Pilihan B: gRPC + Protobuf**

Justifikasi:
- Type safety compile-time via `.proto` schema mencegah contract mismatch antar service.
  Perubahan breaking di proto file akan fail at compile time, bukan runtime.
- Binary encoding Protobuf lebih efisien sekitar 30-40% dibanding JSON untuk payload berulang.
- gRPC mendukung deadline/timeout propagation bawaan — penting untuk request tracing.
- `grpc-go` dan `tonic` (Rust) adalah dua library production-grade yang interoperable.
- Untuk public API yang diakses frontend, tetap REST/JSON via api-gateway — gRPC hanya
  untuk internal service-to-service communication.

#### Konsekuensi

**Pro:**
- Strongly typed contract antar service — breaking changes terdeteksi saat compile.
- Performa lebih baik untuk high-frequency calls (JWT verify, tier check per request).
- Native support untuk deadline propagation dan cancellation.
- Code generation otomatis dari `.proto` untuk Go dan Rust.

**Kontra:**
- Debugging lebih sulit dibanding REST — perlu `grpcurl` atau Postman gRPC support.
- `.proto` file harus di-manage dan di-versioning dengan hati-hati.
- Browser tidak dapat langsung call gRPC tanpa gRPC-Web proxy.

**Risiko:**
- Proto schema versioning yang ceroboh dapat menyebabkan incompatibility. Mitigasi:
  gunakan field numbers yang tidak pernah di-reuse, tambah field dengan `optional`, jangan
  hapus field yang sudah ada (set deprecated).

---

### ADR-006: API Gateway Pattern — Single Entry Point

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Dengan 11 service, client (SvelteKit frontend) tidak boleh mengetahui topologi internal.
Sebuah single entry point diperlukan untuk menangani cross-cutting concerns: auth, rate
limiting, CORS, routing, dan request correlation.

#### Pilihan yang Dipertimbangkan

| Pilihan | Deskripsi |
|---------|-----------|
| A — API Gateway custom Go (dipilih) | Service Go yang menjadi single entry point |
| B — Kong / Traefik sebagai API Gateway | Infrastructure-level gateway |
| C — BFF (Backend for Frontend) | Gateway per client type |

#### Keputusan

**Pilihan A: Custom Go API Gateway**

Justifikasi:
- Kong (B) menambah satu layer infrastructure dengan resource footprint signifikan di VPS.
  Konfigurasi Kong via Admin API lebih kompleks dari code-based routing.
- Traefik sudah digunakan sebagai reverse proxy TLS termination — tidak perlu double up
  sebagai API gateway logic (rate limiting, JWT verify, request enrichment).
- Custom Go gateway memberikan full control atas logic: JWT verify menggunakan shared
  RS256 public key (tanpa roundtrip ke auth-service), tier limit enrichment, correlation ID
  injection, custom rate limiting per endpoint.
- BFF (C) over-engineering untuk satu type client pada tahap ini.

#### Konsekuensi

**Pro:**
- JWT verification di gateway tanpa gRPC roundtrip ke auth-service (menggunakan shared
  public key RSA) — latency lebih rendah untuk setiap request.
- Centralized rate limiting (Redis-backed, per-IP dan per-user-ID).
- Single CORS policy untuk seluruh API.
- Correlation ID (`X-Correlation-ID`) di-inject sekali di gateway dan di-propagate ke semua
  downstream service via request header.

**Kontra:**
- Gateway menjadi single point of failure — harus memiliki health check dan restart policy.
- Business routing logic (mana endpoint ke service mana) perlu di-maintain di satu tempat.
- Setiap service baru harus di-register di gateway routing config.

**Risiko:**
- Gateway bottleneck apabila tidak di-scale. Mitigasi: gateway harus stateless (state ada
  di Redis), sehingga dapat di-replicate dengan mudah jika diperlukan.

---

### ADR-007: Midtrans vs Xendit vs Stripe untuk Payment Indonesia

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

KasKu SaaS memerlukan payment gateway yang mendukung metode pembayaran lokal Indonesia:
QRIS, Virtual Account (BCA, Mandiri, BNI, BRI), GoPay, OVO. Target user adalah pengguna
Indonesia, sehingga gateway yang familiar dan populer di Indonesia menjadi prioritas.

#### Pilihan yang Dipertimbangkan

| Pilihan | MDR QRIS | VA | GoPay/OVO | Settlement | Dokumentasi |
|---------|----------|-----|-----------|------------|-------------|
| A — Midtrans (Gojek Group) (dipilih) | 0.7% | 4000/trx | Tersedia | T+1 | Sangat Baik |
| B — Xendit | 0.7% | 4000/trx | Tersedia | T+1 | Baik |
| C — Stripe | N/A | N/A | Tidak Tersedia | T+2 | Sangat Baik |

#### Keputusan

**Pilihan A: Midtrans**

Justifikasi:
- Midtrans (Gojek Group) adalah payment gateway terpopuler di Indonesia dengan track record
  panjang dan trust level tinggi di kalangan user Indonesia.
- Snap API Midtrans memudahkan implementasi dengan redirect URL — frontend tidak perlu
  handle payment UI sendiri.
- Webhook Midtrans menggunakan signature verification via
  `SHA512(order_id + status_code + gross_amount + server_key)` — mudah diimplementasikan
  dan diverifikasi.
- Stripe (C) tidak mendukung QRIS dan e-wallet lokal Indonesia.
- Xendit (B) adalah alternatif valid, namun Midtrans memiliki ekosistem SDK Go yang lebih
  mature dan dokumentasi lebih lengkap.

#### Konsekuensi

**Pro:**
- Dukungan penuh metode pembayaran Indonesia: QRIS, VA (7 bank), GoPay, OVO, Dana,
  Indomaret, Alfamart.
- Snap API — satu integration, semua payment method.
- Webhook reliable dengan retry mechanism dari Midtrans.
- Sandbox environment lengkap untuk development dan testing.

**Kontra:**
- Terikat vendor Indonesia — jika perlu ekspansi internasional, harus integrasi gateway baru.
- Settlement T+1 (bukan real-time) untuk subscription reconciliation.
- MDR tetap dikenakan per transaksi.

**Risiko:**
- Midtrans downtime saat payment flow menyebabkan subscription tidak aktif. Mitigasi:
  payment webhook dengan idempotency check, timeout handling di billing-service, dan manual
  reconciliation endpoint di admin-service.

---

### ADR-008: JWT RS256 + Refresh Token Rotation

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Authentication di arsitektur microservices memerlukan strategi yang memungkinkan setiap
service memverifikasi token tanpa harus call ke auth-service setiap request (overhead
terlalu tinggi). Sekaligus harus aman terhadap token theft dan replay attack.

#### Pilihan yang Dipertimbangkan

| Pilihan | Deskripsi |
|---------|-----------|
| A — JWT HS256 | Symmetric key, semua service bisa verify dan sign |
| B — JWT RS256 + refresh rotation (dipilih) | Asymmetric: auth-service sign, gateway verify |
| C — Opaque token + token introspection | Token dikirim ke auth-service untuk verify |

#### Keputusan

**Pilihan B: JWT RS256 + Refresh Token Rotation**

Justifikasi:
- RS256 asymmetric: hanya auth-service yang memegang private key (signing). Public key
  di-share ke api-gateway via environment variable atau file mount — gateway dapat verify
  tanpa network roundtrip.
- HS256 (A) memerlukan semua service memegang secret yang sama — jika satu service
  compromised, attacker dapat forge token untuk semua service.
- Opaque token (C) memerlukan network call ke auth-service per request — latency tinggi.
- Refresh token rotation: setiap penggunaan refresh token menghasilkan token baru, dan
  token lama direvoke. Deteksi token theft via "token reuse detection" — jika token revoked
  digunakan lagi, semua session user di-invalidate.
- Access token TTL: 15 menit. Refresh token TTL: 30 hari.

#### Konsekuensi

**Pro:**
- Gateway verify JWT tanpa network call — latency minimal untuk setiap request.
- Private key terisolasi di auth-service saja — blast radius kecil jika service lain compromised.
- Refresh token rotation + reuse detection memberikan keamanan kuat terhadap token theft.
- JWT claims berisi `user_id`, `tenant_schema`, `subscription_tier` — semua info yang
  dibutuhkan downstream service tanpa query tambahan.

**Kontra:**
- Access token tidak dapat di-revoke sebelum expired (TTL 15 menit). Mitigasi: Redis
  blacklist di api-gateway untuk token yang di-logout sebelum expired.
- Private key rotation memerlukan rolling restart auth-service dan update di api-gateway.

**Risiko:**
- JWT dengan claim `subscription_tier` dapat stale selama 15 menit setelah upgrade/downgrade.
  Mitigasi: acceptable untuk use case ini — tier enforcement lag 15 menit tidak signifikan
  bagi user experience.

---

### ADR-009: Docker Compose vs Kubernetes untuk VPS Deployment

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Deployment target adalah VPS (tidak ada managed Kubernetes). Pilihan orkestrasi container
harus seimbang antara kemudahan operasional dan reliability untuk 11 service.

#### Pilihan yang Dipertimbangkan

| Pilihan | Kompleksitas | Resource Overhead | HA Support | Learning Curve |
|---------|-------------|-------------------|------------|----------------|
| A — Docker Compose (dipilih) | Rendah | Minimal | Manual | Rendah |
| B — K3s (lightweight Kubernetes) | Menengah | 512MB+ | Bawaan | Menengah |
| C — Kubernetes full | Tinggi | 2GB+ | Bawaan | Tinggi |

#### Keputusan

**Pilihan A: Docker Compose**

Justifikasi:
- VPS dengan resource terbatas tidak dapat menanggung overhead K3s control plane (512MB+).
- Docker Compose v2 mendukung `depends_on` dengan health check, `profiles` untuk service
  grouping, dan `deploy.restart_policy` — sudah cukup untuk production di VPS.
- Traefik sebagai reverse proxy handle TLS termination, auto-renew Let's Encrypt.
- High availability di level aplikasi: `restart: unless-stopped` di semua service.
- Operasional sederhana: `docker compose up -d`, `docker compose logs`, rolling restart
  `docker compose up -d --no-deps --build {service}`.

#### Konsekuensi

**Pro:**
- Zero overhead orchestration — tidak ada control plane yang memakan resource.
- File `docker-compose.yml` adalah source of truth yang mudah dibaca dan di-audit.
- Tidak ada learning curve K8s untuk operasional sehari-hari.
- Volume mount langsung ke host untuk data persistence tanpa PV/PVC abstraction.

**Kontra:**
- Tidak ada auto-scaling horizontal — scaling manual dengan edit `docker-compose.yml`.
- Rolling update memerlukan brief downtime per service (bukan zero-downtime seperti K8s).
- Tidak ada built-in service mesh, load balancing untuk multi-instance service.

**Risiko:**
- Single VPS adalah single point of failure. Mitigasi: database backup harian ke object
  storage, runbook untuk disaster recovery, monitoring uptime eksternal (UptimeRobot).

---

### ADR-010: Clean Architecture per Service

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Setiap microservice harus memiliki struktur internal yang konsisten untuk memudahkan
onboarding, code review, dan maintenance. Clean Architecture sudah terbukti di monolith
KasKu v1 dan di-carry forward ke setiap service.

#### Keputusan

Setiap service mengadopsi Clean Architecture dengan 4 layer:
1. **domain** — entities, repository interfaces, domain errors (zero external dependency)
2. **usecase** — business logic, use case interfaces (dependency hanya ke domain)
3. **infrastructure** — implementasi repository (DB via pgx/SQLx), gRPC client, MQ publisher
4. **delivery** — HTTP handler (Gin/Axum), gRPC server, middleware

Dependency rule: outer layer boleh import inner, inner tidak boleh import outer.
Domain entity tidak boleh mengandung struct tag dari framework (struct tag json dari
`encoding/json` hanya di delivery layer DTO, bukan di domain entity).

#### Konsekuensi

**Pro:**
- Testability tinggi: usecase dapat di-unit-test dengan mock repository tanpa DB.
- Framework-agnostic domain — migrasi Gin ke framework lain tidak menyentuh domain/usecase.
- Onboarding mudah: developer baru langsung memahami struktur setiap service.

**Kontra:**
- Boilerplate lebih banyak — setiap entity memerlukan domain struct + DTO struct + mapper.
- Overkill untuk service yang sangat simple (health check only). Mitigasi: tetap diterapkan
  untuk konsistensi, walaupun layer tertentu tipis.

**Risiko:**
- Mapping overhead antara domain entity dan DTO di setiap request. Acceptable trade-off
  mengingat benefit isolation dan testability.

---

### ADR-011: Offline-First dengan IndexedDB Dipertahankan di SaaS

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Fitur offline-first (IndexedDB + Dexie.js + Workbox Service Worker) adalah diferensiasi
KasKu dari kompetitor. Di arsitektur SaaS, terdapat pertanyaan apakah offline-first masih
relevan dan feasible dengan multi-tenancy.

#### Keputusan

**Offline-first dipertahankan untuk semua tier (Free, Basic, Pro)**

Justifikasi:
- User Indonesia sering menghadapi koneksi tidak stabil. Offline capability adalah nilai
  jual utama yang tidak boleh hilang saat pivot ke SaaS.
- Di SaaS, offline data disimpan di IndexedDB dengan `user_id` prefix untuk isolasi
  jika suatu hari multi-account di browser (antisipasi masa depan).
- Tier limit (max transaksi, max akun) tetap di-enforce di server saat sync — bukan di
  client. Client boleh input offline, namun saat sync server menolak jika limit tercapai.
- sync-service (Rust) menangani semua kompleksitas konflik dan batch processing.

#### Konsekuensi

**Pro:**
- User tetap bisa input transaksi saat offline, sync saat online kembali.
- Competitive advantage dipertahankan.
- sync-service sebagai dedicated Rust service mampu menangani batch conflict resolution
  dengan performa tinggi.

**Kontra:**
- Tier enforcement di sisi server saat sync dapat mengejutkan user — input offline berhasil
  tapi ditolak saat sync. Mitigasi: frontend menampilkan warning "limit hampir tercapai"
  berdasarkan cached tier info.
- Kompleksitas sync bertambah di multi-tenant context — schema prefix harus di-propagate
  dari JWT claim ke sync-service.

**Risiko:**
- User Free tier yang sudah penuh limit (50 transaksi) dan melakukan input offline banyak
  akan mengalami frustasi saat sync ditolak. Mitigasi: tampilkan quota usage di dashboard
  dan offline status bar di frontend.

---

### ADR-012: Server Wins Conflict Resolution untuk Multi-Tenant Offline Sync

**Status:** Accepted
**Tanggal:** 2026-04-27

#### Konteks

Dalam arsitektur SaaS multi-tenant dengan offline-first, konflik dapat terjadi apabila
user yang sama login di dua device (desktop + mobile) dan kedua device melakukan edit
offline pada record yang sama. Strategi conflict resolution harus dipilih.

#### Pilihan yang Dipertimbangkan

| Pilihan | Deskripsi | Kompleksitas |
|---------|-----------|-------------|
| A — Server Wins (dipilih) | Server state selalu menang jika ada konflik | Rendah |
| B — Last Write Wins (LWW) | Record dengan `updated_at` terbaru menang | Menengah |
| C — CRDT (Conflict-free Replicated Data Type) | Merge otomatis tanpa konflik | Tinggi |

#### Keputusan

**Pilihan A: Server Wins**

Justifikasi:
- KasKu adalah financial tracker — data keuangan harus konsisten dan predictable. CRDT
  merge otomatis bisa menghasilkan angka saldo yang tidak intuitif.
- LWW (B) rawan clock skew jika device user memiliki jam tidak akurat.
- Server Wins sederhana, predictable, dan mudah dijelaskan kepada user: "data di server
  adalah yang paling akurat karena terakhir di-save secara online".
- sync-service Rust mengimplementasikan: saat konflik terdeteksi (record dengan `sync_id`
  yang sama memiliki `updated_at` berbeda), server version dipertahankan dan dikembalikan
  ke client sebagai "resolved" state.
- Client menerima `conflict_resolved` records dan meng-update IndexedDB lokal.

#### Konsekuensi

**Pro:**
- Implementasi sederhana dan dapat diprediksi.
- Tidak ada ambiguitas state — server selalu benar.
- Cocok untuk data keuangan dimana konsistensi lebih penting dari "preserving user input".

**Kontra:**
- User kehilangan input offline jika terjadi konflik dengan data server yang lebih baru.
  Ini dapat mengejutkan user.

**Risiko:**
- User yang edit offline tanpa sadar ada konflik akan kehilangan perubahan mereka.
  Mitigasi: frontend menampilkan toast notification "X record diperbarui dari server saat
  sync" agar user sadar ada conflict resolution yang terjadi.

---

## 2. System Architecture Diagram {#system-architecture-diagram}

```mermaid
graph TB
    subgraph Client["Client Layer"]
        Browser["SvelteKit PWA\n(Browser + IndexedDB)"]
    end

    subgraph Ingress["Ingress Layer"]
        Traefik["Traefik\n(TLS Termination\nLet's Encrypt)\n:443"]
    end

    subgraph Gateway["API Gateway Layer"]
        APIGW["api-gateway\n(Go + Gin)\n:8080\n[JWT Verify, Rate Limit,\nCORS, Correlation ID,\nRequest Routing]"]
    end

    subgraph Services["Service Layer (Internal Network)"]
        AuthSvc["auth-service\n(Go)\n:8081 / gRPC :9081\n[Register, Login,\nJWT RS256, Refresh,\nBrute Force Protect]"]
        UserSvc["user-service\n(Go)\n:8082 / gRPC :9082\n[Profile, Tenant\nProvisioning]"]
        BillingSvc["billing-service\n(Go)\n:8083 / gRPC :9083\n[Subscription,\nMidtrans, Invoice,\nTier Limit Check]"]
        FinanceSvc["finance-service\n(Go)\n:8084 / gRPC :9084\n[Accounts,\nBalance History]"]
        TxSvc["transaction-service\n(Go)\n:8085 / gRPC :9085\n[Transactions,\nCategories]"]
        InvestSvc["investment-service\n(Go)\n:8086 / gRPC :9086\n[Assets,\nUnit History]"]
        PriceSvc["price-service\n(Rust)\n:8087 / gRPC :9087\n[Real-time Price\nCoinGecko/metals.live\nTTL Cache]"]
        SyncSvc["sync-service\n(Rust)\n:8088\n[Batch Push/Pull,\nConflict Resolution\nServer Wins]"]
        NotifSvc["notification-service\n(Go)\n:8089\n[Email via SMTP/\nSendGrid,\nEvent-driven]"]
        AdminSvc["admin-service\n(Go)\n:8090\n[Admin Dashboard,\nUser Mgmt,\nSubscription Override]"]
    end

    subgraph Messaging["Message Broker"]
        RabbitMQ["RabbitMQ\n:5672 (AMQP)\n:15672 (Management UI)"]
    end

    subgraph Databases["Database Layer (PostgreSQL single instance)"]
        AuthDB[("kasku_auth")]
        BillingDB[("kasku_billing")]
        FinanceDB[("kasku_finance\n[schema per tenant]")]
        PriceDB[("kasku_price")]
        AdminDB[("kasku_admin")]
    end

    subgraph Cache["Cache Layer"]
        Redis["Redis\n:6379\n[Rate Limit Counters,\nJWT Blacklist]"]
    end

    subgraph External["External Services"]
        Midtrans["Midtrans\nSnap API"]
        CoinGecko["CoinGecko API"]
        MetalsLive["metals.live API"]
        SMTP["SMTP / SendGrid"]
    end

    Browser -->|HTTPS| Traefik
    Traefik -->|HTTP internal| APIGW

    APIGW -->|gRPC :9081| AuthSvc
    APIGW -->|gRPC :9083| BillingSvc
    APIGW -->|HTTP proxy| UserSvc
    APIGW -->|HTTP proxy| FinanceSvc
    APIGW -->|HTTP proxy| TxSvc
    APIGW -->|HTTP proxy| InvestSvc
    APIGW -->|HTTP proxy| PriceSvc
    APIGW -->|HTTP proxy| SyncSvc
    APIGW -->|HTTP proxy admin-net| AdminSvc
    APIGW --- Redis

    SyncSvc -->|gRPC :9084| FinanceSvc
    SyncSvc -->|gRPC :9085| TxSvc
    SyncSvc -->|gRPC :9086| InvestSvc

    AuthSvc --> AuthDB
    BillingSvc --> BillingDB
    FinanceSvc --> FinanceDB
    TxSvc --> FinanceDB
    InvestSvc --> FinanceDB
    SyncSvc --> FinanceDB
    PriceSvc --> PriceDB
    AdminSvc --> AdminDB
    AdminSvc -.->|read-only| AuthDB
    AdminSvc -.->|read-only| BillingDB

    AuthSvc -->|publish| RabbitMQ
    BillingSvc -->|publish| RabbitMQ
    UserSvc -->|consume| RabbitMQ
    NotifSvc -->|consume| RabbitMQ

    BillingSvc -->|HTTPS| Midtrans
    PriceSvc -->|HTTPS| CoinGecko
    PriceSvc -->|HTTPS| MetalsLive
    NotifSvc -->|SMTP/API| SMTP
```

---

## 3. Service Responsibility Matrix {#service-responsibility-matrix}

| Service | Tech | HTTP Port | gRPC Port | Database | gRPC Dependencies | Events Produced | Events Consumed |
|---------|------|-----------|-----------|----------|-------------------|-----------------|-----------------|
| api-gateway | Go + Gin | 8080 | — | Redis | auth-service (JWT verify opt.), billing-service (tier check) | — | — |
| auth-service | Go + Gin | 8081 | 9081 | kasku_auth | — | `user.registered` | — |
| user-service | Go + Gin | 8082 | 9082 | kasku_finance (DDL only) | — | — | `user.registered`, `subscription.expired` |
| billing-service | Go + Gin | 8083 | 9083 | kasku_billing | — | `payment.succeeded`, `payment.failed`, `subscription.expiring`, `subscription.expired`, `subscription.cancelled` | — |
| finance-service | Go + Gin | 8084 | 9084 | kasku_finance | — | — | — |
| transaction-service | Go + Gin | 8085 | 9085 | kasku_finance | — | — | — |
| investment-service | Go + Gin | 8086 | 9086 | kasku_finance | — | — | — |
| price-service | Rust + Axum | 8087 | 9087 | kasku_price | — | — | — |
| sync-service | Rust + Axum | 8088 | — | kasku_finance | finance-service (:9084), transaction-service (:9085), investment-service (:9086) | — | — |
| notification-service | Go + Gin | 8089 | — | — | — | — | `user.registered`, `payment.succeeded`, `payment.failed`, `subscription.expiring`, `subscription.expired`, `subscription.cancelled` |
| admin-service | Go + Gin | 8090 | — | kasku_admin, kasku_auth (R), kasku_billing (R) | — | — | — |

---

## 4. Inter-Service Communication Map {#inter-service-communication-map}

### 4.1 gRPC Call Matrix

| Caller | Callee | Proto Service | Method | Purpose |
|--------|--------|---------------|--------|---------|
| api-gateway | auth-service | `AuthInternal` | `CheckTokenBlacklist` | Cek Redis blacklist untuk token logout (opsional, bisa langsung Redis) |
| api-gateway | billing-service | `BillingInternal` | `GetUserTierLimits` | Enrich request context dengan tier limits sebelum proxy ke downstream |
| sync-service | finance-service | `FinanceInternal` | `UpsertFinancialAccounts`, `ListFinancialAccounts` | Batch sync financial accounts saat offline sync |
| sync-service | transaction-service | `TransactionInternal` | `UpsertTransactions`, `ListTransactions` | Batch sync transactions saat offline sync |
| sync-service | investment-service | `InvestmentInternal` | `UpsertInvestmentAssets`, `ListInvestmentAssets` | Batch sync investment assets saat offline sync |

### 4.2 RabbitMQ Event Map

Exchange tunggal: `kasku.events` (type: topic, durable: true)

| Routing Key | Producer | Consumer(s) | Payload |
|-------------|----------|-------------|---------|
| `user.registered` | auth-service | user-service, notification-service | `{user_id, email, username, registered_at}` |
| `payment.succeeded` | billing-service | notification-service | `{user_id, order_id, plan_id, plan_name, amount_idr, paid_at}` |
| `payment.failed` | billing-service | notification-service | `{user_id, order_id, reason, failed_at}` |
| `subscription.expiring` | billing-service (cron) | notification-service | `{user_id, email, plan_name, expires_at, days_remaining}` |
| `subscription.expired` | billing-service (cron) | user-service, notification-service | `{user_id, old_plan_name, downgraded_to: "FREE"}` |
| `subscription.cancelled` | billing-service | notification-service | `{user_id, plan_name, cancelled_at, effective_until}` |

Dead Letter Exchange: `kasku.events.dlx` (type: fanout)
Dead Letter Queue: `kasku.events.dlq` — semua unprocessable messages masuk sini untuk manual investigation.

Konfigurasi queue consumer:
- `durable: true`, `auto-delete: false`, `exclusive: false`
- `x-dead-letter-exchange: kasku.events.dlx`
- `x-message-ttl: 86400000` (24 jam)
- Max retry: 3x dengan exponential backoff (10s, 30s, 90s)

### 4.3 Sequence Diagram: User Register → Tenant Provisioning → Welcome Email

```mermaid
sequenceDiagram
    actor Client as SvelteKit Browser
    participant GW as api-gateway
    participant Auth as auth-service
    participant MQ as RabbitMQ
    participant User as user-service
    participant Notif as notification-service
    participant DB_Auth as kasku_auth
    participant DB_Finance as kasku_finance
    participant DB_Billing as kasku_billing
    participant SMTP as SMTP/SendGrid

    Client->>GW: POST /auth/register {email, username, password}
    GW->>GW: validate input format, inject X-Correlation-ID
    GW->>Auth: HTTP proxy POST /register
    Auth->>Auth: validate email uniqueness, password strength
    Auth->>Auth: hash password (Argon2id: time=3, mem=64MB, threads=4)
    Auth->>DB_Auth: INSERT users (email, username, password_hash, is_active=false)
    Auth->>Auth: generate 32-byte random verification token
    Auth->>Auth: compute SHA-256 hash of token
    Auth->>DB_Auth: INSERT email_verifications (user_id, token_hash, expires_at=+24h)
    Auth->>MQ: publish user.registered {user_id, email, username, registered_at}
    Auth-->>GW: 201 Created {user_id, message}
    GW-->>Client: 201 Created {user_id, message: "Silakan cek email untuk verifikasi"}

    par Tenant Provisioning (async)
        MQ->>User: consume user.registered (queue: kasku.user-service)
        User->>DB_Finance: SELECT provision_tenant(user_id::uuid)
        Note over DB_Finance: CREATE SCHEMA tenant_{sanitized_uuid}<br/>DDL semua tabel<br/>Seed categories default
        User->>DB_Billing: INSERT subscriptions (user_id, plan=FREE, status=ACTIVE)
        User->>MQ: ACK message
    and Welcome Email (async)
        MQ->>Notif: consume user.registered (queue: kasku.notification-service)
        Notif->>Notif: render welcome_email.html template
        Notif->>SMTP: send email (to: user email, subject: Selamat Datang di KasKu)
        Notif->>MQ: ACK message
    end
```

### 4.4 Sequence Diagram: Payment → Subscription Activation → Tier Update

```mermaid
sequenceDiagram
    actor Client as SvelteKit Browser
    participant GW as api-gateway
    participant Billing as billing-service
    participant MQ as RabbitMQ
    participant Notif as notification-service
    participant Midtrans as Midtrans Snap API
    participant DB_Billing as kasku_billing

    Client->>GW: POST /billing/subscribe {plan_id: "BASIC"}
    GW->>GW: verify JWT RS256, check rate limit
    GW->>Billing: HTTP proxy POST /subscribe {user_id from JWT claims}
    Billing->>DB_Billing: verify no active non-FREE subscription
    Billing->>DB_Billing: INSERT payments (status=PENDING, order_id=KASKU-{uid_prefix}-{ts})
    Billing->>Midtrans: POST /snap/v1/transactions {order_id, amount=29000, customer_detail}
    Midtrans-->>Billing: {token, redirect_url}
    Billing-->>GW: 201 {payment_id, redirect_url, snap_token}
    GW-->>Client: 201 {redirect_url, snap_token}

    Client->>Midtrans: User completes payment at redirect_url (QRIS/VA/GoPay)

    Midtrans->>GW: POST /billing/webhook/midtrans {order_id, status_code=200, gross_amount, signature_key, transaction_status=settlement}
    GW->>GW: skip JWT auth (webhook path), forward to billing-service
    GW->>Billing: POST /webhook/midtrans (raw Midtrans payload)
    Billing->>Billing: compute expected_sig = SHA512(order_id+status_code+gross_amount+SERVER_KEY)
    Billing->>Billing: verify signature_key == expected_sig
    Billing->>DB_Billing: idempotency check: midtrans_order_id already processed?
    Billing->>DB_Billing: UPDATE payments SET status=SETTLEMENT, midtrans_transaction_id=..., paid_at=now()
    Billing->>DB_Billing: UPDATE subscriptions SET status=ACTIVE, plan_id=BASIC_PLAN_ID, current_period_end=+30days
    Billing->>MQ: publish payment.succeeded {user_id, order_id, plan_id, plan_name, amount_idr, paid_at}
    Billing-->>GW: 200 OK
    GW-->>Midtrans: 200 OK (acknowledge)

    MQ->>Notif: consume payment.succeeded
    Notif->>Notif: render payment_receipt.html template
    Notif->>SMTP: send payment receipt email

    Note over Client,GW: JWT access token masih berisi old tier (max 15 menit)<br/>Setelah POST /auth/refresh: JWT baru berisi subscription_tier=BASIC
```

---

## 5. Tenant Provisioning Flow {#tenant-provisioning-flow}

### Langkah Detail Provisioning Saat User Baru Register

**Step 1 — auth-service: Register User**

```
1. Validasi input: email format (RFC 5322), password strength (min 8 char,
   1 uppercase, 1 digit), username format (alphanumeric + underscore, 3-30 char)
2. Cek duplikasi email di kasku_auth.public.users
3. Hash password menggunakan Argon2id:
   - time=3, memory=65536 KB (64MB), threads=4, keyLen=32
4. INSERT INTO kasku_auth.public.users
   (id=gen_random_uuid(), email, username, password_hash, is_active=false)
5. Generate verification token: 32-byte random via crypto/rand
6. Compute SHA-256 hash dari raw token untuk disimpan di DB
7. INSERT INTO kasku_auth.public.email_verifications
   (id, user_id, token_hash, expires_at=now()+24h)
8. Publish ke RabbitMQ exchange kasku.events, routing key user.registered:
   {user_id, email, username, registered_at}
9. Return HTTP 201: {user_id, message: "Verifikasi email dikirim"}
   Tidak return token (user belum aktif sampai email diverifikasi)
```

**Step 2 — user-service: Schema Provisioning (event-driven)**

```
1. Consume event user.registered dari queue kasku.user-service
2. Sanitize user_id UUID: ganti '-' dengan '_'
   Contoh: 550e8400-e29b-41d4 → 550e8400_e29b_41d4_a716_446655440000
3. Connect ke kasku_finance database
4. Panggil stored function: SELECT provision_tenant('550e8400-e29b-41d4-a716-446655440000'::uuid)
5. Fungsi provision_tenant:
   a. CREATE SCHEMA IF NOT EXISTS tenant_550e8400_e29b_41d4_a716_446655440000
   b. SET search_path = tenant_{sanitized_uuid}
   c. DDL: financial_accounts (lihat databaseScheme.md)
   d. DDL: balance_history
   e. DDL: investment_assets
   f. DDL: unit_history
   g. DDL: transactions
   h. DDL: categories
   i. DDL: sync_log
   j. INSERT seed categories (lihat databaseScheme.md untuk daftar lengkap)
6. INSERT ke kasku_billing.public.subscriptions:
   (user_id, plan_id=FREE_PLAN_ID, status=ACTIVE, current_period_start=now(),
    current_period_end=NULL [FREE tidak expired])
7. ACK message ke RabbitMQ
8. Jika error: NACK tanpa requeue → masuk DLQ untuk manual investigation
   Log: {level: error, event: tenant_provision_failed, user_id, error: ...}
```

**Step 3 — notification-service: Welcome Email (event-driven)**

```
1. Consume event user.registered dari queue kasku.notification-service
2. Load template welcome_email.html dari embedded FS (Go embed)
3. Render template dengan data:
   - username: dari event payload
   - verification_url: https://app.kasku.id/verify-email?token={raw_token}
   Note: raw_token tidak ada di event payload (hanya ada di auth-service DB)
   Perbaikan: auth-service meng-include verification_token (raw) di event payload
   yang hanya dikonsumsi notification-service dan tidak di-log
4. Kirim email via SMTP/SendGrid
5. Log: {level: info, event: welcome_email_sent, user_id, email: "u***@domain.com"}
6. ACK message ke RabbitMQ
```

### Error Handling pada Provisioning

| Step | Failure Scenario | Handling |
|------|-----------------|----------|
| auth-service INSERT users | Database down | HTTP 503, tidak publish event |
| auth-service publish event | RabbitMQ down | Return 503 ke client, roll back INSERT |
| user-service CREATE SCHEMA | Database down | NACK ke RabbitMQ, masuk DLQ |
| user-service CREATE SCHEMA | Schema sudah ada (retry idempotent) | `IF NOT EXISTS` aman |
| user-service INSERT subscription | Database down | NACK ke RabbitMQ, masuk DLQ |
| notification-service send email | SMTP down | NACK ke RabbitMQ, retry 3x, lalu DLQ |
| notification-service send email | Invalid template | NACK ke DLQ, alert ke admin |

---

## 6. Clean Architecture per Service {#clean-architecture-per-service}

### 6.1 Template Go Service

```
{service-name}/
├── cmd/
│   └── server/
│       └── main.go                        # Entry point: load config, init DI, start HTTP + gRPC server
├── internal/
│   ├── domain/
│   │   ├── entity/
│   │   │   └── {entity}.go                # Pure domain structs, tidak ada framework tag
│   │   ├── repository/
│   │   │   └── {entity}_repository.go     # Repository interfaces (ports)
│   │   └── errors/
│   │       └── domain_errors.go           # Typed domain errors: ErrNotFound, ErrConflict, ErrLimitExceeded
│   ├── usecase/
│   │   ├── {feature}_usecase.go           # Business logic implementation
│   │   └── {feature}_usecase_interface.go # Use case interfaces
│   ├── infrastructure/
│   │   ├── persistence/
│   │   │   └── postgres_{entity}_repository.go  # pgx/v5 implementation
│   │   ├── grpc/
│   │   │   └── {service}_grpc_client.go   # gRPC client wrappers (adapters)
│   │   └── messaging/
│   │       └── rabbitmq_publisher.go      # RabbitMQ event publisher
│   └── delivery/
│       ├── http/
│       │   ├── dto/
│       │   │   └── {feature}_dto.go       # Request/Response structs dengan json tags
│       │   ├── handler/
│       │   │   └── {feature}_handler.go   # Gin handlers: validate DTO, call usecase, map response
│       │   ├── middleware/
│       │   │   ├── auth_middleware.go      # JWT extraction dari header
│       │   │   └── correlation_id_middleware.go
│       │   └── router.go
│       └── grpc/
│           └── {service}_grpc_server.go   # gRPC server: proto ↔ domain mapping
├── proto/
│   └── {service}/v1/
│       └── {service}.proto
├── configs/
│   └── config.go                          # Struct config, load dari os.Getenv() dengan validasi
├── Dockerfile
└── go.mod
```

**Aturan wajib Go services:**

- `internal/domain/` tidak boleh import apapun selain stdlib Go. Zero framework dependency.
- `internal/usecase/` tidak boleh import `infrastructure/` atau `delivery/`. Hanya `domain/`.
- Handler di `delivery/http/handler/` tidak boleh mengandung business logic — hanya
  parse request, call usecase, map response.
- Semua repository dibuat via constructor yang menerima `*pgxpool.Pool` sebagai dependency
  (dependency injection via constructor, bukan global variable).
- Config di-load dari `os.Getenv()` dengan validasi di `main.go` — jika env var wajib
  tidak ada, program exit dengan error message yang jelas.
- Structured logging via `zerolog` dengan field `correlation_id`, `service`, `user_id`,
  `event` di setiap log line.
- Untuk service yang akses `kasku_finance`, semua repository method wajib menerima
  parameter `tenantSchema string` yang digunakan sebagai schema prefix di setiap query.

### 6.2 Template Rust Service (price-service, sync-service)

```
{service-name}/
├── src/
│   ├── main.rs                            # Entry point: init tracing, load config, start servers
│   ├── domain/
│   │   ├── mod.rs
│   │   ├── entity.rs                      # Pure domain structs, no serde derives
│   │   ├── repository.rs                  # Repository traits (async_trait)
│   │   └── errors.rs                      # Domain error enum (thiserror)
│   ├── application/
│   │   ├── mod.rs
│   │   └── {feature}_usecase.rs           # Business logic, depends only on domain traits
│   ├── infrastructure/
│   │   ├── mod.rs
│   │   ├── persistence/
│   │   │   └── postgres_{entity}.rs       # SQLx implementation of repository traits
│   │   └── external/
│   │       └── {api_name}_client.rs       # reqwest HTTP client for external APIs
│   └── interfaces/
│       ├── mod.rs
│       ├── http/
│       │   ├── mod.rs
│       │   ├── dto/
│       │   │   └── {feature}_dto.rs       # Serde structs for request/response
│       │   ├── handler/
│       │   │   └── {feature}_handler.rs   # Axum handlers
│       │   └── router.rs
│       └── grpc/
│           └── {service}_server.rs        # Tonic gRPC server implementation
├── proto/
│   └── {service}/v1/
│       └── {service}.proto
├── build.rs                               # tonic_build::compile_protos()
├── Cargo.toml
└── Dockerfile
```

**Aturan wajib Rust services:**

- `domain/` tidak boleh mengandung `use sqlx::*`, `use axum::*`, atau external crate lainnya.
- `application/` hanya depend ke `domain/` traits via generics atau `dyn Trait`.
- Error handling: `thiserror` untuk domain errors, `anyhow` di application layer.
- Tidak ada `.unwrap()` atau `.expect()` di production path — semua `?` operator dengan
  proper `From<E>` implementation.
- `serde::{Serialize, Deserialize}` hanya di `interfaces/http/dto/` (DTO structs) dan
  `interfaces/grpc/` — bukan di domain entity.
- Logging via `tracing` crate dengan `tracing-subscriber` JSON formatter.
- Semua async function menggunakan `tokio` runtime.
- Database queries menggunakan `sqlx::query!` atau `sqlx::query_as!` macro untuk
  compile-time type checking.

---

## 7. Security Architecture {#security-architecture}

### 7.1 JWT RS256 Key Management

```
auth-service:
  - Menyimpan RSA-4096 private key dari env var JWT_PRIVATE_KEY (base64-encoded PEM)
  - Menandatangani access token dan refresh token dengan private key
  - Access token TTL: 15 menit (tidak dapat di-revoke kecuali via Redis blacklist)
  - JWT Claims structure:
    {
      "sub": "550e8400-e29b-41d4-a716-446655440000",  // user_id
      "email": "user@example.com",                    // untuk display, tidak untuk auth
      "tenant_schema": "tenant_550e8400_e29b_...",    // schema prefix untuk kasku_finance queries
      "subscription_tier": "BASIC",                   // FREE | BASIC | PRO
      "iat": 1714211200,
      "exp": 1714212100                               // iat + 900 seconds
    }

api-gateway:
  - Menyimpan RSA public key dari env var JWT_PUBLIC_KEY (base64-encoded PEM)
  - Memverifikasi signature setiap request tanpa network call ke auth-service
  - Memeriksa Redis blacklist key jti:{token_jti} untuk token yang di-logout
  - Me-reject token dengan alg selain RS256 (algorithm confusion attack prevention)
  - Me-inject user_id, tenant_schema, subscription_tier sebagai request header
    ke downstream service: X-User-ID, X-Tenant-Schema, X-Subscription-Tier

Refresh Token:
  - Raw token: 32-byte cryptographically random, disimpan di HttpOnly Secure cookie
  - SHA-256 hash dari raw token disimpan di kasku_auth.public.refresh_tokens
  - TTL: 30 hari (720 jam)
  - Rotation: POST /auth/refresh menghasilkan token_pair baru, token lama di-revoke
  - Reuse detection: jika refresh token yang sudah di-revoke digunakan kembali,
    SEMUA refresh token milik user tersebut di-revoke (account-wide logout)
```

### 7.2 Midtrans Webhook Verification

```
Endpoint: POST /billing/webhook/midtrans

Proses verifikasi (WAJIB sebelum processing apapun):
1. Baca JSON payload dari request body
2. Ekstrak fields: order_id, status_code, gross_amount, signature_key
3. Compute: expected = SHA512(order_id + status_code + gross_amount + MIDTRANS_SERVER_KEY)
4. Compare: signature_key (dari payload) == expected (computed)
5. Jika tidak cocok: return 400 Bad Request, log security event
6. Idempotency check: SELECT 1 FROM payments WHERE midtrans_order_id = $1 AND status != 'PENDING'
7. Jika sudah diproses sebelumnya: return 200 OK (idempotent response)
8. Proses payment state transition sesuai transaction_status dari payload
```

### 7.3 OWASP Top 10 Mitigations

| OWASP | Risk | Mitigasi di KasKu SaaS |
|-------|------|------------------------|
| A01: Broken Access Control | User akses data tenant lain | Schema isolation: tenant_schema dari JWT claim. Setiap query ke kasku_finance wajib gunakan schema dari request context, bukan dari request body. |
| A02: Cryptographic Failures | Password/token exposure | Argon2id untuk password, SHA-256 untuk refresh token storage, RS256 untuk JWT, TLS 1.2+ via Traefik untuk semua komunikasi in-transit. |
| A03: Injection | SQL injection | pgx named parameters (`@name` style atau `$1` positional), SQLx `query!` macro. String interpolation ke SQL dilarang absolut. |
| A04: Insecure Design | Rate limit bypass, SSRF | Redis sliding window rate limit di api-gateway. price-service whitelist domain CoinGecko dan metals.live untuk mencegah SSRF. |
| A05: Security Misconfiguration | Open ports, default credential | Hanya api-gateway yang expose port via Traefik. admin-service di isolated network. Semua credential dari env vars, tidak ada default. |
| A06: Vulnerable Components | CVE di dependency | `govulncheck ./...` dan `cargo audit` di CI pipeline. Dependency update quarterly. |
| A07: Auth Failures | Brute force | Lockout setelah 5 gagal dalam 15 menit di auth-service. Account unlocks otomatis setelah 15 menit. |
| A08: Software Integrity | Supply chain attack | `go.sum` dan `Cargo.lock` di-commit. Dockerfile menggunakan hash image (sha256), bukan tag floating. |
| A09: Logging Failures | PII di log | Structured logging policy: email di-mask di semua log. Password/token tidak pernah di-log. `ip_address` hanya di auth-service untuk security audit events. |
| A10: SSRF | Request ke internal service | price-service hanya boleh request ke domain whitelist: `api.coingecko.com`, `api.metals.live`. Validasi sebelum HTTP request. |

### 7.4 Rate Limiting di API Gateway (Redis Sliding Window)

```
Konfigurasi rate limit per endpoint (per IP kecuali disebutkan):

POST /auth/register:           5 req / 15 menit / IP
POST /auth/login:              10 req / menit / IP, + 5 req / menit / email
POST /auth/refresh:            20 req / menit / user_id
POST /auth/forgot-password:    3 req / jam / email
POST /auth/reset-password:     5 req / 15 menit / IP
POST /billing/webhook/midtrans: 100 req / menit / IP (Midtrans IPs)
GET,POST,PUT,DELETE /accounts: 200 req / menit / user_id
GET,POST /transactions:        200 req / menit / user_id
POST /sync/batch:              30 req / menit / user_id
Default semua endpoint:        200 req / menit / user_id

Response headers pada setiap request:
  X-RateLimit-Limit:     {configured_limit}
  X-RateLimit-Remaining: {remaining_in_window}
  X-RateLimit-Reset:     {unix_timestamp_window_resets}

HTTP 429 Too Many Requests jika limit terlampaui.
Retry-After header disertakan pada response 429.
```

### 7.5 Network Isolation (Docker Networks)

```
kasku-public:
  - Services: traefik, api-gateway
  - Satu-satunya network yang terhubung ke internet

kasku-internal:
  - Services: api-gateway, auth-service, user-service, billing-service,
              finance-service, transaction-service, investment-service,
              price-service, sync-service, notification-service
  - api-gateway bisa proxy ke semua service ini

kasku-admin:
  - Services: api-gateway, admin-service
  - admin-service TIDAK ada di kasku-internal
  - Routing /admin/** hanya di-forward setelah api-gateway verifikasi AdminBearerAuth

kasku-data:
  - Services: semua service + postgres + rabbitmq + redis
  - Database, MQ, Cache hanya accessible dari service layer
  - Tidak ada akses langsung dari kasku-public
```

---

## 8. Deployment Architecture {#deployment-architecture}

### 8.1 Docker Compose Topology (Ringkasan)

Topologi lengkap Docker Compose:

```
Traefik (:443)
  └─> api-gateway (:8080)           # [kasku-public, kasku-internal, kasku-admin, kasku-data]
        ├─> auth-service (:8081)    # [kasku-internal, kasku-data]
        ├─> user-service (:8082)    # [kasku-internal, kasku-data]
        ├─> billing-service (:8083) # [kasku-internal, kasku-data]
        ├─> finance-service (:8084) # [kasku-internal, kasku-data]
        ├─> transaction-service (:8085) # [kasku-internal, kasku-data]
        ├─> investment-service (:8086)  # [kasku-internal, kasku-data]
        ├─> price-service (:8087)   # [kasku-internal, kasku-data]
        ├─> sync-service (:8088)    # [kasku-internal, kasku-data]
        ├─> notification-service (:8089) # [kasku-internal, kasku-data]
        └─> admin-service (:8090)   # [kasku-admin, kasku-data] ONLY

Infrastructure:
  postgres       [kasku-data]  # PostgreSQL 16-alpine, multi-database
  rabbitmq       [kasku-data]  # RabbitMQ 3.13-management-alpine
  redis          [kasku-data]  # Redis 7-alpine
  traefik        [kasku-public]
```

### 8.2 Environment Variables per Service (12-Factor Compliance)

Semua service hanya membaca config dari environment variables. Tidak ada config hardcoded.

**api-gateway:**
```
APP_ENV=production
HTTP_PORT=8080
JWT_PUBLIC_KEY=<RSA-4096 public key PEM, base64url encoded>
REDIS_URL=redis://:${REDIS_PASSWORD}@redis:6379/0
AUTH_SERVICE_GRPC_ADDR=auth-service:9081
BILLING_SERVICE_GRPC_ADDR=billing-service:9083
CORS_ALLOWED_ORIGINS=https://app.kasku.id
RATE_LIMIT_REDIS_KEY_PREFIX=kasku:rl:
LOG_LEVEL=info
```

**auth-service:**
```
APP_ENV=production
HTTP_PORT=8081
GRPC_PORT=9081
DATABASE_URL=postgres://kasku_auth_svc:${AUTH_DB_PASS}@postgres:5432/kasku_auth?sslmode=disable
JWT_PRIVATE_KEY=<RSA-4096 private key PEM, base64url encoded>
JWT_PUBLIC_KEY=<RSA-4096 public key PEM, base64url encoded>
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=720h
RABBITMQ_URL=amqp://kasku_mq_user:${MQ_PASS}@rabbitmq:5672/
ARGON2_TIME=3
ARGON2_MEMORY_KB=65536
ARGON2_THREADS=4
ARGON2_KEY_LENGTH=32
BRUTE_FORCE_MAX_ATTEMPTS=5
BRUTE_FORCE_LOCKOUT_DURATION=15m
LOG_LEVEL=info
```

**user-service:**
```
APP_ENV=production
HTTP_PORT=8082
GRPC_PORT=9082
FINANCE_DATABASE_URL=postgres://kasku_user_svc:${USER_DB_PASS}@postgres:5432/kasku_finance?sslmode=disable
BILLING_DATABASE_URL=postgres://kasku_user_svc:${USER_DB_PASS}@postgres:5432/kasku_billing?sslmode=disable
RABBITMQ_URL=amqp://kasku_mq_user:${MQ_PASS}@rabbitmq:5672/
LOG_LEVEL=info
```

**billing-service:**
```
APP_ENV=production
HTTP_PORT=8083
GRPC_PORT=9083
DATABASE_URL=postgres://kasku_billing_svc:${BILLING_DB_PASS}@postgres:5432/kasku_billing?sslmode=disable
MIDTRANS_SERVER_KEY=${MIDTRANS_SERVER_KEY}
MIDTRANS_CLIENT_KEY=${MIDTRANS_CLIENT_KEY}
MIDTRANS_IS_PRODUCTION=true
MIDTRANS_SNAP_URL=https://app.midtrans.com/snap/v1
RABBITMQ_URL=amqp://kasku_mq_user:${MQ_PASS}@rabbitmq:5672/
SUBSCRIPTION_EXPIRY_WARNING_DAYS=3
LOG_LEVEL=info
```

**finance-service:**
```
APP_ENV=production
HTTP_PORT=8084
GRPC_PORT=9084
DATABASE_URL=postgres://kasku_finance_svc:${FINANCE_DB_PASS}@postgres:5432/kasku_finance?sslmode=disable
LOG_LEVEL=info
```

**transaction-service:**
```
APP_ENV=production
HTTP_PORT=8085
GRPC_PORT=9085
DATABASE_URL=postgres://kasku_transaction_svc:${TX_DB_PASS}@postgres:5432/kasku_finance?sslmode=disable
LOG_LEVEL=info
```

**investment-service:**
```
APP_ENV=production
HTTP_PORT=8086
GRPC_PORT=9086
DATABASE_URL=postgres://kasku_investment_svc:${INVEST_DB_PASS}@postgres:5432/kasku_finance?sslmode=disable
LOG_LEVEL=info
```

**price-service (Rust):**
```
APP_ENV=production
HTTP_PORT=8087
GRPC_PORT=9087
DATABASE_URL=postgres://kasku_price_svc:${PRICE_DB_PASS}@postgres:5432/kasku_price?sslmode=disable
COINGECKO_BASE_URL=https://api.coingecko.com/api/v3
COINGECKO_API_KEY=${COINGECKO_API_KEY}
METALS_LIVE_BASE_URL=https://api.metals.live/v1
PRICE_CACHE_TTL_SECONDS=900
EXTERNAL_REQUEST_TIMEOUT_SECONDS=10
LOG_LEVEL=info
```

**sync-service (Rust):**
```
APP_ENV=production
HTTP_PORT=8088
DATABASE_URL=postgres://kasku_sync_svc:${SYNC_DB_PASS}@postgres:5432/kasku_finance?sslmode=disable
FINANCE_SERVICE_GRPC_ADDR=finance-service:9084
TRANSACTION_SERVICE_GRPC_ADDR=transaction-service:9085
INVESTMENT_SERVICE_GRPC_ADDR=investment-service:9086
SYNC_BATCH_SIZE_LIMIT=500
LOG_LEVEL=info
```

**notification-service:**
```
APP_ENV=production
HTTP_PORT=8089
RABBITMQ_URL=amqp://kasku_mq_user:${MQ_PASS}@rabbitmq:5672/
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=587
SMTP_USERNAME=${SMTP_USERNAME}
SMTP_PASSWORD=${SMTP_PASSWORD}
SMTP_FROM_ADDRESS=noreply@kasku.id
SMTP_FROM_NAME=KasKu
APP_BASE_URL=https://app.kasku.id
LOG_LEVEL=info
```

**admin-service:**
```
APP_ENV=production
HTTP_PORT=8090
ADMIN_DATABASE_URL=postgres://kasku_admin_svc:${ADMIN_DB_PASS}@postgres:5432/kasku_admin?sslmode=disable
AUTH_DATABASE_READ_URL=postgres://kasku_admin_read:${ADMIN_READ_DB_PASS}@postgres:5432/kasku_auth?sslmode=disable
BILLING_DATABASE_READ_URL=postgres://kasku_admin_read:${ADMIN_READ_DB_PASS}@postgres:5432/kasku_billing?sslmode=disable
ADMIN_JWT_SECRET=${ADMIN_JWT_SECRET}
LOG_LEVEL=info
```

---

## 9. Observability {#observability}

### 9.1 Structured Logging

Semua service menggunakan structured JSON logging ke stdout. Log aggregation (Loki, ELK)
dapat di-add kemudian via Docker log driver.

Contoh log line dari billing-service:
```json
{
  "timestamp": "2026-04-27T10:00:00.000Z",
  "level": "info",
  "service": "billing-service",
  "version": "1.0.0",
  "correlation_id": "c3d4e5f6-a1b2-c3d4-e5f6-a1b2c3d4e5f6",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "event": "payment_webhook_processed",
  "order_id": "KASKU-550e8400-1714211200",
  "payment_status": "settlement",
  "duration_ms": 45
}
```

**Kebijakan PII di log (wajib):**

| Field | Policy |
|-------|--------|
| `email` | Mask selalu: `u***@domain.com` |
| `password` | TIDAK PERNAH di-log dalam bentuk apapun |
| `token` / `refresh_token` | TIDAK PERNAH di-log |
| `ip_address` | Hanya di auth-service untuk security audit events |
| `user_id` | Boleh di-log (UUID, bukan PII langsung) |
| `order_id` | Boleh di-log |

**Propagasi `correlation_id`:**

1. api-gateway baca `X-Correlation-ID` dari request header (atau generate UUID v4 baru)
2. Inject ke semua downstream HTTP request sebagai header `X-Correlation-ID`
3. Semua service baca header ini di middleware dan simpan di request context
4. Setiap log line dalam scope request tersebut menyertakan `correlation_id`
5. gRPC: propagasi via metadata key `x-correlation-id`

### 9.2 Health Check Endpoint

Semua service mengimplementasikan `GET /health`:

```json
{
  "status": "healthy",
  "service": "billing-service",
  "version": "1.0.0",
  "timestamp": "2026-04-27T10:00:00.000Z",
  "dependencies": {
    "database": "healthy",
    "rabbitmq": "healthy"
  }
}
```

- HTTP 200 jika semua dependency healthy
- HTTP 503 jika salah satu dependency unhealthy
- Dependency check: ping database (simple `SELECT 1`), ping RabbitMQ connection

Docker Compose healthcheck menggunakan endpoint ini:
```yaml
healthcheck:
  test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:{port}/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 15s
```

### 9.3 RabbitMQ Dead Letter Queue

Arsitektur DLQ:
```
Exchange: kasku.events (topic) → routing → Consumer Queue
                                              ↓ (on NACK / TTL)
Exchange: kasku.events.dlx (fanout) → Queue: kasku.events.dlq

Monitoring: alert via log jika message count di kasku.events.dlq > 0
Manual investigation: akses RabbitMQ Management UI di :15672
```

### 9.4 Graceful Shutdown

Semua service menghandle SIGTERM dan SIGINT:

```
1. Stop accepting new HTTP connections (HTTP server.Shutdown dengan context timeout 30s)
2. Stop gRPC server (grpcServer.GracefulStop())
3. Tunggu in-flight requests selesai (max 30 detik)
4. Stop RabbitMQ consumer (channel.Cancel)
5. Drain koneksi database pool (pool.Close())
6. Flush semua log yang masih di buffer
7. Exit code 0
```

Docker Compose setting: `stop_grace_period: 35s` untuk semua service (5s buffer di atas
30s application shutdown timeout).

---

*Dokumen ini adalah source of truth arsitektur KasKu SaaS v2.0.*
*Update ADR wajib disertai review dan persetujuan dari Principal Architect (TubsAMY).*
*Terakhir diperbarui: 2026-04-27*
