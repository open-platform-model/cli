## ADDED Requirements

### Requirement: mod delete accepts --release-id flag

The `mod delete` command SHALL accept an optional `--release-id` flag. When provided, it SHALL be used as the primary discovery selector. The `--name` flag becomes optional when `--release-id` is provided.

#### Scenario: Delete flag validation

- **WHEN** `opm mod delete -n <namespace>` is run with neither `--name` nor `--release-id`
- **THEN** the command SHALL return a usage error: "either --name or --release-id is required"

#### Scenario: Delete with --release-id flag

- **WHEN** `opm mod delete --release-id <uuid> -n <namespace>` is run
- **THEN** resources SHALL be discovered by the release-id label and deleted

### Requirement: mod delete dual-strategy discovery

When `mod delete` discovers resources, it SHALL use both release-id and name+namespace label selectors (when both are available) and delete the union of both result sets, deduplicated.

#### Scenario: Delete finds orphaned resources via release-id

- **WHEN** a module was applied, then renamed in source (labels changed), then `mod delete` is run with the new name AND the original release-id
- **THEN** resources from both the old name and new name SHALL be found and deleted

## MODIFIED Requirements

### Requirement: mod delete accepts --release-id flag

The `mod delete` command SHALL require at least one of `--name` or `--release-id` for identification. The `--namespace` / `-n` flag remains required in all cases.

#### Scenario: Delete by name only (existing behavior)

- **WHEN** `opm mod delete --name <name> -n <namespace>` is run without `--release-id`
- **THEN** resources SHALL be discovered via the existing 3-label selector (managed-by + name + namespace)

#### Scenario: Delete by release-id only

- **WHEN** `opm mod delete --release-id <uuid> -n <namespace>` is run without `--name`
- **THEN** resources SHALL be discovered via the release-id selector (managed-by + release-id)

#### Scenario: Delete by both

- **WHEN** `opm mod delete --name <name> -n <namespace> --release-id <uuid>` is run
- **THEN** resources SHALL be discovered via both selectors and unioned

### Requirement: Resource labeling includes identity labels

All resources MUST have `module-release.opmodel.dev/uuid: <release-uuid>` when the release identity is available. All resources MUST have `module.opmodel.dev/uuid: <module-uuid>` when the module identity is available.

#### Scenario: Apply stamps identity labels

- **WHEN** `mod apply` is run with a module whose catalog schema provides identity fields
- **THEN** all applied resources SHALL have `module-release.opmodel.dev/uuid` and `module.opmodel.dev/uuid` labels

#### Scenario: Apply without catalog identity support

- **WHEN** `mod apply` is run with a module whose catalog schema does not provide identity fields
- **THEN** resources SHALL NOT have identity labels
- **AND** existing labeling behavior (managed-by, name, namespace, version, component) SHALL be unchanged
