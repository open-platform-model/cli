## MODIFIED Requirements

### Requirement: Gate ordering and dry-run exemption

The gates SHALL run in the order: CRD presence, CRD field floor, operator-version ceiling (skipped only for the operator's own instance during `opm operator install`), ownership resolution, status-RBAC pre-flight, the ownership guard over every rendered resource (capability `apply-pruning`), all before the first resource write. Dry-run applies SHALL skip the gate battery (they write nothing the gates protect).

#### Scenario: Gates precede all writes

- **WHEN** any gate fails during `opm instance apply`
- **THEN** zero resources SHALL have been created or modified in the cluster

#### Scenario: Dry-run skips gates

- **WHEN** `opm instance apply --dry-run` runs against a cluster without the ModuleInstance CRD
- **THEN** the render SHALL be produced without the missing-CRD error
