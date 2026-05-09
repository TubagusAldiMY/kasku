FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o user-service ./cmd/server

FROM alpine:3.19
RUN addgroup -S kasku && adduser -S kasku -G kasku
WORKDIR /app
COPY --from=builder /app/user-service .
USER kasku
EXPOSE 8082 9082
CMD ["./user-service"]
