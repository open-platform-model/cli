## MODIFIED Requirements

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

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Status falls back to label scan

**Reason**: Label-scan returns incorrect results — Kubernetes-generated child resources (Endpoints, EndpointSlice, ReplicaSets, Pods) carry inherited OPM labels but were never applied by OPM. A release without an inventory Secret is not a valid OPM release.

**Migration**: Ensure `opm mod apply` has been run at least once to create the inventory Secret. Any release applied with the current CLI has an inventory and is unaffected.
