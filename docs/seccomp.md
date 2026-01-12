# Seccomp Profiles

This document describes the seccomp (Secure Computing Mode) profiles used by Lokey Client Container components.

## Overview

Seccomp profiles restrict the system calls that containers can make, reducing the attack surface and improving security. We provide seccomp profiles for:

- **Init Container**: Minimal syscalls needed for device creation
- **Client Container**: Syscalls needed for file I/O and network operations

## Profiles

### Init Container Profile (`config/seccomp/init-container.json`)

The init container profile allows only the syscalls necessary for:
- Creating character devices (`mknod`, `mknodat`)
- File operations (`open`, `close`, `stat`, `unlink`, `chmod`)
- Directory operations (`mkdir`, `chdir`)
- Basic process operations

**Key syscalls:**
- `mknod` / `mknodat` - Create character devices
- `mkdir` / `mkdirat` - Create directories
- `chmod` / `fchmod` - Set permissions
- Basic file I/O operations

### Client Container Profile (`config/seccomp/client-container.json`)

The client container profile allows syscalls needed for:
- Network operations (HTTP streaming)
- File I/O (writing to character devices)
- Process management
- Memory management

**Key syscalls:**
- Network: `socket`, `connect`, `sendto`, `recvfrom`, `epoll_*`
- File I/O: `open`, `read`, `write`, `close`
- Process: `clone`, `execve`, `wait4`
- Memory: `mmap`, `munmap`, `brk`

## Installation

### Option 1: Localhost Path (Node-level)

Copy profiles to each Kubernetes node:

```bash
# On each node
sudo mkdir -p /var/lib/kubelet/seccomp/profiles
sudo cp config/seccomp/init-container.json /var/lib/kubelet/seccomp/profiles/lokey-client-init-container.json
sudo cp config/seccomp/client-container.json /var/lib/kubelet/seccomp/profiles/lokey-client-container.json
```

Then reference in pod spec:

```yaml
securityContext:
  seccompProfile:
    type: Localhost
    localhostProfile: lokey-client-container.json
```

### Option 2: ConfigMap (Recommended for Managed Kubernetes)

For managed Kubernetes services (EKS, GKE, AKS), use ConfigMaps:

```bash
kubectl apply -f examples/kubernetes/seccomp-install.yaml
```

Then mount the ConfigMap and reference it (requires Kubernetes 1.19+ with seccomp alpha features).

## Usage in Kubernetes

### With Operator

When using the Kubernetes operator, enable seccomp by setting `enableSeccomp: true` in your `LokeyClient` resource:

```yaml
apiVersion: lokey.io/v1
kind: LokeyClient
metadata:
  name: lokey-client
spec:
  virtioUrl: "http://lokey-virtio:8083"
  enableSidecar: true
  enableSeccomp: true  # Enable seccomp profiles
  devicePaths:
    - "/dev/lokeyrng"
```

The operator will automatically inject seccomp profiles into init and sidecar containers when enabled.

### Manual Configuration

For manual pod configurations, reference seccomp profiles in the security context:

```yaml
# Init container
securityContext:
  seccompProfile:
    type: Localhost
    localhostProfile: lokey-client-init-container.json

# Client container
securityContext:
  seccompProfile:
    type: Localhost
    localhostProfile: lokey-client-container.json
```

## Verification

To verify seccomp is working:

```bash
# Check if seccomp is enforced
kubectl exec <pod> -c lokey-client -- dmesg | grep seccomp

# Test that blocked syscalls are denied
kubectl exec <pod> -c lokey-client -- strace -e trace=all <command>
```

## Customization

You can customize the profiles by:
1. Editing the JSON files in `config/seccomp/`
2. Adding or removing syscalls as needed
3. Testing in a development environment first

## Security Benefits

- **Reduced Attack Surface**: Only necessary syscalls are allowed
- **Defense in Depth**: Even if a container is compromised, limited syscalls reduce impact
- **Compliance**: Meets security best practices for containerized workloads

## Notes

- Seccomp profiles are enforced at the kernel level
- Profiles must be available on the node before pods can use them
- Some managed Kubernetes services may have restrictions on seccomp usage
- Always test profiles in a non-production environment first
