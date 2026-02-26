## Why

There is no way to see what module releases are deployed in a cluster. Users must know the exact release name or ID to interact with a release (`status`, `delete`, `diff`). `opm mod list` closes this gap — it's the entry point for cluster-aware workflows, analogous to `kubectl get pods`.

## What Changes

- Add `opm mod list` command that lists all deployed module releases in a namespace
- Add `-A` / `--all-namespaces` flag to list across all namespaces
- Add `ListInventories` function to the inventory package for label-based discovery of all inventory Secrets
- Export health evaluation primitives from the kubernetes package to enable status summaries
- Add `QuickReleaseHealth` aggregation function for lightweight per-release health summary
- Support output formats: table (default), wide, json, yaml

This is a **MINOR** version change — new command with no breaking changes. All new flags have sensible defaults (namespace from config, table output format).

## Capabilities

### New Capabilities

- `mod-list`: The `opm mod list` command, its flags, output formats, status aggregation, and namespace scoping behavior
- `inventory-listing`: Label-based discovery of all inventory Secrets in a namespace (or all namespaces), building on existing inventory CRUD
- `health-export`: Exporting health evaluation primitives and adding a quick aggregation function for use outside the status command

### Modified Capabilities

_(none — existing specs are unchanged; we build on existing inventory labels and health logic without modifying their contracts)_

## Impact

- **New files**: `internal/inventory/list.go`, `internal/cmd/mod/list.go`, `tests/integration/mod-list/main.go`
- **Modified files**: `internal/kubernetes/health.go` (export types/functions), `internal/kubernetes/status.go` and `internal/kubernetes/tree.go` (update references to exported names), `internal/cmd/mod/mod.go` (register command)
- **Dependencies**: No new external dependencies — uses existing `client-go`, `inventory`, `kubernetes` packages
- **API**: No external API changes. Internal Go API addition only (exported `HealthStatus` type + `EvaluateHealth` + `QuickReleaseHealth`)
