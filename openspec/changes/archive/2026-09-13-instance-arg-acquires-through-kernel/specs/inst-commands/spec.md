## MODIFIED Requirements

### Requirement: instance cluster-query commands accept an instance identifier as positional argument

The cluster-query commands (`status`, `tree`, `events`, `delete`) under `opm instance` SHALL accept an instance identifier as a required positional argument. The CLI SHALL auto-detect whether the identifier is a path, a UUID or an instance name. A path (an `instance.cue` file, or a directory holding one) SHALL be acquired through the library kernel as an instance package with the invocation's resolved registry, and the instance name and namespace SHALL be read from the acquired instance's metadata; the `--namespace` flag SHALL take precedence over the namespace in the file. A path whose package is not a valid instance (wrong kind, no concrete `metadata.name` or `metadata.namespace`, build failure) SHALL be refused with exit code 1 and an error naming the path.

#### Scenario: status by instance name

- **WHEN** `opm instance status jellyfin` is run
- **THEN** the CLI SHALL look up the instance by name via label scan on inventory Secrets
- **AND** display the instance health status

#### Scenario: status by UUID

- **WHEN** `opm instance status 550e8400-e29b-41d4-a716-446655440000` is run
- **THEN** the CLI SHALL look up the instance by UUID via direct Secret GET
- **AND** display the instance health status

#### Scenario: status by instance path

- **WHEN** `opm instance status ./jellyfin/instance.cue` is run and the package declares `metadata.name: "jellyfin"` and `metadata.namespace: "media"`
- **THEN** the CLI SHALL acquire the package through the kernel and look up the instance named `jellyfin` in the `media` namespace

#### Scenario: namespace flag overrides the file

- **WHEN** `opm instance status ./jellyfin -n staging` is run and the package declares `metadata.namespace: "media"`
- **THEN** the CLI SHALL look up the instance in the `staging` namespace

#### Scenario: path that is not a valid instance is refused

- **WHEN** `opm instance status ./jellyfin` is run and the package's `metadata.namespace` is not concrete, or its `kind` is not `ModuleInstance`
- **THEN** the CLI SHALL exit with code 1 and an error naming the path and the refusal

#### Scenario: delete by instance name

- **WHEN** `opm instance delete jellyfin` is run
- **THEN** the CLI SHALL look up the instance by name and delete its resources from the cluster

#### Scenario: identifier auto-detection

- **WHEN** the positional argument exists on disk, ends with `.cue`, contains a path separator, or starts with `.` or `~`
- **THEN** the CLI SHALL treat it as a path
- **WHEN** the positional argument matches the pattern `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
- **THEN** the CLI SHALL treat it as a UUID
- **WHEN** the positional argument is neither
- **THEN** the CLI SHALL treat it as an instance name
