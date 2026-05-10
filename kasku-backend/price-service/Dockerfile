# ──────────────────────────────────────────────────
# KasKu price-service — Multi-stage Rust build
# ──────────────────────────────────────────────────
FROM rust:1.82-slim-bookworm AS builder

RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY Cargo.toml Cargo.lock* ./
COPY build.rs ./
COPY proto/ proto/

# Create dummy source to cache dependency compilation
RUN mkdir src && echo "fn main() {}" > src/main.rs && \
    mkdir -p src/proto_gen && touch src/proto_gen/mod.rs
RUN cargo build --release 2>/dev/null || true
RUN rm -rf src

# Copy actual source and build
COPY src/ src/
COPY migrations/ migrations/
RUN cargo build --release

# ──────────────────────────────────────────────────
# Runtime stage — minimal image
# ──────────────────────────────────────────────────
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/target/release/price-service /app/price-service
COPY migrations/ /app/migrations/

EXPOSE 8087 9087

CMD ["/app/price-service"]
