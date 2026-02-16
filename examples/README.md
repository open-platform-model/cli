# OPM Examples

This directory contains example OPM modules demonstrating various features and patterns. Examples are ordered from beginner to advanced — follow them in sequence for the best learning experience.

## Quick Start

All examples follow the standard 3-file OPM module structure:

```
example-module/
├── cue.mod/module.cue    # CUE module dependencies
├── module.cue            # Module metadata and config schema
├── components.cue        # Component definitions
└── values.cue            # Concrete configuration values
```

### Building an Example

```bash
# Render to stdout (YAML)
opm mod build ./examples/blog

# Render to split files
opm mod build --split ./examples/blog

# Apply to Kubernetes
opm mod apply ./examples/blog
```

---

## Learning Path

### 1. [blog/](blog/) — **Beginner: Multi-Component Stateless App**

**What it demonstrates:**
- Multi-component module (frontend + backend)
- `#Container` resource with ports and environment variables
- `#Replicas` and `#Expose` traits
- Cross-component references (`web` referencing `api` port)
- Config schema (`#config`) and concrete values separation

**Kubernetes output:** 2 Deployments, 1 Service

**Start here if:** You're new to OPM and want to understand the basics.

---

### 2. [jellyfin/](jellyfin/) — **Intermediate: Stateful Application**

**What it demonstrates:**
- Single stateful component (`workload-type: "stateful"`)
- `#Volumes` resource with persistent storage (PVC) and emptyDir
- `#HealthCheck` trait with liveness and readiness probes
- `#Scaling` trait (fixed at 1 for stateful workloads)
- Conditional CUE logic (`if #config.publishedServerUrl != _|_`)
- Dynamic volume mounts from config map
- Resource requests and limits

**Kubernetes output:** 1 StatefulSet, 1 Service, 2+ PersistentVolumeClaims

**Start here if:** You understand the basics and need persistent storage or health checks.

---

### 3. [multi-tier-module/](multi-tier-module/) — **Advanced: All Workload Types**

**What it demonstrates:**
- All 4 workload types: `stateful`, `daemon`, `task`, `scheduled-task`
- `#UpdateStrategy` trait with rolling update config
- `#InitContainers` trait for setup tasks
- `#JobConfig` trait (completions, parallelism, backoffLimit)
- `#CronJobConfig` trait (schedule, concurrency policy, history limits)
- Mix of exec and httpGet health probes

**Kubernetes output:** 1 StatefulSet, 1 DaemonSet, 1 Job, 1 CronJob, 2 PersistentVolumeClaims

**Start here if:** You need batch processing, scheduled tasks, or per-node daemons.

---

### 4. [minecraft/](minecraft/) — **Advanced: Stateful with Backup Sidecar**

**What it demonstrates:**
- Stateful workload with persistent game data
- `#SidecarContainers` trait → Backup container coordinated via RCON
- Flexible storage: PVC (cloud), hostPath (bare-metal), emptyDir (testing)
- Multiple backup methods: tar, rsync, restic (cloud), rclone (remote)
- Conditional trait attachment (`if #config.backup.enabled`)
- Advanced health probes with custom commands
- Production-ready defaults with override examples

**Kubernetes output:** 1 StatefulSet (server + backup sidecar), 1 Service (LoadBalancer), 2 PersistentVolumeClaims

**Start here if:** You need stateful applications with automated backups or want to see advanced sidecar patterns.

---

### 5. [webapp-ingress/](webapp-ingress/) — **Advanced: Production Web App**

**What it demonstrates:**
- `#HttpRoute` trait → Ingress with hostname routing and TLS
- `#Scaling` trait with `auto` → HorizontalPodAutoscaler (CPU-based)
- `#SecurityContext` trait → Non-root user, dropped capabilities
- `#WorkloadIdentity` trait → ServiceAccount creation
- `#SidecarContainers` trait → Log forwarder sidecar (optional)

**Kubernetes output:** 1 Deployment, 1 Service, 1 Ingress, 1 HPA, 1 ServiceAccount

**Start here if:** You need production-grade features (Ingress, autoscaling, security).

