FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY observability-go ./observability-go
COPY user-service/go.mod user-service/go.sum ./user-service/
WORKDIR /app/user-service
RUN go mod download
WORKDIR /app
COPY user-service ./user-service
WORKDIR /app/user-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o user-service ./cmd/server

FROM alpine:3.19
RUN addgroup -S kasku && adduser -S kasku -G kasku
WORKDIR /app
COPY --from=builder /app/user-service/user-service .
USER kasku
EXPOSE 8082 9082
CMD ["./user-service"]
