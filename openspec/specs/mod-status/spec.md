## Purpose

Defines how `opm instance status` discovers tracked resources from persisted instance inventory records and reports instance health without requiring module source.

## Requirements

### Requirement: Status header displays instance ownership

The `opm instance status` command SHALL display instance ownership derived from inventory provenance in the metadata header.

#### Scenario: Header shows controller ownership

- **WHEN** the user runs `opm instance status` for a instance whose inventory records `createdBy: "controller"`
- **THEN** the metadata header SHALL include `Owner: controller`

#### Scenario: Header shows legacy CLI ownership

- **WHEN** the user runs `opm instance status` for a instance whose inventory has no `createdBy`
- **THEN** the metadata header SHALL include `Owner: cli`

### Requirement: Status warns for non-CLI-managed instances

When the CLI reads a controller-managed instance, `opm instance status` SHALL surface a warning that the instance is controller-managed and cannot be mutated by the CLI.

#### Scenario: Controller-managed warning

- **WHEN** the user runs `opm instance status` for a controller-managed instance
- **THEN** the command SHALL display a warning indicating that the instance is controller-managed
- **AND** the command SHALL still show the instance status information

### Requirement: Status supports structured output formats

The command SHALL support `--output`/`-o` with values `table` (default), `yaml`, and `json` for machine-readable output.

#### Scenario: JSON output

- **WHEN** the user runs `opm instance status mod -n ns -o json`
- **THEN** the output SHALL be a valid JSON array of resource status objects

#### Scenario: YAML output

- **WHEN** the user runs `opm instance status mod -n ns -o yaml`
- **THEN** the output SHALL be valid YAML containing resource status entries

### Requirement: Namespace defaults to config

The `--namespace`/`-n` flag SHALL be optional for `opm instance status`. When omitted, the namespace SHALL be resolved using the precedence: flag → `OPM_NAMESPACE` environment variable → `~/.opm/config.cue` kubernetes.namespace → `"default"`.

#### Scenario: Namespace omitted uses config default

- **WHEN** the user runs `opm instance status my-app` without `-n`
- **AND** the config file sets `kubernetes: namespace: "production"`
- **THEN** the command SHALL operate in the `production` namespace

#### Scenario: Namespace omitted uses hardcoded default

- **WHEN** the user runs `opm instance status my-app` without `-n`
- **AND** no config or env sets a namespace
- **THEN** the command SHALL operate in the `default` namespace

### Requirement: Status accepts kubernetes connection flags

The command SHALL accept `--kubeconfig` and `--context` flags for cluster connection. Kubeconfig resolution SHALL follow: explicit flag > `KUBECONFIG` env var > default path (`~/.kube/config`).

#### Scenario: Custom kubeconfig

- **WHEN** the user runs `opm instance status mod -n ns --kubeconfig /path/to/config`
- **THEN** the command SHALL use the specified kubeconfig file

### Requirement: Status fails fast on connectivity errors

The command SHALL fail immediately with a clear error message if the Kubernetes cluster is unreachable.

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with code 3 and display a connectivity error message

### Requirement: Status groups resources by component from inventory

When a persisted instance inventory record is available, the `opm instance status` command SHALL group resources by the `component` field from inventory entries. This eliminates the need to read `component.opmodel.dev/name` labels from the cluster.

#### Scenario: Resources grouped by component

- **WHEN** the user runs `opm instance status` and a persisted instance inventory record exists
- **AND** the ownership inventory tracks 3 resources under component `app` and 2 under component `cache`
- **THEN** the output SHALL group the resources by component name

#### Scenario: Missing resource shown in status

- **WHEN** the ownership inventory tracks a resource that no longer exists on the cluster
- **THEN** the status output SHALL show the resource with status "Missing" or equivalent

### Requirement: Status reads the persisted instance inventory record to discover the tracked resources

