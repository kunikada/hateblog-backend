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

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o admin \
    ./cmd/admin

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set working directory for consistent relative paths
WORKDIR /workspace

# Copy the binary
COPY --from=builder /build/app /workspace/app
COPY --from=builder /build/fetcher /workspace/fetcher
COPY --from=builder /build/updater /workspace/updater
COPY --from=builder /build/admin /workspace/admin

# Copy migrations
COPY --from=builder /build/migrations /workspace/migrations

# Run as non-root user (nonroot user in distroless has UID 65532)
USER nonroot:nonroot

# Expose port (adjust as needed)
EXPOSE 8080

ENTRYPOINT ["/workspace/app"]
