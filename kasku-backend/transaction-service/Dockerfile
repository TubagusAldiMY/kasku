FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY observability-go ./observability-go
COPY transaction-service/go.mod transaction-service/go.sum ./transaction-service/
WORKDIR /app/transaction-service
RUN go mod download
WORKDIR /app
COPY transaction-service ./transaction-service
WORKDIR /app/transaction-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o transaction-service ./cmd/server

FROM alpine:3.19
RUN addgroup -S kasku && adduser -S kasku -G kasku
WORKDIR /app
COPY --from=builder /app/transaction-service/transaction-service .
COPY --from=builder /app/transaction-service/migrations ./migrations
USER kasku
EXPOSE 8085 9085
CMD ["./transaction-service"]
