FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the client binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /app/client ./cmd/client

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/client /app/client

# Use nonroot user (distroless provides this)
USER nonroot:nonroot

# Set environment variables with defaults
ENV LOKEY_VIRTIO_URL=http://localhost:8083 \
    LOKEY_STREAM_ENDPOINT=/stream \
    DEVICE_PATH=/dev/lokeyrng \
    CHUNK_SIZE=1024 \
    RECONNECT_INTERVAL=5s \
    LOG_LEVEL=INFO

# Run the client
ENTRYPOINT ["/app/client"]
