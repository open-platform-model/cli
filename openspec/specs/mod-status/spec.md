
## Requirements

### Requirement: Status discovers resources via inventory

The `opm mod status` command SHALL read the inventory Secret for the release to discover its resources. If `--release-id` is provided, it SHALL use `inventory.GetInventory` (direct GET by name, with UUID label fallback). If only `--release-name` is provided, it SHALL use `inventory.FindInventoryByReleaseName` (label scan on inventory Secrets only). Once the inventory is found, it SHALL perform one targeted GET per tracked entry via `inventory.DiscoverResourcesFromInventory`. It MUST NOT require module source or re-rendering. It MUST NOT use a cluster-wide label-scan to discover workload resources.

#### Scenario: Status shows deployed resources via inventory (release-name path)

- **WHEN** the user runs `opm mod status --release-name my-app -n production`
- **AND** an inventory Secret exists labeled `module-release.opmodel.dev/name=my-app`
- **THEN** the command SHALL fetch each tracked resource via targeted GET
- **AND** only resources explicitly tracked in the inventory SHALL appear in the output

#### Scenario: Status shows deployed resources via inventory (release-id path)

- **WHEN** the user runs `opm mod status --release-id <uuid> -n production`
- **AND** an inventory Secret exists with that UUID
- **THEN** the command SHALL fetch each tracked resource via targeted GET

#### Scenario: Release not found

- **WHEN** the user runs `opm mod status --release-name my-app -n production`
- **AND** no inventory Secret exists for that release name in that namespace
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

The default output format SHALL be a table showing: resource kind, name, namespace, health status, and age. Columns SHALL be aligned and human-readable.

#### Scenario: Default table output

- **WHEN** the user runs `opm mod status -n ns --name mod` without `--output`
- **THEN** the output SHALL be a formatted table with KIND, NAME, NAMESPACE, STATUS, and AGE columns

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

When an inventory Secret is available, the `opm mod status` command SHALL group resources by the `component` field from inventory entries. This eliminates the need to read `component.opmodel.dev/name` labels from the cluster.

#### Scenario: Resources grouped by component

- **WHEN** the user runs `opm mod status` and an inventory exists
- **AND** the inventory tracks 3 resources under component `app` and 2 under component `cache`
- **THEN** the output SHALL group the resources by component name

#### Scenario: Missing resource shown in status

- **WHEN** the inventory tracks a resource that no longer exists on the cluster
- **THEN** the status output SHALL show the resource with status "Missing" or equivalent
