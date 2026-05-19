FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY observability-go ./observability-go
COPY notification-service/go.mod notification-service/go.sum ./notification-service/
WORKDIR /app/notification-service
RUN go mod download
WORKDIR /app
COPY notification-service ./notification-service
WORKDIR /app/notification-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o notification-service ./cmd/server

FROM alpine:3.19
RUN addgroup -S kasku && adduser -S kasku -G kasku
WORKDIR /app
COPY --from=builder /app/notification-service/notification-service .
USER kasku
EXPOSE 8089
CMD ["./notification-service"]
