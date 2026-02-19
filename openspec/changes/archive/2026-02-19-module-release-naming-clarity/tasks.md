## 1. inventory package — type and field renames

- [x] 1.1 Rename `ModuleRef` → `ChangeSource` in `internal/inventory/types.go`
- [x] 1.2 Rename field `ChangeSource.Name` → `ChangeSource.ReleaseName` with JSON tag `json:"name"` preserved
- [x] 1.3 Rename field `ChangeEntry.Module` → `ChangeEntry.Source` with JSON tag `json:"module"` preserved
- [x] 1.4 Rename field `InventoryMetadata.Name` → `InventoryMetadata.ModuleName` with JSON tag `json:"name"` preserved
- [x] 1.5 Rename field `InventoryMetadata.Namespace` → `InventoryMetadata.ReleaseNamespace` with JSON tag `json:"namespace"` preserved
- [x] 1.6 Update `internal/inventory/changeid.go`: parameter `module ModuleRef` → `source ChangeSource`, variable `module` → `source`, field accesses updated
- [x] 1.7 Update `internal/inventory/secret.go`: all `inv.Metadata.Name` → `inv.Metadata.ModuleName`, `inv.Metadata.Namespace` → `inv.Metadata.ReleaseNamespace`
- [x] 1.8 Update `internal/inventory/crud_test.go`: assertion field references updated
- [x] 1.9 Update `internal/inventory/types_test.go`: struct literal field names updated
- [x] 1.10 Update `internal/inventory/changeid_test.go`: struct literal field names and variable names updated

## 2. output package — function rename

- [x] 2.1 Rename `ModuleLogger` → `ReleaseLogger` in `internal/output/log.go` (keep `m:` prefix in rendered output unchanged)
- [x] 2.2 Update `internal/output/log_test.go`: test function names and callsites updated

## 3. cmdutil package — function and type renames

- [x] 3.1 Rename `RenderModule` → `RenderRelease` in `internal/cmdutil/render.go`
- [x] 3.2 Rename `RenderModuleOpts` → `RenderReleaseOpts` in `internal/cmdutil/render.go`
- [x] 3.3 Update `internal/cmdutil/render.go`: variable `modLog` → `releaseLog`, debug message `"rendering module"` → `"rendering release"`, key `"module"` → `"module-path"`
- [x] 3.4 Update `internal/cmdutil/output.go`: variables `modLog` → `releaseLog`, log message `modLog.Info("module", ...)` → `releaseLog.Info("release", ...)`
- [x] 3.5 Update `internal/cmdutil/render_test.go`: callsite `RenderModule` → `RenderRelease`, type `RenderModuleOpts` → `RenderReleaseOpts`

## 4. kubernetes package — function rename

- [x] 4.1 Rename `GetModuleStatus` → `GetReleaseStatus` in `internal/kubernetes/status.go`
- [x] 4.2 Update `internal/kubernetes/diff_integration_test.go`: callsite `GetModuleStatus` → `GetReleaseStatus`, variable `moduleName` → `releaseName` (4 occurrences)

## 5. release builder — variable and error message fixes

- [x] 5.1 Rename all `concreteModule` → `concreteRelease` in `internal/build/release/builder.go` (5 occurrences)
- [x] 5.2 Rename all `concreteModule` parameter → `concreteRelease` in `internal/build/release/metadata.go` (11 occurrences across function signatures and usages)
- [x] 5.3 Change error message string `"module validation failed"` → `"release validation failed"` in `internal/build/release/builder.go:149`

## 6. build package — dead code removal

- [x] 6.1 Remove `ModuleValidationError` type and all its methods from `internal/build/errors.go`
- [x] 6.2 Remove corresponding test cases for `ModuleValidationError` from `internal/build/errors_test.go`

## 7. cmd package — callsite updates

- [x] 7.1 Update `internal/cmd/mod_apply.go`: `cmdutil.RenderModule` → `cmdutil.RenderRelease`, `cmdutil.RenderModuleOpts` → `cmdutil.RenderReleaseOpts`, `modLog` → `releaseLog`, `inventory.ModuleRef` → `inventory.ChangeSource`, field `.Name` → `.ReleaseName`
- [x] 7.2 Update `internal/cmd/mod_build.go`: `cmdutil.RenderModule` → `cmdutil.RenderRelease`, `cmdutil.RenderModuleOpts` → `cmdutil.RenderReleaseOpts`, `modLog` → `releaseLog`
- [x] 7.3 Update `internal/cmd/mod_vet.go`: `cmdutil.RenderModule` → `cmdutil.RenderRelease`, `cmdutil.RenderModuleOpts` → `cmdutil.RenderReleaseOpts`, `modLog` → `releaseLog`
- [x] 7.4 Update `internal/cmd/mod_diff.go`: `cmdutil.RenderModule` → `cmdutil.RenderRelease`, `cmdutil.RenderModuleOpts` → `cmdutil.RenderReleaseOpts`, `modLog` → `releaseLog`
- [x] 7.5 Update `internal/cmd/mod_delete.go`: `output.ModuleLogger` → `output.ReleaseLogger`, `modLog` → `releaseLog`
- [x] 7.6 Update `internal/cmd/mod_status.go`: `kubernetes.GetModuleStatus` → `kubernetes.GetReleaseStatus`, `output.ModuleLogger` → `output.ReleaseLogger`, `modLog` → `releaseLog` (3 occurrences)
- [x] 7.7 Update `internal/kubernetes/apply.go`: `output.ModuleLogger` → `output.ReleaseLogger`, `modLog` → `releaseLog`
- [x] 7.8 Update `internal/kubernetes/delete.go`: `output.ModuleLogger` → `output.ReleaseLogger`, `modLog` → `releaseLog`

## 8. integration tests — variable renames

- [x] 8.1 Update `tests/integration/deploy/main.go`: variable `moduleName` used as release name → `releaseName`
- [x] 8.2 Update `tests/integration/inventory-ops/main.go`: `inventory.ModuleRef` → `inventory.ChangeSource`, field `.Name` → `.ReleaseName`
- [x] 8.3 Update `tests/integration/inventory-apply/main.go`: `inventory.ModuleRef` → `inventory.ChangeSource`, field `.Name` → `.ReleaseName`

## 9. Validation

- [x] 9.1 Run `task build` — binary compiles without errors
- [x] 9.2 Run `task test` — all tests pass
- [x] 9.3 Run `task check` — fmt + vet + test all green
