# Raspberry Pi (ARM32 / ARM64) — energy-sender
# Multi-arch: soporta linux/arm/v7 (RPi 3/4 32-bit) y linux/arm64 (RPi 4/5 64-bit)

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod ./
RUN go mod download || true

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o energy-sender .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/energy-sender .

EXPOSE 8086
CMD ["./energy-sender"]
