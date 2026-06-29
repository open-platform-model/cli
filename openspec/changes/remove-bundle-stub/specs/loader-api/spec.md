## MODIFIED Requirements

### Requirement: DetectInstanceKind identifies instance type
The `pkg/loader` package SHALL export a `DetectInstanceKind` function that reads the `kind` field from a loaded instance package. It SHALL recognize `"ModuleInstance"` and reject every other kind with an error; bundle kinds are no longer recognized (enhancement 0002 D15 — the bundle path is removed).

#### Scenario: ModuleInstance kind detection
- **WHEN** `DetectInstanceKind(pkg)` is called and the `kind` field is "ModuleInstance"
- **THEN** it returns "ModuleInstance"

#### Scenario: Unknown kind
- **WHEN** `DetectInstanceKind(pkg)` is called with an unrecognized kind (including the former `"BundleRelease"`)
- **THEN** it returns an `unknown instance kind` error

## REMOVED Requirements

### Requirement: LoadBundleReleaseFromValue builds a BundleRelease
**Reason**: The bundle path is removed (enhancement 0002 D15, supersedes D7). There is no `pkg/bundle` package and no bundle renderer for this function to return into; X1 explicitly deferred this bundle surface to X2 for removal. (The function had no implementation in the current loader — it was stale spec residue of the paused single-build refactor.)
**Migration**: None. Module instances are loaded via the module-loader functions (`LoadInstanceFile` / `LoadModuleInstanceFromValue`). A bundle-loading entrypoint is reintroduced only if bundle support is built.
