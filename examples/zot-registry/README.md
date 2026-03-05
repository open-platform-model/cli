# Zot OCI Registry Module

Production-ready [Zot](https://zotregistry.dev) OCI-native container registry module for Kubernetes.

## Overview

Zot is a production-ready, vendor-neutral OCI-native container registry. This OPM module provides a fully-featured, production-ready deployment with:

- **Authentication & Access Control** - htpasswd-based authentication with fine-grained repository policies
- **Metrics** - Prometheus metrics endpoint for monitoring
- **Storage Management** - Garbage collection, deduplication, and data integrity scrubbing
- **Registry Sync** - Mirror images from upstream registries (on-demand or periodic)
- **Production Defaults** - Persistent storage, health probes, security context, resource limits

## Features

| Feature | Description |
|---------|-------------|
| **Image Variants** | Full (with all extensions) or minimal (lightweight) |
| **Storage** | PVC or emptyDir, with GC, dedupe, and scrub support |
| **Auth** | htpasswd-based with admin/user policies |
| **Metrics** | Prometheus `/metrics` endpoint |
| **Sync** | Mirror upstream registries (Docker Hub, GHCR, etc.) |
| **Health Probes** | Startup, liveness, and readiness probes |
| **Security** | Non-root, drop capabilities, read-only root FS (where possible) |
| **Ingress** | Optional HTTPRoute for Gateway API |

## Quick Start

### 1. Install with Default Values (Production)

```bash
# Deploy with production defaults (PVC storage, auth, metrics)
opm mod apply ./zot-registry
```

This creates:
- 1 replica StatefulSet with 20Gi PVC storage
- htpasswd authentication (admin:admin, user:user)
- Garbage collection every 24h
- Data integrity scrubbing every 24h
- Prometheus metrics on `/metrics`

### 2. Minimal Development Setup

```bash
# Deploy with minimal config (no persistence, no auth)
opm mod apply ./zot-registry -f values_minimal.cue
```

This creates:
- 1 replica with emptyDir storage (ephemeral)
- No authentication
- Debug logging
- Minimal resource requests

## Configuration

### Image Variants

Zot provides two image variants:

| Variant | Image | Features |
|---------|-------|----------|
| `full` | `ghcr.io/project-zot/zot` | All extensions (search UI, sync, scrub, metrics) |
| `minimal` | `ghcr.io/project-zot/zot-minimal` | Core registry only (smaller image) |

Set via `image.variant: "full"` or `"minimal"`.

### Storage

Configure storage type and features:

```cue
storage: {
    type: "pvc"              // or "emptyDir"
    size: "50Gi"             // PVC size
    storageClass: "fast-ssd" // Storage class
    
    dedupe: true             // Enable hard-link deduplication
    
    gc: {
        enabled:  true
        delay:    "2h"       // Wait before removing blobs
        interval: "12h"      // GC frequency
    }
    
    scrub: {
        enabled:  true
        interval: "24h"      // Data integrity check frequency
    }
}
```

### Authentication & Access Control

**Basic Authentication:**

```cue
auth: {
    htpasswd: {
        credentials: {
            $secretName: "zot-htpasswd"
            $dataKey:    "htpasswd"
            value: """
                admin:$2y$05$...
                user:$2y$05$...
                """
        }
    }
}
```

Generate htpasswd entries:
```bash
htpasswd -nbB username password
```

**Access Control Policies:**

```cue
auth: {
    accessControl: {
        adminUsers: ["admin", "ci-bot"]
        
        repositories: {
            "library/**": {
                policies: [{
                    users:   ["developer"]
                    actions: ["read"]
                }]
                defaultPolicy: []
            }
            "private/**": {
                policies: [{
                    users:   ["admin"]
                    actions: ["read", "create", "update", "delete"]
                }]
                defaultPolicy: []
            }
        }
    }
}
```

### Registry Sync (Mirroring)

Mirror upstream registries:

```cue
sync: {
    registries: [{
        urls:         ["https://docker.io"]
        onDemand:     true        // Pull on first request
        tlsVerify:    true
        pollInterval: "6h"        // Periodic sync interval
        
        content: [{
            prefix: "library/**" // Mirror only official images
        }]
    }, {
        urls:      ["https://ghcr.io"]
        onDemand:  true
        tlsVerify: true
        
        content: [{
            prefix:      "myorg/**"
            destination: "mirror/myorg" // Remap path
        }]
    }]
}
```

### Ingress

Expose via Gateway API HTTPRoute:

```cue
httpRoute: {
    hostnames: ["registry.example.com"]
    
    tls: {
        secretName: "registry-tls"
    }
    
    gatewayRef: {
        name:      "cluster-gateway"
        namespace: "gateway-system"
    }
}
```

## Production Recommendations

### 1. Use External Secrets

Don't commit htpasswd credentials in `values.cue`. Instead, use External Secrets Operator or sealed secrets:

```cue
auth: {
    htpasswd: {
        credentials: {
            $secretName: "zot-htpasswd"
            $dataKey:    "htpasswd"
            // Reference external secret - value will be injected
            externalPath: "registry/htpasswd"
            remoteKey:    "htpasswd"
        }
    }
}
```

### 2. Enable All Production Features

```cue
storage: {
    type:         "pvc"
    size:         "100Gi"        // Size based on image volume
    storageClass: "fast-ssd"     // Use fast storage
    dedupe:       true           // Save space
    
    gc: {
        enabled:  true
        delay:    "1h"
        interval: "24h"
    }
    
    scrub: {
        enabled:  true           // Catch bit rot early
        interval: "24h"
    }
}

log: {
    level: "info"
    audit: {
        enabled: true            // Audit trail for compliance
    }
}

metrics: {
    enabled: true                // Monitor with Prometheus
}
```

### 3. Resource Sizing

Scale resources based on usage:

| Usage Level | Memory | CPU |
|-------------|--------|-----|
| Small (< 100 images) | 256Mi - 512Mi | 100m - 250m |
| Medium (< 1000 images) | 512Mi - 1Gi | 250m - 500m |
| Large (> 1000 images) | 1Gi - 4Gi | 500m - 2000m |

### 4. High Availability

For HA, use `replicas: 3` with `ReadWriteMany` storage:

```cue
replicas: 3

storage: {
    type:         "pvc"
    storageClass: "nfs"  // RWX-capable storage class
}
```

⚠️ **Note**: Zot doesn't have built-in clustering. Multiple replicas share the same storage but don't coordinate. For true HA, consider a distributed storage backend.

## Monitoring

Zot exposes Prometheus metrics at `/metrics`:

| Metric | Description |
|--------|-------------|
| `zot_http_requests_total` | Total HTTP requests by status code |
| `zot_http_request_duration_seconds` | Request duration histogram |
| `zot_storage_blobs_total` | Total number of blobs |
| `zot_storage_bytes_total` | Total storage used |
| `zot_gc_duration_seconds` | Garbage collection duration |

Example ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: zot-registry
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: zot
  endpoints:
  - port: api
    path: /metrics
    interval: 30s
```

## Configuration Reference

See [module.cue](./module.cue) for the complete `#config` schema.

## Examples

### Example 1: Private Registry with Auth

```cue
values: {
    image.variant: "full"
    storage: {
        type: "pvc"
        size: "50Gi"
    }
    auth: {
        htpasswd.credentials: { /* ... */ }
        accessControl: {
            adminUsers: ["admin"]
            repositories: {
                "**": {
                    policies: [{
                        users:   ["developer"]
                        actions: ["read"]
                    }]
                    defaultPolicy: []
                }
            }
        }
    }
    metrics.enabled: true
}
```

### Example 2: Public Mirror

```cue
values: {
    image.variant: "full"
    storage.type: "pvc"
    sync: {
        registries: [{
            urls:      ["https://docker.io"]
            onDemand:  true
            tlsVerify: true
            content: [{
                prefix: "library/**"
            }]
        }]
    }
    // No auth - public read-only mirror
}
```

### Example 3: CI/CD Registry

```cue
values: {
    image.variant: "minimal"
    storage.type: "emptyDir"  // Ephemeral builds
    log.level:    "debug"
    replicas:     1
    resources: {
        requests.memory: "512Mi"
        requests.cpu:    "250m"
    }
}
```

## Resources

- [Zot Documentation](https://zotregistry.dev)
- [Zot GitHub](https://github.com/project-zot/zot)
- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec)
- [Helm Chart](https://github.com/project-zot/helm-charts)

## License

This module follows the same license as the OPM project. Zot itself is Apache 2.0 licensed.
