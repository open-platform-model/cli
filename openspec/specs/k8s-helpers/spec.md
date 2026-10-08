# Kubernetes Shared Helpers

## Purpose

Shared utility functions in `internal/kubernetes/` that eliminate duplication across K8s operations (apply, delete). Provides resource resolution from API discovery, a `ResourceClient` abstraction for namespace-vs-cluster scoping, and consolidates path utilities to a single source of truth.

---

## Requirements

### Requirement: ResourceClient method eliminates namespaced-vs-cluster-scoped branching

The `*Client` type SHALL provide a `ResourceClient` method that accepts a `schema.GroupVersionResource` and a namespace string, and returns the appropriate `dynamic.ResourceInterface`. When namespace is non-empty, it SHALL return a namespace-scoped client. When namespace is empty, it SHALL return a cluster-scoped client. All K8s operations (apply, delete) SHALL use this method instead of inline branching.

#### Scenario: Namespaced resource client

- **WHEN** `ResourceClient` is called with namespace `"production"`
- **THEN** it SHALL return a `dynamic.ResourceInterface` scoped to the `"production"` namespace

#### Scenario: Cluster-scoped resource client

- **WHEN** `ResourceClient` is called with an empty namespace `""`
- **THEN** it SHALL return a cluster-scoped `dynamic.ResourceInterface`

#### Scenario: Apply uses ResourceClient for GET and PATCH

- **WHEN** `ApplyOne` performs a GET to check existing state and a PATCH to apply
- **THEN** both operations SHALL use `client.ResourceClient(gvr, ns)` instead of inline `if ns != ""` branching

#### Scenario: Delete uses ResourceClient

- **WHEN** `deleteResource` deletes a resource
- **THEN** it SHALL use `client.ResourceClient(gvr, ns).Delete(...)` instead of inline branching

### Requirement: Single expandTilde implementation across the codebase

There SHALL be exactly one implementation of tilde expansion (`~` to home directory) used by both the config and kubernetes packages. The `config.ExpandTilde` function SHALL be the canonical implementation. The `kubernetes` package SHALL import and call `config.ExpandTilde` instead of maintaining its own copy.

#### Scenario: Kubernetes kubeconfig resolution uses config.ExpandTilde

- **WHEN** `resolveKubeconfig` in `kubernetes/client.go` needs to expand a tilde in a path
- **THEN** it SHALL call `config.ExpandTilde(path)` from `internal/config`

#### Scenario: No duplicate expandTilde exists in the codebase

- **WHEN** the codebase is searched for functions named `expandTilde` (case-insensitive)
- **THEN** only `config.ExpandTilde` SHALL exist as an implementation (the kubernetes copy SHALL be removed)

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

When a discovery request fails (the API server denies it, is unavailable, or the request fails in any other way), the cli SHALL NOT read the failure as "the kind is not served" or as "the object does not exist". The API error SHALL stay in the error chain. An apply SHALL stop at a failed discovery request, in its first-install check, in its apply of objects and in its prune, and SHALL exit with code 4 when access is denied, 3 when the server is unavailable and 1 otherwise. A command that reads recorded objects SHALL report the failure through the same path as a failed read of the object.

#### Scenario: Discovery is forbidden during delete

- **WHEN** `opm instance delete` runs and the discovery request for the group and version of a recorded object answers Forbidden
- **THEN** the command SHALL report a failure for that object, with the Forbidden error in its message, and keep the `ModuleInstance` record
- **AND** it SHALL exit non-zero, with the code it gives any failed read of a tracked resource

#### Scenario: Discovery is forbidden during diff

- **WHEN** `opm instance diff` runs and the discovery request for the group and version of a rendered object answers Forbidden
- **THEN** the command SHALL report that object as a read failure, not as added
- **AND** it SHALL exit with code 4

#### Scenario: Discovery is unavailable during the first-install check

- **WHEN** the first-install check runs and the discovery request for a rendered object answers ServiceUnavailable
- **THEN** the apply SHALL stop before it applies anything
- **AND** the error chain SHALL carry the ServiceUnavailable error

#### Scenario: Discovery fails during the apply of objects

- **WHEN** an apply sends its objects and the discovery request for one of them answers ServiceUnavailable
- **THEN** the apply SHALL send no further object, SHALL NOT prune and SHALL NOT write the record
- **AND** it SHALL exit with code 3
- **AND** a dry run SHALL stop in the same way

#### Scenario: Discovery fails during prune

- **WHEN** an apply prunes stale entries and the discovery request for the group and version of one of them answers Forbidden
- **THEN** the prune SHALL delete nothing more
- **AND** the record written after it SHALL still list that entry and every stale entry not yet pruned
- **AND** the command SHALL exit with code 4

### Requirement: An apply waits until discovery serves the kinds its definitions add

Outside a dry run, after the CustomResourceDefinitions of an apply report Established, the apply SHALL wait until discovery serves every kind that the remaining objects of the apply use and that one of those definitions defines and serves at the object's version. The wait SHALL be bounded by the same deadline as the wait for Established. A discovery failure during the wait SHALL stop the apply.

#### Scenario: A custom resource applied with its definition

- **WHEN** an apply holds a CustomResourceDefinition and an object of the kind it defines
- **AND** discovery starts to serve the kind shortly after the definition reports Established
- **THEN** the apply SHALL apply the object without an error

#### Scenario: An object at a version its definition does not serve

- **WHEN** an apply holds a CustomResourceDefinition that serves `v1` and an object of its kind at `v1alpha1`
- **THEN** the apply SHALL NOT wait for that kind
- **AND** it SHALL report an error for that object that names the kind and the API version, and SHALL apply the other objects

#### Scenario: Discovery never serves the kind

- **WHEN** the deadline passes and discovery still does not serve the kind
- **THEN** the apply SHALL stop with an error that names the kind

### Requirement: Resource resolution has one implementation

The `internal/kubernetes` package SHALL hold the single implementation that resolves a group, version and kind to a resource, and every package that addresses a Kubernetes object by kind SHALL use it. No function that derives a plural from a kind name SHALL exist in non-test code that reaches a cluster.

#### Scenario: No guessed plural on a cluster path

- **WHEN** the non-test Go code of the cli is searched for `KindToResource`, `HeuristicPluralize` and `UnsafeGuessKindToResource`
- **THEN** the only match SHALL be the test-support resolver for fake clients
