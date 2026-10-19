# OPM CLI

> **WARNING: UNDER HEAVY DEVELOPMENT** - This project is actively being developed and APIs may change frequently.

Command-line interface for the Open Platform Model (OPM). Build, validate, deploy, and inspect portable application releases defined with CUE.

## Quick Start

```bash
# Build the CLI
task build

# Initialize a new module (fetches the standard template and re-identifies it)
./bin/opm mod init example.com/modules/my_module@v0

# Validate a module
./bin/opm module vet ./my_module

# Create an instance package for a published module
./bin/opm instance init web opmodel.dev/modules/web_app -n demo

# Validate the instance
./bin/opm instance vet ./web/instance.cue

# Render the instance
./bin/opm instance build ./web/instance.cue

# Apply the instance
./bin/opm instance apply ./web/instance.cue
```

## Features

- **Type-safe definitions** using CUE
- **Kubernetes-native** resource management
- **Portable blueprints** across providers
- **OCI-based distribution** for modules and definitions
- **Interactive CLI** with rich terminal output

## Commands

### Module Operations (`opm module`)

`opm mod` remains available as a compatibility alias.

Use `opm module` when you are starting from module source. For rendering, deploying, or inspecting instances, use `opm instance`.

| Command | Description |
|---------|-------------|
| `module init` | Create a new module from a template |
| `module vet` | Validate a module without rendering manifests |
| `module eval` | Print the evaluated module as formatted CUE; `-e <path>` prints one value (`#config`, `metadata.name`) |
| `module build` | Render a module directory or a published module to manifests through a synthetic instance (`debugValues` or `-f` values) |
| `module apply` | Deploy a module directory or a published module to a cluster through a synthetic instance |
| `module tidy` | Resolve, pin and prune the module's CUE dependencies as `cue mod tidy` does, without the `cue` binary; `--check` fails without writing |
| `module version set` | Set the version the module declares in `identity/identity.cue` — surgical, idempotent, offline |
| `module publish` | Publish the module from its committed source, at the coordinates it declares |

### Catalog Operations (`opm catalog`)

Use `opm catalog` when you are starting from catalog source: tidy its dependencies, declare a version, or publish a release from the committed tree.

| Command | Description |
|---------|-------------|
| `catalog tidy` | Resolve, pin and prune the catalog's CUE dependencies as `cue mod tidy` does, without the `cue` binary; `--check` fails without writing |
| `catalog version set` | Set the version the catalog declares in `identity/identity.cue` — surgical, idempotent, offline |
| `catalog publish` | Publish the catalog from its committed source, at the coordinates it declares |

### Instance Operations (`opm instance`)

<!-- Renamed from `opm release` / `opm rel` (enhancement 0002). The old `release`/`rel` verb is removed — no back-compat alias. -->

`opm i`, `opm ins`, and `opm inst` are the short aliases.

Use `opm instance` when you are starting from an instance file or when you want to inspect, list, or delete deployed instances.

| Command | Description |
|---------|-------------|
| `instance init` | Create a standalone instance package (`cue.mod/module.cue`, `instance.cue`, `values.cue`) for a published module, pinned and ready to build |
| `instance vet` | Validate an instance file without generating manifests |
| `instance build` | Render an instance file or instance package directory to manifests (a module directory is refused: use `module build`) |
| `instance apply` | Deploy an instance file to a cluster (`--wait` blocks until every resource is healthy; for a CLI-managed instance the prune keeps PersistentVolumeClaims unless `--delete-data`) |
| `instance diff` | Compare an instance file with live cluster state |
| `instance status` | Show resource status for a deployed instance |
| `instance tree` | Show instance resource hierarchy |
| `instance delete` | Delete instance resources from a cluster (for a CLI-managed instance, PersistentVolumeClaims are kept unless `--delete-data`; `--wait` returns only when the deleted resources are gone) |
| `instance list` | List deployed instances |
| `instance events` | Show events for an instance |

#### CLI-managed vs controller-managed instances

Every deployed instance is managed by exactly one of two actors, recorded as
`spec.owner` on its `ModuleInstance`. The CLI behaves differently against each,
and resolves which one it is before doing anything.

