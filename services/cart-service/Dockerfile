# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev}" \
    -o /app/cart-service \
    ./cmd/cart-service

# Build minimal health check tool (simple HTTP client, statically linked for distroless)
RUN cat > /tmp/healthcheck.go << 'EOF' && \
package main
import (
    "net/http"
    "os"
    "io"
    "time"
)
func main() {
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get("http://localhost:8080/health")
    if err != nil {
        os.Exit(1)
    }
    defer resp.Body.Close()
    io.Copy(io.Discard, resp.Body)
    if resp.StatusCode != 200 {
        os.Exit(1)
    }
}
EOF
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -tags netgo -installsuffix netgo -o /app/healthcheck /tmp/healthcheck.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/cart-service /cart-service

# Copy health check tool (ensure executable)
COPY --from=builder --chmod=755 /app/healthcheck /healthcheck

# Use non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/cart-service", "-health-check"] || exit 1

# Run the application
ENTRYPOINT ["/cart-service"]
