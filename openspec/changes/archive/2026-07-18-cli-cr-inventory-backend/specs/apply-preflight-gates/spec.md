# apply-preflight-gates

## Purpose

The gate battery that runs on the CLI's real (non-dry-run) apply path before any resource is written: CRD presence, CRD field-presence floor, operator-version ceiling, and the status-RBAC pre-flight. Enhancement 0006 D24/D27/D23/D33 — these gates give the `ModuleInstance` CRD its role as a hard prerequisite for CLI apply.

## ADDED Requirements

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

Before applying, the CLI SHALL read the cluster-scoped singleton `Platform` and compare `status.operatorVersion` to its own version. If the Platform or the field is absent, the check SHALL be skipped (solo cluster semantics). If the operator version is semver-greater than the CLI version, apply SHALL refuse with an error telling the user to upgrade the CLI. If the CLI's own version is not valid semver (dev build), the check SHALL be skipped with a warning. If reading the Platform fails due to RBAC, the check SHALL be skipped with a warning (a namespace-scoped user must remain able to apply).

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

The gates SHALL run in the order: CRD presence, CRD field floor, operator-version ceiling, ownership resolution, status-RBAC pre-flight, pre-apply existence check — all before the first resource write. Dry-run applies SHALL skip the gate battery (they write nothing the gates protect).

#### Scenario: Gates precede all writes

- **WHEN** any gate fails during `opm instance apply`
- **THEN** zero resources SHALL have been created or modified in the cluster

#### Scenario: Dry-run skips gates

- **WHEN** `opm instance apply --dry-run` runs against a cluster without the ModuleInstance CRD
- **THEN** the render SHALL be produced without the missing-CRD error