**CLI-managed** (`spec.owner: cli`): the CLI renders, applies, prunes, and
records the inventory itself. Every instance the CLI creates is CLI-managed —
`opm instance apply` writes `spec.owner: cli` on create and never rewrites an
existing owner. A CLI-managed apply returns once the cluster accepts the
objects; pass `--wait` to block until every applied resource is healthy (a
finished Deployment, StatefulSet or DaemonSet rollout, a completed Job, a bound
claim), bounded by `--timeout` (default 5m). On timeout it exits non-zero and
names the resources still pending.

**Controller-managed** (`spec.owner: controller`): the controller reconciles the
instance, and the CLI edits its spec rather than the cluster. A
controller-managed instance is one created outside the CLI — kubectl, GitOps,
the controller's own manifests. The CLI offers no command that moves an instance
between the two.

| Command | Against a controller-managed instance |
|---------|--------------------------------------|
| `instance apply` | Acts as a spec editor: writes `spec.module` and `spec.values`, waits for the controller's reconcile, reports the result. Applies and prunes nothing itself. Refuses a module that resolves from local bytes — the controller can only fetch published modules. |
| `instance delete` | Deletes the `ModuleInstance` and lets the controller's cleanup finalizer act. Refuses when the controller is not running, because deleting a finalizer-armed resource with no controller wedges it in `Terminating` with its workloads orphaned. Whether the workloads are actually removed depends on `spec.prune` — see below. |

> **`spec.prune` decides whether a controller-owned delete removes anything.**
> The field has no default and the CLI does not write it, so for a
> CLI-created instance the controller removes the `ModuleInstance` and
> deliberately **leaves the workloads running**. `opm instance delete` reports
> which of the two happened rather than assuming. To have the controller remove
> workloads on delete, set it first:
>
> ```bash
> kubectl patch moduleinstance <name> -n <ns> --type=merge -p '{"spec":{"prune":true}}'
> ```

Both wait for the controller, bounded by `--timeout` (default 5m).

### Configuration (`opm config`)

| Command | Description |
|---------|-------------|
| `config init` | Initialize OPM configuration |
| `config vet` | Validate configuration |

### Registry Operations (`opm registry`)

Use `opm registry` to manage credentials for the OCI registries OPM publishes to and pulls from.

| Command | Description |
|---------|-------------|
| `registry login [host]` | Verify a credential against the registry, then store it in the standard docker credential file — the store push and pull both read. Interactive by design; in CI use `docker login`, which writes the same file |

Without a host argument, the configured registry mapping is resolved to its host set: one host proceeds, several refuse with each listed as a runnable command. Append `+insecure` for a registry served over plain HTTP.

### Controller Lifecycle (`opm controller`)

Use `opm controller` to put the opm-controller (and its CRDs) onto a cluster — a prerequisite for any `opm instance apply`.

`opm controller install` installs the controller from its OPM module, `opmodel.dev/modules/opm_controller`, as the CLI-owned ModuleInstance `opm-controller` in `opm-system`. The controller never reconciles that instance, so re-running the CLI is always the way to repair or upgrade the controller.

- **Install needs a registry.** The module is pulled from the configured registry (`--registry`, `OPM_REGISTRY` or the config file). An air-gapped cluster installs from a mirror that serves the module and its dependencies.
- **`--crds-only` needs no registry.** It applies the CRDs of the module version this CLI pins, which the CLI carries, and talks only to the cluster. It takes no `--version`, `-f` or `--reset-values`, and refuses CRDs a newer module version installed (run `--crds-only` from a CLI that pins that version or newer, or install that version in full). A CRD that exists is judged as on a full install: one the controller's instance record lists is applied (a denied read of the record is a warning, and the CRDs are then judged without it); one without OPM's managed-by label, or with another instance's identity, is refused, and the refusal prints the annotation to set on it. `--crds-only` adopts nothing by itself and writes no ModuleInstance.
- **`--version` takes a controller module version**: `--version 0.2.0` pins, `--version v0` floats. Without it, install uses the module version this CLI pins. Install prints the controller release the module deploys.
- **Settings are instance values, not Deployment patches.** `-f/--values` files are layered over the values recorded on the controller's instance, so a reinstall keeps every recorded value it does not change; `--reset-values` starts from the module's defaults. The module's `#config` holds `registry` (the controller's own `--registry` mapping), `image.repository` (for a mirror), `defaultServiceAccount`, `resources`, `replicas` and `extraArgs`.
- **Install does not take over a controller applied from an earlier release manifest.** An object the module renders that already exists passes when the instance's record lists it, when it carries OPM's managed-by label and no other instance's UUID (a stopped install, a `--crds-only` install), or when it carries the annotation `opmodel.dev/adopt=<instance UUID>`; otherwise install refuses with exit 2, changes nothing, and prints the annotation to set for each object. The earlier manifest used other names and another namespace, so only its four CRDs are refused. Remove that controller first (its Deployment, Namespace and role objects, by hand: install does not delete or report objects the module does not render), then annotate the CRDs; with the CRDs annotated and the earlier controller left in place, two controllers run.
- **Uninstall deletes what the instance recorded**, except the CRDs and the Namespace, then the record. A controller with no record, applied with kubectl or by an older CLI, is refused: delete its objects by hand, or annotate them as install's refusal prints and install it with this CLI first.

