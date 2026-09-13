## Purpose

Defines how `opm instance list` discovers deployed instances from persisted instance inventory records and reports instance metadata and health without requiring module source.

## Requirements

### Requirement: List command discovers instances via persisted ownership inventory

The `opm instance list` command SHALL discover all deployed module instances by listing persisted instance inventory records in the target namespace. It SHALL use the `ListRecords` function from the inventory package. It MUST NOT require module source, re-rendering, knowledge of specific instance names, or inventory change-history fields to identify the current owned resource set for an instance. <!-- Was: "discovers releases", "release inventory records", "release names" (0002 D9) -->

#### Scenario: List discovers via ListRecords

- **WHEN** the user runs `opm instance list -n production`
- **THEN** the command SHALL list deployed module instances via `ListRecords` for `production`
- **AND** SHALL NOT require module source or re-rendering

### Requirement: List command supports all-namespaces flag

The command SHALL accept `-A` / `--all-namespaces` to list instances across all namespaces. When `-A` is used, the NAMESPACE column SHALL be included in the table output. When `-A` is not used, the NAMESPACE column SHALL be hidden.

#### Scenario: All-namespaces listing

- **WHEN** the user runs `opm instance list -A`
- **AND** instances exist in namespaces `media` and `games`
- **THEN** the output SHALL include instances from both namespaces
- **AND** the table SHALL include a NAMESPACE column

#### Scenario: Single namespace hides namespace column

- **WHEN** the user runs `opm instance list -n media`
- **THEN** the table output SHALL NOT include a NAMESPACE column

### Requirement: List command shows health status for each instance

The command SHALL evaluate the health of each instance by discovering its tracked resources and evaluating their health status. The STATUS column SHALL display the aggregate health and ready/total count in the format `Ready (N/N)` or `NotReady (N/N)`.

#### Scenario: All resources healthy

- **WHEN** a instance has 5 tracked resources and all are healthy
- **THEN** the STATUS column SHALL display `Ready (5/5)`

#### Scenario: Some resources unhealthy

- **WHEN** a instance has 5 tracked resources and 2 are not healthy
- **THEN** the STATUS column SHALL display `NotReady (3/5)`

#### Scenario: Missing resource counts as unhealthy

- **WHEN** a instance tracks a resource that no longer exists on the cluster
- **THEN** that resource SHALL count toward the total but NOT toward the ready count

#### Scenario: Zero resources

- **WHEN** a instance inventory record has no tracked resources in its ownership inventory
- **THEN** the STATUS column SHALL display `Unknown (0/0)`

### Requirement: List command displays instance ownership

The `opm instance list` command SHALL expose instance ownership derived from inventory provenance. Table outputs SHALL include an OWNER column, and structured outputs SHALL include an `owner` field.

#### Scenario: Table output shows controller ownership

- **WHEN** the user runs `opm instance list`
- **AND** a instance inventory records `createdBy: "controller"`
- **THEN** the instance row SHALL display `controller` in the OWNER column

#### Scenario: Legacy inventory shows CLI ownership

- **WHEN** the user runs `opm instance list`
- **AND** a instance inventory has no `createdBy`
- **THEN** the instance row SHALL display `cli` in the OWNER column

### Requirement: List command default table output

The default output format SHALL be a table with columns: NAME, MODULE, OWNER, VERSION, STATUS, AGE. When `-A` is used, a NAMESPACE column SHALL be prepended. Results SHALL be sorted alphabetically by instance name. The table SHALL use space-padded columns consistent with kubectl output conventions.

#### Scenario: Default table columns

- **WHEN** the user runs `opm instance list -n production`
- **THEN** the table SHALL have columns: NAME, MODULE, OWNER, VERSION, STATUS, AGE

#### Scenario: All-namespaces table columns

- **WHEN** the user runs `opm instance list -A`
- **THEN** the table SHALL have columns: NAMESPACE, NAME, MODULE, OWNER, VERSION, STATUS, AGE

#### Scenario: Sorted by name

- **WHEN** instances `zebra`, `alpha`, and `middle` exist
- **THEN** the table SHALL display them in order: `alpha`, `middle`, `zebra`

