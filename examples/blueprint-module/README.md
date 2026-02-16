# Blueprint Module — Simplified Module Authoring

**Complexity:** Intermediate (for understanding) / Beginner (for authoring)  
**Workload Types:** `stateless`, `stateful`

Demonstrates "easy mode" module authoring using **Blueprints** — pre-composed bundles of resources and traits that dramatically reduce boilerplate.

## What This Example Demonstrates

### Core Concepts
- **`#StatelessWorkload` blueprint** — Bundles `#Container` + `#Scaling` + optional traits
- **`#SimpleDatabase` blueprint** — Auto-generates database config from engine + credentials
- **Blueprint composition** — Single `statelessWorkload` / `simpleDatabase` field instead of multiple `spec` fields
- **Auto-generated behavior** — Env vars, volume mounts, health checks based on database engine

### OPM Patterns
- Blueprints for rapid prototyping
- Blueprint + additional traits (e.g., `#Expose` not in `#StatelessWorkload`)
- Side-by-side comparison with manual composition

## Architecture

```
┌──────────────────────────────────────────┐
│ api (StatelessWorkload Blueprint)       │
│   Container: node:20-alpine             │
│   Scaling: 3 replicas                   │
│   Health: /healthz, /ready              │
│   Expose: Service on port 3000          │
└──────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────┐
│ database (SimpleDatabase Blueprint)     │
│   Engine: postgres:15-alpine            │
│   Auto-generated:                       │
│     - POSTGRES_DB, POSTGRES_USER env    │
│     - Volume mount to /var/lib/postgres │
│     - Health check: pg_isready          │
│     - PVC: 10Gi                         │
└──────────────────────────────────────────┘
```

## Configuration Schema

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `api.image` | string | `"node:20-alpine"` | API container image |
| `api.replicas` | int | `3` | Number of API replicas |
| `api.port` | int | `3000` | API service port |
| `database.engine` | string | `"postgres"` | Database engine (postgres/mysql/mongodb/redis) |
| `database.version` | string | `"15-alpine"` | Database version tag |
| `database.dbName` | string | `"myapp"` | Database name |
| `database.username` | string | `"appuser"` | Database username |
| `database.password` | string | `"change-me-in-production"` | Database password (SENSITIVE) |
| `database.storage.size` | string | `"10Gi"` | PVC size |
| `database.storage.storageClass` | string | `"standard"` | Storage class |

## Rendered Kubernetes Resources

| Resource | Name | Type | Notes |
|----------|------|------|-------|
| Deployment | `api` | `apps/v1` | 3 replicas |
| Service | `api` | `v1` | ClusterIP (port 3000) |
| StatefulSet | `database` | `apps/v1` | 1 replica (auto-set by blueprint) |
| PersistentVolumeClaim | `database-data-0` | `v1` | 10Gi |

**Total:** 4 Kubernetes resources

## Usage

### Build (render to YAML)

```bash
# Render to stdout
opm mod build ./examples/blueprint-module

# Render to split files
opm mod build --split ./examples/blueprint-module
```

### Apply to Kubernetes

```bash
# Apply with defaults
opm mod apply ./examples/blueprint-module

# Apply to specific namespace
opm mod apply --namespace apps ./examples/blueprint-module
```

### Switch Database Engine

Edit `values.cue` to use MySQL instead of PostgreSQL:

```cue
values: {
    database: {
        engine:  "mysql"      // ← Changed from "postgres"
        version: "8.0"
        // ... rest stays the same
    }
}
```

The blueprint automatically:
- Changes image to `mysql:8.0`
- Sets `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD` env vars (instead of `POSTGRES_*`)
- Changes volume mount to `/var/lib/mysql`
- Changes health check to `mysqladmin ping -u root`

## Files

```
blueprint-module/
├── cue.mod/module.cue    # CUE dependencies (includes opmodel.dev/blueprints@v0)
├── module.cue            # Module metadata and config schema
├── components.cue        # Components using blueprints
└── values.cue            # Default configuration values
```

## Blueprint vs. Manual Composition

### With Blueprint (This Example)

