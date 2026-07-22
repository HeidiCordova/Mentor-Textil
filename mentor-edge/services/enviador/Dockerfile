# Jetson Orin - enviador
# Multi-stage build for ARM64

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o enviador ./cmd/main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=America/Lima

WORKDIR /root/

COPY --from=builder /app/enviador .

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -q --spider http://localhost:8003/health || exit 1

EXPOSE 8003

CMD ["./enviador"]
