# ──────────────────────────────────────────────────
# KasKu sync-service — Multi-stage Rust build
# ──────────────────────────────────────────────────
FROM rust:1.88-slim-bookworm AS builder

RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY Cargo.toml Cargo.lock* ./

# Create dummy source to cache dependency compilation
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release 2>/dev/null || true
RUN rm -rf src

# Copy actual source and build
COPY src/ src/
RUN find src -name "*.rs" -exec touch {} + && cargo build --release

# ──────────────────────────────────────────────────
# Runtime stage — minimal image
# ──────────────────────────────────────────────────
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r kasku && useradd -r -g kasku kasku

WORKDIR /app
COPY --from=builder --chown=kasku:kasku /app/target/release/sync-service /app/sync-service

USER kasku
EXPOSE 8088
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8088/health || exit 1
CMD ["/app/sync-service"]
