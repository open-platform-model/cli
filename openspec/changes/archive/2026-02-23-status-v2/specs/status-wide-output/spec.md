## ADDED Requirements

### Requirement: Wide format shows replica counts for workloads

When the user specifies `-o wide`, the status table SHALL include a REPLICAS column. For workload resources (Deployment, StatefulSet, DaemonSet), this column SHALL display the ready/desired replica ratio. For non-workload resources, the column SHALL display `-`.

The replica values SHALL be extracted from the resource's status and spec fields:
- **Deployment**: `status.readyReplicas` / `spec.replicas`
- **StatefulSet**: `status.readyReplicas` / `spec.replicas`
- **DaemonSet**: `status.numberReady` / `status.desiredNumberScheduled`

If a status field is missing or zero, it SHALL be displayed as `0`.

#### Scenario: Deployment replicas in wide format

- **WHEN** the user runs `opm mod status --release-name my-app -n prod -o wide`
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

#### Scenario: Non-workload resource in wide format

- **WHEN** a ConfigMap or Service is listed in wide format
- **THEN** the REPLICAS column SHALL display `-`

### Requirement: Wide format shows container images for workloads

When the user specifies `-o wide`, the status table SHALL include an IMAGE column. For workload resources (Deployment, StatefulSet, DaemonSet), this column SHALL display the image of the first container in the pod template. For non-workload resources, the column SHALL display `-`.

The image SHALL be extracted from `spec.template.spec.containers[0].image`.

#### Scenario: Deployment image in wide format

- **WHEN** a Deployment has `spec.template.spec.containers[0].image: nginx:1.25`
- **THEN** the IMAGE column SHALL display `nginx:1.25`

#### Scenario: Workload with no containers

- **WHEN** a workload has an empty containers list
- **THEN** the IMAGE column SHALL display `-`

### Requirement: Wide format shows PVC capacity

When the user specifies `-o wide`, PersistentVolumeClaim resources SHALL display their capacity and phase in the REPLICAS column. The IMAGE column SHALL display `-`.

The capacity SHALL be extracted from `status.capacity.storage` and the phase from `status.phase`.

#### Scenario: Bound PVC in wide format

- **WHEN** a PVC has `status.capacity.storage: 10Gi` and `status.phase: Bound`
- **THEN** the REPLICAS column SHALL display `10Gi (Bound)`

#### Scenario: Pending PVC in wide format

- **WHEN** a PVC has `status.phase: Pending` and no capacity
- **THEN** the REPLICAS column SHALL display `Pending`

### Requirement: Wide format shows Ingress hosts

When the user specifies `-o wide`, Ingress resources SHALL display their first host rule in the IMAGE column. The REPLICAS column SHALL display `-`.

The host SHALL be extracted from `spec.rules[0].host`.

#### Scenario: Ingress with host in wide format

- **WHEN** an Ingress has `spec.rules[0].host: app.example.com`
- **THEN** the IMAGE column SHALL display `app.example.com`

#### Scenario: Ingress without host rules

- **WHEN** an Ingress has no rules or the first rule has no host
- **THEN** the IMAGE column SHALL display `-`

### Requirement: Wide format table columns

When `-o wide` is specified, the table SHALL display the following columns in order: KIND, NAME, COMPONENT, STATUS, REPLICAS, IMAGE, AGE. This is a superset of the default table columns with REPLICAS and IMAGE added.

#### Scenario: Wide table column order

- **WHEN** the user runs `opm mod status --release-name my-app -n prod -o wide`
- **THEN** the table headers SHALL be KIND, NAME, COMPONENT, STATUS, REPLICAS, IMAGE, AGE in that order