| Command | Description |
|---------|-------------|
| `controller install` | Install the controller module (`--version`, `-f/--values`, `--reset-values`, `--crds-only`, `--rbac [--user\|--group]`, `--skip-platform`, `--catalog-prerelease`, `--timeout`) |
| `controller uninstall` | Remove the recorded controller, preserving CRDs and its Namespace (`--remove-finalizers`) |

```bash
# Install the pinned controller module and wait for it to roll out
opm controller install

# Let the controller resolve modules through a mirror (recorded on its instance)
echo 'values: registry: "opmodel.dev=mirror.example/opm"' > controller-values.cue
opm controller install -f controller-values.cue

# CLI-solo path: install just the CRDs, no running controller (works offline)
opm controller install --crds-only

# Grant a non-admin user access to ModuleInstances
opm controller install --crds-only --rbac --user alice

# Remove the controller (refuses while any ModuleInstance is still active)
opm controller uninstall
```

## Example Published Module Workflow

`module build` and `module apply` take a published module by its module path,
without a major. `--version v1` takes the newest release of major 1,
`--version 1.0.4` pins that release, and no `--version` takes the newest
release of the highest major built on this CLI's core. The chosen version is
reported on standard error, so manifests on standard output stay parseable.

```bash
# Render the newest compatible release
opm module build opmodel.dev/modules/web_app > manifests.yaml

# Render a pinned release with your own values
opm module build opmodel.dev/modules/web_app --version 1.0.4 -f values.cue

# Deploy the newest v1 release; without -f this applies the module's
# debugValues and warns about it
opm module apply opmodel.dev/modules/web_app --version v1 --name hello -n demo
```

A local directory is always spelled as a path (`.`, `./web.app`, `../x`, or
absolute); a bare argument whose first element holds a dot is a module path.

## Example Instance Workflow

`instance init` writes a package that pins the module at the resolved
version, core at the version the module declares, and the rest of their
dependencies, so it builds with no `cue` command. `values.cue` starts from the
module's `initValues`, else its `debugValues` when they are fully concrete,
else empty; review it before deploying. The target directory must not exist or sit inside
another CUE module.

```bash
# Create the package in ./web (--version and --dir work as elsewhere)
opm instance init web opmodel.dev/modules/web_app -n demo

# Validate an instance file
opm instance vet ./web/instance.cue

# Render manifests from an instance file
opm instance build ./web/instance.cue

# Apply an instance file to the cluster
opm instance apply ./web/instance.cue

# Inspect deployed state by file, name, or UUID
opm instance status ./web/instance.cue
opm instance status web -n demo
```

## Documentation

For development guidelines, architecture details, and agent instructions, see `AGENTS.md`.

## Build And Test

```bash
# Run all checks (format, vet, lint, test)
task check

# Build binary
task build

# Install binary
task install

# Run tests
task test

# Run tests with coverage
task test:coverage
```

## Container Image

Each release publishes a multi-arch (linux/amd64, linux/arm64) distroless image to GHCR, usable as a CI job or Kubernetes `Job` image. It runs as a non-root user and carries a CA bundle for HTTPS registries; registry configuration and credentials come from the pipeline.

```bash
docker run --rm ghcr.io/open-platform-model/opm:1.0.0-beta.2 version
```

Every release is tagged with its version (no `v` prefix); stable releases also move `latest` and the major tag (`v1`).

## Requirements

- Go 1.25+
- Kubernetes cluster for deployment and integration-test workflows

## License

This project is licensed under the Apache License 2.0. See `LICENSE`.
