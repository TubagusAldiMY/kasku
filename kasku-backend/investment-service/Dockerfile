FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/investment-service ./cmd/server/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /bin/investment-service /app/investment-service
EXPOSE 8086 9086
CMD ["/app/investment-service"]
