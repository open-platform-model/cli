# OPM CLI

> **WARNING: UNDER HEAVY DEVELOPMENT** - This project is actively being developed and APIs may change frequently.

Command-line interface for the Open Platform Model (OPM). Build, validate, deploy, and inspect portable application releases defined with CUE.

## Quick Start

```bash
# Build the CLI
task build

# Initialize a new module
./bin/opm module init ./my-module

# Validate a module
./bin/opm module vet ./my-module

# Validate an instance file
./bin/opm instance vet ./instance.cue

# Render an instance file
./bin/opm instance build ./instance.cue

# Apply an instance file
./bin/opm instance apply ./instance.cue
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

### Instance Operations (`opm instance`)

<!-- Renamed from `opm release` / `opm rel` (enhancement 0002 D6). The old `release`/`rel` verb is removed — no back-compat alias (D8). -->

`opm inst` is the short alias.

Use `opm instance` when you are starting from an instance file or when you want to inspect, list, or delete deployed instances.

| Command | Description |
|---------|-------------|
| `instance vet` | Validate an instance file without generating manifests |
| `instance build` | Render an instance file to manifests |
| `instance apply` | Deploy an instance file to a cluster |
| `instance diff` | Compare an instance file with live cluster state |
| `instance status` | Show resource status for a deployed instance |
| `instance tree` | Show instance resource hierarchy |
| `instance delete` | Delete instance resources from a cluster |
| `instance list` | List deployed instances |
| `instance events` | Show events for an instance |
| `instance handoff` | Transfer a CLI-managed instance to the operator |

#### CLI-managed vs operator-managed instances

Every deployed instance is managed by exactly one of two actors, recorded as
`spec.owner` on its `ModuleInstance`. The CLI behaves differently against each,
and resolves which one it is before doing anything.

**CLI-managed** (`spec.owner: cli`, the default for `opm instance apply`): the
CLI renders, applies, prunes, and records the inventory itself.

**Operator-managed** (`spec.owner: operator`): the operator reconciles the
instance, and the CLI edits its spec rather than the cluster.

| Command | Against an operator-managed instance |
|---------|--------------------------------------|
| `instance apply` | Acts as a spec editor: writes `spec.module` and `spec.values`, waits for the operator's reconcile, reports the result. Applies and prunes nothing itself. Refuses a module that resolves from local bytes — the operator can only fetch published modules. |
| `instance delete` | Deletes the `ModuleInstance` and lets the operator's cleanup finalizer act. Refuses when the operator is not running, because deleting a finalizer-armed resource with no controller wedges it in `Terminating` with its workloads orphaned. Whether the workloads are actually removed depends on `spec.prune` — see below. |

> **`spec.prune` decides whether an operator-owned delete removes anything.**
> The field has no default and the CLI does not write it, so for a
> CLI-created instance the operator removes the `ModuleInstance` and
> deliberately **leaves the workloads running**. `opm instance delete` reports
> which of the two happened rather than assuming. To have the operator remove
> workloads on delete, set it first:
>
> ```bash
> kubectl patch moduleinstance <name> -n <ns> --type=merge -p '{"spec":{"prune":true}}'
> ```

Both wait for the operator, bounded by `--timeout` (default 5m).

#### Graduating an instance to the operator (`instance handoff`)

`opm instance handoff <name>` moves a CLI-managed instance to operator
management. It verifies the operator can take over safely *before* changing
anything, checking in order:

1. the operator is installed and ready
2. the `ModuleInstance` exists and the CLI owns it
3. the instance was not last applied from local module bytes
4. `spec.module` resolves from the registry — ignoring any local replacement
   and any cached copy, since the operator gets neither
5. re-rendering the published module against the cluster `Platform` reproduces
   what is currently deployed

Only then does it set `spec.owner: operator` and wait for the operator's first
reconcile, which must leave the instance's resource set unchanged.

```bash
# Install the operator, then graduate an existing CLI-managed instance
opm operator install
opm instance handoff jellyfin -n media

# Afterwards, update values through the operator
opm instance apply ./instances/jellyfin/instance.cue -n media
```

Adoption relabels the instance's resources to the operator's `managed-by`
identity. Nothing is restarted, created, or removed — the command reports the
relabel as information, not as a change.

Two things to know before running it:

- **Handoff is forward-only.** There is no reverse mode and no flag for one. If
  the operator's first reconcile fails, ownership stays with the operator and
  the CLI tells you so rather than silently undoing the transfer.
- **`--platform` is rejected.** Verification renders against the cluster
  `Platform`, because that is what the operator will use; verifying against
  anything else would prove nothing.

If step 5 reports a digest mismatch, the cluster is running something the
registry no longer describes. Re-apply with the CLI to reconcile them, or pass
`--force` to hand off anyway with the mismatch displayed. `--force` bypasses
only that check — the ownership, provenance, and resolvability gates have no
override, since failing them makes the transfer unsafe rather than merely
unverified.

### Configuration (`opm config`)

| Command | Description |
|---------|-------------|
| `config init` | Initialize OPM configuration |
| `config vet` | Validate configuration |

### Operator Lifecycle (`opm operator`)

Use `opm operator` to put the opm-operator (and its CRDs) onto a cluster — a prerequisite for any `opm instance apply`.

| Command | Description |
|---------|-------------|
| `operator install` | Install the opm-operator (`--crds-only`, `--rbac [--user\|--group]`, `--version`, `--timeout`) |
| `operator uninstall` | Remove the opm-operator, preserving CRDs and its Namespace (`--remove-finalizers`) |

```bash
# Install the full operator and wait for it to become ready
opm operator install

# CLI-solo path: install just the CRDs, no running operator
opm operator install --crds-only

# Grant a non-admin user access to ModuleInstances
opm operator install --crds-only --rbac --user alice

# Remove the operator (refuses while any ModuleInstance is still active)
opm operator uninstall
```

## Example Instance Workflow

```bash
# Validate an instance file
opm instance vet ./instances/jellyfin/instance.cue

# Render manifests from an instance file
opm instance build ./instances/jellyfin/instance.cue

# Apply an instance file to the cluster
opm instance apply ./instances/jellyfin/instance.cue

# Inspect deployed state by file, name, or UUID
opm instance status ./instances/jellyfin/instance.cue
opm instance status jellyfin -n media

# Hand the instance over to the operator once you want it reconciled
opm instance handoff jellyfin -n media
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

## Requirements

- Go 1.25+
- Kubernetes cluster for deployment and integration-test workflows

## License

This project is licensed under the Apache License 2.0. See `LICENSE`.
