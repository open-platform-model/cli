## REMOVED Requirements

### Requirement: First-time apply refuses existing objects OPM does not manage
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## ADDED Requirements

### Requirement: First-time apply refuses existing objects OPM does not manage, on every command

On first-time apply (no previous inventory), the system SHALL check each rendered resource against the cluster. If a resource exists with a `deletionTimestamp` (terminating), or exists without OPM labels (untracked), the apply SHALL fail with a clear error message. This check SHALL be skipped entirely when a previous inventory exists. The check SHALL take no admission set and no flag SHALL bypass it, for every command alike, `opm controller install` included. The untracked-resource error SHALL NOT name a flag that would bypass it, since no flag does; it SHALL tell the user to remove or rename the existing resource, or to change the module so it renders a different name.

#### Scenario: Untracked resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked

#### Scenario: Terminating resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource exists on the cluster with a `deletionTimestamp`
- **THEN** the command SHALL fail with an error indicating the resource is terminating

#### Scenario: Check skipped when inventory exists

- **WHEN** performing a subsequent apply (previous inventory exists)
- **THEN** the pre-apply existence check SHALL be skipped entirely

#### Scenario: Untracked-resource error names no bypass flag

- **WHEN** performing a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the error SHALL NOT mention `--force`
- **AND** the error SHALL tell the user to remove or rename the existing resource, or to change the module to render a different name

#### Scenario: Controller install meets the same check

- **WHEN** `opm controller install` performs the first apply of the controller instance
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked, as any other first-time apply does
