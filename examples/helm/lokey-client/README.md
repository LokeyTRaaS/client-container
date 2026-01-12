# Lokey Client Helm Chart

Helm chart for deploying Lokey Client Container and Operator.

## Installation

```bash
helm install lokey-client ./examples/helm/lokey-client
```

## Configuration

See `values.yaml` for all configurable options.

## Usage

After installation, create a `LokeyClient` resource:

```yaml
apiVersion: lokey.io/v1
kind: LokeyClient
metadata:
  name: lokey-client-default
spec:
  virtioUrl: "http://lokey-virtio:8083"
  devicePaths:
    - "/dev/lokeyrng"
```
