# Values Layering — Environment-Specific Configuration

**Complexity:** Intermediate  
**Workload Types:** `stateless` (Deployment)

Demonstrates environment-specific configuration using CUE value overlays. A single module definition with separate value files for dev, staging, and production environments.

## What This Example Demonstrates

### Core Concepts
- **Values layering** — Base `values.cue` + environment-specific overrides
- **`-f` flag pattern** — Multiple values files unified at build time
- **CUE constraints** — Schema validation preventing invalid environment configs
- **Conditional constraints** — Production-specific requirements (min replicas, TLS)
- **Environment labels** — Tracking environment via metadata labels

### OPM Patterns
- Single module definition, multiple deployment profiles
- Schema-enforced production safety (e.g., `replicas >= 2`, `tls.enabled = true`)
- Development-optimized defaults with production overrides
- Image pinning strategies (tag → version → SHA digest)

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ SAME MODULE DEFINITION (module.cue + components.cue)        │
└──────────────────────────────────────────────────────────────┘
                              │
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
         ▼                    ▼                    ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ values.cue (dev) │  │ values_staging   │  │ values_production│
│                  │  │                  │  │                  │
│ replicas: 1      │  │ replicas: 3      │  │ replicas: 5      │
│ cpu: 50m         │  │ cpu: 100m        │  │ cpu: 200m        │
│ image: :alpine   │  │ image: :1.25.3   │  │ image: @sha256   │
│ hostname: .local │  │ hostname: -stg   │  │ hostname: .com   │
│ tls: false       │  │ tls: true        │  │ tls: true        │
└──────────────────┘  └──────────────────┘  └──────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ Dev K8s          │  │ Staging K8s      │  │ Production K8s   │
│ 1 replica        │  │ 3 replicas       │  │ 5 replicas       │
│ No Ingress       │  │ Ingress + TLS    │  │ Ingress + TLS    │
│ 50m CPU          │  │ 100m CPU         │  │ 200m CPU         │
└──────────────────┘  └──────────────────┘  └──────────────────┘
```

## Configuration Files

### Base Values (`values.cue`) — Development Defaults

```cue
values: {
    environment: "dev"
    web: {
        image:    "nginx:1.25-alpine"       // Lightweight dev image
        replicas: 1                         // Single replica
        resources: {
            requests: { cpu: "50m", memory: "64Mi" }
            limits:   { cpu: "200m", memory: "128Mi" }
        }
        ingress: {
            hostname: "webapp-dev.local"
            tls: enabled: false             // No TLS in dev
        }
    }
}
```

### Staging Overrides (`values_staging.cue`)

```cue
values: {
    environment: "staging"
    web: {
        image:    "nginx:1.25.3-alpine"     // Pinned version
        replicas: 3                         // Scale for load testing
        resources: {
            requests: { cpu: "100m", memory: "128Mi" }
            limits:   { cpu: "500m", memory: "256Mi" }
        }
        ingress: {
            hostname: "webapp-staging.example.com"
            tls: {
                enabled:    true            // TLS enabled
                secretName: "webapp-staging-tls"
            }
        }
    }
}
```

### Production Overrides (`values_production.cue`)

```cue
values: {
    environment: "production"
    web: {
        image:    "nginx:1.25.3-alpine@sha256:a592..."  // SHA digest for immutability
        replicas: 5                         // High availability (enforced: >= 2)
        resources: {
            requests: { cpu: "200m", memory: "256Mi" }
            limits:   { cpu: "1000m", memory: "512Mi" }
        }
        ingress: {
            hostname: "webapp.example.com"
            tls: {
                enabled:    true            // TLS required (enforced)
                secretName: "webapp-production-tls"
            }
        }
    }
}
```

## Schema Constraints (Environment-Specific)

The module schema enforces production-specific requirements:

```cue
#config: {
    environment: "dev" | "staging" | "production"

    web: {
        // Production requires at least 2 replicas
        if environment == "production" {
            replicas: >=2
        }

        // Production requires TLS
        if environment == "production" {
            ingress: tls: enabled: true
        }
    }
}
```

**What this means:**
- Dev/staging can use 1 replica and no TLS
- Production **MUST** have 2+ replicas and TLS enabled
- Build fails if you try `values_production.cue` with `replicas: 1` or `tls.enabled: false`

## Usage

### Build for Development (default)

```bash
# Uses values.cue (dev defaults)
opm mod build ./examples/values-layering

# Explicit dev values
opm mod build -f values.cue ./examples/values-layering
```

### Build for Staging

```bash
# Override with staging values
opm mod build -f values_staging.cue ./examples/values-layering

# Render to split files
opm mod build --split -f values_staging.cue ./examples/values-layering
```

### Build for Production

```bash
# Override with production values
opm mod build -f values_production.cue ./examples/values-layering