The `opm instance status` command SHALL read the persisted instance inventory record (the `ModuleInstance` CR) for the instance named by its positional `<file|name|uuid>` argument to discover its tracked resources. <!-- Was: release inventory record, --release-id/--instance-name (enhancement 0002 ("Hard rename, no alias window; `semver: major` (resolves OQ3, OQ4)")) -->

#### Scenario: Status reads the inventory record
- **WHEN** the user runs `opm instance status <name> -n production`
- **THEN** the command SHALL read the persisted instance inventory record (the `ModuleInstance` CR) to discover its tracked resources

### Requirement: A UUID argument resolves by listing the CRs and a name argument by a direct GET

A UUID argument of the `opm instance status` command SHALL resolve via `inventory.FindRecordByInstanceUUID` (list the CRs in the namespace, match `status.instanceUUID`); a name argument SHALL resolve via `inventory.GetRecord` (direct GET by name).

#### Scenario: Discover by instance UUID

- **WHEN** the user runs `opm instance status <uuid> -n production`
- **THEN** the command SHALL resolve the instance inventory record via `inventory.FindRecordByInstanceUUID`
- **AND** SHALL perform one targeted GET per tracked entry

#### Scenario: Discover by instance name

- **WHEN** the user runs `opm instance status my-app -n production`
- **THEN** the command SHALL resolve the instance inventory record by a direct GET by name
- **AND** SHALL NOT require module source or re-rendering

### Requirement: Status performs one targeted GET per tracked entry once the record is found

Once the persisted instance inventory record is found, the `opm instance status` command SHALL perform one targeted GET per tracked entry via `inventory.DiscoverResourcesFromInventory`.

#### Scenario: One targeted GET per tracked entry
- **WHEN** the record is found
- **THEN** the command SHALL perform one targeted GET per tracked entry via `inventory.DiscoverResourcesFromInventory`

### Requirement: Status requires no module source or re-rendering and uses no cluster-wide label-scan

The `opm instance status` command MUST NOT require module source or re-rendering. It MUST NOT use a cluster-wide label-scan to discover workload resources.

#### Scenario: No module source and no label-scan
- **WHEN** the command discovers the tracked resources of an instance
- **THEN** it SHALL NOT require module source or re-rendering
- **AND** it SHALL NOT use a cluster-wide label-scan to discover workload resources

### Requirement: A workload is healthy only when its rollout has finished

The `opm instance status` command SHALL evaluate resource health using category-specific criteria: **Workloads** (Deployment, StatefulSet, DaemonSet): healthy only when the rollout has finished, never on a condition alone. The controller SHALL have observed the current generation (`status.observedGeneration >= metadata.generation`) and the replica counters SHALL match `spec.replicas` (default 1).

#### Scenario: Workload generation not observed
- **WHEN** the controller has not observed the current generation of a Deployment, StatefulSet or DaemonSet (`status.observedGeneration >= metadata.generation` does not hold)
- **THEN** the workload SHALL NOT be healthy, because a workload is healthy only when the rollout has finished

### Requirement: A Deployment is healthy when its replica counters match and no old replicas remain

The `opm instance status` command SHALL evaluate resource health using category-specific criteria: Deployment: `updatedReplicas` and `availableReplicas` equal `spec.replicas`, no old replicas remain, and the `Progressing` condition does not report `ProgressDeadlineExceeded`. See the requirement "A workload is healthy only when its rollout has finished".

#### Scenario: Deployment not yet ready

- **WHEN** a Deployment has `Ready` condition set to False
- **THEN** the status output SHALL show the Deployment as "NotReady"

#### Scenario: Deployment upgrade stuck behind an available old ReplicaSet

- **WHEN** a Deployment has `Available` True but `updatedReplicas` is lower than `spec.replicas`
- **THEN** the status output SHALL show the Deployment as "NotReady"

### Requirement: A StatefulSet is healthy when its replica counters and revisions match

The `opm instance status` command SHALL evaluate resource health using category-specific criteria: StatefulSet: `readyReplicas` and `updatedReplicas` equal `spec.replicas` and `updateRevision` equals `currentRevision` (revision and `updatedReplicas` checks relax for a partitioned `RollingUpdate` and are skipped for `OnDelete`). See the requirement "A workload is healthy only when its rollout has finished".

