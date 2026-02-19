## Why

The inventory Secret's `data.metadata` blob conflates Module and ModuleRelease identity into a single struct, with `"name"` storing the module name and `"releaseName"` storing the release name — the opposite of what `kind: ModuleRelease` implies. The Secret labels are similarly confused, mixing module-scoped labels (`module.opmodel.dev/name`, `module.opmodel.dev/namespace`) onto an object that represents a release. This makes the data model harder to reason about and inconsistent with the K8s convention that `metadata.name` identifies the object itself.

## What Changes

- **BREAKING**: Rename `data.metadata` key → `data.releaseMetadata`
- **BREAKING**: In `releaseMetadata` JSON: `"name"` now holds the release name (was module name); `"releaseName"` field removed; `"releaseId"` renamed to `"uuid"`; `"namespace"` stays (holds release namespace, unchanged semantics)
- **BREAKING**: Add new `data.moduleMetadata` Secret data key containing `kind: Module`, `apiVersion`, `name` (module name), and `uuid` (module identity, omitempty)
- **BREAKING**: Remove `module.opmodel.dev/name` and `module.opmodel.dev/namespace` labels from inventory Secret
- **BREAKING**: Add `module-release.opmodel.dev/namespace` label (replaces `module.opmodel.dev/namespace`)
- `InventoryMetadata` struct renamed to `ReleaseMetadata`; new `ModuleMetadata` struct added
- `MarshalToSecret` / `UnmarshalFromSecret` updated to handle two data keys
- `WriteInventory` updated: `moduleName` passed only at create time; metadata is write-once and preserved on updates

## Capabilities

### New Capabilities

- none

### Modified Capabilities

- `release-inventory`: Secret data model restructured — `metadata` key split into `releaseMetadata` + `moduleMetadata`; field names cleaned up; labels revised

## Impact

- `internal/inventory/types.go` — struct changes
- `internal/inventory/secret.go` — serialization key names, label set
- `internal/inventory/crud.go` — `WriteInventory` signature
- `internal/cmd/mod/apply.go` — create path passes `moduleName` and `moduleUUID`
- `internal/kubernetes/discovery.go` — label constants added/removed
- `internal/inventory/crud_test.go`, `types_test.go` — updated fixtures
- `tests/integration/deploy/`, `inventory-ops/`, `inventory-apply/` — updated fixtures
- SemVer: **PATCH** (internal data format; no public CLI API change)
