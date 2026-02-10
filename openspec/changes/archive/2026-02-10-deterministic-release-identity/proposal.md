## Why

The `opm mod delete` command discovers resources via label matching on module name and namespace. While this works for the common case, it breaks when the module or release is renamed, and provides no secondary identification if labels are tampered with. Additionally, there is no stable, collision-proof identifier for a deployment slot that survives version upgrades — making it impossible to reliably track "this deployment" across its lifecycle. A deterministic UUID v5 identity, computed from immutable inputs using CUE's `uuid.SHA1` builtin, solves this by giving every module definition and every release a reproducible, stable identifier that becomes a label on cluster resources.

This is the **CLI-side** of a two-part change. A companion change with the same name covers the CUE catalog schema additions (`#Module.metadata.identity` and `#ModuleRelease.metadata.identity`). This change consumes those computed identities and uses them for labeling, discovery, and status display.

## What Changes

- Add `module-release.opmodel.dev/uuid` label to all applied resources (the release identity UUID)
- Add `module.opmodel.dev/uuid` label to all applied resources (the module identity UUID)
- Enhance `mod delete` discovery to use `release-id` label as primary selector, with current name+namespace labels as fallback — union both result sets
- Add `--release-id` flag to `mod delete` for direct UUID-based deletion
- Show release ID and module ID in `mod status` output
- Read identity fields from CUE evaluation output (populated by catalog schema)

**Not in scope:**

- The CUE schema changes themselves (separate catalog change)
- Inventory ConfigMap / state storage (future enhancement)
- ModuleRelease CR / ownerReferences (future controller work)

## Capabilities

### New Capabilities

- `release-identity-labeling`: Stamping release-id and module-id labels on resources during apply, and reading identity from CUE build output
- `identity-based-discovery`: Using release-id label for resource discovery during delete, status, and diff — with fallback to existing label selectors

### Modified Capabilities

- `deploy`: New labels added to resource labeling requirements (FR-D-060 series). Delete discovery enhanced with release-id selector. New `--release-id` flag on `mod delete`. Status output includes identity fields.

## Impact

- **SemVer**: MINOR — new labels, new optional flag, new status output. No breaking changes. Existing `--name`/`-n` delete workflow unchanged.
- **Packages affected**:
  - `internal/kubernetes/discovery.go` — new selector builder, union discovery logic
  - `internal/kubernetes/apply.go` — read and inject identity labels from module metadata
  - `internal/kubernetes/delete.go` — dual-strategy discovery
  - `internal/kubernetes/status.go` — display identity in output
  - `internal/cmd/mod_delete.go` — `--release-id` flag
  - `internal/cmd/mod_status.go` — identity display
  - `internal/build/types.go` — identity fields on metadata structs
- **Dependencies**: Requires the catalog `deterministic-release-identity` change to be published first (adds `identity` field to `#Module.metadata` and `#ModuleRelease.metadata` in the CUE schemas).
- **Backwards compatibility**: Resources applied before this change won't have release-id labels. The fallback to name+namespace discovery ensures these are still found. New applies will stamp both old and new labels.
