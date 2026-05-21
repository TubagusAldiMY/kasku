FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY observability-go ./observability-go
COPY investment-service/go.mod investment-service/go.sum* ./investment-service/
WORKDIR /app/investment-service
RUN go mod download 2>/dev/null || true
WORKDIR /app
COPY investment-service ./investment-service
WORKDIR /app/investment-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/investment-service ./cmd/server/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget && \
    addgroup -S kasku && adduser -S -G kasku kasku
COPY --from=builder --chown=kasku:kasku /bin/investment-service /app/investment-service
USER kasku
EXPOSE 8086 9086
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8086/health || exit 1
CMD ["/app/investment-service"]
