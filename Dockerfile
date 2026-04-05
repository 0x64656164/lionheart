# Multi-stage build for Lionheart CLI
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY core/go.mod core/go.sum ./core/
COPY cmd/lionheart/go.mod cmd/lionheart/go.sum ./cmd/lionheart/

# Download dependencies
RUN cd core && go mod download
RUN cd cmd/lionheart && go mod download

# Copy source code
COPY . .

# Build the binary
RUN cd cmd/lionheart && \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/lionheart-vpn/lionheart/core.Version=$(git describe --tags --always 2>/dev/null || echo 'docker')" \
    -o /build/lionheart .

# Final stage
FROM scratch

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /build/lionheart /lionheart

# Expose default port
EXPOSE 8443

# Set entrypoint
ENTRYPOINT ["/lionheart"]
