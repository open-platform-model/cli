## ADDED Requirements

### Requirement: Module metadata follows OPM conventions

The module SHALL declare valid OPM metadata including `apiVersion`, `name`, `version`, `description`, and `defaultNamespace`. The module SHALL unify with `core.#Module`.

#### Scenario: Module passes CUE validation

- **WHEN** `cue vet ./...` is run against the `testing/jellyfin/` directory with the OPM schema registry
- **THEN** validation SHALL succeed with no errors

#### Scenario: Module metadata is complete

- **WHEN** the module is loaded
- **THEN** `metadata.name` SHALL be `"jellyfin"`
- **THEN** `metadata.version` SHALL be a valid SemVer string
- **THEN** `metadata.apiVersion` SHALL follow the `<domain>/<name>@v<N>` pattern

### Requirement: Module uses three-file convention

The module SHALL separate concerns across three CUE files: `module.cue` (metadata and config schema), `components.cue` (component definitions), and `values.cue` (concrete default values). A `cue.mod/module.cue` SHALL declare the CUE module path and dependencies.

#### Scenario: File structure matches convention

- **WHEN** the `testing/jellyfin/` directory is listed
- **THEN** it SHALL contain `module.cue`, `components.cue`, `values.cue`, and `cue.mod/module.cue`

### Requirement: Config schema validates user input

The module SHALL define a `#config` schema that constrains all user-configurable values. The schema SHALL enforce type safety for all fields at definition time.

#### Scenario: Image field is required

- **WHEN** a user omits the `image` field from values
- **THEN** CUE validation SHALL fail because `image` is constrained as `string` with no default

#### Scenario: Port is bounded

- **WHEN** a user sets `port` to a value outside the range 1-65535
- **THEN** CUE validation SHALL fail

#### Scenario: PUID and PGID have defaults

- **WHEN** a user does not specify `puid` or `pgid`
- **THEN** the values SHALL default to `1000`

#### Scenario: Timezone is required

- **WHEN** a user omits the `timezone` field
- **THEN** CUE validation SHALL fail because `timezone` is constrained as `string` with no default

### Requirement: Config storage uses a persistent volume claim

The module SHALL define a persistent volume for the Jellyfin `/config` directory. The PVC size SHALL be configurable via the `#config` schema with a sensible default.

#### Scenario: Config PVC is provisioned

- **WHEN** the module is rendered with default values
- **THEN** a volume named `config` SHALL exist with a `persistentClaim` of size `10Gi`

#### Scenario: Config PVC size is overridable

- **WHEN** a user sets `configStorageSize` to `"50Gi"`
- **THEN** the config volume's `persistentClaim.size` SHALL be `"50Gi"`

### Requirement: Media libraries are modeled as named volume mounts

The module SHALL define media libraries as a struct-keyed map in `#config` where each key is a library name and the value specifies a `mountPath`. Each media library entry SHALL produce a corresponding volume and volume mount on the container.

#### Scenario: Default media libraries are present

- **WHEN** the module is rendered with default values
- **THEN** at least two media library volumes SHALL exist (e.g., `tvshows` and `movies`)
- **THEN** each SHALL be mounted at its specified `mountPath` under `/data/`

#### Scenario: User adds a custom media library

- **WHEN** a user adds `music: { mountPath: "/data/music" }` to the `media` config
- **THEN** a volume named `music` SHALL be created
- **THEN** it SHALL be mounted at `/data/music` on the container

### Requirement: Component is labeled as stateful workload

The component SHALL carry the label `"core.opmodel.dev/workload-type": "stateful"`. This ensures the Kubernetes provider renders a StatefulSet.

#### Scenario: Workload type label is set

- **WHEN** the module's component metadata is inspected
- **THEN** the label `core.opmodel.dev/workload-type` SHALL have the value `"stateful"`

### Requirement: Web UI is exposed via network service

The module SHALL use the `#Expose` trait to expose the Jellyfin web UI port. The service type SHALL default to `ClusterIP`.

#### Scenario: HTTP port is exposed

- **WHEN** the module is rendered with default values
- **THEN** port 8096 SHALL be exposed as a service
- **THEN** the service type SHALL be `"ClusterIP"`

#### Scenario: Exposed port is configurable

- **WHEN** a user overrides the `port` config value to `9096`
- **THEN** the exposed port SHALL reflect the new value

### Requirement: Health checks probe the Jellyfin API

The module SHALL define both liveness and readiness probes using HTTP GET against the Jellyfin health endpoint.

#### Scenario: Liveness probe is configured

- **WHEN** the module's component spec is inspected
- **THEN** a liveness probe SHALL exist using `httpGet` on path `/health` at port `8096`
- **THEN** `initialDelaySeconds` SHALL be at least `30` (Jellyfin startup can be slow)

#### Scenario: Readiness probe is configured

- **WHEN** the module's component spec is inspected
- **THEN** a readiness probe SHALL exist using `httpGet` on path `/health` at port `8096`
- **THEN** `initialDelaySeconds` SHALL be at least `5`

### Requirement: LinuxServer environment variables are configured

The module SHALL map `puid`, `pgid`, and `timezone` config fields to the container's `PUID`, `PGID`, and `TZ` environment variables respectively. An optional `publishedServerUrl` SHALL map to `JELLYFIN_PublishedServerUrl` when provided.

#### Scenario: Default environment is set

- **WHEN** the module is rendered with default values
- **THEN** the container SHALL have env var `PUID` with value from `puid` config
- **THEN** the container SHALL have env var `PGID` with value from `pgid` config
- **THEN** the container SHALL have env var `TZ` with value from `timezone` config

#### Scenario: Published server URL is optional

- **WHEN** `publishedServerUrl` is not set in config
- **THEN** the `JELLYFIN_PublishedServerUrl` env var SHALL NOT be present on the container

#### Scenario: Published server URL is set

- **WHEN** `publishedServerUrl` is set to `"http://192.168.1.100:8096"`
- **THEN** the container SHALL have env var `JELLYFIN_PublishedServerUrl` with that value

### Requirement: Scaling is fixed to a single replica

The module SHALL use the `#Scaling` trait with `count: 1`. The config schema SHALL NOT expose replica count as a user-configurable value because Jellyfin does not support horizontal scaling.

#### Scenario: Single replica enforced

- **WHEN** the module is rendered
- **THEN** the scaling count SHALL be `1`

### Requirement: Default values produce a valid deployable module

The `values.cue` file SHALL provide concrete defaults for all required `#config` fields such that the module is valid and deployable without any user overrides.

#### Scenario: Module validates with defaults only

- **WHEN** `cue vet ./...` is run with no user overrides
- **THEN** validation SHALL succeed
- **THEN** all `#config` fields SHALL have concrete values