```cue
api: {
    blueprints.#StatelessWorkload
    traits_network.#Expose

    spec: {
        statelessWorkload: {         // ← Single field
            container: { ... }
            scaling: { count: 3 }
            healthCheck: { ... }
        }
        expose: { ... }
    }
}
```

**Lines of code:** ~50 lines

### Without Blueprint (Manual)

```cue
api: {
    resources_workload.#Container    // ← Attach resources
    traits_workload.#Scaling         // ← Attach traits one-by-one
    traits_workload.#HealthCheck
    traits_workload.#RestartPolicy
    traits_workload.#UpdateStrategy
    traits_network.#Expose

    metadata: {
        labels: "core.opmodel.dev/workload-type": "stateless"  // ← Manual label
    }

    spec: {
        container: { ... }           // ← Multiple top-level fields
        scaling: { count: 3 }
        healthCheck: { ... }
        restartPolicy: "Always"
        updateStrategy: { ... }
        expose: { ... }
    }
}
```

**Lines of code:** ~80 lines

**Reduction:** ~40% fewer lines with blueprints

## SimpleDatabase Blueprint Magic

The `#SimpleDatabase` blueprint auto-generates configuration based on the `engine` field:

### PostgreSQL (`engine: "postgres"`)

```cue
spec: {
    simpleDatabase: {
        engine: "postgres"
        version: "15-alpine"
        dbName: "myapp"
        username: "appuser"
        password: "secret"
    }
}
```

**Auto-generated:**
- Image: `postgres:15-alpine`
- Env vars: `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`
- Volume mount: `/var/lib/postgresql/data`
- Health check: `pg_isready -U appuser`

### MySQL (`engine: "mysql"`)

```cue
spec: {
    simpleDatabase: {
        engine: "mysql"
        version: "8.0"
        // ... same fields
    }
}
```

**Auto-generated:**
- Image: `mysql:8.0`
- Env vars: `MYSQL_DATABASE`, `MYSQL_USER`, `MYSQL_PASSWORD`
- Volume mount: `/var/lib/mysql`
- Health check: `mysqladmin ping -u root`

### MongoDB (`engine: "mongodb"`)

**Auto-generated:**
- Image: `mongo:<version>`
- Env vars: `MONGO_INITDB_DATABASE`, `MONGO_INITDB_ROOT_USERNAME`, `MONGO_INITDB_ROOT_PASSWORD`
- Volume mount: `/data/db`
- Health check: `mongo --eval "db.adminCommand('ping')"`

### Redis (`engine: "redis"`)

**Auto-generated:**
- Image: `redis:<version>`
- Env vars: _(none, Redis has no init env vars)_
- Volume mount: `/data`
- Health check: `redis-cli ping`

## Key Code Snippets

### StatelessWorkload Blueprint Usage

```cue
import (
    blueprints "opmodel.dev/blueprints@v0"
)

#components: {
    api: {
        blueprints.#StatelessWorkload  // ← Attach blueprint

        spec: {
            statelessWorkload: {               // ← Single spec field
                container: {
                    name:  "api"
                    image: #config.api.image
                    ports: http: { targetPort: 3000 }
                }
                scaling: { count: #config.api.replicas }
                healthCheck: { ... }
            }
        }
    }
}
```

The blueprint handles:
- Attaching `#Container` resource
- Attaching `#Scaling` trait
- Setting `workload-type: stateless` label
- Mapping `statelessWorkload.container` → `spec.container`
- Mapping `statelessWorkload.scaling` → `spec.scaling`

### SimpleDatabase Blueprint Usage

```cue
import (
    blueprints "opmodel.dev/blueprints@v0"
)

#components: {
    database: {
        blueprints.#SimpleDatabase  // ← Attach blueprint

        spec: {
            simpleDatabase: {          // ← Single spec field
                engine:   "postgres"
                version:  "15-alpine"
                dbName:   "myapp"
                username: "appuser"
                password: "secret"
                persistence: {
                    enabled: true
                    size:    "10Gi"
                }
            }
        }
    }
}
```

