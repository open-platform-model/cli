## Why

The `deploy-v1` change bundles four commands (apply, delete, diff, status) into a single change. Splitting `mod diff` and `mod status` into their own change reduces scope per change, enables parallel work, and recognizes that diff/status are read-only inspection commands with distinct concerns from the write-oriented apply/delete pair.

## What Changes

- Add `opm mod diff` command — renders locally and compares against live cluster state using dyff for semantic, colorized output
- Add `opm mod status` command — discovers deployed resources via OPM labels and evaluates health per resource category
- Add `internal/kubernetes/diff.go` — live state fetching and comparison logic
- Add `internal/kubernetes/health.go` — resource health evaluation by category (workloads, jobs, passive, custom)
- Add `internal/kubernetes/status.go` — resource discovery, health aggregation, and formatted output
- Add `internal/cmd/mod/diff.go` — CLI command wiring for diff
- Add `internal/cmd/mod/status.go` — CLI command wiring for status

## Capabilities

### New Capabilities

- `mod-diff`: Render a module locally and diff each resource against live cluster state, showing additions, deletions, and modifications with colorized semantic output via dyff
- `mod-status`: Discover deployed module resources via OPM labels and report health status per resource, with table/yaml/json output and optional watch mode

### Modified Capabilities

_None. This change introduces new commands without modifying existing spec-level behavior._

## Impact

- **New packages**: `internal/kubernetes/diff.go`, `internal/kubernetes/health.go`, `internal/kubernetes/status.go`
- **New commands**: `internal/cmd/mod/diff.go`, `internal/cmd/mod/status.go`
- **Dependencies**: Requires `render-pipeline-v1` (Pipeline interface), `build-v1` (Pipeline implementation), and shared Kubernetes client infrastructure from `deploy-v1` (`internal/kubernetes/client.go`, `internal/kubernetes/discovery.go`)
- **External deps**: `homeport/dyff` for diff output, `k8s.io/client-go` for cluster interaction
- **SemVer**: MINOR — adds new commands with no breaking changes
- **deploy-v1 scope reduction**: Phases 4 (Diff) and 5 (Status) move here; deploy-v1 retains apply, delete, client infrastructure, and resource labeling
