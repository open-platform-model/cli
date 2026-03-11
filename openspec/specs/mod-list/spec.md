## Purpose

Defines how `opm mod list` discovers deployed releases from persisted release inventory records and reports release metadata and health without requiring module source.

## Requirements

### Requirement: List command discovers releases via persisted ownership inventory

The `opm mod list` command SHALL discover all deployed module releases by listing persisted release inventory records in the target namespace. It SHALL use the `ListInventories` function from the inventory package. It MUST NOT require module source, re-rendering, knowledge of specific release names, or inventory change-history fields to identify the current owned resource set for a release.

#### Scenario: List releases in a namespace

- **WHEN** the user runs `opm mod list -n production`
- **AND** 3 persisted release inventory records exist in the `production` namespace
- **THEN** the command SHALL display all 3 releases

#### Scenario: No releases found

- **WHEN** the user runs `opm mod list -n empty-namespace`
- **AND** no persisted release inventory records exist in that namespace
- **THEN** the command SHALL print `No releases found in namespace "empty-namespace"` and exit with code 0

#### Scenario: List works with ownership-only inventory

- **WHEN** the user runs `opm mod list` in a namespace containing releases with ownership-only inventory
- **THEN** the command SHALL still discover those releases and evaluate their health from the tracked resource set

### Requirement: List command supports all-namespaces flag

The command SHALL accept `-A` / `--all-namespaces` to list releases across all namespaces. When `-A` is used, the NAMESPACE column SHALL be included in the table output. When `-A` is not used, the NAMESPACE column SHALL be hidden.

#### Scenario: All-namespaces listing

- **WHEN** the user runs `opm mod list -A`
- **AND** releases exist in namespaces `media` and `games`
- **THEN** the output SHALL include releases from both namespaces
- **AND** the table SHALL include a NAMESPACE column

#### Scenario: Single namespace hides namespace column

- **WHEN** the user runs `opm mod list -n media`
- **THEN** the table output SHALL NOT include a NAMESPACE column

### Requirement: List command shows health status for each release

The command SHALL evaluate the health of each release by discovering its tracked resources and evaluating their health status. The STATUS column SHALL display the aggregate health and ready/total count in the format `Ready (N/N)` or `NotReady (N/N)`.

#### Scenario: All resources healthy

- **WHEN** a release has 5 tracked resources and all are healthy
- **THEN** the STATUS column SHALL display `Ready (5/5)`

#### Scenario: Some resources unhealthy

- **WHEN** a release has 5 tracked resources and 2 are not healthy
- **THEN** the STATUS column SHALL display `NotReady (3/5)`

#### Scenario: Missing resource counts as unhealthy

- **WHEN** a release tracks a resource that no longer exists on the cluster
- **THEN** that resource SHALL count toward the total but NOT toward the ready count

#### Scenario: Zero resources

- **WHEN** a release inventory record has no tracked resources in its ownership inventory
- **THEN** the STATUS column SHALL display `Unknown (0/0)`

### Requirement: List command displays release ownership

The `opm mod list` command SHALL expose release ownership derived from inventory provenance. Table outputs SHALL include an OWNER column, and structured outputs SHALL include an `owner` field.

#### Scenario: Table output shows controller ownership

- **WHEN** the user runs `opm mod list`
- **AND** a release inventory records `createdBy: "controller"`
- **THEN** the release row SHALL display `controller` in the OWNER column

#### Scenario: Legacy inventory shows CLI ownership

- **WHEN** the user runs `opm mod list`
- **AND** a release inventory has no `createdBy`
- **THEN** the release row SHALL display `cli` in the OWNER column

### Requirement: List command default table output

The default output format SHALL be a table with columns: NAME, MODULE, OWNER, VERSION, STATUS, AGE. When `-A` is used, a NAMESPACE column SHALL be prepended. Results SHALL be sorted alphabetically by release name. The table SHALL use space-padded columns consistent with kubectl output conventions.

#### Scenario: Default table columns

- **WHEN** the user runs `opm mod list -n production`
- **THEN** the table SHALL have columns: NAME, MODULE, OWNER, VERSION, STATUS, AGE

#### Scenario: All-namespaces table columns

- **WHEN** the user runs `opm mod list -A`
- **THEN** the table SHALL have columns: NAMESPACE, NAME, MODULE, OWNER, VERSION, STATUS, AGE

