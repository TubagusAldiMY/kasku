# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Install ca-certificates for HTTPS/gRPC calls
RUN apk add --no-cache ca-certificates git

WORKDIR /build

# Download dependencies terlebih dahulu (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary — CGO disabled untuk static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o api-gateway ./cmd/server/main.go

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates untuk koneksi TLS/gRPC ke billing-service
RUN apk add --no-cache ca-certificates tzdata

# Non-root user untuk keamanan
RUN addgroup -g 1001 -S kasku && \
    adduser -u 1001 -S kasku -G kasku

WORKDIR /app

# Copy binary dari builder
COPY --from=builder /build/api-gateway .

# Ganti ke non-root user
USER kasku

EXPOSE 8080

ENTRYPOINT ["/app/api-gateway"]
