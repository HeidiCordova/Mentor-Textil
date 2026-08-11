# Jetson Orin - resiliencia
# Multi-stage build for ARM64

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go test ./...

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o resiliencia ./cmd/main.go

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

ENV TZ=America/Lima

WORKDIR /root/

COPY --from=builder /app/resiliencia .

HEALTHCHECK --interval=15s --timeout=3s --retries=5 \
    CMD wget -q --spider http://localhost:8002/health || exit 1

EXPOSE 8002

CMD ["./resiliencia"]
