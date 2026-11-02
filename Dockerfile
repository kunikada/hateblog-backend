# Build stage
FROM golang:1.27-trixie AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -tags timetzdata to embed timezone database
# -ldflags for stripping debug info and reducing size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags timetzdata \
    -ldflags='-w -s -extldflags "-static"' \
    -o app \
    ./cmd/app

# Build batch tools
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags timetzdata \
    -ldflags='-w -s -extldflags "-static"' \
    -o fetcher \
    ./cmd/fetcher

RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags timetzdata \
    -ldflags='-w -s -extldflags "-static"' \
    -o updater \
    ./cmd/updater

RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags timetzdata \
    -ldflags='-w -s -extldflags "-static"' \
    -o admin \
    ./cmd/admin

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static-debian13:nonroot AS runtime

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Set working directory for consistent relative paths
WORKDIR /workspace

# Copy the binary
COPY --from=builder /build/app /workspace/app
COPY --from=builder /build/fetcher /workspace/fetcher
COPY --from=builder /build/updater /workspace/updater
COPY --from=builder /build/admin /workspace/admin

# Copy migrations with runtime user ownership to avoid permission issues.
COPY --from=builder --chown=nonroot:nonroot /build/migrations /workspace/migrations

# Run as non-root user (nonroot user in distroless has UID 65532)
USER nonroot:nonroot

# Expose port (adjust as needed)
EXPOSE 8080

ENTRYPOINT ["/workspace/app"]
