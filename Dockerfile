# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags for stripping debug info and reducing size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o app \
    ./cmd/app

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/app /app

# Run as non-root user (nonroot user in distroless has UID 65532)
USER nonroot:nonroot

# Expose port (adjust as needed)
EXPOSE 8080

# Health check endpoint (adjust as needed)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app", "healthcheck"]

ENTRYPOINT ["/app"]