#### Scenario: Sorted by name

- **WHEN** releases `zebra`, `alpha`, and `middle` exist
- **THEN** the table SHALL display them in order: `alpha`, `middle`, `zebra`

### Requirement: List command supports wide output

When `--output wide` / `-o wide` is specified, the table SHALL include additional columns: RELEASE-ID and LAST-APPLIED. RELEASE-ID SHALL display the full release UUID. LAST-APPLIED SHALL display the `LastTransitionTime` from the release metadata.

#### Scenario: Wide output columns without -A

- **WHEN** the user runs `opm mod list -n production -o wide`
- **THEN** the table SHALL have columns: NAME, MODULE, OWNER, VERSION, STATUS, AGE, RELEASE-ID, LAST-APPLIED

#### Scenario: Wide output columns with -A

- **WHEN** the user runs `opm mod list -A -o wide`
- **THEN** the table SHALL have columns: NAMESPACE, NAME, MODULE, OWNER, VERSION, STATUS, AGE, RELEASE-ID, LAST-APPLIED

### Requirement: List command supports structured output formats

The command SHALL support `--output`/`-o` with values `json` and `yaml` for machine-readable output. The structured output SHALL include all fields: name, module, namespace, owner, version, status, readyCount, totalCount, releaseID, lastApplied.

#### Scenario: JSON output

- **WHEN** the user runs `opm mod list -n production -o json`
- **THEN** the output SHALL be a valid JSON array of release summary objects

#### Scenario: YAML output

- **WHEN** the user runs `opm mod list -n production -o yaml`
- **THEN** the output SHALL be valid YAML containing release summary entries

### Requirement: List command namespace resolution

The `--namespace`/`-n` flag SHALL be optional. When omitted, the namespace SHALL be resolved using the precedence: flag -> `OPM_NAMESPACE` environment variable -> `~/.opm/config.cue` kubernetes.namespace -> `"default"`. The `-A` flag SHALL override any namespace selection and list across all namespaces.

#### Scenario: Namespace from config

- **WHEN** the user runs `opm mod list` without `-n` or `-A`
- **AND** the config file sets `kubernetes: namespace: "production"`
- **THEN** the command SHALL list releases in the `production` namespace

#### Scenario: -A overrides namespace

- **WHEN** the user runs `opm mod list -n production -A`
- **THEN** the command SHALL list releases across ALL namespaces, ignoring `-n`

### Requirement: List command accepts kubernetes connection flags

The command SHALL accept `--kubeconfig` and `--context` flags for cluster connection, following the same resolution precedence as other `opm mod` commands.

#### Scenario: Custom context

- **WHEN** the user runs `opm mod list --context staging-cluster -n default`
- **THEN** the command SHALL connect to the `staging-cluster` context

### Requirement: List command evaluates health in parallel

The command SHALL evaluate release health concurrently using a bounded worker pool to keep latency reasonable. The concurrency limit SHALL prevent overwhelming the Kubernetes API server.

#### Scenario: Multiple releases evaluated concurrently

- **WHEN** 10 releases exist in a namespace
- **THEN** the command SHALL discover resources and evaluate health for multiple releases concurrently, not sequentially

### Requirement: List metadata extraction does not depend on inventory change history

The command SHALL extract display metadata from each persisted release inventory record: release name from `releaseMetadata.name`, module name from `moduleMetadata.name`, version from `moduleMetadata.version`, release ID from `releaseMetadata.uuid`, last applied time from `releaseMetadata.lastTransitionTime`, and age computed from `lastTransitionTime`. Owner display metadata SHALL come from the top-level `createdBy` field, defaulting to `cli` when that field is omitted for legacy inventories. The command SHALL NOT require inventory change-history metadata such as latest change source version, raw values, or per-change timestamps.

#### Scenario: List remains functional with ownership-only inventory

- **WHEN** a release has ownership-only inventory and no latest change entry
- **THEN** `opm mod list` SHALL still be able to display the release and compute health from tracked resources

#### Scenario: List reads deployed module version from module metadata

- **WHEN** a persisted release inventory record includes `moduleMetadata.version`
- **THEN** the VERSION column SHALL display that value

#### Scenario: No version recorded

- **WHEN** a persisted release inventory record omits `moduleMetadata.version`
- **THEN** the VERSION column SHALL display `-`
