# WebApp with Ingress — Production Web Application

**Complexity:** Advanced  
**Workload Types:** `stateless` (Deployment)

A production-grade stateless web application demonstrating HTTP Ingress routing, Horizontal Pod Autoscaling (HPA), security hardening, service account management, and sidecar containers.

## What This Example Demonstrates

### Core Concepts
- **`#HttpRoute` trait** → IngressTransformer → `networking.k8s.io/v1 Ingress`
- **`#Scaling` trait with `auto`** → HpaTransformer → `autoscaling/v2 HorizontalPodAutoscaler`
- **`#SecurityContext` trait** — Non-root user, dropped capabilities, restricted filesystem
- **`#WorkloadIdentity` trait** → ServiceAccountTransformer → `v1 ServiceAccount`
- **`#SidecarContainers` trait** — Log forwarder sidecar (optional)
- **`#HealthCheck` trait** — HTTP liveness and readiness probes
- **Resource requests and limits** — CPU and memory constraints

### OPM Patterns
- Conditional sidecar attachment (`if #config.web.sidecar.enabled`)
- CPU-based autoscaling with min/max bounds
- Path-based HTTP routing with hostname matching
- TLS termination at ingress (optional)
- Security best practices (runAsNonRoot, drop ALL capabilities)

## Architecture

```
Internet
    │
    ▼
┌─────────────────────────────┐
│ Ingress (nginx)             │  Hostname: webapp.example.com
│   Path: / → web:8080        │  TLS: optional
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│ Service (ClusterIP)         │  Port: 8080
│   Selector: web             │
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ HorizontalPodAutoscaler                 │  Min: 2, Max: 10
│   Target: 70% CPU utilization           │  Metrics: CPU
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Deployment: web (2-10 replicas)         │
│                                          │
│  ┌──────────────────────────────────┐   │
│  │ Container: web (nginx:1.25)      │   │
│  │   Port: 8080                     │   │
│  │   Health: /healthz, /ready       │   │
│  │   Resources: 100m CPU, 128Mi RAM │   │
│  │   Security: runAsUser 1000       │   │
│  │             drop ALL caps        │   │
│  └──────────────────────────────────┘   │
│                                          │
│  ┌──────────────────────────────────┐   │
│  │ Sidecar: log-forwarder (optional)│   │
│  │   Image: fluent/fluent-bit:2.0   │   │
│  └──────────────────────────────────┘   │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│ ServiceAccount: webapp      │
│   automountToken: false     │
└─────────────────────────────┘
```

## Configuration Schema

| Field | Type | Constraint | Default | Description |
|-------|------|------------|---------|-------------|
| `web.image` | string | - | `"nginx:1.25-alpine"` | Container image |
| `web.port` | int | 1-65535 | `8080` | Service port |
| `web.scaling.min` | int | 1-100 | `2` | Minimum replicas |
| `web.scaling.max` | int | 1-100 | `10` | Maximum replicas |
| `web.scaling.targetCPUUtilization` | int | 1-100 | `70` | Target CPU % for autoscaling |
| `web.resources.requests.cpu` | string | - | `"100m"` | CPU request (100 millicores) |
| `web.resources.requests.memory` | string | - | `"128Mi"` | Memory request |
| `web.resources.limits.cpu` | string | - | `"500m"` | CPU limit |
| `web.resources.limits.memory` | string | - | `"512Mi"` | Memory limit |
| `web.ingress.hostname` | string | - | `"webapp.example.com"` | Ingress hostname |
| `web.ingress.path` | string | - | `"/"` | URL path prefix |
| `web.ingress.ingressClassName` | string | - | `"nginx"` | Ingress controller class |
| `web.ingress.tls.enabled` | bool | - | `false` | Enable TLS termination |
| `web.ingress.tls.secretName` | string? | - | `"webapp-tls"` | TLS certificate secret name |
| `web.security.runAsUser` | int | - | `1000` | User ID to run container as |
| `web.security.runAsGroup` | int | - | `1000` | Group ID |
| `web.security.fsGroup` | int | - | `1000` | Filesystem group ID |
| `web.serviceAccount.name` | string | - | `"webapp"` | Service account name |
| `web.sidecar.enabled` | bool | - | `false` | Enable log forwarder sidecar |
| `web.sidecar.image` | string | - | `"fluent/fluent-bit:2.0-distroless"` | Sidecar image |

## Rendered Kubernetes Resources

| Resource | Name | Type | Notes |
|----------|------|------|-------|
| Deployment | `web` | `apps/v1` | 2-10 replicas (autoscaled) |
| Service | `web` | `v1` | ClusterIP (port 8080) |
| Ingress | `web` | `networking.k8s.io/v1` | HTTP routing to service |
| HorizontalPodAutoscaler | `web` | `autoscaling/v2` | CPU-based scaling |
| ServiceAccount | `webapp` | `v1` | Workload identity |

**Total:** 5 Kubernetes resources

## Usage

### Build (render to YAML)

```bash
# Render to stdout
opm mod build ./examples/webapp-ingress

# Render to split files
opm mod build --split ./examples/webapp-ingress
```

### Apply to Kubernetes

```bash
# Apply with defaults
opm mod apply ./examples/webapp-ingress

# Apply to production namespace
opm mod apply --namespace production ./examples/webapp-ingress
```

### Enable TLS

Edit `values.cue` or create a custom values file:

```cue
// values_prod.cue
package main

values: {
    web: {
        ingress: {
            hostname: "webapp.example.com"
            tls: {
                enabled:    true
                secretName: "webapp-tls"  // Must exist in namespace
            }
        }
    }
}
```

Apply with TLS:

```bash
opm mod apply -f values_prod.cue ./examples/webapp-ingress
```

### Enable Sidecar Container

