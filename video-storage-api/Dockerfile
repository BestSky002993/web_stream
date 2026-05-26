# ---------- Build stage ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---------- Runtime stage ----------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata wget ffmpeg && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/api /app/api

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api"]
