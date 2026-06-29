# Quickstart

Get up and running with OPM in a few minutes. The catalog and core modules
are published to a public registry, so you only need the CLI and a CUE
toolchain to start authoring and rendering your first module.

## Prerequisites

- **Go 1.25+** - for building the CLI
- **[Task](https://taskfile.dev)** - task runner (`go install github.com/go-task/task/v3/cmd/task@latest`)
- **[CUE](https://cuelang.org)** - CUE toolchain (`go install cuelang.org/go/cmd/cue@latest`)

The "Deploy to a Cluster" section additionally needs:

- **kind** - local Kubernetes (`go install sigs.k8s.io/kind@latest`)
- **kubectl** - Kubernetes CLI

## Build and Install the CLI

Clone the CLI repo and install the binary:

```bash
git clone https://github.com/open-platform-model/cli.git
cd cli
task build && task install
```

This builds `./bin/opm` and installs `opm` into `$GOPATH/bin`.

## Configure the Registry

OPM resolves catalog schemas and modules via the CUE module proxy. Point
both CUE and OPM at the public OPM registry on GHCR:

```bash
export CUE_REGISTRY='testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY='testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
```

`opmodel.dev` resolves the stable catalog and module releases.
`testing.opmodel.dev` is reserved for prerelease testing artifacts and can
be omitted if you do not need it. `registry.cue.works` is the upstream
public CUE registry used as a fallback for non-OPM modules.

Add these exports to your shell profile if you plan to use OPM regularly.

## Initialize Configuration

```bash
opm config init
```

This creates `~/.opm/config.cue` with default settings.

Next you need to tidy to pull all dependencies down.

```bash
cd ~/.opm/
cue mod tidy
```

## Create Your First Module

The fastest way to learn OPM is to scaffold a module, render it, and read
the generated manifests.

### Scaffold

```bash
opm module init my-app
```

This creates `./my-app/` from the `standard` template:

```text
my-app/
  cue.mod/module.cue        CUE module metadata
  module.cue                Module definition (metadata + #config + debugValues)
  components.cue            Component definitions
```

Other templates are available via `--template`:

- `simple` - single-file module, good for learning
- `standard` - separated concerns (default)
- `advanced` - multi-package architecture for complex platforms

### Inspect

Open `my-app/module.cue` and `my-app/components.cue`. The scaffold defines
a minimal workload, a default `#config` schema, and `debugValues` that
populate that schema with placeholder values.

### Build

Render the module to Kubernetes manifests using its `debugValues`:

```bash
cd my-app
opm module build
```

`opm module build` synthesizes a `#ModuleRelease` around the module so you
do not need an `instance.cue` while iterating. The rendered YAML is written
to stdout by default.

Useful variants:

```bash
# Run from anywhere by passing a module directory
opm module build ./my-app

# Override the synthetic release name
opm module build ./my-app --name my-app-debug

# Provide explicit values instead of debugValues
opm module build ./my-app -f ./my-overrides.cue

# Write each resource to its own file under ./manifests
opm module build ./my-app --split --out-dir ./manifests
```

`opm instance build` accepts the same module-directory form, so
`opm instance build ./my-app` is equivalent for symmetry with the release
workflow.

`opm mod` is an alias for `opm module`, so all of the commands above also
work as `opm mod init`, `opm mod build`, etc.

## Working from a Release File

When you already have a release definition, `opm instance` is the canonical
workflow. The example instance files under `examples/instances/` are small
release definitions that import published modules straight from the
public catalog (`opmodel.dev/modules/<name>@v1`) — no local module sources
required, no `task publish`.

Available examples:

- `examples/instances/jellyfin/` — single-container stateful app (storage,
  optional gateway route, optional K8up backup). Used throughout this
  guide.
- `examples/instances/garage/` — stateless S3-compatible object store;
  showcases the "required secret" pattern (`adminToken`, `rpcSecret`).
- `examples/instances/mc_java_fleet/` — multi-instance Minecraft fleet
  with a shared mc-router. Has a default single-server `values.cue` and
  a multi-server `values_multi.cue`.

### Build

```bash
opm instance build ./examples/instances/jellyfin/instance.cue
```

The release directory contains a sibling `values.cue` which is loaded
automatically. To use a different values file, pass it explicitly:

```bash
opm instance build ./examples/instances/mc_java_fleet/instance.cue \
  -f ./examples/instances/mc_java_fleet/values_multi.cue
```

A release file pulling a public module looks like the `jellyfin` example:

```cue
package jellyfin

import (
    mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
    m  "opmodel.dev/modules/jellyfin@v1"
)

mr.#ModuleRelease

metadata: {
    name:      "jellyfin"
    namespace: "default"
}

#module: m
```

With the public registry configured, `opm instance build` and
`opm instance apply` resolve `opmodel.dev/modules/jellyfin@v1` from
`ghcr.io/open-platform-model` automatically.

## Deploy to a Cluster

The remaining steps require a Kubernetes cluster. The fastest path is a
local `kind` cluster, which the CLI repo provides a Task for:

```bash
task cluster:create
```

This creates a local kind cluster named `opm-dev`.

### Apply

```bash
opm instance apply ./examples/instances/jellyfin/instance.cue --create-namespace
```

The `garage` and `mc_java_fleet` examples follow the same pattern — point
`opm instance apply` at their `instance.cue`. Note that `garage` requires
you to replace the `adminToken` and `rpcSecret` placeholders in
`values.cue` before applying.

### Inspect

The `jellyfin` example release uses release name `jellyfin` in namespace
`default`.

```bash
opm instance status jellyfin -n default
opm instance tree   jellyfin -n default
opm instance events jellyfin -n default
```

### Delete

```bash
opm instance delete jellyfin -n default --force
```

## Cleanup

Delete the kind cluster:

```bash
task cluster:delete
```

## Next Steps

- Run `opm instance --help` and `opm module --help`
- See `README.md` for command overview
- See `AGENTS.md` for architecture and development guidance
- Browse the public module catalog at
  <https://github.com/open-platform-model/modules>
