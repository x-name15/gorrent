FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the daemon statically
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/gorrentd ./cmd/daemon
# Build the cli statically
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/gorrent ./cmd/cli

# Certificates are needed for HTTPS (DoH, Scrapers)
FROM alpine:latest AS certs
RUN apk --no-cache add ca-certificates

# Final optimized scratch image
FROM scratch
WORKDIR /
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/bin/gorrentd /gorrentd
COPY --from=builder /app/bin/gorrent /gorrent

# Optional: Default config fallback inside the image
COPY config.yaml /config.yaml

EXPOSE 7800
CMD ["/gorrentd"]
