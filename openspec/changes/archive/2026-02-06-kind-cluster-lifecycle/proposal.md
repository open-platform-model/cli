## Why

The CLI's stub commands (`mod apply`, `mod delete`, `mod diff`, `mod status`) will soon need a live Kubernetes cluster for integration and end-to-end testing. Today there is no developer tooling to create, inspect, or tear down a local cluster — the `test:integration` Taskfile task exists as a placeholder that points at an empty directory. Adding kind cluster lifecycle tasks to the Taskfile gives every contributor a one-command path to stand up a disposable test cluster, removing a manual prerequisite that blocks integration test development.

## What Changes

- Add a `cluster:create` Taskfile task that provisions a kind cluster with a configurable name and Kubernetes version
- Add a `cluster:delete` Taskfile task that tears down the cluster by name
- Add a `cluster:status` Taskfile task that reports whether the cluster is running and its connection details
- Add a `cluster:recreate` convenience task that deletes then creates (clean-slate reset)
- Add a kind cluster configuration file that pins node image version and any required cluster settings (e.g., extra port mappings for future webhook testing)
- Update `test:integration` and `test:e2e` tasks to document the cluster prerequisite (but do NOT auto-create — keep cluster lifecycle explicit per Principle VII)

## Capabilities

### New Capabilities

- `kind-cluster-tasks`: Taskfile tasks for creating, deleting, inspecting, and recreating a local kind cluster for development and testing

### Modified Capabilities

_None. No existing spec-level requirements are changing._

## Impact

- **Taskfile.yml**: New `cluster:*` task namespace (4 tasks)
- **New file**: Kind cluster configuration YAML (e.g., `hack/kind-config.yaml` or `tests/kind-config.yaml`)
- **Dependencies**: Requires `kind` binary on PATH (not a Go module dependency — invoked via Taskfile shell commands)
- **SemVer**: **PATCH** — developer tooling only, no changes to CLI commands, flags, or public API
- **Justification (Principle VII)**: The four stub commands in `internal/cmd/mod_stubs.go` all require a live cluster for meaningful testing. Without these tasks, every contributor must manually manage kind clusters, which is error-prone and undocumented. This is the minimum viable tooling to unblock integration test development.