#### Scenario: StatefulSet with OnDelete
- **WHEN** a StatefulSet uses `OnDelete`
- **THEN** the revision and `updatedReplicas` checks SHALL be skipped

### Requirement: A DaemonSet is healthy when its scheduled and available counters match

The `opm instance status` command SHALL evaluate resource health using category-specific criteria: DaemonSet: `updatedNumberScheduled` and `numberAvailable` equal `desiredNumberScheduled`. See the requirement "A workload is healthy only when its rollout has finished".

#### Scenario: DaemonSet counters
- **WHEN** the `updatedNumberScheduled` and `numberAvailable` of a DaemonSet equal its `desiredNumberScheduled`
- **THEN** the DaemonSet criteria for health SHALL be met

### Requirement: A Job is healthy when its Complete condition is True

The `opm instance status` command SHALL evaluate resource health using category-specific criteria: **Jobs** (Job): healthy when `Complete` condition is True.

#### Scenario: Job completed

- **WHEN** a Job has `Complete` condition set to True
- **THEN** the status output SHALL show the Job as "Complete"

### Requirement: A CronJob is reported as Applied

The `opm instance status` command SHALL evaluate resource health using category-specific criteria: **CronJobs**: reported as `Applied` (scheduled, no readiness phase).

#### Scenario: CronJob reported as Applied
- **WHEN** a CronJob is evaluated
- **THEN** it SHALL be reported as `Applied` (scheduled, no readiness phase)

### Requirement: Passive kinds are reported as Applied and custom resources are healthy on a True Ready condition

The `opm instance status` command SHALL evaluate resource health using category-specific criteria: **Passive** (ConfigMap, Secret, Service, ServiceAccount, RBAC kinds, and similar kinds with no readiness concept): reported as `Applied` on creation, which counts as healthy but does not claim readiness. **Custom** (CRD instances): healthy if `Ready` condition is present and True, otherwise treated as passive.

#### Scenario: ConfigMap always healthy

- **WHEN** a ConfigMap exists in the cluster
- **THEN** the status output SHALL show the ConfigMap as "Applied"
- **AND** the instance aggregate status SHALL still be "Ready" when every other resource is healthy

#### Scenario: Custom resource with Ready condition

- **WHEN** a CRD instance has a `Ready` condition set to True
- **THEN** the status output SHALL show the resource as "Ready"

#### Scenario: Custom resource without Ready condition

- **WHEN** a CRD instance has no `Ready` condition in its status
- **THEN** the status output SHALL treat the resource as passive and show it as "Applied"

### Requirement: The default status output is a table with plain space-padded columns

The default output format of `opm instance status` SHALL be a table showing: resource kind, name, namespace, component, health status, and age. The COMPONENT column SHALL be populated from the inventory entry for each resource. Resources without a component SHALL display `-` in the COMPONENT column. Columns SHALL be aligned and human-readable. The table SHALL render as plain, space-padded columns with no border characters, consistent with kubectl output conventions.

#### Scenario: Default table output includes component column

- **WHEN** the user runs `opm instance status my-app -n production` without `--output`
- **THEN** the output SHALL be a formatted table with KIND, NAME, COMPONENT, STATUS, and AGE columns
- **AND** the COMPONENT column SHALL show the component name from the inventory entry for each resource

#### Scenario: Resource without component

- **WHEN** a resource does not have a component recorded in the inventory
- **THEN** the COMPONENT column SHALL display `-` for that resource

### Requirement: The output flag accepts wide and wide renders additional columns

The `--output`/`-o` flag of `opm instance status` SHALL accept `wide` as a valid value in addition to `table`, `yaml`, and `json`. When `-o wide` is specified, the `opm instance status` command SHALL render a table format with additional columns.

#### Scenario: Wide format accepted as output value

- **WHEN** the user runs `opm instance status my-app -n production -o wide`
- **THEN** the command SHALL render a table with additional columns beyond the default format

