FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY shared/ /shared/
COPY services/cloud-gateway/go.mod services/cloud-gateway/go.sum ./
RUN go mod download
COPY services/cloud-gateway/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cloud-gateway ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /cloud-gateway /cloud-gateway
EXPOSE 8888
ENTRYPOINT ["/cloud-gateway"]
