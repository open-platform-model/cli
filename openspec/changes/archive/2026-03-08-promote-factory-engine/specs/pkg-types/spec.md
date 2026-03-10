## ADDED Requirements

### Requirement: Core types exported in pkg/
All shared domain types SHALL be exported under `pkg/` for reuse by external tools. The package structure SHALL be:
- `pkg/core/` — `Resource`, label constants, GVK weights
- `pkg/module/` — `Module`, `ModuleMetadata`
- `pkg/modulerelease/` — `ModuleRelease`, `ReleaseMetadata`
- `pkg/bundle/` — `Bundle`, `BundleMetadata`
- `pkg/bundlerelease/` — `BundleRelease`, `BundleReleaseMetadata`
- `pkg/provider/` — `Provider`, `ProviderMetadata`
- `pkg/errors/` — all error types
- `pkg/loader/` — loading functions
- `pkg/engine/` — rendering engine

#### Scenario: External tool imports pkg/core
- **WHEN** an external Go module imports `github.com/opmodel/cli/pkg/core`
- **THEN** it can access `Resource`, label constants, and `GetWeight()` without importing any `internal/` packages

#### Scenario: External tool imports pkg/module
- **WHEN** an external Go module imports `github.com/opmodel/cli/pkg/module`
- **THEN** it can access `Module` and `ModuleMetadata` types

### Requirement: ModuleRelease has typed component accessors
`ModuleRelease` SHALL expose components via typed accessor methods, NOT raw public fields.

#### Scenario: MatchComponents returns schema-preserving value
- **WHEN** `release.MatchComponents()` is called
- **THEN** it returns the CUE value that preserves `#resources`, `#traits`, and `#blueprints` definition fields needed for matching

#### Scenario: ExecuteComponents returns finalized data value
- **WHEN** `release.ExecuteComponents()` is called
- **THEN** it returns the finalized, constraint-free CUE value suitable for `FillPath` injection into transformers

### Requirement: Bundle and BundleRelease types
`pkg/bundle/` SHALL export `Bundle` and `BundleMetadata`. `pkg/bundlerelease/` SHALL export `BundleRelease` and `BundleReleaseMetadata`. `BundleRelease` SHALL carry a `Releases map[string]*modulerelease.ModuleRelease` field.

#### Scenario: BundleRelease carries per-instance releases
- **WHEN** a `BundleRelease` is loaded from a CUE bundle release definition
- **THEN** `BundleRelease.Releases` contains one `*ModuleRelease` per instance, keyed by instance name

### Requirement: Provider is a thin CUE wrapper
`pkg/provider.Provider` SHALL carry `*ProviderMetadata` and a `Data cue.Value` field. It SHALL NOT have a Go-side `Match()` method.

#### Scenario: Provider has no Match method
- **WHEN** code accesses a `*provider.Provider`
- **THEN** there is no `Match()` method available — matching is done via CUE `#MatchPlan` in the engine

### Requirement: No Component Go type
There SHALL be no `Component` struct type in `pkg/`. Component information for display purposes SHALL be derived from the `MatchPlan` result or CUE value iteration.

#### Scenario: No component package exists
- **WHEN** code attempts to import `pkg/component`
- **THEN** compilation fails — the package does not exist
