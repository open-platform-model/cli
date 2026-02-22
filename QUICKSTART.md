# Quickstart

Get up and running with OPM in 5 minutes.

## Prerequisites

- **Go 1.23+** — for building the CLI
- **[Task](https://taskfile.dev)** — task runner (`go install github.com/go-task/task/v3/cmd/task@latest`)
- **[CUE](https://cuelang.org)** — for module publishing (`go install cuelang.org/go/cmd/cue@latest`)
- **Docker** — for running the local OCI registry
- **kind** — local Kubernetes (`go install sigs.k8s.io/kind@latest`)
- **kubectl** — Kubernetes CLI
- **jq** — JSON processor (optional, for `task registry:status`)

## Clone Repositories

Clone both the CLI and catalog repositories:

```bash
git clone https://github.com/open-platform-model/cli.git
git clone https://github.com/open-platform-model/catalog.git
```

Expected layout:

```text
├── cli/        # CLI source code
└── catalog/    # CUE module definitions
```

## Start the OCI Registry

The catalog repository includes tasks for managing a local Docker OCI registry.

```bash
cd catalog
task registry:start
```

This creates and starts a Docker container running a local OCI registry at `localhost:5000`.

**Registry tasks available:**

| Task | Description |
|------|-------------|
| `task registry:start` | Start the registry |
| `task registry:stop` | Stop the registry (preserves data) |
| `task registry:status` | Show status and published modules |
| `task registry:health` | Check registry health |
| `task registry:list` | List all published modules |
| `task registry:cleanup` | Remove registry and all data |

## Configure Environment

Set the registry environment variables to use the local registry:

```bash
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
export OPM_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
```

Add these to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) for persistence.

## Publish CUE Modules

Publish the OPM CUE modules to your local registry:

```bash
cd catalog
task publish:all:local
```

This publishes all catalog modules (`core`, `schemas`, `resources`, `traits`, etc.) in dependency order.

Verify with:

```bash
task registry:status
```

## Build & Install

Build and install the CLI:

```bash
cd cli
task build && task install
```

This builds the `opm` binary to `./bin/opm` and installs it to `$GOPATH/bin/opm`.

## Initialize Configuration

Initialize the OPM configuration for first-time use:

```bash
opm config init
```

This creates `~/.opm/config.cue` with default settings.

## Create a Dev Cluster

```bash
task cluster:create
```

Creates a kind cluster named `opm-dev` with Kubernetes v1.34.0.

## Example: Jellyfin Module

The `examples/jellyfin` module demonstrates a stateful single-container application with persistent storage.

### Build (render manifests locally)

Without `--release-name` (defaults to the module name):

```bash
opm mod build ./examples/jellyfin -n default
```

With a custom release name:

```bash
opm mod build ./examples/jellyfin -n default --release-name jellyfin-media
```

Renders Kubernetes manifests from the module definition.

### Apply (deploy to cluster)

Without `--release-name` (defaults to the module name):

```bash
opm mod apply ./examples/jellyfin -n default
```

With a custom release name:

```bash
opm mod apply ./examples/jellyfin -n default --release-name jellyfin-media
```

Applies resources to the cluster using server-side apply.

### Diff (compare local vs. live state)

```bash
opm mod diff ./examples/jellyfin -n default
```

Shows semantic differences between your local definition and what's running in the cluster.

> **Note:** The diff command is currently broken and shows excessive differences.

### Status (check resource health)

```bash
opm mod status --name jellyfin -n default
```

Displays health and readiness of all module resources.

> **Note:** The status command is not particularly useful at the moment.

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

With a custom release name:

```bash
opm mod build ./examples/blog -n default --release-name my-blog
opm mod apply ./examples/blog -n default --release-name my-blog
opm mod status --name my-blog -n default
opm mod delete --name my-blog -n default --force
```

**Multi-tier** — demonstrates all four workload types (stateful, daemon, task, scheduled-task):

```bash
opm mod build ./examples/multi-tier-module -n default
opm mod apply ./examples/multi-tier-module -n default
opm mod status --name multi-tier-module -n default
opm mod delete --name multi-tier-module -n default --force
```

With a custom release name:

```bash
opm mod build ./examples/multi-tier-module -n default --release-name my-app
opm mod apply ./examples/multi-tier-module -n default --release-name my-app
opm mod status --name my-app -n default
opm mod delete --name my-app -n default --force
```

## Cleanup

```bash
task cluster:delete
```

Deletes the `opm-dev` kind cluster.

To also stop the registry:

```bash
cd catalog
task registry:stop
```

## Next Steps

- Run `opm --help` to explore all commands
- See [README.md](./README.md) for detailed documentation
- Check [AGENTS.md](./AGENTS.md) for CLI architecture and development guidelines
