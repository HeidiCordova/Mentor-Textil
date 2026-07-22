FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY shared/ /shared/
COPY services/cloud-config/go.mod services/cloud-config/go.sum ./
RUN go mod download
COPY services/cloud-config/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /cloud-config ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /cloud-config /cloud-config
EXPOSE 8083
ENTRYPOINT ["/cloud-config"]
