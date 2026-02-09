# Design

## Context

The `testing/` directory has two example modules:

- **blog** — stateless, two components (web + api), demonstrates `#Container`, `#Replicas`, `#Expose`
- **multi-tier-module** — all workload types (stateful, daemon, task, scheduled-task), demonstrates `#Volumes`, `#Scaling`, `#HealthCheck`, `#RestartPolicy`, `#UpdateStrategy`, `#InitContainers`, `#JobConfig`, `#CronJobConfig`

Neither demonstrates a real-world single-container stateful application — the most common pattern for self-hosted software (media servers, wikis, home automation). Jellyfin is a good fit: it's a single process with persistent state, media volume mounts, health-checkable HTTP UI, and a well-documented container image from LinuxServer.io.

The module uses the same three-file convention (`module.cue`, `components.cue`, `values.cue`) established by the existing examples.

## Goals / Non-Goals

**Goals:**

- Produce a valid, deployable OPM module that passes `cue vet` against the schema registry
- Demonstrate the stateful single-container pattern with realistic defaults
- Exercise storage (`#Volumes` with `persistentClaim`), networking (`#Expose`), and health checks (`#HealthCheck`) together
- Keep the config schema simple enough to serve as a reference for module authors

**Non-Goals:**

- GPU/device passthrough (no OPM type support yet)
- UDP service discovery ports (adds complexity without demonstrating new OPM patterns)
- HTTPS termination (certificate management is out of scope)
- Ingress/routing (`#HttpRoute`) — exposure via `#Expose` is sufficient for a test module

## Decisions

### 1. Workload type: `stateful`

**Decision**: Label the component as `"stateful"` workload type.

**Rationale**: Jellyfin writes persistent config/metadata to `/config` (can exceed 50GB). A StatefulSet provides stable network identity and ordered PVC lifecycle. Even though Jellyfin runs a single replica, `stateful` is semantically correct and the Kubernetes provider will render a StatefulSet — giving us proper PVC management.

**Alternative considered**: `stateless` with an external PVC — possible but misrepresents the workload's nature and loses StatefulSet PVC binding.

### 2. Media volumes: struct-based with named keys (not a list)

**Decision**: Model media libraries as a CUE struct with string keys:

```text
media: [Name=string]: {
    mountPath: string
}
```

Each key becomes a volume mount name, and the value specifies the mount path inside the container.

**Rationale**: This follows the same pattern used by `ports`, `env`, and `volumeMounts` throughout OPM schemas — struct-keyed, not list-based. It's idiomatic CUE (keys are merge-friendly), and users can add/remove libraries by adding/removing struct fields without index management.

**Alternative considered**: Fixed fields (`tvshows`, `movies`, `music`) — too rigid, different users have different library structures. A list (`[...{name, path}]`) — less idiomatic in CUE and harder to override specific entries.

### 3. Config volume: dedicated PVC with configurable size

**Decision**: The `/config` directory gets its own `persistentClaim` volume with a configurable size (default `10Gi`).

**Rationale**: Jellyfin's config directory stores metadata, thumbnails, and transcoding cache. LinuxServer docs warn it can grow to 50GB+ for large collections. A PVC is essential — ephemeral storage would lose all library metadata on restart.

### 4. Media volumes modeled as `emptyDir` in the OPM module

**Decision**: Media library volumes use `emptyDir` (not `persistentClaim`) in the OPM definition.

**Rationale**: In real deployments, media volumes point to pre-existing NFS mounts, host paths, or external storage — not dynamically provisioned PVCs. The OPM module defines the mount points; the actual volume backing is a platform/deployment concern. Using `emptyDir` keeps the module portable and lets operators override with their actual storage at release time. The multi-tier-module's database already demonstrates the `persistentClaim` pattern.

**Alternative considered**: `persistentClaim` for media — would force dynamic provisioning for what are typically pre-existing mounts. Host path volumes — not portable and not modeled in OPM schemas.

### 5. LinuxServer PUID/PGID as environment variables

**Decision**: Expose `puid`, `pgid`, and `timezone` as top-level config fields, mapped to the container's `PUID`, `PGID`, and `TZ` environment variables.

**Rationale**: These are LinuxServer.io image conventions, not Kubernetes security context fields. `PUID`/`PGID` trigger an internal usermod inside the container — they don't map to `runAsUser`/`runAsGroup`. Modeling them as env vars is honest about what they are.

### 6. Health check: HTTP GET on port 8096

**Decision**: Liveness and readiness probes both use `httpGet` on `/health` at port 8096.

**Rationale**: Jellyfin exposes a `/health` endpoint that returns 200 when the server is ready. This is more reliable than a TCP socket check because it validates the application is actually serving, not just that the port is open. Initial delay of 30s for liveness (Jellyfin startup can be slow with large libraries) and 10s for readiness.

### 7. Single component, no init containers or sidecars

**Decision**: One component named `jellyfin`. No init containers, no sidecars.

**Rationale**: Jellyfin is self-contained. There's no database to wait for, no config to pre-generate. Adding init containers would add complexity without demonstrating a pattern not already covered by multi-tier-module.

## Risks / Trade-offs

- **[No GPU transcoding]** → Accept for v1. Document as a known limitation. Users needing transcoding can override at the platform/provider level. A future `#DevicePassthrough` trait would address this properly.
- **[emptyDir for media]** → Media is lost on pod deletion if not overridden. This is intentional — the module defines mount points, operators provide backing storage. The values.cue comments should make this clear.
- **[Single replica only]** → Jellyfin doesn't support horizontal scaling. The `#Scaling` trait with `count: 1` makes this explicit. Attempting to scale beyond 1 would cause data corruption.
- **[No ingress]** → Users wanting external access need to add `#HttpRoute` or configure ingress separately. `#Expose` with ClusterIP is sufficient for the module's scope.
