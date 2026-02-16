# Blog — Multi-Component Stateless Application

**Complexity:** Beginner  
**Workload Types:** `stateless` (Deployment)

A simple two-tier web application demonstrating the basics of OPM module structure, multi-component composition, and cross-component references.

## What This Example Demonstrates

### Core Concepts

- **Multi-component module** — Two components (`web` frontend, `api` backend) in a single module
- **`#Container` resource** — Container definitions with images, ports, and environment variables
- **`#Replicas` trait** — Simple replica count for horizontal scaling
- **`#Expose` trait** — Service exposure with port mapping
- **Cross-component references** — `web` component referencing `api` port via `#config`

### OPM Patterns

- Standard 3-file structure (`module.cue`, `components.cue`, `values.cue`)
- Config schema (`#config`) separating constraints from concrete values
- Static environment variable wiring (`API_URL` pointing to backend service)

## Architecture

```
┌─────────────┐
│ web (nginx) │────────────┐
│   port 8080 │            │
└─────────────┘            │
                           │ API_URL=http://api:3000
                           ▼
              ┌────────────────────┐
              │ api (node:alpine)  │
              │      port 3000     │
              └────────────────────┘
```

## Configuration Schema

| Field | Type | Constraint | Default | Description |
|-------|------|------------|---------|-------------|
| `web.image` | string | - | `"nginx:1.25"` | Web frontend container image |
| `web.replicas` | int | >= 1 | `4` | Number of web replicas |
| `web.port` | int | 1-65535 | `8080` | Exposed service port for web |
| `api.image` | string | - | `"node:20-alpine"` | API backend container image |
| `api.replicas` | int | >= 1 | `4` | Number of API replicas |
| `api.port` | int | 1-65535 | `3000` | API service port |

## Rendered Kubernetes Resources

| Resource | Name | Type | Replicas |
|----------|------|------|----------|
| Deployment | `web` | `apps/v1` | 4 |
| Deployment | `api` | `apps/v1` | 4 |
| Service | `web` | `v1` | ClusterIP (port 8080) |

**Total:** 3 Kubernetes resources

## Usage

### Build (render to YAML)

```bash
# Render to stdout
opm mod build ./examples/blog

# Render to split files (output: ./manifests/*.yaml)
opm mod build --split ./examples/blog
```

### Apply to Kubernetes

```bash
# Apply with defaults
opm mod apply ./examples/blog

# Apply to specific namespace
opm mod apply --namespace blog-prod ./examples/blog
```

### Customize Values

Override defaults by editing `values.cue` or creating a separate values file:

```cue
// values_dev.cue
package main

values: {
    web: replicas: 1     // Dev uses fewer replicas
    api: replicas: 1
}
```

Then build with the override:

```bash
opm mod build -f values_dev.cue ./examples/blog
```

## Files

```
blog/
├── cue.mod/module.cue    # CUE dependencies (opmodel.dev/core@v0, etc.)
├── module.cue            # Module metadata and config schema
├── components.cue        # Component definitions (web, api)
└── values.cue            # Default configuration values
```

## Key Code Snippets

### Cross-Component Reference

The `web` component references the `api` port from config:

```cue
// components.cue (web component)
spec: {
    container: {
        env: apiUrl: {
            name:  "API_URL"
            value: "http://api:\(#config.api.port)"  // ← References api.port
        }
    }
}
```

At build time, `#config.api.port` resolves to the concrete value `3000` from `values.cue`, producing:

```yaml
env:
  - name: API_URL
    value: "http://api:3000"
```

### Port Mapping

The `web` component exposes its container port (80) as a service on port 8080:

```cue
spec: {
    container: {
        ports: http: { targetPort: 80 }
    }
    expose: {
        ports: http: container.ports.http & {
            exposedPort: #config.web.port  // From values: 8080
        }
        type: "ClusterIP"
    }
}
```

Produces a Kubernetes Service:

```yaml
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 8080        # External service port
      targetPort: 80    # Container port
```

## Next Steps

- **Add health checks:** See [jellyfin/](../jellyfin/) for liveness/readiness probes
- **Add persistent storage:** See [jellyfin/](../jellyfin/) for volume examples
- **Explore workload types:** See [multi-tier-module/](../multi-tier-module/) for StatefulSet, DaemonSet, Job, and CronJob

## Related Examples

- [jellyfin/](../jellyfin/) — Stateful workload with volumes and health checks
- [multi-tier-module/](../multi-tier-module/) — All workload types with advanced traits
