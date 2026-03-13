## Why

The `internal/resourceorder` package provides GVK-based apply ordering weights — pure data with zero dependencies beyond `k8s.io/apimachinery`. Both CLI and controller need deterministic resource ordering (CRDs before workloads, webhooks last). Making this public allows sharing the same ordering policy.

## What Changes

- Move `internal/resourceorder/` to `pkg/resourceorder/`
- No renames — `resourceorder.GetWeight()` reads fine as a public API
- Update all 4 callers

## Capabilities

### New Capabilities

- `pkg-resourceorder`: GVK-based resource ordering weights are publicly importable from `pkg/resourceorder/`. External consumers can call `resourceorder.GetWeight(gvk)` to get deterministic apply ordering.

### Modified Capabilities

## Impact

- **Files moved**: `internal/resourceorder/weights.go`, `internal/resourceorder/weights_test.go` → `pkg/resourceorder/`
- **Internal callers updated** (4 files):
  - `internal/inventory/stale.go`
  - `internal/inventory/digest.go`
  - `internal/output/manifest.go`
  - `internal/kubernetes/delete.go`
- **SemVer**: MINOR (new public API surface)