---

### 6. [app-config/](app-config/) — **Intermediate: Configuration Management**

**What it demonstrates:**
- `#ConfigMaps` resource → ConfigMap generation
- `#Secrets` resource → Secret generation (base64 encoded)
- Volume-mounted ConfigMaps → Config files in containers
- Environment variables from config → Static env var wiring

**Kubernetes output:** 1 Deployment, 1 Service, 2 ConfigMaps, 2 Secrets

**Start here if:** You need externalized configuration or secret management.

---

### 7. [values-layering/](values-layering/) — **Intermediate: Environment-Specific Config**

**What it demonstrates:**
- Base `values.cue` with dev defaults
- Override files: `values_staging.cue`, `values_production.cue`
- `-f` flag pattern for layered configuration
- Schema constraints enforcing production requirements (min replicas, TLS)
- Environment label propagation

**Kubernetes output:** Same resources, different configurations per environment

**Start here if:** You need to deploy the same module to dev/staging/prod with different configs.

---

### 8. [blueprint-module/](blueprint-module/) — **Intermediate: Simplified Authoring**

**What it demonstrates:**
- `#StatelessWorkload` blueprint → Single field replaces multiple resources/traits
- `#SimpleDatabase` blueprint → Auto-generated DB config based on engine
- Blueprint + additional traits → Combining blueprints with manual traits
- 40% less boilerplate compared to manual composition

**Kubernetes output:** 1 Deployment, 1 Service, 1 StatefulSet, 1 PVC

**Start here if:** You want rapid prototyping with less boilerplate.

---

### 9. [multi-package-module/](multi-package-module/) — **Advanced: Large Module Organization**

**What it demonstrates:**
- Multi-package CUE architecture → Separate `main` and `components` packages
- One file per component → `components/frontend.cue`, `components/backend.cue`, etc.
- Component aggregation → `components/components.cue` exports `#all`
- Package imports → `import "example.com/multi-package-module@v0/components"`

**Kubernetes output:** 3 Deployments, 2 Services

**Start here if:** You have 10+ components and need better organization.

---

## Common Patterns

### Config Schema vs. Values

OPM separates **constraints** (`#config`) from **concrete values** (`values`):

```cue
// module.cue - Schema with constraints
#config: {
    replicas: int & >=1 & <=100  // Constraint: 1-100
    image:    string              // Required string
}

// values.cue - Concrete values
values: {
    replicas: 3           // Satisfies constraint
    image:    "nginx:1.25"
}
```

### Component Structure

Every component follows this pattern:

```cue
#components: {
    myComponent: {
        // 1. Attach resources (REQUIRED)
        resources_workload.#Container

        // 2. Attach traits (OPTIONAL)
        traits_workload.#Scaling
        traits_network.#Expose

        // 3. Set workload type label (REQUIRED for Container resources)
        metadata: labels: "core.opmodel.dev/workload-type": "stateless"

        // 4. Define spec (merged from all resources + traits)
        spec: {
            container: { name: "app", image: "myimage:v1", ... }
            scaling:   { count: 3 }
            expose:    { ports: http: { targetPort: 8080 }, type: "ClusterIP" }
        }
    }
}
```

### Workload Types

OPM uses a label to determine the Kubernetes resource type:

| Label Value | Kubernetes Resource | Use Case |
|-------------|---------------------|----------|
| `stateless` | Deployment | Web apps, APIs, stateless services |
| `stateful` | StatefulSet | Databases, message queues, ordered services |
| `daemon` | DaemonSet | Log collectors, monitoring agents, per-node services |
| `task` | Job | Migrations, batch processing, one-off tasks |
| `scheduled-task` | CronJob | Backups, periodic cleanup, scheduled reports |

---

## Resources

- [OPM Documentation](https://openplatformmodel.org/docs)
- [CUE Language](https://cuelang.org)
- [OPM Catalog](https://github.com/open-platform-model/catalog) (resource/trait/blueprint definitions)

---

## Feedback

Found an issue or have a suggestion? [Open an issue](https://github.com/open-platform-model/cli/issues) or submit a PR.
