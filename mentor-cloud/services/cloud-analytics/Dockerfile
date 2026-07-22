FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY shared/ /shared/
COPY services/cloud-analytics/go.mod services/cloud-analytics/go.sum ./
RUN go mod download
COPY services/cloud-analytics/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cloud-analytics ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /cloud-analytics /cloud-analytics
EXPOSE 8084
ENTRYPOINT ["/cloud-analytics"]
