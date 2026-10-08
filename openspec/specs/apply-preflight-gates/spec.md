## Purpose

The gate battery that runs on the CLI's real (non-dry-run) apply path before any resource is written: CRD presence, CRD field-presence floor, operator-version ceiling, and the status-RBAC pre-flight. Enhancement 0006 D24/D27/D23/D33 — these gates give the `ModuleInstance` CRD its role as a hard prerequisite for CLI apply.

## Requirements

### Requirement: Missing-CRD gate with install hint

Before applying any resource, the CLI SHALL verify the `moduleinstances.opmodel.dev` CustomResourceDefinition exists. When absent, apply SHALL fail with the one-line hint: `ModuleInstance CRD not found — run 'opm operator install --crds-only'`, and do nothing else.

#### Scenario: CRD absent

- **WHEN** `opm instance apply` runs against a cluster without the ModuleInstance CRD
- **THEN** the command SHALL exit non-zero with the install hint
- **AND** no resource SHALL have been applied

### Requirement: CRD field-presence floor

Before applying, the CLI SHALL verify the installed `ModuleInstance` CRD's served storage-version schema contains the `spec.owner` and `status.inventory` properties. When either is missing, apply SHALL refuse with: `ModuleInstance CRD is missing required fields — run 'opm operator install --crds-only'`.

#### Scenario: Outdated CRD refused

- **WHEN** the installed CRD schema lacks `spec.owner`
- **THEN** apply SHALL exit non-zero with the missing-fields error before any resource is applied

#### Scenario: Current CRD passes

- **WHEN** the installed CRD schema contains both `spec.owner` and `status.inventory`
- **THEN** the floor gate SHALL pass silently

### Requirement: Operator-version ceiling

Before applying, the CLI SHALL read the cluster-scoped singleton `Platform` and compare `status.operatorVersion` to its own version on MAJOR.MINOR only: the patch number, the prerelease suffix and any build metadata SHALL be ignored on both sides, because the CLI and the operator share MAJOR.MINOR and release patches and prerelease counters independently. If the Platform or the field is absent, the check SHALL be skipped (solo cluster semantics). If the operator's MAJOR.MINOR is greater than the CLI's MAJOR.MINOR, apply SHALL refuse with an error telling the user to upgrade the CLI; the error SHALL name both full versions as read. If the CLI's own version is not valid semver (dev build), the check SHALL be skipped with a warning. If the operator version is not valid semver, the check SHALL be skipped with a warning. If reading the Platform fails due to RBAC, the check SHALL be skipped with a warning (a namespace-scoped user must remain able to apply). The instance apply that `opm operator install` performs for the operator's own instance SHALL skip this check, because install replaces the operator the Platform reports rather than driving it; install applies the same `MAJOR.MINOR` rule to the operator version of the module it installs instead (see the `operator-lifecycle` capability). Every other apply keeps the check, and the CRD-presence and CRD field-floor gates still run for the operator's instance. Source: 0021:D9:R3, 0021:D9:R4.

#### Scenario: Solo cluster skips the ceiling

- **WHEN** no `Platform` CR exists or `status.operatorVersion` is absent
- **THEN** the ceiling check SHALL be skipped and apply SHALL proceed

#### Scenario: Older CLI refused

- **WHEN** `status.operatorVersion` is `1.2.0` and the CLI version is `1.1.0`
- **THEN** apply SHALL exit non-zero with an upgrade-the-CLI error

#### Scenario: Dev build skips with warning

- **WHEN** the CLI version is `dev`
- **THEN** the ceiling check SHALL be skipped and a warning SHALL be printed

#### Scenario: Platform read denied degrades to warning

- **WHEN** reading the `Platform` returns a forbidden error
- **THEN** the ceiling check SHALL be skipped with a warning and apply SHALL proceed

#### Scenario: Newer prerelease counter on the same line passes

- **WHEN** `status.operatorVersion` is `1.0.0-beta.3` and the CLI version is `1.0.0-beta.2`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Beta operator against an alpha CLI of the same line passes

- **WHEN** `status.operatorVersion` is `1.0.0-beta.1` and the CLI version is `1.0.0-alpha.27`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Newer operator patch passes

- **WHEN** `status.operatorVersion` is `1.0.4` and the CLI version is `1.0.1`
- **THEN** the ceiling check SHALL pass and apply SHALL proceed

#### Scenario: Newer operator minor refused

- **WHEN** `status.operatorVersion` is `1.1.0` (or `1.1.0-beta.1`) and the CLI version is `1.0.0`
- **THEN** apply SHALL exit non-zero with an upgrade-the-CLI error naming both versions

#### Scenario: Older CLI repairs a newer operator

- **WHEN** `Platform/cluster` reports `status.operatorVersion` `1.1.0`, the CLI is `1.0.0`, and `opm operator install` targets a module that deploys operator `1.0.2`
- **THEN** no ceiling refusal occurs and install proceeds

#### Scenario: Ordinary apply against the same cluster still refused

- **WHEN** the same CLI runs `opm instance apply` for any other instance against that cluster
- **THEN** apply SHALL exit non-zero with the upgrade-the-CLI error

### Requirement: Status-RBAC pre-flight

In CLI-executor mode, before applying any resource, the CLI SHALL issue a `SelfSubjectAccessReview` for `patch` on `moduleinstances/status` in the target namespace. On denial, apply SHALL abort with an actionable error explaining that inventory cannot be recorded and naming the remedies (grant `moduleinstances/status`, or `opm operator install --crds-only --rbac`). The pre-flight guarantees resources are never deployed without a recordable inventory.

#### Scenario: Denied status access aborts before apply

- **WHEN** the SSAR reports `patch moduleinstances/status` is denied
- **THEN** apply SHALL exit non-zero before any resource is applied
- **AND** the error SHALL name the `--rbac` remedy

#### Scenario: Allowed status access proceeds silently

- **WHEN** the SSAR reports the access is allowed
- **THEN** apply SHALL proceed with no additional output

### Requirement: Gate ordering and dry-run exemption

The gates SHALL run in the order: CRD presence, CRD field floor, operator-version ceiling (skipped only for the operator's own instance during `opm operator install`), ownership resolution, status-RBAC pre-flight, the ownership guard over every rendered resource (capability `apply-pruning`), all before the first resource write. Dry-run applies SHALL skip the CRD presence, CRD field floor, operator-version ceiling and status-RBAC gates (they write nothing those gates protect). A dry run SHALL still resolve ownership and SHALL still run the ownership guard, so that it previews the refusal of the real run (capability `apply-pruning`).

#### Scenario: Gates precede all writes

- **WHEN** any gate fails during `opm instance apply`
- **THEN** zero resources SHALL have been created or modified in the cluster

#### Scenario: Dry-run skips gates

- **WHEN** `opm instance apply --dry-run` runs against a cluster without the ModuleInstance CRD
- **THEN** the render SHALL be produced without the missing-CRD error

#### Scenario: Dry-run runs the ownership guard

- **WHEN** `opm instance apply --dry-run` runs and a rendered resource exists without an OPM managed-by label
- **THEN** the dry run SHALL report the refusal of the ownership guard and SHALL exit 1
