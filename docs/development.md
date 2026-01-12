# Development Guide

This guide covers development setup, building, testing, and contributing to Lokey Client Container.

## Prerequisites

- Go 1.25 or later
- Docker (for container builds)
- Docker Buildx (for multi-architecture builds)
- Task (taskfile.dev) - Task runner
- kubectl (for operator development)
- Kubernetes cluster (for operator testing)

## Project Structure

```
client-container/
├── cmd/
│   ├── client/          # Main client application
│   ├── init/            # Init container (device creation)
│   └── operator/        # Kubernetes operator
├── pkg/
│   ├── config/          # Configuration management
│   ├── device/          # Character device writer
│   ├── virtio/          # VirtIO HTTP client
│   └── operator/        # Operator logic
├── api/v1/              # Kubernetes CRDs
├── tests/               # Test suites
├── examples/            # Usage examples
└── docs/                # Documentation
```

## Getting Started

### 1. Clone and Initialize

```bash
git clone <repository-url>
cd client-container
task init
task download
```

### 2. Build Binaries

```bash
# Build all binaries
task build

# Build specific component
cd cmd/client && go build
cd cmd/init && go build
cd cmd/operator && go build
```

### 3. Run Tests

```bash
# Run all tests
task test

# Run with coverage
task test-coverage

# Run specific test package
go test ./tests/config_test/... -v
```

### 4. Build Containers

```bash
# Build all containers locally
task docker-build-all

# Build individual containers
task docker-build-client
task docker-build-init
task docker-build-operator

# Build for multiple architectures
task docker-build-multi
```

## Development Workflow

### Code Quality

```bash
# Format code
task fmt

# Run linters
task lint

# Run go vet
task vet

# Run all checks
task prepare-all
```

### Complete Development Cycle

```bash
# Run all development tasks
task all
```

This will:
1. Initialize project
2. Clean build artifacts
3. Download dependencies
4. Format code
5. Run vet
6. Run linters
7. Run tests
8. Build binaries

## Building Containers

### Local Builds

Build containers for your local architecture:

```bash
# Build all containers
task docker-build-all

# Individual builds
task docker-build-client    # Client container
task docker-build-init      # Init container
task docker-build-operator  # Operator container
```

### Multi-Architecture Builds

Build for multiple architectures (AMD64, ARM64, ARMv7, ARMv8):

```bash
task docker-build-multi
```

**Note**: Requires Docker Buildx. Set up with:
```bash
docker buildx create --name multiarch --use
docker buildx inspect --bootstrap
```

### Testing Containers Locally

```bash
# Test client container
docker run --rm \
  -e LOKEY_VIRTIO_URL=http://host.docker.internal:8083 \
  -v /dev:/dev \
  lokey-client-container:latest

# Test init container
docker run --rm --privileged \
  -v /tmp/dev:/dev \
  lokey-client-container-init:latest \
  -devices /tmp/dev/lokeyrng
```

## Operator Development

### Build Operator

```bash
task operator-build
```

### Generate CRD Manifests

If using kubebuilder:

```bash
task operator-manifests
```

### Install CRDs

```bash
task operator-install
```

### Run Operator Locally

```bash
# Requires kubeconfig configured
task operator-run
```

### Test Operator

```bash
task operator-test
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./tests/... -v

# Run with race detector
go test -race ./tests/...

# Run with coverage
go test -cover ./tests/...
```

### Integration Testing

1. Start Lokey VirtIO service (see Lokey project)
2. Run client container:
   ```bash
   docker run --rm \
     -e LOKEY_VIRTIO_URL=http://<virtio-url>:8083 \
     -v /tmp/dev:/dev \
     lokey-client-container:latest
   ```
3. Test device creation:
   ```bash
   docker run --rm --privileged \
     -v /tmp/dev:/dev \
     lokey-client-container-init:latest \
     -devices /tmp/dev/lokeyrng
   ```

## Code Style

- Follow Go standard formatting (`gofmt`)
- Use `golangci-lint` for linting
- Write tests for all new features
- Document public APIs
- Keep functions focused and small

## Adding New Features

1. Create feature branch
2. Implement feature with tests
3. Update documentation
4. Run all checks: `task all`
5. Submit pull request

## Debugging

### Client Container

```bash
# Run with debug logging
docker run --rm \
  -e LOG_LEVEL=DEBUG \
  -e LOKEY_VIRTIO_URL=http://virtio:8083 \
  lokey-client-container:latest
```

### Operator

```bash
# Run operator with verbose logging
./build/operator --zap-devel
```

## CI/CD

The project uses GitHub Actions for:
- Building multi-arch containers
- Running tests
- Linting code
- Creating releases

See `.github/workflows/` for workflow definitions.

## Troubleshooting

### Build Issues

```bash
# Clean and rebuild
task clean
task build

# Verify Go version
go version  # Should be 1.25+

# Check dependencies
go mod verify
```

### Docker Build Issues

```bash
# Check Docker is running
docker info

# For multi-arch builds, verify buildx
docker buildx ls
```

### Test Failures

```bash
# Run tests with verbose output
go test ./tests/... -v

# Check test coverage
task test-coverage
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Update documentation
6. Run `task all` to verify
7. Submit a pull request

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Kubernetes Operator SDK](https://sdk.operatorframework.io/)
- [Docker Buildx](https://docs.docker.com/buildx/)
- [Task Documentation](https://taskfile.dev/)
