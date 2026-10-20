## REMOVED Requirements

### Requirement: Wide format shows replica counts for workloads
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Wide format adds a REPLICAS column to the status table", "Replica values come from the status and spec fields of each workload kind", "A missing or zero status replica field displays as 0".

## ADDED Requirements

### Requirement: Wide format adds a REPLICAS column to the status table

When the user specifies `-o wide`, the status table SHALL include a REPLICAS column. For workload resources (Deployment, StatefulSet, DaemonSet), this column SHALL display the ready/desired replica ratio. For non-workload resources, the column SHALL display `-`.

#### Scenario: Non-workload resource in wide format

- **WHEN** a ConfigMap or Service is listed in wide format
- **THEN** the REPLICAS column SHALL display `-`

### Requirement: Replica values come from the status and spec fields of each workload kind

The replica values SHALL be extracted from the resource's status and spec fields:
- **Deployment**: `status.readyReplicas` / `spec.replicas`
- **StatefulSet**: `status.readyReplicas` / `spec.replicas`
- **DaemonSet**: `status.numberReady` / `status.desiredNumberScheduled`
(Requirement: Wide format adds a REPLICAS column to the status table)

#### Scenario: Deployment replicas in wide format

- **WHEN** the user runs `opm instance status my-app -n prod -o wide`
- **AND** a Deployment has `spec.replicas: 3` and `status.readyReplicas: 3`
- **THEN** the REPLICAS column for that Deployment SHALL display `3/3`

#### Scenario: Deployment with partial readiness

- **WHEN** a Deployment has `spec.replicas: 3` and `status.readyReplicas: 1`
- **THEN** the REPLICAS column SHALL display `1/3`

#### Scenario: StatefulSet replicas in wide format

- **WHEN** a StatefulSet has `spec.replicas: 1` and `status.readyReplicas: 1`
- **THEN** the REPLICAS column SHALL display `1/1`

#### Scenario: DaemonSet replicas in wide format

- **WHEN** a DaemonSet has `status.desiredNumberScheduled: 5` and `status.numberReady: 5`
- **THEN** the REPLICAS column SHALL display `5/5`

### Requirement: A missing or zero status replica field displays as 0

If a status field is missing or zero, it SHALL be displayed as `0`.
(Requirement: Replica values come from the status and spec fields of each workload kind)

#### Scenario: A missing status field displays as 0

- **WHEN** a status field is missing or zero
- **THEN** it SHALL be displayed as `0`