### Requirement: List command supports wide output

When `--output wide` / `-o wide` is specified, the table SHALL include additional columns: INSTANCE-ID and LAST-APPLIED. INSTANCE-ID SHALL display the full instance UUID. LAST-APPLIED SHALL display the `LastTransitionTime` from the instance metadata.

#### Scenario: Wide output columns without -A

- **WHEN** the user runs `opm instance list -n production -o wide`
- **THEN** the table SHALL have columns: NAME, MODULE, OWNER, VERSION, STATUS, AGE, INSTANCE-ID, LAST-APPLIED

#### Scenario: Wide output columns with -A

- **WHEN** the user runs `opm instance list -A -o wide`
- **THEN** the table SHALL have columns: NAMESPACE, NAME, MODULE, OWNER, VERSION, STATUS, AGE, INSTANCE-ID, LAST-APPLIED

### Requirement: List command supports structured output formats

The command SHALL support `--output`/`-o` with values `json` and `yaml` for machine-readable output. The structured output SHALL include all fields: name, module, namespace, owner, version, status, readyCount, totalCount, instanceID, lastApplied.

#### Scenario: JSON output

- **WHEN** the user runs `opm instance list -n production -o json`
- **THEN** the output SHALL be a valid JSON array of instance summary objects

#### Scenario: YAML output

- **WHEN** the user runs `opm instance list -n production -o yaml`
- **THEN** the output SHALL be valid YAML containing instance summary entries

### Requirement: List command namespace resolution

The `--namespace`/`-n` flag SHALL be optional. When omitted, the namespace SHALL be resolved using the precedence: flag -> `OPM_NAMESPACE` environment variable -> `~/.opm/config.cue` kubernetes.namespace -> `"default"`. The `-A` flag SHALL override any namespace selection and list across all namespaces.

#### Scenario: Namespace from config

- **WHEN** the user runs `opm instance list` without `-n` or `-A`
- **AND** the config file sets `kubernetes: namespace: "production"`
- **THEN** the command SHALL list instances in the `production` namespace

#### Scenario: -A overrides namespace

- **WHEN** the user runs `opm instance list -n production -A`
- **THEN** the command SHALL list instances across ALL namespaces, ignoring `-n`

### Requirement: List command accepts kubernetes connection flags

The command SHALL accept `--kubeconfig` and `--context` flags for cluster connection, following the same resolution precedence as the other `opm instance` commands.

#### Scenario: Custom context

- **WHEN** the user runs `opm instance list --context staging-cluster -n default`
- **THEN** the command SHALL connect to the `staging-cluster` context

### Requirement: List command evaluates health in parallel

The command SHALL evaluate instance health concurrently using a bounded worker pool to keep latency reasonable. The concurrency limit SHALL prevent overwhelming the Kubernetes API server.

#### Scenario: Multiple instances evaluated concurrently

- **WHEN** 10 instances exist in a namespace
- **THEN** the command SHALL discover resources and evaluate health for multiple instances concurrently, not sequentially

### Requirement: List metadata extraction does not depend on inventory change history

The command SHALL extract display metadata from each persisted instance inventory record: instance name from `instanceMetadata.name`, module name from `moduleMetadata.name`, version from `moduleMetadata.version`, instance ID from `instanceMetadata.uuid`, last applied time from `instanceMetadata.lastTransitionTime`, and age computed from `lastTransitionTime`. Owner display metadata SHALL come from the top-level `createdBy` field, defaulting to `cli` when that field is omitted for legacy inventories. The command SHALL NOT require inventory change-history metadata such as latest change source version, raw values, or per-change timestamps. <!-- Was: release inventory record, releaseMetadata (0002 D8/D9) -->

#### Scenario: Display metadata sourced from persisted inventory

- **WHEN** the user runs `opm instance list -n production`
- **AND** a persisted instance inventory record exists
- **THEN** each row SHALL source instance name from `instanceMetadata.name`, module name/version from `moduleMetadata`, and owner from top-level `createdBy` (defaulting to `cli` for legacy inventories)
