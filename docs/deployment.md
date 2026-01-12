# Deployment Guide

This guide covers deploying Lokey Client Container in various environments.

## Prerequisites

- Lokey VirtIO service running and accessible
- Docker or Kubernetes cluster
- Character device creation capability (for init container)
- Seccomp profiles installed (optional, for enhanced security)

## Building Containers

### Local Build

```bash
# Build all containers
task docker-build-all

# Or build individually
task docker-build-client
task docker-build-init
task docker-build-operator
```

### Multi-Architecture Build

```bash
# Build for AMD64, ARM64, ARMv7, ARMv8
task docker-build-multi
```

## Docker Compose

### Basic Setup

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

### With Init Container

```yaml
services:
  device-init:
    image: ghcr.io/lokey/client-container-init:latest
    command: ["/app/init", "-devices", "/dev/lokeyrng"]
    volumes:
      - dev-shared:/dev
    privileged: true

  lokey-client:
    image: ghcr.io/lokey/client-container:latest
    depends_on:
      - device-init
    volumes:
      - dev-shared:/dev
```

## Kubernetes Sidecar

### 1. Create Namespace

```bash
kubectl create namespace lokey-example
```

### 2. Deploy with Sidecar

```bash
kubectl apply -f examples/kubernetes/sidecar.yaml
```

### 3. Verify

```bash
kubectl logs -n lokey-example deployment/example-app -c lokey-client
```

## Kubernetes Daemonset

### 1. Deploy Daemonset

```bash
kubectl apply -f examples/kubernetes/daemonset.yaml
```

### 2. Verify

```bash
kubectl get daemonset -n lokey-system lokey-client
kubectl logs -n lokey-system daemonset/lokey-client
```

## Configuration

### Environment Variables

Set via ConfigMap or directly in deployment:

```yaml
env:
- name: LOKEY_VIRTIO_URL
  value: "http://lokey-virtio:8083"
- name: DEVICE_PATH
  value: "/dev/lokeyrng,/dev/hwrng"
```

### Resource Limits

Recommended resource limits:

```yaml
resources:
  requests:
    memory: "32Mi"
    cpu: "50m"
  limits:
    memory: "128Mi"
    cpu: "200m"
```

## Troubleshooting

### Connection Issues

```bash
# Check VirtIO service is accessible
kubectl exec -it <pod> -- wget -O- http://lokey-virtio:8083/health

# Check client logs
kubectl logs <pod> -c lokey-client
```

### Device Issues

```bash
# Check device exists
kubectl exec -it <pod> -- ls -l /dev/lokeyrng

# Check device permissions
kubectl exec -it <pod> -- stat /dev/lokeyrng
```

### Network Issues

```bash
# Test connectivity
kubectl exec -it <pod> -- nc -zv lokey-virtio 8083
```

## Seccomp Profiles

For enhanced security, use seccomp profiles:

1. Install profiles on nodes (see [Seccomp Guide](seccomp.md))
2. Profiles are automatically referenced in Kubernetes examples

The seccomp profiles restrict system calls to only what's necessary for operation.

## Security

### Pod Security

For sidecars, use non-root user:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
```

For daemonsets, root may be required:

```yaml
securityContext:
  privileged: true
  runAsUser: 0
```

### Network Policies

Restrict network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: lokey-client
spec:
  podSelector:
    matchLabels:
      app: lokey-client
  policyTypes:
  - Egress
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: lokey-virtio
    ports:
    - protocol: TCP
      port: 8083
```

## Monitoring

### Logs

```bash
# Follow logs
kubectl logs -f <pod> -c lokey-client

# Check for errors
kubectl logs <pod> -c lokey-client | grep ERROR
```

### Metrics

Monitor via VirtIO service health endpoint:

```bash
curl http://lokey-virtio:8083/health
```

## Upgrades

### Rolling Update

```bash
kubectl set image deployment/lokey-client \
  lokey-client=ghcr.io/lokey/client-container:v1.1.0
```

### Daemonset Update

```bash
kubectl set image daemonset/lokey-client \
  lokey-client=ghcr.io/lokey/client-container:v1.1.0
```
