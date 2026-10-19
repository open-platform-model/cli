## REMOVED Requirements

### Requirement: Operator-version ceiling
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only", "Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR", "The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC", "The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver", "The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check", "Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates".

### Requirement: Gate ordering and dry-run exemption
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The gates run in a fixed order, all before the first resource write", "Dry-run applies skip four gates and still resolve ownership and run the ownership guard".

## ADDED Requirements

### Requirement: Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only

Before applying, the CLI SHALL read the cluster-scoped singleton `Platform` and compare `status.operatorVersion` to its own version on MAJOR.MINOR only: the patch number, the prerelease suffix and any build metadata SHALL be ignored on both sides, because the CLI and the operator share MAJOR.MINOR and release patches and prerelease counters independently.

**Source**: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Newer prerelease counter on the same line passes

- **WHEN** `status.operatorVersion` is `1.0.0-beta.3` and the CLI version is `1.0.0-beta.2`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Beta operator against an alpha CLI of the same line passes

- **WHEN** `status.operatorVersion` is `1.0.0-beta.1` and the CLI version is `1.0.0-alpha.27`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Newer operator patch passes

- **WHEN** `status.operatorVersion` is `1.0.4` and the CLI version is `1.0.1`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

### Requirement: Apply refuses with an upgrade-the-CLI error when the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR

If the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR, apply SHALL refuse with an error telling the user to upgrade the CLI; the error SHALL name both full versions as read. (Requirement: Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only)

**Source**: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Older CLI refused

- **WHEN** `status.operatorVersion` is `1.2.0` and the CLI version is `1.1.0`
- **THEN** apply SHALL exit non-zero with an upgrade-the-CLI error

#### Scenario: Newer operator minor refused

- **WHEN** `status.operatorVersion` is `1.1.0` (or `1.1.0-beta.1`) and the CLI version is `1.0.0`
- **THEN** apply SHALL exit non-zero with an upgrade-the-CLI error naming both versions

### Requirement: The operator-version ceiling check is skipped when the Platform or the field is absent or when reading the Platform fails due to RBAC

If the Platform or the field is absent, the check SHALL be skipped (solo cluster semantics). If reading the Platform fails due to RBAC, the check SHALL be skipped with a warning (a namespace-scoped user must remain able to apply). (Requirement: Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only)

**Source**: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Solo cluster skips the ceiling

- **WHEN** no `Platform` CR exists or `status.operatorVersion` is absent
- **THEN** the ceiling check SHALL be skipped and apply SHALL proceed

#### Scenario: Platform read denied degrades to warning

- **WHEN** reading the `Platform` returns a forbidden error
- **THEN** the ceiling check SHALL be skipped with a warning and apply SHALL proceed

### Requirement: The operator-version ceiling check is skipped with a warning when the CLI version or the operator version is not valid semver

If the CLI's own version is not valid semver (dev build), the check SHALL be skipped with a warning. If the operator version is not valid semver, the check SHALL be skipped with a warning. (Requirement: Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only)

**Source**: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Dev build skips with warning

- **WHEN** the CLI version is `dev`
- **THEN** the ceiling check SHALL be skipped and a warning SHALL be printed

### Requirement: The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check

The instance apply that `opm operator install` performs for the operator's own instance SHALL skip this check, because install replaces the operator the Platform reports rather than driving it; install applies the same `MAJOR.MINOR` rule to the operator version of the module it installs instead (see the `operator-lifecycle` capability). (Requirement: Before applying, the CLI compares the operator version of the `Platform` to its own version on MAJOR.MINOR only)

**Source**: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Older CLI repairs a newer operator

- **WHEN** `Platform/cluster` reports `status.operatorVersion` `1.1.0`, the CLI is `1.0.0`, and `opm operator install` targets a module that deploys operator `1.0.2`
- **THEN** no ceiling refusal occurs and install proceeds

### Requirement: Every other apply keeps the operator-version ceiling check and the operator's instance still runs the CRD gates

Every other apply SHALL keep the check, and the CRD-presence and CRD field-floor gates SHALL still run for the operator's instance. (Requirement: The instance apply that `opm operator install` performs for the operator's own instance skips the operator-version ceiling check)

**Source**: enhancement 0021 ("The CLI's ceiling on operator versions compares `MAJOR.MINOR` only").

#### Scenario: Ordinary apply against the same cluster still refused

- **WHEN** the same CLI runs `opm instance apply` for any other instance against that cluster
- **THEN** apply SHALL exit non-zero with the upgrade-the-CLI error

### Requirement: The gates run in a fixed order, all before the first resource write

The gates SHALL run in the order: CRD presence, CRD field floor, operator-version ceiling (skipped only for the operator's own instance during `opm operator install`), ownership resolution, status-RBAC pre-flight, the ownership guard over every rendered resource (capability `apply-pruning`), all before the first resource write.

#### Scenario: Gates precede all writes

- **WHEN** any gate fails during `opm instance apply`
- **THEN** zero resources SHALL have been created or modified in the cluster

### Requirement: Dry-run applies skip four gates and still resolve ownership and run the ownership guard

Dry-run applies SHALL skip the CRD presence, CRD field floor, operator-version ceiling and status-RBAC gates (they write nothing those gates protect). A dry run SHALL still resolve ownership and SHALL still run the ownership guard, so that it previews the refusal of the real run (capability `apply-pruning`).

#### Scenario: Dry-run skips gates

- **WHEN** `opm instance apply --dry-run` runs against a cluster without the ModuleInstance CRD
- **THEN** the render SHALL be produced without the missing-CRD error

#### Scenario: Dry-run runs the ownership guard

- **WHEN** `opm instance apply --dry-run` runs and a rendered resource exists without an OPM managed-by label
- **THEN** the dry run SHALL report the refusal of the ownership guard and SHALL exit 1
