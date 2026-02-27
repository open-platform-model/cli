## ADDED Requirements

### Requirement: Test fixtures use v1alpha1-compatible metadata structure

All CUE test fixtures in `internal/loader/testdata/`, `internal/pipeline/testdata/`, and `tests/fixtures/valid/` SHALL use the v1alpha1 metadata structure: `metadata.modulePath` (not `metadata.apiVersion`), kebab-case `metadata.name`, and `metadata.version`.

#### Scenario: Loader test module has modulePath

- **WHEN** the test module at `internal/loader/testdata/test-module/module.cue` is loaded
- **THEN** the metadata SHALL contain `modulePath` (not `apiVersion` or `cueModulePath`)
- **THEN** the metadata `name` SHALL be kebab-case
- **THEN** the metadata SHALL contain `version` (semver for modules)

#### Scenario: Pipeline test module has modulePath

- **WHEN** the test module at `internal/pipeline/testdata/test-module/module.cue` is loaded
- **THEN** the metadata SHALL contain `modulePath` (not `apiVersion` or `cueModulePath`)

### Requirement: Test fixtures use v1alpha1 FQN format for resources and traits

Resource and trait FQN keys in test fixture `#resources` and `#traits` maps SHALL use the v1alpha1 primitive FQN format `<modulePath>/<name>@<majorVersion>` (e.g., `"opmodel.dev/resources/workload/container@v1"`).

#### Scenario: Resource FQN in v1alpha1 format

- **WHEN** a test fixture defines `#resources`
- **THEN** the keys SHALL use `path/name@version` format (e.g., `"opmodel.dev/resources/workload/container@v1"`)
- **THEN** no `@v0` FQN keys SHALL remain
- **THEN** no `#PascalName` suffix SHALL appear in FQN keys

#### Scenario: Trait FQN in v1alpha1 format

- **WHEN** a test fixture defines `#traits`
- **THEN** the keys SHALL use `path/name@version` format (e.g., `"opmodel.dev/traits/network/expose@v1"`)

### Requirement: Module FQN uses container-style format

Module metadata `fqn` SHALL use the container-style format `modulePath/name:semver` (e.g., `"opmodel.dev/modules/test-module:0.1.0"`). This is `#ModuleFQNType`, distinct from primitive `#FQNType`.

#### Scenario: Module FQN computed correctly

- **WHEN** a test fixture defines a module with `modulePath: "example.com"`, `name: "my-app"`, `version: "1.0.0"`
- **THEN** `metadata.fqn` SHALL equal `"example.com/my-app:1.0.0"`

### Requirement: Go tests assert v1alpha1 values

All Go test assertions for FQN strings, apiVersion values, provider apiVersions, and metadata fields SHALL use v1alpha1 values. No test SHALL assert on `@v0` FQN strings or legacy metadata field names.

#### Scenario: Component test FQN assertions

- **WHEN** `internal/core/component/component_test.go` asserts on resource FQNs
- **THEN** the expected FQN values SHALL use `path/name@version` format (e.g., `"opmodel.dev/resources/workload/container@v1"`)

#### Scenario: Provider test apiVersion assertions

- **WHEN** `internal/loader/provider_test.go` asserts on provider apiVersion
- **THEN** the expected value SHALL be `"core.opmodel.dev/v1alpha1"` (not `"opmodel.dev/providers/kubernetes@v0"`)

#### Scenario: Pipeline test provider CUE strings

- **WHEN** `internal/pipeline/pipeline_test.go` uses inline CUE provider definitions
- **THEN** the CUE strings SHALL use `@v1` resource/trait FQNs in `path/name@version` format

### Requirement: Example modules rewritten for v1alpha1

All 9 example modules SHALL be rewritten with v1alpha1 schema:
- Imports use `@v1` catalog paths
- Metadata uses `modulePath` and kebab-case `name`
- Components use `#Scaling` (not `#Replicas`), `#Volumes` (not `#PersistentVolume`)
- Components using `#Container` include `"core.opmodel.dev/workload-type"` label
- Container images use structured `{repository, tag, digest}` format
- Port/env definitions use struct-keyed maps
- `cue.mod/module.cue` deps reference `@v1` packages (or are left empty for `cue mod tidy`)

#### Scenario: Jellyfin example uses v1alpha1 schema

- **WHEN** `examples/jellyfin/module.cue` is inspected
- **THEN** it SHALL import `"opmodel.dev/core@v1"`
- **THEN** it SHALL NOT contain any `@v0` import paths

#### Scenario: Example components have workload-type labels

- **WHEN** any example component uses `resources_workload.#Container`
- **THEN** it SHALL have `metadata: labels: "core.opmodel.dev/workload-type": <type>` set

#### Scenario: Example values use structured images

- **WHEN** any example `values.cue` provides an image value
- **THEN** it SHALL use `{repository: "...", tag: "...", digest: ""}` format (not a plain string)

### Requirement: Documentation comments updated

Doc comments in `internal/pipeline/types.go` and other files that contain example FQN strings SHALL be updated to use v1alpha1 format.

#### Scenario: Pipeline types.go comments

- **WHEN** `internal/pipeline/types.go` is inspected
- **THEN** example FQN strings in comments SHALL use `path/name@version` format (e.g., `"opmodel.dev/resources/workload/container@v1"`)
