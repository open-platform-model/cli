# Package Types (pkg/)

## Purpose

Defines the exported `pkg/` package structure that makes all shared domain types available for external tools. Replaces the `internal/core/` subpackages with public equivalents.

## Requirements

### Requirement: Core types exported in pkg/
Shared domain types the CLI owns SHALL be exported under `pkg/` for reuse by external tools. The package structure SHALL be:
- `pkg/core/` — `Resource`, label constants, unstructured conversion helpers (ordering weights live in `pkg/resourceorder`)
- `pkg/errors/` — CLI error types, sentinels and grouped CUE error helpers
- `pkg/inventory/` — the public inventory package
- `pkg/loader/` — instance-file loading and local-replacement provenance readers
- `pkg/resourceorder/` — apply and delete ordering weights

Module and instance metadata types SHALL NOT be declared in `pkg/`: they are the library's `opm/schema.ModuleMetadata` and `opm/schema.InstanceMetadata` (aliased in `opm/module`), carried by every acquired artifact, and the CLI SHALL use those types wherever it holds decoded metadata. Where the CLI holds an acquired `*module.Module`, it SHALL read the module's identity fields (`name`, `modulePath`, `version`) from the module's decoded `Metadata` and SHALL NOT look them up in the module's `Package` value.

There SHALL be no `pkg/bundle/` package — bundle support is not implemented (enhancement 0002 D15 removed the bundle path; D15 supersedes D7).

#### Scenario: External tool imports pkg/core
- **WHEN** an external Go module imports `github.com/open-platform-model/cli/pkg/core`
- **THEN** it can access `Resource`, label constants, and `GetWeight()` without importing any `internal/` packages

#### Scenario: External tool imports pkg/module
- **WHEN** code attempts to import `github.com/open-platform-model/cli/pkg/module`
- **THEN** compilation fails — the package does not exist; the metadata types are the library's `ModuleMetadata` and `InstanceMetadata`

#### Scenario: pkg/bundle does not exist
- **WHEN** code attempts to import `github.com/open-platform-model/cli/pkg/bundle`
- **THEN** compilation fails — the package does not exist

#### Scenario: Scaffold reads module identity from decoded metadata
- **WHEN** `opm module init` checks that a scaffolded or cloned tree derives its new identity, or reads the version a tree states to create its identity package
- **THEN** it reads the identity fields (`modulePath`, `version`) from the acquired module's `Metadata`, not from its `Package` value
- **AND** a clone source whose metadata does not derive the new identity still refuses with the existing "does not derive metadata" refusal

### Requirement: No Component Go type
There SHALL be no `Component` struct type in `pkg/`. Component information for display purposes SHALL be derived from the `MatchPlan` result or CUE value iteration.

#### Scenario: No component package exists
- **WHEN** code attempts to import `pkg/component`
- **THEN** compilation fails — the package does not exist
