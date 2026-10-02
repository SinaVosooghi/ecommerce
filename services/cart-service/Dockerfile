# Requires BuildKit (the default builder since Docker 23) for cache mounts.

# Build stage
FROM golang:1.24-alpine AS builder

# Set automatically by BuildKit for the target platform
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

# Install build dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Download dependencies first so they are cached independently of source changes
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
COPY . .

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /app/cart-service \
    ./cmd/cart-service

# Runtime stage
FROM gcr.io/distroless/static:nonroot

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/cart-service /cart-service

# Use non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 8080

# Health check: distroless has no shell or wget, so the binary probes itself
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/cart-service", "-health-check"]

# Run the application
ENTRYPOINT ["/cart-service"]
