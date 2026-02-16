# Multi-Tier Module — All Workload Types

**Complexity:** Advanced  
**Workload Types:** `stateful`, `daemon`, `task`, `scheduled-task`

A comprehensive example demonstrating all four OPM workload types with advanced traits including init containers, update strategies, and job configurations.

## What This Example Demonstrates

### Core Concepts
- **All 4 workload types** — StatefulSet, DaemonSet, Job, CronJob in a single module
- **`#Scaling` trait** — Horizontal scaling for StatefulSet
- **`#UpdateStrategy` trait** — RollingUpdate with maxUnavailable and partition
- **`#HealthCheck` trait** — Both `exec` and `httpGet` probe styles
- **`#InitContainers` trait** — Pre-start setup tasks
- **`#JobConfig` trait** — Job-specific configuration (completions, parallelism, backoff)
- **`#CronJobConfig` trait** — Scheduled job configuration (cron schedule, concurrency policy)
- **`#RestartPolicy` trait** — Different restart policies per workload type

### OPM Patterns
- Mix of workload types in one module
- Volume mounts with `readOnly` flag
- Exec-based health checks (shell commands)
- Init containers for dependency checks and setup
- Job completion tracking and retry logic
- Cron scheduling with history limits

## Architecture

```
┌────────────────────────────┐
│ database (StatefulSet)     │   Persistent PostgreSQL
│   replicas: 3              │   with rolling updates
│   /var/lib/postgresql/data │
└────────────────────────────┘

┌────────────────────────────┐
│ log-agent (DaemonSet)      │   Per-node log collector
│   Runs on every node       │   with host volume mounts
│   /var/log → read-only     │
└────────────────────────────┘

┌────────────────────────────┐
│ setup-job (Job)            │   One-time migration task
│   completions: 1           │   with retry logic
│   backoffLimit: 4          │
└────────────────────────────┘

┌────────────────────────────┐
│ backup-job (CronJob)       │   Scheduled backup task
│   schedule: "0 2 * * *"    │   Runs daily at 2 AM
│   concurrency: Forbid      │
└────────────────────────────┘
```

## Configuration Schema

| Field | Type | Constraint | Default | Description |
|-------|------|------------|---------|-------------|
| `database.image` | string | - | `"postgres:15-alpine"` | PostgreSQL container image |
| `database.scaling` | int | >= 1 | `3` | Number of database replicas |
| `logAgent.image` | string | - | `"fluent/fluent-bit:2.0"` | Log agent container image |
| `setupJob.image` | string | - | `"migrate/migrate:v4.15"` | Migration tool image |
| `backupJob.image` | string | - | `"postgres:15-alpine"` | Backup script image |
| `backupJob.schedule` | string | - | `"0 2 * * *"` | Cron schedule (daily 2 AM) |

## Rendered Kubernetes Resources

| Resource | Name | Type | Notes |
|----------|------|------|-------|
| StatefulSet | `database` | `apps/v1` | 3 replicas, rolling update |
| DaemonSet | `log-agent` | `apps/v1` | Runs on every node |
| Job | `setup-job` | `batch/v1` | Run-to-completion, max 4 retries |
| CronJob | `backup-job` | `batch/v1` | Daily at 2 AM, forbid concurrent runs |
| PersistentVolumeClaim | `database-data-0` | `v1` | 10Gi for first replica |
| PersistentVolumeClaim | `database-data-1` | `v1` | 10Gi for second replica |
| PersistentVolumeClaim | `database-data-2` | `v1` | 10Gi for third replica |

**Total:** 7 Kubernetes resources

## Usage

### Build (render to YAML)

```bash
# Render to stdout
opm mod build ./examples/multi-tier-module

# Render to split files
opm mod build --split ./examples/multi-tier-module
```

### Apply to Kubernetes

```bash
# Apply all components
opm mod apply ./examples/multi-tier-module

# Apply to specific namespace
opm mod apply --namespace production ./examples/multi-tier-module
```

### Check Status

```bash
# Watch all components
opm mod status --watch ./examples/multi-tier-module

# Check specific workload
kubectl get statefulset database
kubectl get daemonset log-agent
kubectl get job setup-job
kubectl get cronjob backup-job
```

## Files

```
multi-tier-module/
├── cue.mod/module.cue    # CUE dependencies
├── module.cue            # Module metadata and config schema
├── components.cue        # All component definitions (4 workload types)
└── values.cue            # Default configuration values
```

## Key Code Snippets

### StatefulSet with Rolling Update Strategy

```cue
database: {
    resources_workload.#Container
    resources_storage.#Volumes
    traits_workload.#Scaling
    traits_workload.#UpdateStrategy
    traits_workload.#HealthCheck
    traits_workload.#InitContainers

    metadata: labels: "core.opmodel.dev/workload-type": "stateful"

    spec: {
        scaling: count: #config.database.scaling

        updateStrategy: {
            type: "RollingUpdate"
            rollingUpdate: {
                maxUnavailable: 1   // Only 1 pod down at a time
                partition:      0   // Update all pods (0 = no partition)
            }
        }

        healthCheck: {
            livenessProbe: {
                exec: command: ["pg_isready", "-U", "postgres"]
                initialDelaySeconds: 30
                periodSeconds:       10
            }
            readinessProbe: {
                exec: command: ["pg_isready", "-U", "postgres"]
                initialDelaySeconds: 5
                periodSeconds:       5
            }
        }
    }
}
```

