## Purpose

Defines how `opm mod status` discovers tracked resources from persisted release inventory records and reports release health without requiring module source.

## Requirements

### Requirement: Status discovers resources via ownership inventory

The `opm mod status` command SHALL read the persisted release inventory record for the release to discover its tracked resources. If `--release-id` is provided, it SHALL use `inventory.GetInventory` (direct GET by name, with UUID label fallback). If only `--release-name` is provided, it SHALL use `inventory.FindInventoryByReleaseName` (inventory-record lookup by release-name label). Once the inventory is found, it SHALL perform one targeted GET per tracked entry via `inventory.DiscoverResourcesFromInventory`. It MUST NOT require module source or re-rendering. It MUST NOT use a cluster-wide label-scan to discover workload resources.

#### Scenario: Status shows deployed resources via ownership inventory (release-name path)

- **WHEN** the user runs `opm mod status --release-name my-app -n production`
- **AND** a persisted release inventory record exists labeled `module-release.opmodel.dev/name=my-app`
- **THEN** the command SHALL fetch each tracked resource via targeted GET
- **AND** only resources explicitly tracked in the ownership inventory SHALL appear in the output

#### Scenario: Status shows deployed resources via ownership inventory (release-id path)

- **WHEN** the user runs `opm mod status --release-id <uuid> -n production`
- **AND** a persisted release inventory record exists with that UUID
- **THEN** the command SHALL fetch each tracked resource via targeted GET

#### Scenario: Release not found

- **WHEN** the user runs `opm mod status --release-name my-app -n production`
- **AND** no persisted release inventory record exists for that release name in that namespace
- **THEN** the command SHALL exit with error: `"release 'my-app' not found in namespace 'production'"`

#### Scenario: Kubernetes-generated children not shown

- **WHEN** a release includes a Service that caused Kubernetes to create Endpoints and EndpointSlice resources
- **AND** those child resources were not tracked by the inventory
- **THEN** `opm mod status` SHALL NOT include Endpoints or EndpointSlice in the output

### Requirement: Status evaluates health per resource category

The command SHALL evaluate resource health using category-specific criteria:
- **Workloads** (Deployment, StatefulSet, DaemonSet): healthy when `Ready` condition is True
- **Jobs** (Job): healthy when `Complete` condition is True
- **CronJobs**: always reported as healthy (scheduled)
- **Passive** (ConfigMap, Secret, Service, PVC): healthy on creation
- **Custom** (CRD instances): healthy if `Ready` condition is present and True, otherwise treated as passive

#### Scenario: Deployment not yet ready

- **WHEN** a Deployment has `Ready` condition set to False
- **THEN** the status output SHALL show the Deployment as "NotReady"

#### Scenario: Job completed

- **WHEN** a Job has `Complete` condition set to True
- **THEN** the status output SHALL show the Job as "Complete"

#### Scenario: ConfigMap always healthy

- **WHEN** a ConfigMap exists in the cluster
- **THEN** the status output SHALL show the ConfigMap as "Ready"

#### Scenario: Custom resource with Ready condition

- **WHEN** a CRD instance has a `Ready` condition set to True
- **THEN** the status output SHALL show the resource as "Ready"

#### Scenario: Custom resource without Ready condition

- **WHEN** a CRD instance has no `Ready` condition in its status
- **THEN** the status output SHALL treat the resource as passive and show it as "Ready"

### Requirement: Status supports table output format

The default output format SHALL be a table showing: resource kind, name, namespace, component, health status, and age. The COMPONENT column SHALL be populated from the inventory entry for each resource. Resources without a component SHALL display `-` in the COMPONENT column. Columns SHALL be aligned and human-readable. The table SHALL render as plain, space-padded columns with no border characters, consistent with kubectl output conventions.

The `--output`/`-o` flag SHALL accept `wide` as a valid value in addition to `table`, `yaml`, and `json`. When `-o wide` is specified, the command SHALL render a table format with additional columns.

#### Scenario: Default table output includes component column

