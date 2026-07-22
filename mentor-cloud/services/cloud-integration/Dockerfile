FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY shared/ /shared/
COPY services/cloud-integration/go.mod services/cloud-integration/go.sum ./
RUN go mod download
COPY services/cloud-integration/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /cloud-integration ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /cloud-integration /cloud-integration
EXPOSE 8085
ENTRYPOINT ["/cloud-integration"]
