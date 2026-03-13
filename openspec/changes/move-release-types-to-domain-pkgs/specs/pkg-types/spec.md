## MODIFIED Requirements

### Requirement: Core types exported in pkg/
All shared domain types SHALL be exported under `pkg/` for reuse by external tools. The package structure SHALL be:
- `pkg/core/` — `Resource`, label constants, GVK weights
- `pkg/module/` — `Module`, `ModuleMetadata`, `Release`, `ReleaseMetadata`
- `pkg/bundle/` — `Bundle`, `BundleMetadata`, `Release`, `ReleaseMetadata`
- `pkg/provider/` — `Provider`, `ProviderMetadata`
- `pkg/errors/` — all error types
- `pkg/loader/` — loading functions
- `pkg/render/` — rendering engine (matching, execution, validation)

#### Scenario: External tool imports pkg/core
- **WHEN** an external Go module imports `github.com/opmodel/cli/pkg/core`
- **THEN** it can access `Resource`, label constants, and `GetWeight()` without importing any `internal/` packages

#### Scenario: External tool imports pkg/module
- **WHEN** an external Go module imports `github.com/opmodel/cli/pkg/module`
- **THEN** it can access `Module`, `ModuleMetadata`, `Release`, and `ReleaseMetadata` types

#### Scenario: External tool imports pkg/bundle
- **WHEN** an external Go module imports `github.com/opmodel/cli/pkg/bundle`
- **THEN** it can access `Bundle`, `BundleMetadata`, `Release`, and `ReleaseMetadata` types

### Requirement: Bundle and Release types
`pkg/bundle/` SHALL export `Bundle`, `BundleMetadata`, `Release`, and `ReleaseMetadata`. `Release` SHALL carry a `Releases map[string]*module.Release` field, importing `pkg/module`.

#### Scenario: bundle.Release carries per-instance releases
- **WHEN** a `bundle.Release` is loaded from a CUE bundle release definition
- **THEN** `bundle.Release.Releases` contains one `*module.Release` per instance, keyed by instance name

#### Scenario: pkg/bundle imports pkg/module
- **WHEN** `pkg/bundle/` is compiled
- **THEN** it imports `pkg/module/` for the `Release` type reference in `bundle.Release.Releases`
- **THEN** no circular dependency exists

## REMOVED Requirements

### Requirement: Separate modulerelease and bundlerelease packages
**Reason**: `ModuleRelease` and `BundleRelease` types are colocated with their parent domain packages (`pkg/module/`, `pkg/bundle/`) instead of separate `pkg/modulerelease/` and `pkg/bundlerelease/` packages. The types are renamed to `Release` and `ReleaseMetadata`. Fewer packages, same cohesion.
**Migration**: Import `pkg/module.Release` instead of `pkg/modulerelease.ModuleRelease`; import `pkg/bundle.Release` instead of `pkg/bundlerelease.BundleRelease`.
