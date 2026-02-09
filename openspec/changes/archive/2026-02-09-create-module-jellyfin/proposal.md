# Proposal

## Why

The `testing/` directory contains example OPM modules (`blog`, `multi-tier-module`) that demonstrate stateless and multi-workload patterns. There is no example of a real-world **single-container stateful media application** — a common deployment pattern for self-hosted software. A Jellyfin module fills this gap, exercising stateful workloads, persistent storage, network exposure, health checks, and the LinuxServer.io container convention (PUID/PGID/TZ), while serving as a practical reference module that people could actually deploy.

## What Changes

- Add a new OPM module at `testing/jellyfin/` that deploys the [linuxserver/jellyfin](https://docs.linuxserver.io/images/docker-jellyfin/) media server image
- Single stateful component with persistent config storage and media volume mounts
- HTTP health check probe against the Jellyfin web UI (port 8096)
- Network exposure for the web UI via the `#Expose` trait
- Configurable media library paths, LinuxServer environment (PUID/PGID/TZ), and published server URL
- Sensible defaults that produce a valid, deployable module out of the box

### Out of scope

- **Hardware acceleration / GPU passthrough** — OPM has no device passthrough resource or trait; this is deferred until the type system supports it
- **UDP discovery ports** (7359, 1900) — optional Jellyfin features that require host networking or NodePort; deferred to keep the initial module simple
- **HTTPS (port 8920)** — requires user-provided certificates; deferred
- **Multi-instance / HA deployment** — Jellyfin is a single-instance application

## Capabilities

### New Capabilities

- `jellyfin-module`: The complete Jellyfin OPM module definition — metadata, config schema, stateful component, storage, networking, health checks, and default values

### Modified Capabilities

_(none — this is a new module in `testing/`, no existing specs are affected)_

## Impact

- **New files**: `testing/jellyfin/{cue.mod/module.cue, module.cue, components.cue, values.cue}`
- **Dependencies**: `opmodel.dev/core@v0`, `opmodel.dev/resources/{workload,storage}@v0`, `opmodel.dev/traits/{workload,network}@v0`, `opmodel.dev/schemas@v0`
- **CLI**: No CLI code changes — this is a CUE module, not a Go change
- **SemVer**: N/A (test fixture, not a CLI release)
- **Validation**: Module must pass `cue vet ./...` against the OPM schema registry
