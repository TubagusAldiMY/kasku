FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY observability-go ./observability-go
COPY auth-service/go.mod auth-service/go.sum ./auth-service/
WORKDIR /app/auth-service
RUN go mod download

WORKDIR /app
COPY auth-service ./auth-service
WORKDIR /app/auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /auth-service ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kasku && adduser -S -G kasku kasku

WORKDIR /app
COPY --from=builder /auth-service .
COPY --from=builder --chown=kasku:kasku /app/auth-service/migrations ./migrations/

USER kasku
EXPOSE 8081
ENTRYPOINT ["./auth-service"]
