FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy go.mod dan go.sum terlebih dahulu untuk memanfaatkan Docker layer cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o billing-service ./cmd/server

# ─── Runtime image ─────────────────────────────────────────────────────────────
FROM alpine:3.19

# Jalankan sebagai non-root user untuk keamanan
RUN addgroup -S kasku && adduser -S kasku -G kasku

WORKDIR /app
COPY --from=builder /app/billing-service .
COPY --from=builder /app/migrations ./migrations

USER kasku

EXPOSE 8083 9083

CMD ["./billing-service"]