- **WHEN** the user runs `opm mod status --release-name my-app -n production` without `--output`
- **THEN** the output SHALL be a formatted table with KIND, NAME, COMPONENT, STATUS, and AGE columns
- **AND** the COMPONENT column SHALL show the component name from the inventory entry for each resource

#### Scenario: Resource without component

- **WHEN** a resource does not have a component recorded in the inventory
- **THEN** the COMPONENT column SHALL display `-` for that resource

#### Scenario: Wide format accepted as output value

- **WHEN** the user runs `opm mod status --release-name my-app -n production -o wide`
- **THEN** the command SHALL render a table with additional columns beyond the default format

### Requirement: Status header does not depend on inventory change history

The status output SHALL display a metadata header above the resource table containing release name, namespace, aggregate health status, and a resource summary. Module version and ownership metadata SHALL come from the persisted release inventory record (`releaseMetadata`, `moduleMetadata`, `createdBy`) when present. The command SHALL NOT require inventory change-history metadata such as source version, raw values, or per-change timestamps, and it MUST NOT require module source or re-rendering.

The header SHALL include:
- **Release**: from the release name resolved from the persisted release inventory record
- **Version**: from `moduleMetadata.version` when present (omitted when not recorded)
- **Namespace**: from the resolved Kubernetes configuration
- **Status**: the aggregate health status (Ready, NotReady, Unknown)
- **Resources**: total count with breakdown (e.g., "6 total (6 ready)" or "6 total (5 ready, 1 not ready)")

#### Scenario: Status remains functional with ownership-only inventory

- **WHEN** a release has ownership-only inventory and no history-bearing inventory fields
- **THEN** `opm mod status` SHALL still be able to enumerate resources and show their health

#### Scenario: Status reads deployed module version from module metadata

- **WHEN** a persisted release inventory record includes `moduleMetadata.version`
- **THEN** the metadata header SHALL use that field for deployed module version display

#### Scenario: Header shows release metadata

- **WHEN** the user runs `opm mod status --release-name jellyfin -n media`
- **AND** the persisted release inventory record includes `moduleMetadata.version: "1.2.0"`
- **THEN** the output SHALL begin with a header showing `Release: jellyfin`, `Version: 1.2.0`, `Namespace: media`, the aggregate status, and a resource count summary

#### Scenario: Header omits version when not recorded in inventory

- **WHEN** the persisted release inventory record omits `moduleMetadata.version`
- **THEN** the header SHALL omit the Version line entirely

#### Scenario: Header shows not ready count

- **WHEN** 2 out of 6 resources have a health status of NotReady
- **THEN** the Resources line SHALL display "6 total (4 ready, 2 not ready)"

### Requirement: Status header displays release ownership

The `opm mod status` command SHALL display release ownership derived from inventory provenance in the metadata header.

#### Scenario: Header shows controller ownership

- **WHEN** the user runs `opm mod status` for a release whose inventory records `createdBy: "controller"`
- **THEN** the metadata header SHALL include `Owner: controller`

#### Scenario: Header shows legacy CLI ownership

- **WHEN** the user runs `opm mod status` for a release whose inventory has no `createdBy`
- **THEN** the metadata header SHALL include `Owner: cli`

### Requirement: Status warns for non-CLI-managed releases

When the CLI reads a controller-managed release, `opm mod status` SHALL surface a warning that the release is controller-managed and cannot be mutated by the CLI.

#### Scenario: Controller-managed warning

- **WHEN** the user runs `opm mod status` for a controller-managed release
- **THEN** the command SHALL display a warning indicating that the release is controller-managed
- **AND** the command SHALL still show the release status information

### Requirement: Status output uses color

The status table and header SHALL use color-coded output when stdout is a TTY. Color SHALL be disabled when stdout is not a TTY or when the `NO_COLOR` environment variable is set.

The color mapping SHALL be:
- Health status `Ready` and `Complete`: green
- Health status `NotReady` and `Missing`: red
- Health status `Unknown`: yellow
- Component names: cyan
- Resource names: cyan
- Structural elements (borders, separators): dim gray

