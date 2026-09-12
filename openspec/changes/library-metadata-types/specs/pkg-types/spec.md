## MODIFIED Requirements

### Requirement: Core types exported in pkg/
Shared domain types the CLI owns SHALL be exported under `pkg/` for reuse by external tools. The package structure SHALL be:
- `pkg/core/` — `Resource`, label constants, GVK-weighted conversion helpers
- `pkg/errors/` — CLI error types, sentinels and grouped CUE error helpers
- `pkg/inventory/` — the public inventory package
- `pkg/loader/` — module-root and local-replacement provenance readers
- `pkg/resourceorder/` — apply and delete ordering weights

Module and instance metadata types SHALL NOT be declared in `pkg/`: they are the library's `opm/schema.ModuleMetadata` and `opm/schema.InstanceMetadata` (aliased in `opm/module`), carried by every acquired artifact, and the CLI SHALL use those types wherever it holds decoded metadata.

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
