# Quickstart

Get up and running with OPM in 5 minutes.

## Prerequisites

- **Go 1.23+** — for building the CLI
- **[Task](https://taskfile.dev)** — task runner (`go install github.com/go-task/task/v3/cmd/task@latest`)
- **kind** — local Kubernetes (`go install sigs.k8s.io/kind@latest`)
- **kubectl** — Kubernetes CLI
- **OPM OCI registry** — Relies on ´registry.opmodel.dev´
- **Access to OCI registry** - Must set CUE_REGISTRY and OPM_REGISTRY

```bash
export OPM_REGISTRY='opmodel.dev=registry.opmodel.dev,registry.cue.works'
export CUE_REGISTRY='opmodel.dev=registry.opmodel.dev,registry.cue.works'
```

## Build & Install

```bash
task build && task install
```

This builds the `opm` binary to `./bin/opm` and installs it to `$GOPATH/bin/opm`.

## Create a Dev Cluster

```bash
task cluster:create
```

Creates a kind cluster named `opm-dev` with Kubernetes v1.34.0.

## Example: Jellyfin Module

The `examples/jellyfin` module demonstrates a stateful single-container application with persistent storage.

### Build (render manifests locally)

```bash
opm mod build ./examples/jellyfin -n default
```

Renders Kubernetes manifests from the module definition.

### Apply (deploy to cluster)

```bash
opm mod apply ./examples/jellyfin -n default
```

Applies resources to the cluster using server-side apply.

### Diff (compare local vs. live state)

```bash
opm mod diff ./examples/jellyfin -n default
```

Shows semantic differences between your local definition and what's running in the cluster.

### Status (check resource health)

```bash
opm mod status --name jellyfin -n default
```

Displays health and readiness of all module resources.

### Delete (remove from cluster)

```bash
opm mod delete --name jellyfin -n default --force
```

Deletes all module resources in reverse weight order.

## Other Examples

**Blog** — two-tier stateless application (web frontend + API backend):

```bash
opm mod build ./examples/blog -n default
opm mod apply ./examples/blog -n default
opm mod status --name Blog -n default
opm mod delete --name Blog -n default --force
```

**Multi-tier** — demonstrates all four workload types (stateful, daemon, task, scheduled-task):

```bash
opm mod build ./examples/multi-tier-module -n default
opm mod apply ./examples/multi-tier-module -n default
opm mod status --name multi-tier-module -n default
opm mod delete --name multi-tier-module -n default --force
```

## Cleanup

```bash
task cluster:delete
```

Deletes the `opm-dev` kind cluster.

## Next Steps

- Run `opm --help` to explore all commands
- See [README.md](./README.md) for detailed documentation
- Check [AGENTS.md](./AGENTS.md) for CLI architecture and development guidelines
