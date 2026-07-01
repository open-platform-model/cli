## MODIFIED Requirements

### Requirement: Import path correctness

Subpackages SHALL use correct import paths for internal and external dependencies.

#### Scenario: Subpackage imports shared types

- **WHEN** a subpackage needs LoadedComponent
- **THEN** it SHALL import `"github.com/open-platform-model/cli/internal/build"`

#### Scenario: Import order maintained

- **WHEN** files have imports
- **THEN** imports SHALL be ordered: stdlib, external, internal (per Go conventions)
