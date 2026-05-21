# ──────────────────────────────────────────────────
# KasKu price-service — Multi-stage Rust build
# ──────────────────────────────────────────────────
FROM rust:1.88-slim-bookworm AS builder

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
# Touch source files so cargo sees them as newer than the dummy-build binary.
# Docker COPY sets mtime=0 (epoch), which is older than the dummy binary — cargo
# would incorrectly skip recompilation without this.
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
COPY --from=builder --chown=kasku:kasku /app/target/release/price-service /app/price-service
COPY --chown=kasku:kasku migrations/ /app/migrations/

USER kasku
EXPOSE 8087 9087
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8087/health || exit 1
CMD ["/app/price-service"]