The blueprint handles:
- Attaching `#Container` + `#Volumes` resources
- Attaching `#Scaling` + `#RestartPolicy` + `#HealthCheck` traits
- Setting `workload-type: stateful` label
- Generating engine-specific image, env vars, volume mounts, health checks
- Configuring PVC from `persistence.size`

### Combining Blueprint + Additional Traits

Blueprints don't include every possible trait. You can attach additional traits:

```cue
api: {
    blueprints.#StatelessWorkload  // ← Blueprint provides: Container, Scaling
    traits_network.#Expose         // ← Add trait not in blueprint

    spec: {
        statelessWorkload: { ... }         // ← Blueprint field
        expose: { ... }                    // ← Additional trait field
    }
}
```

## When to Use Blueprints

### Use Blueprints When:
- **Rapid prototyping** — You want to spin up a module quickly
- **Standard workloads** — Your use case fits a common pattern (stateless app, simple DB)
- **Learning OPM** — You're new and want to avoid boilerplate
- **Internal tooling** — Dev/staging environments with simple requirements

### Use Manual Composition When:
- **Custom requirements** — You need fine-grained control over resource/trait attachment
- **Production workloads** — You want explicit visibility into every config field
- **Complex patterns** — Blueprints don't support your use case (e.g., multi-container pods)
- **Advanced traits** — You need traits not included in blueprints (e.g., `#UpdateStrategy`, `#InitContainers`)

## Available Blueprints

| Blueprint | FQN | Workload Type | Composed Resources | Composed Traits |
|-----------|-----|---------------|-------------------|----------------|
| **#StatelessWorkload** | `opmodel.dev/blueprints@v0#StatelessWorkload` | stateless | Container | Scaling |
| **#StatefulWorkload** | `opmodel.dev/blueprints@v0#StatefulWorkload` | stateful | Container, Volumes | Scaling, RestartPolicy, UpdateStrategy, HealthCheck, SidecarContainers, InitContainers |
| **#DaemonWorkload** | `opmodel.dev/blueprints@v0#DaemonWorkload` | daemon | Container | RestartPolicy, UpdateStrategy, HealthCheck, SidecarContainers, InitContainers |
| **#TaskWorkload** | `opmodel.dev/blueprints@v0#TaskWorkload` | task | Container | JobConfig, RestartPolicy, SidecarContainers, InitContainers |
| **#ScheduledTaskWorkload** | `opmodel.dev/blueprints@v0#ScheduledTaskWorkload` | scheduled-task | Container | CronJobConfig, RestartPolicy, SidecarContainers, InitContainers |
| **#SimpleDatabase** | `opmodel.dev/blueprints@v0#SimpleDatabase` | stateful | Container, Volumes | Scaling, RestartPolicy, HealthCheck |

## Dependencies

To use blueprints, add `opmodel.dev/blueprints@v0` to your `cue.mod/module.cue`:

```cue
deps: {
    "opmodel.dev/blueprints@v0": {
        v: "v0.1.7"
    }
}
```

## Limitations

1. **Blueprints are opinionated** — They make decisions for you (e.g., `SimpleDatabase` sets scaling.count = 1)
2. **Less flexibility** — You can't remove composed resources/traits from a blueprint
3. **Learning curve** — You need to learn blueprint-specific field names (`statelessWorkload`, `simpleDatabase`)
4. **Catalog dependency** — Blueprints are defined in the catalog, not in your module

## Next Steps

- **Multi-package structure:** See upcoming `multi-package-module/` example
- **Manual composition:** Compare with [blog/](../blog/), [jellyfin/](../jellyfin/), [multi-tier-module/](../multi-tier-module/)
- **Advanced traits:** See [webapp-ingress/](../webapp-ingress/) for `#HttpRoute`, `#SecurityContext`, etc.

## Related Examples

- [blog/](../blog/) — Manual composition with `#Container` + `#Replicas` + `#Expose`
- [jellyfin/](../jellyfin/) — Manual composition with `#Volumes` + `#HealthCheck`
- [multi-tier-module/](../multi-tier-module/) — Manual composition with all workload types
- [app-config/](../app-config/) — ConfigMaps and Secrets
