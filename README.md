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
| `module version set` | Set the version the module declares in `identity/identity.cue` — surgical, idempotent, offline |
| `module publish` | Publish the module from its committed source, at the coordinates it declares |

### Catalog Operations (`opm catalog`)

Use `opm catalog` when you are starting from catalog source: declare a version or publish a release from the committed tree.

| Command | Description |
|---------|-------------|
| `catalog version set` | Set the version the catalog declares in `identity/identity.cue` — surgical, idempotent, offline |
| `catalog publish` | Publish the catalog from its committed source, at the coordinates it declares |

### Instance Operations (`opm instance`)

<!-- Renamed from `opm release` / `opm rel` (enhancement 0002 D6). The old `release`/`rel` verb is removed — no back-compat alias (D8). -->

`opm i`, `opm ins`, and `opm inst` are the short aliases.

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

#### CLI-managed vs operator-managed instances

Every deployed instance is managed by exactly one of two actors, recorded as
`spec.owner` on its `ModuleInstance`. The CLI behaves differently against each,
and resolves which one it is before doing anything.

**CLI-managed** (`spec.owner: cli`): the CLI renders, applies, prunes, and
records the inventory itself. Every instance the CLI creates is CLI-managed —
`opm instance apply` writes `spec.owner: cli` on create and never rewrites an
existing owner.

**Operator-managed** (`spec.owner: operator`): the operator reconciles the
instance, and the CLI edits its spec rather than the cluster. An
operator-managed instance is one created outside the CLI — kubectl, GitOps,
the operator's own manifests. The CLI offers no command that moves an instance
between the two.

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
