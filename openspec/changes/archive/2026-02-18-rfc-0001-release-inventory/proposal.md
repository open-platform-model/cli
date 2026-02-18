## Why

OPM is stateless: after `opm mod apply`, no record of the applied resource set is stored. This causes three problems. First, renaming a resource in a module (e.g. changing a component name) leaves orphans on the cluster — the old-named resources persist forever. Second, there is no automatic pruning of resources removed from a module between applies. Third, `diff`, `status`, and `delete` must scan every API type on the server via label selectors, which is slow (hundreds of API calls) and can surface false positives like Endpoints. RFC-0001 introduces a lightweight inventory stored as a Kubernetes Secret that records the exact set of resources for each apply, enabling pruning, fast discovery, and correct lifecycle management.

## What Changes

- New `internal/inventory/` package with Go types for the inventory data model, Secret serialization/deserialization, identity equality functions, deterministic manifest digest computation, change ID + history management, and inventory Secret CRUD operations
- `opm mod apply` gains a post-apply inventory write and stale resource pruning: render -> compute digest -> compute change ID -> read previous inventory -> compute stale set -> apply resources -> prune stale -> write inventory
- New flags on `opm mod apply`: `--no-prune` (skip pruning), `--max-history=N` (default 10, cap change history), `--force` (allow empty render to prune all)
- Safety checks: component-rename detection (prevents spurious deletes when only the component name changes), pre-apply existence check on first install (catches untracked or terminating resources)
- `opm mod delete` switches to inventory-based resource enumeration (deletes only tracked resources, then the inventory Secret itself), with label-scan fallback for pre-inventory modules
- `opm mod diff` orphan detection switches to inventory set-difference (fast, no API scan), with label-scan fallback
- `opm mod status` resource enumeration switches to inventory-based targeted GETs with component grouping, with label-scan fallback
- `DiscoverResources()` updated to exclude Secrets labeled `opmodel.dev/component: inventory` from workload queries
- Pipeline sort in `pipeline.go` upgraded to deterministic total ordering (5-key sort) as a side benefit of the digest algorithm

## Capabilities

### New Capabilities

- `release-inventory`: Inventory data model (types, JSON serialization), Secret marshal/unmarshal, identity equality, Secret name convention and labels, deterministic manifest digest, change ID computation, history management (index ordering, pruning, idempotent re-apply), and inventory Secret CRUD operations (get with fallback, write with optimistic concurrency, idempotent delete)
- `apply-pruning`: Updated apply flow integrating inventory — stale set computation, component-rename safety check, pre-apply existence check (first install only), create-then-prune ordering, write-nothing-on-failure semantics, `--no-prune` / `--max-history` / `--force` flags

### Modified Capabilities

- `deploy`: `opm mod apply` orchestration adds inventory read/write and pruning steps; `opm mod delete` switches from label-scan to inventory-based resource enumeration and deletes the inventory Secret last; both fall back to label-scan when no inventory exists
- `resource-discovery`: `DiscoverResources()` gains exclusion filter for Secrets with `opmodel.dev/component: inventory` to prevent the inventory Secret from appearing in workload queries
- `mod-diff`: `findOrphans()` switches from full API scan to inventory set-difference with targeted GETs, falling back to label-scan when no inventory exists
- `mod-status`: `GetModuleStatus()` switches from label-scan to inventory-based targeted GETs and groups resources by component using inventory data, falling back to label-scan when no inventory exists

## Impact

- **New package:** `internal/inventory/` — types, serialization, digest, change ID, CRUD
- **Modified packages:** `internal/kubernetes/` (apply, delete, diff, status, discovery), `internal/cmd/` (mod_apply.go flags and orchestration), `internal/build/pipeline.go` (deterministic sort)
- **New direct dependency:** `k8s.io/api` (for typed `corev1.Secret` in inventory CRUD)
- **SemVer:** MINOR — new behavior is additive (inventory write, pruning behind flags), all existing behavior preserved via label-scan fallback for pre-inventory modules; no breaking changes
- **Resolves:** #16 (delete no longer catches Endpoints), partially addresses #17 (write-nothing-on-failure keeps inventory consistent; partial apply to cluster remains a separate concern)
- **Justification (Principle VII):** Three new flags add complexity, but `--max-history` and `--no-prune` have safe defaults (10 and false), and `--force` is a safety gate for a destructive edge case. The inventory Secret itself is a single well-defined object per release.