### Requirement: The status header shows the instance name, namespace, health and summary from the persisted inventory record

The status output SHALL display a metadata header above the resource table containing instance name, namespace, aggregate health status, and a resource summary. Module version and ownership metadata SHALL come from the persisted instance inventory record (`instanceMetadata`, `moduleMetadata`, `createdBy`) when present. <!-- Was: release name, release inventory record, releaseMetadata (enhancement 0002: hard rename with no alias window; instance files are `instance.cue`) -->

#### Scenario: Header sourced from persisted inventory

- **WHEN** the user runs `opm instance status my-app -n production`
- **AND** a persisted instance inventory record exists
- **THEN** the header SHALL display the instance name, namespace, aggregate health, and a resource summary sourced from `instanceMetadata`/`moduleMetadata`/`createdBy`

### Requirement: The status header requires no inventory change history, module source or re-rendering

The `opm instance status` command SHALL NOT require inventory change-history metadata such as source version, raw values, or per-change timestamps, and it MUST NOT require module source or re-rendering.

#### Scenario: Header needs no change history
- **WHEN** the command renders the metadata header
- **THEN** it SHALL NOT require inventory change-history metadata such as source version, raw values, or per-change timestamps

### Requirement: The status table and header are color-coded on a TTY and plain otherwise

The status table and header SHALL use color-coded output when stdout is a TTY. Color SHALL be disabled when stdout is not a TTY or when the `NO_COLOR` environment variable is set.

#### Scenario: Color output on TTY

- **WHEN** the user runs `opm instance status` with stdout connected to a TTY
- **AND** the `NO_COLOR` environment variable is not set
- **THEN** the STATUS column SHALL render `Ready` in green and `NotReady` in red
- **AND** the COMPONENT column SHALL render component names in cyan

#### Scenario: Color disabled on pipe

- **WHEN** the user pipes the output (e.g., `opm instance status ... | cat`)
- **THEN** all color/ANSI escape codes SHALL be stripped from the output

#### Scenario: Color disabled by NO_COLOR

- **WHEN** the `NO_COLOR` environment variable is set
- **THEN** all color/ANSI escape codes SHALL be stripped from the output

### Requirement: The color mapping of the status output

The color mapping of the status table and header SHALL be:
- Health status `Ready` and `Complete`: green
- Health status `Applied`: dim (healthy, but carries no readiness signal)
- Health status `NotReady` and `Missing`: red
- Health status `Unknown`: yellow
- Component names: cyan
- Resource names: cyan
- Structural elements (borders, separators): dim gray

#### Scenario: Unknown renders yellow
- **WHEN** a resource has health status `Unknown`
- **THEN** the color mapping SHALL show it in yellow

### Requirement: An unreadable tracked resource is warned about and listed as Unknown

When a tracked resource cannot be read (its GET fails with an error other than NotFound), `opm instance status` SHALL print a warning naming the resource's kind, namespace and name and the read error, and SHALL list the resource with health `Unknown`.

#### Scenario: Forbidden resource makes the instance not ready

- **WHEN** the user runs `opm instance status demo -n apps`
- **AND** every tracked resource is healthy except a ConfigMap whose read fails with Forbidden
- **THEN** the output SHALL list the ConfigMap with status `Unknown`
- **AND** a warning SHALL name the ConfigMap and the Forbidden error
- **AND** the aggregate status SHALL be `NotReady`
- **AND** the command SHALL exit 2

### Requirement: An Unknown resource makes the instance NotReady and the command exit 2

An `Unknown` resource is not healthy: the instance's aggregate status SHALL be `NotReady`, the summary SHALL count it as not ready, and the `opm instance status` command SHALL exit 2. An instance whose tracked resources all failed to read SHALL be reported this way, not as not found.

#### Scenario: Every read fails

- **WHEN** no tracked resource of an instance can be read
- **THEN** the command SHALL list each with status `Unknown`
- **AND** SHALL exit 2, not 5
