## 1. Types (`internal/inventory/types.go`)

- [x] 1.1 Rename `InventoryMetadata` → `ReleaseMetadata`; update JSON tags: `ReleaseName` → `json:"name"`, `ReleaseNamespace` → `json:"namespace"`, `ReleaseID` → `json:"uuid"`; remove `ModuleName` field and `ReleaseName`/`name` duplication
- [x] 1.2 Add `ModuleMetadata` struct with fields: `Kind string \`json:"kind"\``, `APIVersion string \`json:"apiVersion"\``, `Name string \`json:"name"\``, `UUID string \`json:"uuid,omitempty"\``
- [x] 1.3 Update `InventorySecret` struct: rename `Metadata` field to `ReleaseMetadata ReleaseMetadata`; add `ModuleMetadata ModuleMetadata`

## 2. Serialization (`internal/inventory/secret.go`)

- [x] 2.1 Replace `secretKeyMetadata = "metadata"` with two consts: `secretKeyReleaseMetadata = "releaseMetadata"` and `secretKeyModuleMetadata = "moduleMetadata"`
- [x] 2.2 Update `InventoryLabels`: remove `moduleName` parameter, remove `module.opmodel.dev/name` entry, rename `module.opmodel.dev/namespace` → `module-release.opmodel.dev/namespace`
- [x] 2.3 Update `MarshalToSecret`: marshal `inv.ReleaseMetadata` to `secretKeyReleaseMetadata`; marshal `inv.ModuleMetadata` to `secretKeyModuleMetadata`; update `SecretName` and `InventoryLabels` call sites (remove `moduleName` arg, use `inv.ReleaseMetadata.ReleaseName`, `inv.ReleaseMetadata.ReleaseID`, `inv.ReleaseMetadata.ReleaseNamespace`)
- [x] 2.4 Update `UnmarshalFromSecret`: read `releaseMetadata` key (required, error if missing); read `moduleMetadata` key (optional, zero value if missing); remove old `metadata` key handling

## 3. Label constants (`internal/kubernetes/discovery.go`)

- [x] 3.1 Remove `LabelModuleName` constant (`module.opmodel.dev/name`)
- [x] 3.2 Verify `LabelModuleNamespace` points to `module-release.opmodel.dev/namespace` (update if needed)

## 4. CRUD (`internal/inventory/crud.go`)

- [x] 4.1 Update `WriteInventory` signature: add `moduleName string` and `moduleUUID string` parameters
- [x] 4.2 In `WriteInventory`: pass `moduleName` and `moduleUUID` only when constructing a new `InventorySecret` (i.e. when called with `inv` that has empty `ReleaseMetadata`); pass empty string to `InventoryLabels` for `moduleName` (now removed from that func)

## 5. Apply command (`internal/cmd/mod/apply.go`)

- [x] 5.1 On the create path (`prevInventory == nil`): populate `ReleaseMetadata` with `Kind`, `APIVersion`, `ReleaseName: result.Release.Name`, `ReleaseNamespace: namespace`, `ReleaseID: releaseID`; populate `ModuleMetadata` with `Kind: "Module"`, `APIVersion: "core.opmodel.dev/v1alpha1"`, `Name: result.Release.ModuleName`, `UUID: result.Release.Identity`
- [x] 5.2 Update `WriteInventory` call to pass `result.Release.ModuleName` and `result.Release.Identity`

## 6. Unit tests (`internal/inventory/`)

- [x] 6.1 Update `types_test.go`: replace `InventoryMetadata` with `ReleaseMetadata`; add `ModuleMetadata` assertions; verify `kind`/`apiVersion` on both structs
- [x] 6.2 Update `crud_test.go`: remove `ModuleName` from metadata literals; update `WriteInventory` calls with new signature; update field access from `Metadata.*` to `ReleaseMetadata.*` and `ModuleMetadata.*`
- [x] 6.3 Add roundtrip test in `crud_test.go` (or `secret_test.go`) verifying `moduleMetadata` key is present/absent correctly and that missing key on unmarshal is not an error

## 7. Integration test fixtures (`tests/integration/`)

- [x] 7.1 Update `deploy/main.go`: replace `InventoryMetadata` literal with `ReleaseMetadata` + `ModuleMetadata`; update `WriteInventory` call
- [x] 7.2 Update `inventory-ops/main.go`: same structural update across all `WriteInventory` call sites
- [x] 7.3 Update `inventory-apply/main.go`: same structural update

## 8. Validation

- [x] 8.1 `task fmt` — all files formatted
- [x] 8.2 `task test` — all unit tests pass
