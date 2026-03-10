# Quickstart

Get up and running with OPM in a few minutes using the example release files.

## Prerequisites

- **Go 1.23+** - for building the CLI
- **[Task](https://taskfile.dev)** - task runner (`go install github.com/go-task/task/v3/cmd/task@latest`)
- **[CUE](https://cuelang.org)** - for module publishing (`go install cuelang.org/go/cmd/cue@latest`)
- **Docker** - for running the local OCI registry
- **kind** - local Kubernetes (`go install sigs.k8s.io/kind@latest`)
- **kubectl** - Kubernetes CLI
- **jq** - optional, for `task registry:status`

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

This starts a local OCI registry at `localhost:5000`.

## Configure Environment

Point CUE and OPM at the local registry:

```bash
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
export OPM_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
```

## Publish CUE Modules

Publish the catalog modules to your local registry:

```bash
cd catalog
task publish:all:local
```

Verify with:

```bash
task registry:status
```

## Build and Install the CLI

```bash
cd cli
task build && task install
```

This builds `./bin/opm` and installs `opm` into `$GOPATH/bin`.

## Initialize Configuration

```bash
opm config init
```

This creates `~/.opm/config.cue` with default settings.

## Create a Dev Cluster

```bash
task cluster:create
```

This creates a local kind cluster named `opm-dev`.

## Release-Based Workflow

This quickstart uses the example release files under `examples/releases/`.

- Jellyfin release: `examples/releases/jellyfin/release.cue`
- Minecraft release: `examples/releases/minecraft/release.cue`
- Referenced example modules:
  - `examples/modules/jellyfin`
  - `examples/modules/mc_java`

`opm module` is still useful for authoring and direct module workflows, but for this quickstart we start from release definitions.

## Example: Jellyfin Release

The Jellyfin example release lives in `examples/releases/jellyfin/release.cue` and references the module in `examples/modules/jellyfin`.

### Build

```bash
opm release build ./examples/releases/jellyfin/release.cue
```

Build with explicit values:

```bash
opm release build ./examples/releases/jellyfin/release.cue -f ./examples/releases/jellyfin/values.cue
```

### Apply

```bash
opm release apply ./examples/releases/jellyfin/release.cue --create-namespace
```

Apply with explicit values:

```bash
opm release apply ./examples/releases/jellyfin/release.cue -f ./examples/releases/jellyfin/values.cue --create-namespace
```

### Status

The Jellyfin release file uses release name `jf` in namespace `jellyfin`.

```bash
opm release status jf -n jellyfin
```

### Tree

```bash
opm release tree jf -n jellyfin
```

### Events

```bash
opm release events jf -n jellyfin
```

### Delete

```bash
opm release delete jf -n jellyfin --force
```

## Example: Minecraft Release

The Minecraft example release lives in `examples/releases/minecraft/release.cue` and references the module in `examples/modules/mc_java`.

### Build

```bash
opm release build ./examples/releases/minecraft/release.cue
```

Build with one of the example values files:

```bash
opm release build ./examples/releases/minecraft/release.cue -f ./examples/releases/minecraft/values_forge.cue
```

Other example values files:

- `examples/releases/minecraft/values.cue`
- `examples/releases/minecraft/values_fabric_modrinth.cue`
- `examples/releases/minecraft/values_paper_restic.cue`

### Apply

```bash
opm release apply ./examples/releases/minecraft/release.cue -f ./examples/releases/minecraft/values_forge.cue
```

### Status

The Minecraft release file uses release name `minecraft` in namespace `default`.

```bash
opm release status minecraft -n default
```

### Delete

```bash
opm release delete minecraft -n default --force
```

## Notes

- `opm release` is the canonical workflow when you already have a release definition.
- `opm module` remains the canonical workflow when you are starting from module source.
- `opm mod` still works as an alias for `opm module`.

## Cleanup

Delete the cluster:

```bash
task cluster:delete
```

Stop the registry if needed:

```bash
cd catalog
task registry:stop
```

## Next Steps

- Run `opm release --help` and `opm module --help`
- See `README.md` for command overview
- See `AGENTS.md` for architecture and development guidance
