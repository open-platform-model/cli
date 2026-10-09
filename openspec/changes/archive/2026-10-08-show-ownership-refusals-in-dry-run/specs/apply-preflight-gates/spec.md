## MODIFIED Requirements

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
