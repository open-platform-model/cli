## ADDED Requirements

### Requirement: Public ownership policy check
The `pkg/ownership` package SHALL export an `EnsureCLIMutable` function that enforces the no-takeover policy: if a release was created by a controller, the CLI MUST refuse to mutate it. The function SHALL accept plain string parameters (not internal types).

#### Scenario: CLI-created release is mutable
- **WHEN** `ownership.EnsureCLIMutable("cli", "my-release", "default")` is called
- **THEN** it SHALL return nil (no error)

#### Scenario: Controller-created release is immutable to CLI
- **WHEN** `ownership.EnsureCLIMutable("controller", "my-release", "default")` is called
- **THEN** it SHALL return an error stating the release is controller-managed

#### Scenario: Nil/empty createdBy is mutable
- **WHEN** `ownership.EnsureCLIMutable("", "my-release", "default")` is called
- **THEN** it SHALL return nil (legacy releases without provenance are assumed CLI-owned)

### Requirement: Ownership constants
The `pkg/ownership` package SHALL export `CreatedByCLI` and `CreatedByController` string constants for use by both CLI and controller.

#### Scenario: Constants match inventory values
- **WHEN** code references `ownership.CreatedByCLI`
- **THEN** its value SHALL be `"cli"`
- **WHEN** code references `ownership.CreatedByController`
- **THEN** its value SHALL be `"controller"`

### Requirement: No internal dependencies
The `pkg/ownership` package SHALL NOT import any `internal/` packages.

#### Scenario: Clean dependency tree
- **WHEN** `pkg/ownership/` is compiled
- **THEN** its dependency tree contains only standard library packages
