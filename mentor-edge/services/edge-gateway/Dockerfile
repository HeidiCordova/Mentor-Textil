FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o edge-gateway ./cmd/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata ffmpeg
ENV TZ=America/Lima
WORKDIR /root/
COPY --from=builder /app/edge-gateway .
HEALTHCHECK --interval=15s --timeout=3s --retries=5 \
    CMD wget -q --spider http://localhost:8005/health || exit 1
EXPOSE 8005
CMD ["./edge-gateway"]
