FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY observability-go ./observability-go
COPY admin-service/go.mod admin-service/go.sum ./admin-service/
WORKDIR /app/admin-service
RUN go mod download
WORKDIR /app
COPY admin-service ./admin-service
WORKDIR /app/admin-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o admin-service ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache wget ca-certificates && \
    addgroup -S kasku && adduser -S kasku -G kasku
WORKDIR /app
COPY --from=builder /app/admin-service/admin-service .
COPY --from=builder /app/admin-service/migrations ./migrations
USER kasku
EXPOSE 8090
CMD ["./admin-service"]