# Apply to production namespace
opm mod apply --namespace production -f values_production.cue ./examples/values-layering
```

### Validate Schema Constraints

Try to build production with invalid values:

```bash
# This FAILS: replicas < 2
opm mod build -f <(cat <<EOF
package main
values: {
    environment: "production"
    web: replicas: 1  // ❌ Violates constraint (production requires >= 2)
}
EOF
) ./examples/values-layering
```

Error:

```
values.web.replicas: invalid value 1 (out of bound >=2)
```

## Rendered Kubernetes Resources

| Environment | Deployment Replicas | Ingress | TLS | CPU Request | Memory Request |
|-------------|---------------------|---------|-----|-------------|----------------|
| **Dev** | 1 | webapp-dev.local | No | 50m | 64Mi |
| **Staging** | 3 | webapp-staging.example.com | Yes | 100m | 128Mi |
| **Production** | 5 | webapp.example.com | Yes | 200m | 256Mi |

Each environment produces the same resource types:
- 1 Deployment
- 1 Service
- 1 Ingress

## Files

```
values-layering/
├── cue.mod/module.cue       # CUE dependencies
├── module.cue               # Module metadata + schema (with constraints)
├── components.cue           # Environment-agnostic component definitions
├── values.cue               # Base values (dev defaults)
├── values_staging.cue       # Staging overrides
└── values_production.cue    # Production overrides
```

## Key Code Snippets

### Conditional Schema Constraints

```cue
// module.cue
#config: {
    environment: "dev" | "staging" | "production"

    web: {
        replicas: int & >=1 & <=100

        // Production-specific constraints
        if environment == "production" {
            replicas: >=2  // Strengthen constraint for production
        }

        ingress: {
            tls: {
                enabled: bool | *false

                // Production requires TLS
                if environment == "production" {
                    enabled: true  // Force TLS in production
                }
            }
        }
    }
}
```

When `environment == "production"`, CUE unifies:
- `replicas: int & >=1 & <=100` (base)
- `replicas: >=2` (production-specific)
- Result: `replicas: int & >=2 & <=100`

### Environment Label Propagation

```cue
// components.cue
#components: {
    web: {
        metadata: {
            labels: {
                "core.opmodel.dev/workload-type": "stateless"
                "app.example.com/environment":    #config.environment  // ← Propagates to all resources
            }
        }
    }
}
```

Produces labels on Deployment/Service/Ingress:

```yaml
metadata:
  labels:
    core.opmodel.dev/workload-type: stateless
    app.example.com/environment: production  # ← From values
```

### Values Override Pattern

CUE unifies multiple value files:

```
Base (values.cue):
  environment: "dev"
  web.replicas: 1

Override (values_production.cue):
  environment: "production"
  web.replicas: 5

Result (after unification):
  environment: "production"  // ← Override wins
  web.replicas: 5            // ← Override wins
  web.resources: { ... }     // ← Inherited from base
```

Fields not specified in override inherit from base.

## Image Pinning Strategies

| Strategy | Example | Use Case | Pros | Cons |
|----------|---------|----------|------|------|
| **Tag only** | `nginx:alpine` | Dev | Easy updates | Non-deterministic |
| **Versioned tag** | `nginx:1.25.3-alpine` | Staging | Specific version | Mutable |
| **SHA digest** | `nginx:1.25.3@sha256:a592...` | Production | Immutable | Verbose |

**Best practice:** Use SHA digests in production to prevent tag mutation attacks.

## Environment-Specific Workflows

### Development Workflow

```bash
# Build and apply to dev namespace
opm mod apply --namespace dev ./examples/values-layering

# Watch logs
kubectl logs -n dev -l app.example.com/environment=dev --tail=100 -f
```

### Staging Workflow

```bash
# Build with staging values
opm mod build -f values_staging.cue --split ./examples/values-layering

# Review manifests before applying
cat manifests/*.yaml

# Apply to staging namespace
opm mod apply --namespace staging -f values_staging.cue ./examples/values-layering
```

### Production Workflow

```bash
# Validate production values against schema
opm mod vet -f values_production.cue ./examples/values-layering

# Build production manifests
opm mod build -f values_production.cue --split ./examples/values-layering

# Review critical changes (diff against live)
opm mod diff --namespace production -f values_production.cue ./examples/values-layering

# Apply to production (with caution!)
opm mod apply --namespace production -f values_production.cue ./examples/values-layering
```

## Advanced: Multi-File Layering

You can stack multiple override files:

```bash
# Base + region-specific + env-specific
opm mod build \
  -f values_us_east.cue \
  -f values_production.cue \
  ./examples/values-layering
```

**Order matters:** Later files override earlier ones.

## Next Steps

- **Simplify with Blueprints:** See upcoming `blueprint-module/` example
- **Add secrets management:** See [app-config/](../app-config/) for ConfigMaps/Secrets
- **Multi-package structure:** See upcoming `multi-package-module/` example

## Related Examples

- [blog/](../blog/) — Simple multi-component stateless app
- [webapp-ingress/](../webapp-ingress/) — Production web app with Ingress and HPA
- [app-config/](../app-config/) — ConfigMaps, Secrets, and volume-mounted config
