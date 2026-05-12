FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /auth-service ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S kasku && adduser -S -G kasku kasku

WORKDIR /app
COPY --from=builder /auth-service .
COPY --chown=kasku:kasku migrations/ ./migrations/

USER kasku
EXPOSE 8081
ENTRYPOINT ["./auth-service"]
