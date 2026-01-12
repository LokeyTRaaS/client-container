# Lokey Client Container

A Go-based container service that connects to Lokey VirtIO service and feeds true random data to character devices (`/dev/lokeyrng`, `/dev/hwrng`) for use in Kubernetes sidecars and daemonsets.

[![Build Status](https://github.com/lokey/client-container/actions/workflows/build.yml/badge.svg)](https://github.com/lokey/client-container/actions/workflows/build.yml)
[![Lint](https://github.com/lokey/client-container/actions/workflows/lint.yml/badge.svg)](https://github.com/lokey/client-container/actions/workflows/lint.yml)
[![Test](https://github.com/lokey/client-container/actions/workflows/test.yml/badge.svg)](https://github.com/lokey/client-container/actions/workflows/test.yml)

## Overview

Lokey Client Container bridges the gap between Lokey's VirtIO service and applications that need hardware random number generator (RNG) devices. It provides:

- **HTTP Streaming Client**: Connects to Lokey VirtIO service and streams random data
- **Character Device Writer**: Writes random data to one or more character devices
- **Automatic Reconnection**: Handles network failures with exponential backoff
- **Multi-Device Support**: Can write to multiple devices simultaneously
- **Kubernetes Ready**: Designed for sidecar and daemonset patterns
- **Kubernetes Operator**: Declarative management of randomness injection

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

### Kubernetes Operator

```yaml
apiVersion: lokey.io/v1
kind: LokeyClient
metadata:
  name: lokey-client-default
spec:
  virtioUrl: "http://lokey-virtio:8083"
  devicePaths:
    - "/dev/lokeyrng"
  injectionMode: "annotation"
```

## Architecture

```mermaid
graph LR
    A[Lokey VirtIO Service] -->|HTTP Stream| B[Client Container]
    B -->|Writes| C[/dev/lokeyrng]
    B -->|Writes| D[/dev/hwrng]
    C -->|Reads| E[Application]
    D -->|Reads| E
```

## Features

- ✅ HTTP streaming from Lokey VirtIO service
- ✅ Multi-device support (`/dev/lokeyrng`, `/dev/hwrng`)
- ✅ Automatic reconnection with exponential backoff
- ✅ Kubernetes sidecar pattern
- ✅ Kubernetes daemonset pattern
- ✅ Kubernetes operator with CRDs
- ✅ Selective control (sidecars, daemonsets, or both)
- ✅ Automatic sidecar injection
- ✅ Multi-architecture support (AMD64, ARM64, ARMv7, ARMv8)
- ✅ Hardened container images (distroless)
- ✅ Optional seccomp profiles for enhanced security
- ✅ Comprehensive testing

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

## Documentation

- [Architecture](docs/architecture.md) - Detailed architecture and design
- [Deployment](docs/deployment.md) - Deployment guides
- [Development](docs/development.md) - Development guide and build instructions
- [Operator](docs/operator.md) - Kubernetes operator documentation
- [Seccomp Profiles](docs/seccomp.md) - Security profiles for containers
- [Examples](examples/) - Usage examples and patterns

## Development

```bash
# Build binaries
task build

# Test
task test

# Lint
task lint

# Build Docker containers locally
task docker-build-all

# Build for multiple architectures
task docker-build-multi

# Run all checks
task all
```

### Available Tasks

- **Build**: `task build` - Build all Go binaries
- **Test**: `task test` - Run unit tests
- **Lint**: `task lint` - Run linters
- **Docker Build**: `task docker-build-all` - Build all container images
- **Multi-arch Build**: `task docker-build-multi` - Build for AMD64/ARM64/ARMv7/ARMv8
- **Operator**: `task operator-build` - Build operator binary
- **All**: `task all` - Run all development checks

See `task --list-all` for complete list of tasks.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Related Projects

- [Lokey](https://github.com/lokey/Lokey) - True random number generation service