### DaemonSet with Host Volume Mounts

```cue
logAgent: {
    resources_workload.#Container
    resources_storage.#Volumes
    traits_workload.#RestartPolicy
    traits_workload.#UpdateStrategy
    traits_workload.#HealthCheck

    metadata: labels: "core.opmodel.dev/workload-type": "daemon"

    spec: {
        restartPolicy: "Always"

        volumes: {
            varlog: {
                name: "varlog"
                emptyDir: {}  // In real use: hostPath for /var/log
            }
        }

        container: {
            volumeMounts: varlog: {
                name:      "varlog"
                mountPath: "/var/log"
                readOnly:  true  // Read-only mount for security
            }
        }
    }
}
```

### Job with Init Container and Retry Logic

```cue
setupJob: {
    resources_workload.#Container
    traits_workload.#JobConfig
    traits_workload.#RestartPolicy
    traits_workload.#InitContainers

    metadata: labels: "core.opmodel.dev/workload-type": "task"

    spec: {
        jobConfig: {
            completions:            1    // Run once successfully
            parallelism:            1    // Single pod
            backoffLimit:           4    // Retry up to 4 times
            activeDeadlineSeconds:  300  // Timeout after 5 minutes
            ttlSecondsAfterFinished: 100 // Delete job 100s after completion
        }

        restartPolicy: "OnFailure"

        initContainers: [{
            name:  "check-database"
            image: "busybox:1.35"
            command: [
                "sh", "-c",
                "until nslookup database; do echo waiting for database; sleep 2; done",
            ]
        }]
    }
}
```

### CronJob with Concurrency Control

```cue
backupJob: {
    resources_workload.#Container
    traits_workload.#CronJobConfig
    traits_workload.#RestartPolicy
    traits_workload.#InitContainers

    metadata: labels: "core.opmodel.dev/workload-type": "scheduled-task"

    spec: {
        cronJobConfig: {
            scheduleCron:              #config.backupJob.schedule  // "0 2 * * *"
            concurrencyPolicy:         "Forbid"  // Don't run if previous job still active
            startingDeadlineSeconds:   300       // Start within 5 min of schedule
            successfulJobsHistoryLimit: 3        // Keep last 3 successful jobs
            failedJobsHistoryLimit:    1         // Keep last 1 failed job
        }

        restartPolicy: "OnFailure"

        initContainers: [{
            name:  "pre-backup-check"
            image: "postgres:15-alpine"
            command: ["pg_isready", "-h", "database", "-U", "postgres"]
        }]
    }
}
```

## Workload Type Details

### StatefulSet (`workload-type: "stateful"`)

**Use cases:** Databases, message queues, ordered services

**Behavior:**
- Pods start sequentially: `database-0`, then `database-1`, then `database-2`
- Each pod gets a persistent volume claim (PVC)
- Pods retain identity across restarts (stable hostname)
- Updates roll through pods in reverse order (highest index first)

**Key traits:** `#Scaling`, `#UpdateStrategy`, `#HealthCheck`, `#InitContainers`, `#RestartPolicy`

### DaemonSet (`workload-type: "daemon"`)

**Use cases:** Log collectors, monitoring agents, per-node services

**Behavior:**
- One pod per node in the cluster
- New pods automatically scheduled on new nodes
- Pods deleted when nodes are removed

**Key traits:** `#UpdateStrategy`, `#HealthCheck`, `#RestartPolicy`

### Job (`workload-type: "task"`)

**Use cases:** Migrations, batch processing, one-off tasks

**Behavior:**
- Runs to completion (not long-running)
- Retries on failure (up to `backoffLimit`)
- Deleted after TTL (if `ttlSecondsAfterFinished` set)
- Can run multiple completions in parallel

**Key traits:** `#JobConfig`, `#RestartPolicy`, `#InitContainers`

### CronJob (`workload-type: "scheduled-task"`)

**Use cases:** Backups, periodic cleanup, scheduled reports

**Behavior:**
- Creates a Job on schedule (cron syntax)
- Can forbid/allow/replace concurrent executions
- Manages job history (keeps last N successful/failed jobs)
- Supports starting deadline (skip if too late)

**Key traits:** `#CronJobConfig`, `#RestartPolicy`, `#InitContainers`

## Health Check Probe Types

### Exec Probe (shell command)

```cue
healthCheck: {
    livenessProbe: {
        exec: command: ["pg_isready", "-U", "postgres"]
        initialDelaySeconds: 30
        periodSeconds:       10
    }
}
```

Kubernetes runs the command inside the container. Exit code 0 = healthy.

### HTTP Probe (GET request)

```cue
healthCheck: {
    readinessProbe: {
        httpGet: {
            path: "/health"
            port: 8080
        }
        initialDelaySeconds: 10
        periodSeconds:       5
    }
}
```

Kubernetes sends `GET /health` to port 8080. HTTP 200-399 = healthy.

## Next Steps

- **Add Ingress routing:** Upcoming `webapp-ingress/` example
- **Add ConfigMaps/Secrets:** Upcoming `app-config/` example
- **Add HPA autoscaling:** Upcoming `webapp-ingress/` example
- **Simplify with Blueprints:** Upcoming `blueprint-module/` example

## Related Examples

- [blog/](../blog/) — Simple multi-component stateless app
- [jellyfin/](../jellyfin/) — Stateful workload with persistent storage
