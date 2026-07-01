## MODIFIED Requirements

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
- **WHEN** an external Go module imports `github.com/open-platform-model/cli/pkg/core`
- **THEN** it can access `Resource`, label constants, and `GetWeight()` without importing any `internal/` packages

#### Scenario: External tool imports pkg/module
- **WHEN** an external Go module imports `github.com/open-platform-model/cli/pkg/module`
- **THEN** it can access `Module`, `ModuleMetadata`, `Instance`, and `InstanceMetadata` types

#### Scenario: pkg/bundle does not exist
- **WHEN** code attempts to import `github.com/open-platform-model/cli/pkg/bundle`
- **THEN** compilation fails — the package does not exist
