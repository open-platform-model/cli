## ADDED Requirements

### Requirement: CLI mutating workflows refuse controller-managed releases

Before mutating an existing release, CLI workflows that apply or delete a release SHALL inspect inventory ownership. If the inventory indicates `createdBy: "controller"`, the CLI MUST refuse the mutation.

#### Scenario: Apply blocked for controller-managed release
- **WHEN** the user runs `opm mod apply` for a release whose inventory records `createdBy: "controller"`
- **THEN** the command SHALL fail before mutating cluster resources
- **AND** the error SHALL state that the release is controller-managed and cannot be changed by the CLI

#### Scenario: Delete blocked for controller-managed release
- **WHEN** the user runs `opm mod delete` for a release whose inventory records `createdBy: "controller"`
- **THEN** the command SHALL fail before deleting tracked resources
- **AND** the error SHALL state that the release is controller-managed and cannot be deleted by the CLI

### Requirement: Legacy CLI-managed releases remain mutable by the CLI

If an existing inventory has `createdBy: "cli"` or does not contain `createdBy`, CLI mutating workflows SHALL continue to operate normally.

#### Scenario: Apply allowed for legacy inventory
- **WHEN** the user runs `opm mod apply` for a release whose inventory has no `createdBy`
- **THEN** the CLI SHALL treat the release as CLI-managed and proceed normally
