# Examples

This document provides practical examples for using Lokey Client Container.

## Building Examples

Before using the examples, you can build the containers locally:

```bash
# Build all containers
task docker-build-all

# Or use pre-built images from GitHub Container Registry
# Images are automatically built on releases
```

## Docker Compose

See [examples/docker-compose.yaml](../examples/docker-compose.yaml) for a complete Docker Compose setup.

## Kubernetes Sidecar

### Basic Sidecar

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-app
spec:
  initContainers:
  - name: device-init
    image: ghcr.io/lokey/client-container-init:latest
    command: ["/app/init", "-devices", "/dev/lokeyrng"]
    securityContext:
      capabilities:
        add: ["MKNOD"]
      runAsUser: 0
      seccompProfile:
        type: Localhost
        localhostProfile: lokey-client-init-container.json
    volumeMounts:
    - name: dev
      mountPath: /dev
  containers:
  - name: lokey-client
    image: ghcr.io/lokey/client-container:latest
    env:
    - name: LOKEY_VIRTIO_URL
      value: "http://lokey-virtio:8083"
    volumeMounts:
    - name: dev
      mountPath: /dev
    securityContext:
      seccompProfile:
        type: Localhost
        localhostProfile: lokey-client-container.json
  - name: app
    image: alpine:latest
    command: ["sh", "-c", "while true; do head -c 32 /dev/lokeyrng | hexdump -C; sleep 5; done"]
    volumeMounts:
    - name: dev
      mountPath: /dev
  volumes:
  - name: dev
    emptyDir: {}
```

**Note**: Before using seccomp profiles, ensure they are installed on your Kubernetes nodes. See [Seccomp Profiles](seccomp.md) for installation instructions.

## Kubernetes Daemonset

See [examples/kubernetes/daemonset.yaml](../examples/kubernetes/daemonset.yaml) for a complete daemonset configuration with seccomp profiles.

**Note**: The daemonset example includes seccomp profiles for enhanced security. Ensure seccomp profiles are installed on your Kubernetes nodes before deployment.

## Operator Usage

### Sidecar Only

```yaml
apiVersion: lokey.io/v1
kind: LokeyClient
metadata:
  name: lokey-client-sidecar
spec:
  virtioUrl: "http://lokey-virtio:8083"
  enableSidecar: true
  enableDaemonset: false
  # enableSeccomp: false  # Optional: enable seccomp profiles (default: false)
  devicePaths:
    - "/dev/lokeyrng"
  podSelector:
    matchLabels:
      lokey.io/inject: "true"
```

### Daemonset Only

```yaml
apiVersion: lokey.io/v1
kind: LokeyClient
metadata:
  name: lokey-client-daemonset
spec:
  virtioUrl: "http://lokey-virtio:8083"
  enableSidecar: false
  enableDaemonset: true
  # enableSeccomp: false  # Optional: enable seccomp profiles (default: false)
  devicePaths:
    - "/dev/lokeyrng"
    - "/dev/hwrng"
```

### Both Sidecar and Daemonset

```yaml
apiVersion: lokey.io/v1
kind: LokeyClient
metadata:
  name: lokey-client-both
spec:
  virtioUrl: "http://lokey-virtio:8083"
  enableSidecar: true
  enableDaemonset: true
  # enableSeccomp: false  # Optional: enable seccomp profiles (default: false)
  devicePaths:
    - "/dev/lokeyrng"
  podSelector:
    matchLabels:
      lokey.io/inject: "true"
```

### With Seccomp Enabled

```yaml
apiVersion: lokey.io/v1
kind: LokeyClient
metadata:
  name: lokey-client-secure
spec:
  virtioUrl: "http://lokey-virtio:8083"
  enableSidecar: true
  enableSeccomp: true  # Enable seccomp profiles (requires profiles on nodes)
  devicePaths:
    - "/dev/lokeyrng"
  podSelector:
    matchLabels:
      lokey.io/inject: "true"
```

**Note**: Seccomp profiles are **optional** and disabled by default. Only enable if profiles are installed on all Kubernetes nodes. See [Seccomp Profiles](seccomp.md) for installation instructions.

### Annotate Pod for Injection

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    lokey.io/inject: "true"
spec:
  containers:
  - name: app
    image: my-app:latest
```

## Use Cases

### Cryptographic Key Generation

```bash
# Read random bytes for key generation
head -c 32 /dev/lokeyrng > key.bin
```

### Secure Token Generation

```python
import os

# Read random bytes
with open('/dev/lokeyrng', 'rb') as f:
    token = f.read(32)
    print(token.hex())
```

### Testing Randomness

```bash
# Test random data quality
head -c 1024 /dev/lokeyrng | ent
```
