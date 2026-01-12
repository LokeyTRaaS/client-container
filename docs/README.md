# Lokey Client Container

A Go-based container service that connects to Lokey VirtIO service and feeds true random data to character devices (`/dev/lokeyrng`, `/dev/hwrng`) for use in Kubernetes sidecars and daemonsets.

## Overview

Lokey Client Container bridges the gap between Lokey's VirtIO service and applications that need hardware random number generator (RNG) devices. It provides:

- **HTTP Streaming Client**: Connects to Lokey VirtIO service and streams random data
- **Character Device Writer**: Writes random data to one or more character devices
- **Automatic Reconnection**: Handles network failures with exponential backoff
- **Multi-Device Support**: Can write to multiple devices simultaneously
- **Kubernetes Ready**: Designed for sidecar and daemonset patterns

## Architecture

```mermaid
graph LR
    A[Lokey VirtIO Service] -->|HTTP Stream| B[Client Container]
    B -->|Writes| C[/dev/lokeyrng]
    B -->|Writes| D[/dev/hwrng]
    C -->|Reads| E[Application]
    D -->|Reads| E
```

## Quick Start

### Docker Compose

```yaml
services:
  lokey-client:
    image: ghcr.io/lokey/client-container:latest
    environment:
      - LOKEY_VIRTIO_URL=http://virtio:8083
      - DEVICE_PATH=/dev/lokeyrng
    volumes:
      - /dev:/dev
```

### Kubernetes Sidecar

```yaml
containers:
- name: lokey-client
  image: ghcr.io/lokey/client-container:latest
  env:
  - name: LOKEY_VIRTIO_URL
    value: "http://lokey-virtio:8083"
  volumeMounts:
  - name: dev
    mountPath: /dev
```

See [examples](examples/) for complete configurations.

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOKEY_VIRTIO_URL` | `http://localhost:8083` | URL of Lokey VirtIO service |
| `LOKEY_STREAM_ENDPOINT` | `/stream` | HTTP endpoint for streaming |
| `DEVICE_PATH` | `/dev/lokeyrng` | Comma-separated device paths |
| `CHUNK_SIZE` | `1024` | Stream chunk size in bytes |
| `RECONNECT_INTERVAL` | `5s` | Reconnection interval |
| `LOG_LEVEL` | `INFO` | Log level (DEBUG, INFO, WARN, ERROR) |

## Usage Patterns

### 1. Sidecar Pattern

Inject the client as a sidecar container alongside your application:

```yaml
containers:
- name: lokey-client  # Sidecar
  image: ghcr.io/lokey/client-container:latest
- name: app  # Your application
  image: your-app:latest
```

### 2. Daemonset Pattern

Deploy as a daemonset for node-level randomness:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: lokey-client
spec:
  template:
    spec:
      containers:
      - name: lokey-client
        image: ghcr.io/lokey/client-container:latest
```

## Documentation

- [Architecture](architecture.md) - Detailed architecture and design
- [Deployment](deployment.md) - Deployment guides
- [Development](development.md) - Development guide and build instructions
- [Operator](operator.md) - Kubernetes operator documentation
- [Seccomp Profiles](seccomp.md) - Security profiles for containers
- [Examples](examples.md) - Usage examples and patterns

## Development

```bash
# Build binaries
task build

# Test
task test

# Lint
task lint

# Build Docker containers
task docker-build-all

# Run all checks
task all
```

See [Development Guide](development.md) for detailed development instructions.

## License

MIT License - See [LICENSE](../LICENSE) for details.
