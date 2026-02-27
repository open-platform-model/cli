## ADDED Requirements

### Requirement: Templates import v1 catalog packages

All init templates (simple, standard, advanced) SHALL use `@v1` import paths for catalog packages:
- `"opmodel.dev/core@v1"`
- `"opmodel.dev/schemas@v1"`
- `"opmodel.dev/resources/workload@v1"`
- `"opmodel.dev/resources/storage@v1"`
- `"opmodel.dev/traits/workload@v1"`
- `"opmodel.dev/traits/network@v1"`

No `@v0` catalog import paths SHALL remain in any template file.

#### Scenario: Simple template imports

- **WHEN** `opm mod init my-app --template simple` is run
- **THEN** the generated `module.cue` SHALL import `"opmodel.dev/core@v1"`, `"opmodel.dev/schemas@v1"`, `"opmodel.dev/resources/workload@v1"`, and `"opmodel.dev/traits/workload@v1"`

#### Scenario: Standard template imports

- **WHEN** `opm mod init my-app --template standard` is run
- **THEN** the generated `components.cue` SHALL import `"opmodel.dev/resources/workload@v1"`, `"opmodel.dev/traits/workload@v1"`, and `"opmodel.dev/traits/network@v1"`

#### Scenario: Advanced template component subpackage imports

- **WHEN** `opm mod init my-app --template advanced` is run
- **THEN** each file in `components/` SHALL import from `@v1` catalog paths
- **THEN** `components.cue` SHALL import the user's subpackage as `"{{.ModulePath}}@v0/components"` (user module version stays `@v0`)

### Requirement: Templates generate v1alpha1 metadata structure

All templates SHALL generate `metadata` with `modulePath` (plain path, no version) and `name` in kebab-case. The `version` field SHALL be present in metadata. The top-level `apiVersion` and `kind` fields SHALL NOT be generated explicitly (they are inherited from `core.#Module`).

#### Scenario: Module metadata uses modulePath

- **WHEN** `opm mod init my-app` is run
- **THEN** the generated `module.cue` SHALL contain `metadata: modulePath: "example.com"` (the domain part of the module path)
- **THEN** the generated `module.cue` SHALL contain `metadata: name: "my-app"` (kebab-case)
- **THEN** the generated `module.cue` SHALL contain `metadata: version: "{{.Version}}"`
- **THEN** the generated `module.cue` SHALL NOT contain `metadata: apiVersion:`

### Requirement: Templates use Scaling trait instead of Replicas

All templates SHALL use `traits_workload.#Scaling` instead of `traits_workload.#Replicas`. The spec field SHALL be `scaling: count: <value>` instead of `replicas: <value>`.

#### Scenario: Simple template scaling

- **WHEN** `opm mod init my-app --template simple` is run
- **THEN** the generated `module.cue` SHALL contain `traits_workload.#Scaling`
- **THEN** the spec SHALL contain `scaling: count: #config.replicas`

#### Scenario: Advanced template component scaling

- **WHEN** `opm mod init my-app --template advanced` is run
- **THEN** each component template file SHALL use `traits_workload.#Scaling`
- **THEN** the spec SHALL use `scaling: count:` instead of `replicas:`

### Requirement: Templates use Volumes resource instead of PersistentVolume

The advanced template's database component SHALL use `storage_resources.#Volumes` instead of `resources_storage.#PersistentVolume`. The spec field SHALL use `volumes: <name>: { persistentClaim: { size: ... } }` instead of `storage: { name, size }`.

#### Scenario: Database component uses Volumes

- **WHEN** `opm mod init my-app --template advanced` is run
- **THEN** `components/db.cue` SHALL import `storage_resources "opmodel.dev/resources/storage@v1"`
- **THEN** `components/db.cue` SHALL use `storage_resources.#Volumes`
- **THEN** the spec SHALL contain `volumes: data: { name: "data", persistentClaim: { size: ... } }`

### Requirement: Templates generate workload-type labels

All components that use `#Container` SHALL include a `"core.opmodel.dev/workload-type"` label in their metadata.

#### Scenario: Stateless component has workload-type label

- **WHEN** `opm mod init my-app --template simple` is run
- **THEN** the `app` component SHALL contain `metadata: labels: "core.opmodel.dev/workload-type": "stateless"`

#### Scenario: Stateful component has workload-type label

- **WHEN** `opm mod init my-app --template advanced` is run
- **THEN** the `db` component SHALL contain `metadata: labels: "core.opmodel.dev/workload-type": "stateful"`

### Requirement: Templates use structured image format

All templates SHALL use the structured image format `schemas.#Image` in `#config` and provide structured values `{repository, tag, digest}` in `values.cue`.

#### Scenario: Config schema uses Image type

- **WHEN** `opm mod init my-app --template simple` is run
- **THEN** `#config` SHALL contain `image!: schemas.#Image`
- **THEN** `module.cue` SHALL import `schemas "opmodel.dev/schemas@v1"`

#### Scenario: Values provide structured image

- **WHEN** `opm mod init my-app --template simple` is run
- **THEN** `values.cue` SHALL contain `image: { repository: "busybox", tag: "latest", digest: "" }`

#### Scenario: Standard template structured images

- **WHEN** `opm mod init my-app --template standard` is run
- **THEN** `values.cue` SHALL contain `web: image: { repository: "nginx", tag: "latest", digest: "" }`
- **THEN** `values.cue` SHALL contain `api: image: { repository: "node", tag: "20-alpine", digest: "" }`

### Requirement: Templates generate container name and port name fields

All container specs SHALL include an explicit `name` field. All port definitions SHALL be struct-keyed maps.

#### Scenario: Container has name field

- **WHEN** `opm mod init my-app --template simple` is run
- **THEN** the container spec SHALL contain `name: "app"`

#### Scenario: Port has name via struct key

- **WHEN** `opm mod init my-app --template standard` is run
- **THEN** ports SHALL be defined as struct-keyed maps (e.g., `ports: http: { targetPort: 80 }`)

### Requirement: User module cue.mod stays at v0

The generated `cue.mod/module.cue` SHALL declare the user's module at `@v0`: `module: "{{.ModulePath}}@v0"`. The user's module version is independent of the catalog version they import.

#### Scenario: Module path version unchanged

- **WHEN** `opm mod init my-app` is run
- **THEN** `cue.mod/module.cue` SHALL contain `module: "example.com/my-app@v0"`
