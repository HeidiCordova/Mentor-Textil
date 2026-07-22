FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY shared/ /shared/
COPY services/energy-ingest/go.mod services/energy-ingest/go.sum ./
RUN go mod download
COPY services/energy-ingest/cmd/ ./cmd/
COPY services/energy-ingest/internal/ ./internal/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /energy-ingest ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /energy-ingest /energy-ingest
EXPOSE 8086
ENTRYPOINT ["/energy-ingest"]
