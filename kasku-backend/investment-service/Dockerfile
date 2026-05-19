FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY observability-go ./observability-go
COPY investment-service/go.mod investment-service/go.sum* ./investment-service/
WORKDIR /app/investment-service
RUN go mod download 2>/dev/null || true
WORKDIR /app
COPY investment-service ./investment-service
WORKDIR /app/investment-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/investment-service ./cmd/server/main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /bin/investment-service /app/investment-service
EXPOSE 8086 9086
CMD ["/app/investment-service"]
