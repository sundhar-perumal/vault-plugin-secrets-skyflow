# Multi-stage Dockerfile for Skyflow Vault Plugin

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && \
    go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o skyflow-plugin \
    ./main.go

# Run tests
RUN go test -v ./...

# Runtime stage
FROM hashicorp/vault:1.16.0

# Add metadata
LABEL org.opencontainers.image.title="Skyflow Vault Plugin" \
      org.opencontainers.image.description="HashiCorp Vault plugin for Skyflow token generation" \
    org.opencontainers.image.vendor="Community" \
    org.opencontainers.image.source="https://github.com/<org>/vault-plugin-secrets-skyflow" \
      org.opencontainers.image.version="${VERSION}"

# Copy plugin binary from builder
COPY --from=builder /build/skyflow-plugin /vault/plugins/skyflow-plugin

# Set plugin permissions
RUN chmod +x /vault/plugins/skyflow-plugin

# Create plugin directory
RUN mkdir -p /vault/plugins

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD vault status || exit 1

# Expose Vault port
EXPOSE 8200

# Set environment variables
ENV VAULT_DEV_ROOT_TOKEN_ID=root \
    VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200 \
    VAULT_ADDR=http://127.0.0.1:8200

# Start Vault in dev mode with plugin directory
CMD ["vault", "server", "-dev", "-dev-plugin-dir=/vault/plugins", "-dev-root-token-id=root"]