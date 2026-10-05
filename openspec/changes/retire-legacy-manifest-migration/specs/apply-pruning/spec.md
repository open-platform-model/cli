## REMOVED Requirements

### Requirement: Pre-apply existence check on first install
**Reason**: Its admission set existed only for `opm operator install` to take over an operator installed from an earlier release manifest (0012:D8:R6, withdrawn 2026-10-05). The requirement is restated without the admission set as "First-time apply refuses existing objects OPM does not manage".
**Migration**: None. No caller passes an admission set; the check behaves as it did for every command other than operator install.

## ADDED Requirements

### Requirement: First-time apply refuses existing objects OPM does not manage

On first-time apply (no previous inventory), the system SHALL check each rendered resource against the cluster. If a resource exists with a `deletionTimestamp` (terminating), or exists without OPM labels (untracked), the apply SHALL fail with a clear error message. This check SHALL be skipped entirely when a previous inventory exists. The check SHALL take no admission set and no flag SHALL bypass it, for every command alike, `opm operator install` included. The untracked-resource error SHALL NOT name a flag that would bypass it, since no flag does; it SHALL tell the user to remove or rename the existing resource, or to change the module so it renders a different name.

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

#### Scenario: Operator install meets the same check

- **WHEN** `opm operator install` performs the first apply of the operator instance
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked, as any other first-time apply does