```cue
// values_sidecar.cue
package main

values: {
    web: {
        sidecar: {
            enabled: true
            image:   "fluent/fluent-bit:2.0-distroless"
        }
    }
}
```

Apply with sidecar:

```bash
opm mod apply -f values_sidecar.cue ./examples/webapp-ingress
```

### Watch Autoscaling

```bash
# Generate load to trigger autoscaling
kubectl run load-generator --image=busybox -- /bin/sh -c "while true; do wget -q -O- http://web:8080; done"

# Watch HPA scale up
kubectl get hpa web --watch

# Clean up
kubectl delete pod load-generator
```

## Files

```
webapp-ingress/
├── cue.mod/module.cue    # CUE dependencies
├── module.cue            # Module metadata and config schema
├── components.cue        # Web component definition
└── values.cue            # Default configuration values
```

## Key Code Snippets

### Horizontal Pod Autoscaler (HPA)

```cue
spec: {
    scaling: {
        count: #config.web.scaling.min  // Initial replica count
        auto: {
            min: #config.web.scaling.min      // Scale down to 2
            max: #config.web.scaling.max      // Scale up to 10
            metrics: [{
                type: "cpu"
                target: {
                    averageUtilization: #config.web.scaling.targetCPUUtilization  // 70%
                }
            }]
        }
    }
}
```

The HPA monitors CPU usage across all pods. When average CPU exceeds 70%, it adds replicas (up to 10). When CPU drops below 70%, it removes replicas (down to 2).

### HTTP Ingress with Path-Based Routing

```cue
spec: {
    httpRoute: {
        if #config.web.ingress.hostname != "" {
            hostnames: [#config.web.ingress.hostname]  // ["webapp.example.com"]
        }
        rules: [{
            matches: [{
                path: {
                    type:  "Prefix"     // Match all paths starting with "/"
                    value: #config.web.ingress.path
                }
            }]
            backendPort: #config.web.port  // Route to service port 8080
        }]
        if #config.web.ingress.ingressClassName != "" {
            ingressClassName: #config.web.ingress.ingressClassName  // "nginx"
        }
        if #config.web.ingress.tls.enabled {
            tls: {
                mode: "Terminate"
                if #config.web.ingress.tls.secretName != _|_ {
                    certificateRef: {
                        name: #config.web.ingress.tls.secretName
                    }
                }
            }
        }
    }
}
```

Produces an Ingress resource:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web
spec:
  ingressClassName: nginx
  rules:
    - host: webapp.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web
                port:
                  number: 8080
  tls:  # Only if tls.enabled=true
    - hosts:
        - webapp.example.com
      secretName: webapp-tls
```

### Security Context (Hardening)

```cue
spec: {
    securityContext: {
        runAsNonRoot:             true    // Refuse to run as root
        runAsUser:                #config.web.security.runAsUser      // UID 1000
        runAsGroup:               #config.web.security.runAsGroup     // GID 1000
        readOnlyRootFilesystem:   false   // Set true if app writes to /tmp only
        allowPrivilegeEscalation: false   // Prevent gaining privileges
        capabilities: {
            drop: ["ALL"]  // Drop all Linux capabilities
        }
    }
}
```

Produces a securityContext in the pod spec:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
```

### Service Account

```cue
spec: {
    workloadIdentity: {
        name:           #config.web.serviceAccount.name  // "webapp"
        automountToken: false  // Don't auto-mount token (security best practice)
    }
}
```

Produces:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: webapp
automountServiceAccountToken: false
```

The pod spec will reference this ServiceAccount:

```yaml
spec:
  serviceAccountName: webapp
```

### Conditional Sidecar Container

```cue
spec: {
    if #config.web.sidecar.enabled {
        sidecarContainers: [{
            name:  "log-forwarder"
            image: #config.web.sidecar.image
            env: {
                SIDECAR_MODE: {
                    name:  "SIDECAR_MODE"
                    value: "forwarder"
                }
            }
        }]
    }
}
```

When `sidecar.enabled = true`, produces an additional container in the pod:

```yaml
spec:
  containers:
    - name: web
      image: nginx:1.25-alpine
      # ... main container spec ...
    - name: log-forwarder
      image: fluent/fluent-bit:2.0-distroless
      env:
        - name: SIDECAR_MODE
          value: forwarder
```

## Security Best Practices Demonstrated

1. **Non-root execution** — `runAsNonRoot: true`, `runAsUser: 1000`
2. **Dropped capabilities** — `drop: ["ALL"]` removes all Linux capabilities
3. **No privilege escalation** — `allowPrivilegeEscalation: false`
4. **Service account token** — `automountToken: false` (only mount if needed)
5. **Resource limits** — Prevents resource exhaustion attacks
6. **Health checks** — Auto-restart unhealthy containers

## Autoscaling Behavior

HPA calculates desired replicas:

```
desiredReplicas = ceil(currentReplicas * (currentCPU / targetCPU))
```

Example:
- Current: 2 replicas, average CPU 85%
- Target: 70%
- Calculation: `ceil(2 * (85 / 70)) = ceil(2.43) = 3 replicas`

HPA adds 1 replica. If CPU drops to 60%:
- Calculation: `ceil(3 * (60 / 70)) = ceil(2.57) = 3 replicas`

No change (below target, but not enough to remove a replica yet).

## Next Steps

- **Add ConfigMaps/Secrets:** See upcoming `app-config/` example
- **Add environment-specific config:** See upcoming `values-layering/` example
- **Simplify with Blueprints:** See upcoming `blueprint-module/` example

## Related Examples

- [blog/](../blog/) — Simple multi-component stateless app
- [jellyfin/](../jellyfin/) — Stateful workload with persistent storage
- [multi-tier-module/](../multi-tier-module/) — All workload types with advanced traits
