## Why

The codebase has a systematic naming confusion between the Module concept (a CUE definition directory) and the ModuleRelease concept (a concrete deployed instance). Internal Go identifiers — function names, type names, variable names, and error messages — routinely say "Module" when they mean "ModuleRelease", making the code actively misleading to anyone reading or extending it. A thorough audit identified 27 distinct violations across 15 files.

## What Changes

- Rename `inventory.ModuleRef` → `inventory.ChangeSource` (type name reflects its actual role: recording the source context of a change entry)
- Rename `ChangeSource.Name` field → `ChangeSource.ReleaseName` (JSON tag `"name"` preserved for backward compat)
- Rename `ChangeEntry.Module` field → `ChangeEntry.Source` (JSON tag `"module"` preserved)
- Rename `InventoryMetadata.Name` → `InventoryMetadata.ModuleName` (JSON tag `"name"` preserved)
- Rename `InventoryMetadata.Namespace` → `InventoryMetadata.ReleaseNamespace` (JSON tag `"namespace"` preserved)
- Rename `cmdutil.RenderModule()` → `cmdutil.RenderRelease()`
- Rename `cmdutil.RenderModuleOpts` → `cmdutil.RenderReleaseOpts`
- Rename `output.ModuleLogger()` → `output.ReleaseLogger()` (visible `m:` prefix unchanged — UX stability)
- Rename `kubernetes.GetModuleStatus()` → `kubernetes.GetReleaseStatus()`
- Rename `concreteModule` variable → `concreteRelease` in `release/builder.go` and `release/metadata.go`
- Fix error message `"module validation failed"` → `"release validation failed"` in `release/builder.go`
- Fix log message `modLog.Info("module", ...)` → `releaseLog.Info("release", ...)` in `cmdutil/output.go`
- Fix debug message `"rendering module"` → `"rendering release"` in `cmdutil/render.go`
- Remove dead type `build.ModuleValidationError` — defined but never instantiated or type-asserted
- Rename all `modLog` local variables → `releaseLog` at callsites
- Rename `moduleName` test variables used as release names → `releaseName`

No user-facing behavior changes. The terminal output (`m:` prefix, "Module applied", "Module up to date", "Deploy module to cluster") is intentionally preserved — users think in terms of modules and the UX language reflects that.

## Capabilities

### New Capabilities
<!-- none — pure refactor -->

### Modified Capabilities
<!-- none — no spec-level behavior changes -->

## Impact

- `internal/inventory/types.go` — type and field renames
- `internal/inventory/changeid.go` — parameter type rename
- `internal/inventory/secret.go` — field access renames
- `internal/inventory/crud_test.go` — assertion updates
- `internal/inventory/types_test.go` — struct literal updates
- `internal/inventory/changeid_test.go` — struct literal updates
- `internal/cmdutil/render.go` — function and type rename
- `internal/cmdutil/output.go` — variable and log message rename
- `internal/output/log.go` — function rename
- `internal/output/log_test.go` — test updates
- `internal/kubernetes/status.go` — function rename
- `internal/build/release/builder.go` — variable and error message rename
- `internal/build/release/metadata.go` — parameter rename
- `internal/build/errors.go` — dead type removal
- `internal/build/errors_test.go` — dead test removal
- `internal/cmd/mod_apply.go`, `mod_build.go`, `mod_vet.go`, `mod_diff.go`, `mod_delete.go`, `mod_status.go` — callsite updates
- `tests/integration/deploy/main.go`, `inventory-ops/main.go`, `inventory-apply/main.go` — variable renames
- `internal/kubernetes/diff_integration_test.go` — variable renames
- SemVer: **PATCH** — internal refactor, no public interface or behavior changes
