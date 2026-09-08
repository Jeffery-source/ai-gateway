# =========================
# Build stage
# =========================
FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /ai_gateway ./cmd/gateway


# =========================
# Runtime stage
# =========================
FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /ai_gateway /app/ai_gateway
COPY config.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/ai_gateway"]