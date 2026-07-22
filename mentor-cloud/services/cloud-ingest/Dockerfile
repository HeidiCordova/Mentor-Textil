FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY shared/ /shared/
COPY services/cloud-ingest/go.mod services/cloud-ingest/go.sum ./
RUN go mod download
COPY services/cloud-ingest/cmd/ ./cmd/
COPY services/cloud-ingest/internal/ ./internal/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /cloud-ingest ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /cloud-ingest /cloud-ingest
EXPOSE 8082
ENTRYPOINT ["/cloud-ingest"]
