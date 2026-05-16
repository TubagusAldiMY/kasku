FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o admin-service ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache wget ca-certificates && \
    addgroup -S kasku && adduser -S kasku -G kasku
WORKDIR /app
COPY --from=builder /app/admin-service .
COPY --from=builder /app/migrations ./migrations
USER kasku
EXPOSE 8090
CMD ["./admin-service"]
