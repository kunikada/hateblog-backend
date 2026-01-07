# Build stage
FROM golang:1.25-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates tzdata sudo && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user for development (UID 1000)
RUN groupadd -g 1000 vscode && \
    useradd -m -d /home/vscode -u 1000 -g vscode vscode && \
    echo "vscode ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/vscode && \
    chmod 0440 /etc/sudoers.d/vscode

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

# Build batch tools
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o fetcher \
    ./cmd/fetcher

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o updater \
    ./cmd/updater

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/app /app
COPY --from=builder /build/fetcher /fetcher
COPY --from=builder /build/updater /updater

# Copy migrations
COPY --from=builder /build/migrations /migrations

# Run as non-root user (nonroot user in distroless has UID 65532)
USER nonroot:nonroot

# Expose port (adjust as needed)
EXPOSE 8080

# Health check endpoint (adjust as needed)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app", "healthcheck"]

ENTRYPOINT ["/app"]