#### Scenario: Color output on TTY

- **WHEN** the user runs `opm mod status` with stdout connected to a TTY
- **AND** the `NO_COLOR` environment variable is not set
- **THEN** the STATUS column SHALL render `Ready` in green and `NotReady` in red
- **AND** the COMPONENT column SHALL render component names in cyan

#### Scenario: Color disabled on pipe

- **WHEN** the user pipes the output (e.g., `opm mod status ... | cat`)
- **THEN** all color/ANSI escape codes SHALL be stripped from the output

#### Scenario: Color disabled by NO_COLOR

- **WHEN** the `NO_COLOR` environment variable is set
- **THEN** all color/ANSI escape codes SHALL be stripped from the output

### Requirement: Status supports structured output formats

The command SHALL support `--output`/`-o` with values `table` (default), `yaml`, and `json` for machine-readable output.

#### Scenario: JSON output

- **WHEN** the user runs `opm mod status -n ns --name mod -o json`
- **THEN** the output SHALL be a valid JSON array of resource status objects

#### Scenario: YAML output

- **WHEN** the user runs `opm mod status -n ns --name mod -o yaml`
- **THEN** the output SHALL be valid YAML containing resource status entries

### Requirement: Status supports watch mode

The command SHALL support `--watch` for continuous monitoring. In watch mode, the status table SHALL refresh at a regular interval (2 seconds), clearing the previous output and displaying the updated table.

#### Scenario: Watch mode updates on change

- **WHEN** the user runs `opm mod status -n ns --name mod --watch`
- **THEN** the status table SHALL refresh every 2 seconds until the user interrupts (Ctrl+C)

#### Scenario: Watch mode exits cleanly on interrupt

- **WHEN** the user presses Ctrl+C during watch mode
- **THEN** the command SHALL exit with code 0 and restore the terminal state

### Requirement: Namespace defaults to config

The `--namespace`/`-n` flag SHALL be optional for `opm mod status`. When omitted, the namespace SHALL be resolved using the precedence: flag → `OPM_NAMESPACE` environment variable → `~/.opm/config.cue` kubernetes.namespace → `"default"`.

#### Scenario: Namespace omitted uses config default

- **WHEN** the user runs `opm mod status --release-name my-app` without `-n`
- **AND** the config file sets `kubernetes: namespace: "production"`
- **THEN** the command SHALL operate in the `production` namespace

#### Scenario: Namespace omitted uses hardcoded default

- **WHEN** the user runs `opm mod status --release-name my-app` without `-n`
- **AND** no config or env sets a namespace
- **THEN** the command SHALL operate in the `default` namespace

### Requirement: Status accepts kubernetes connection flags

The command SHALL accept `--kubeconfig` and `--context` flags for cluster connection. Kubeconfig resolution SHALL follow: explicit flag > `KUBECONFIG` env var > default path (`~/.kube/config`).

#### Scenario: Custom kubeconfig

- **WHEN** the user runs `opm mod status --kubeconfig /path/to/config -n ns --name mod`
- **THEN** the command SHALL use the specified kubeconfig file

### Requirement: Status fails fast on connectivity errors

The command SHALL fail immediately with a clear error message if the Kubernetes cluster is unreachable.

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with code 3 and display a connectivity error message

### Requirement: Status groups resources by component from inventory

When a persisted release inventory record is available, the `opm mod status` command SHALL group resources by the `component` field from inventory entries. This eliminates the need to read `component.opmodel.dev/name` labels from the cluster.

#### Scenario: Resources grouped by component

- **WHEN** the user runs `opm mod status` and a persisted release inventory record exists
- **AND** the ownership inventory tracks 3 resources under component `app` and 2 under component `cache`
- **THEN** the output SHALL group the resources by component name

#### Scenario: Missing resource shown in status

- **WHEN** the ownership inventory tracks a resource that no longer exists on the cluster
- **THEN** the status output SHALL show the resource with status "Missing" or equivalent
