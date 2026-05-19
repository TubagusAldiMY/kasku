FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY observability-go ./observability-go
COPY finance-service/go.mod finance-service/go.sum ./finance-service/
WORKDIR /app/finance-service
RUN go mod download
WORKDIR /app
COPY finance-service ./finance-service
WORKDIR /app/finance-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o finance-service ./cmd/server

FROM alpine:3.19
RUN addgroup -S kasku && adduser -S kasku -G kasku
WORKDIR /app
COPY --from=builder /app/finance-service/finance-service .
COPY --from=builder /app/finance-service/migrations ./migrations
USER kasku
EXPOSE 8084 9084
CMD ["./finance-service"]
