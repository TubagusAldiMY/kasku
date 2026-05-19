FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy go.mod dan go.sum terlebih dahulu untuk memanfaatkan Docker layer cache
COPY observability-go ./observability-go
COPY billing-service/go.mod billing-service/go.sum ./billing-service/
WORKDIR /app/billing-service
RUN go mod download

WORKDIR /app
COPY billing-service ./billing-service
WORKDIR /app/billing-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o billing-service ./cmd/server

# ─── Runtime image ─────────────────────────────────────────────────────────────
FROM alpine:3.19

# Jalankan sebagai non-root user untuk keamanan
RUN addgroup -S kasku && adduser -S kasku -G kasku

WORKDIR /app
COPY --from=builder /app/billing-service/billing-service .
COPY --from=builder /app/billing-service/migrations ./migrations

USER kasku

EXPOSE 8083 9083

CMD ["./billing-service"]
