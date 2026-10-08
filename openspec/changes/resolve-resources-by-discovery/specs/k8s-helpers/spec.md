## REMOVED Requirements

### Requirement: Unified GVR resolution from unstructured objects

**Reason**: The requirement prescribed a guessed plural (a table of known kinds, then "lowercase + s"). A wrong guess made a delete or a prune read 404 as "already gone" and forget an object that still existed.

**Migration**: See "Resource names come from API discovery". No user action.

### Requirement: Shared resource utilities live in a dedicated file

**Reason**: The requirement listed the guessing functions (`kindToResource`, `knownKindResources`, `heuristicPluralize`) as required content of `resource.go`. They are removed.

**Migration**: See "Resource resolution has one implementation". No user action.

## ADDED Requirements

### Requirement: Resource names come from API discovery

Every command that addresses a Kubernetes object by group, version and kind SHALL take the resource name from the cluster's API discovery for that group and version. The cli SHALL NOT derive a resource name from the spelling of a kind. Discovery answers SHALL be cached in memory for the life of the command, so that a command makes at most one successful discovery request per group and version it touches. A dry run SHALL resolve resource names in the same way as a real run.

#### Scenario: A kind with an irregular plural

- **WHEN** the cluster serves kind `Prometheus` of `monitoring.coreos.com/v1` as resource `prometheuses`
- **AND** a command addresses an object of that kind
- **THEN** the request SHALL address the resource `prometheuses`

#### Scenario: One discovery request per group and version

- **WHEN** a command addresses a `Deployment` and a `StatefulSet`, both of `apps/v1`
- **THEN** the cli SHALL make one discovery request for `apps/v1`

#### Scenario: A kind served after the first answer

- **WHEN** discovery for a group and version was answered without kind `Widget`
- **AND** the cluster starts to serve `Widget` in that group and version during the same command
- **AND** the command then addresses a `Widget`
- **THEN** the cli SHALL ask discovery again and SHALL resolve the resource

### Requirement: A kind the cluster does not serve is an error that names it

When discovery answers and the group and version, or the kind in it, is not served, a command that changes or deletes the object, or that reads a recorded object, SHALL treat the object as unreadable with an error that names the kind and its API version. `opm instance delete` and the stale-resource prune SHALL NOT count such an object as already gone: the delete SHALL report a failure for that resource and keep the `ModuleInstance` record, and the prune SHALL keep the entry in the record it writes. The error SHALL NOT be a NotFound error.

#### Scenario: Delete with a recorded kind the cluster does not serve

- **WHEN** `opm instance delete` runs and the record lists an object whose kind the cluster does not serve at the recorded version
- **THEN** the command SHALL report a failure for that object, with a message that names the kind and the API version
- **AND** it SHALL NOT delete the `ModuleInstance` record
- **AND** it SHALL exit non-zero

#### Scenario: Prune with a stale kind the cluster does not serve

- **WHEN** an apply prunes a stale entry whose kind the cluster does not serve at the recorded version
- **THEN** the prune SHALL report a failure for that entry, with a message that names the kind and the API version
- **AND** the record written after the prune SHALL still list the entry

#### Scenario: Apply of a kind the cluster does not serve

- **WHEN** an apply sends an object whose kind the cluster does not serve and no CustomResourceDefinition of the same apply defines
- **THEN** the apply SHALL report an error for that object that names the kind and the API version

#### Scenario: An existence check before an apply

- **WHEN** the first-install check or `opm instance diff` reads a rendered object whose kind the cluster does not serve
- **THEN** the object SHALL count as not existing in the cluster

### Requirement: A failed discovery request is an error

When a discovery request fails (the API server denies it, is unavailable, or the request fails in any other way), the cli SHALL NOT read the failure as "the kind is not served" or as "the object does not exist". The failure SHALL be reported through the same path as a failed read of the object, with the API error in the error chain, so that the command exits with code 4 when access is denied, 3 when the server is unavailable and 1 otherwise, where that path sets the exit code from the error.

#### Scenario: Discovery is forbidden during delete

- **WHEN** `opm instance delete` runs and the discovery request for the group and version of a recorded object answers Forbidden
- **THEN** the command SHALL report a failure for that object and keep the `ModuleInstance` record
- **AND** it SHALL exit with code 4

#### Scenario: Discovery is unavailable during the first-install check

- **WHEN** the first-install check runs and the discovery request for a rendered object answers ServiceUnavailable
- **THEN** the apply SHALL stop before it applies anything
- **AND** the error chain SHALL carry the ServiceUnavailable error

#### Scenario: Discovery fails during prune

- **WHEN** an apply prunes a stale entry and the discovery request for its group and version fails
- **THEN** the prune SHALL report a failure for that entry and the record SHALL still list it

### Requirement: An apply waits until discovery serves the kinds its definitions add

Outside a dry run, after the CustomResourceDefinitions of an apply report Established, the apply SHALL wait until discovery serves every kind that the remaining objects of the apply use and that one of those definitions defines. The wait SHALL be bounded by the same deadline as the wait for Established. A discovery failure during the wait SHALL stop the apply.

#### Scenario: A custom resource applied with its definition

- **WHEN** an apply holds a CustomResourceDefinition and an object of the kind it defines
- **AND** discovery starts to serve the kind shortly after the definition reports Established
- **THEN** the apply SHALL apply the object without an error

#### Scenario: Discovery never serves the kind

- **WHEN** the deadline passes and discovery still does not serve the kind
- **THEN** the apply SHALL stop with an error that names the kind

### Requirement: Resource resolution has one implementation

The `internal/kubernetes` package SHALL hold the single implementation that resolves a group, version and kind to a resource, and every package that addresses a Kubernetes object by kind SHALL use it. No function that derives a plural from a kind name SHALL exist in non-test code that reaches a cluster.

#### Scenario: No guessed plural on a cluster path

- **WHEN** the non-test Go code of the cli is searched for `KindToResource`, `HeuristicPluralize` and `UnsafeGuessKindToResource`
- **THEN** the only match SHALL be the test-support resolver for fake clients
