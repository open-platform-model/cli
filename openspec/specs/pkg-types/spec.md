# Package Types (pkg/)

## Purpose

Defines the exported `pkg/` package structure that makes all shared domain types available for external tools. Replaces the `internal/core/` subpackages with public equivalents.

## Requirements

### Requirement: Core types exported in pkg/
All shared domain types SHALL be exported under `pkg/` for reuse by external tools. The package structure SHALL be:
- `pkg/core/` — `Resource`, label constants, GVK weights
- `pkg/module/` — `Module`, `ModuleMetadata`, `Instance`, `InstanceMetadata`
- `pkg/provider/` — `Provider`, `ProviderMetadata`
- `pkg/errors/` — all error types
- `pkg/loader/` — loading functions
- `pkg/render/` — rendering engine (matching, execution, validation)

There SHALL be no `pkg/bundle/` package — bundle support is not implemented (enhancement 0002 D15 removed the bundle path; D15 supersedes D7).

#### Scenario: External tool imports pkg/core
- **WHEN** an external Go module imports `github.com/opmodel/cli/pkg/core`
- **THEN** it can access `Resource`, label constants, and `GetWeight()` without importing any `internal/` packages

#### Scenario: External tool imports pkg/module
- **WHEN** an external Go module imports `github.com/opmodel/cli/pkg/module`
- **THEN** it can access `Module`, `ModuleMetadata`, `Instance`, and `InstanceMetadata` types

#### Scenario: pkg/bundle does not exist
- **WHEN** code attempts to import `github.com/opmodel/cli/pkg/bundle`
- **THEN** compilation fails — the package does not exist

### Requirement: ModuleInstance has typed component accessors
`ModuleInstance` SHALL expose components via typed accessor methods, NOT raw public fields.

#### Scenario: MatchComponents returns schema-preserving value
- **WHEN** `release.MatchComponents()` is called
- **THEN** it returns the CUE value that preserves `#resources`, `#traits`, and `#blueprints` definition fields needed for matching

#### Scenario: ExecuteComponents returns finalized data value
- **WHEN** `release.ExecuteComponents()` is called
- **THEN** it returns the finalized, constraint-free CUE value suitable for `FillPath` injection into transformers

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
