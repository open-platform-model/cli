# Kubernetes Shared Helpers

## Purpose

Shared utility functions in `internal/kubernetes/` that eliminate duplication across K8s operations (apply, delete, diff). Provides unified GVR resolution, a `ResourceClient` abstraction for namespace-vs-cluster scoping, and consolidates path utilities to a single source of truth.

---

## Requirements

### Requirement: Unified GVR resolution from unstructured objects

The `internal/kubernetes` package SHALL provide a single `gvrFromUnstructured` function that derives a `schema.GroupVersionResource` from an `*unstructured.Unstructured` object. This function SHALL be used by all operations that need to resolve a dynamic resource client (apply, delete, diff). There SHALL NOT be duplicate implementations of this logic.

#### Scenario: GVR resolution for namespaced core resource

- **WHEN** `gvrFromUnstructured` is called with an `*unstructured.Unstructured` having apiVersion `v1` and kind `ConfigMap`
- **THEN** the returned `GroupVersionResource` SHALL have Group `""`, Version `"v1"`, and Resource `"configmaps"`

#### Scenario: GVR resolution for apps group resource

- **WHEN** `gvrFromUnstructured` is called with an `*unstructured.Unstructured` having apiVersion `apps/v1` and kind `Deployment`
- **THEN** the returned `GroupVersionResource` SHALL have Group `"apps"`, Version `"v1"`, and Resource `"deployments"`

#### Scenario: GVR resolution for unknown custom resource

- **WHEN** `gvrFromUnstructured` is called with an `*unstructured.Unstructured` having kind `MyCustomThing`
- **THEN** the returned Resource SHALL be the heuristic plural form of the kind (lowercase + "s")

### Requirement: ResourceClient method eliminates namespaced-vs-cluster-scoped branching

The `*Client` type SHALL provide a `ResourceClient` method that accepts a `schema.GroupVersionResource` and a namespace string, and returns the appropriate `dynamic.ResourceInterface`. When namespace is non-empty, it SHALL return a namespace-scoped client. When namespace is empty, it SHALL return a cluster-scoped client. All K8s operations (apply, delete, diff) SHALL use this method instead of inline branching.

#### Scenario: Namespaced resource client

- **WHEN** `ResourceClient` is called with namespace `"production"`
- **THEN** it SHALL return a `dynamic.ResourceInterface` scoped to the `"production"` namespace

#### Scenario: Cluster-scoped resource client

- **WHEN** `ResourceClient` is called with an empty namespace `""`
- **THEN** it SHALL return a cluster-scoped `dynamic.ResourceInterface`

#### Scenario: Apply uses ResourceClient for GET and PATCH

- **WHEN** `applyResource` performs a GET to check existing state and a PATCH to apply
- **THEN** both operations SHALL use `client.ResourceClient(gvr, ns)` instead of inline `if ns != ""` branching

#### Scenario: Delete uses ResourceClient

- **WHEN** `deleteResource` deletes a resource
- **THEN** it SHALL use `client.ResourceClient(gvr, ns).Delete(...)` instead of inline branching

#### Scenario: Diff uses ResourceClient for live state fetch

- **WHEN** `fetchLiveState` fetches a resource from the cluster
- **THEN** it SHALL use `client.ResourceClient(gvr, ns).Get(...)` instead of inline branching

### Requirement: Single expandTilde implementation across the codebase

There SHALL be exactly one implementation of tilde expansion (`~` to home directory) used by both the config and kubernetes packages. The `config.ExpandTilde` function SHALL be the canonical implementation. The `kubernetes` package SHALL import and call `config.ExpandTilde` instead of maintaining its own copy.

#### Scenario: Kubernetes kubeconfig resolution uses config.ExpandTilde

- **WHEN** `resolveKubeconfig` in `kubernetes/client.go` needs to expand a tilde in a path
- **THEN** it SHALL call `config.ExpandTilde(path)` from `internal/config`

#### Scenario: No duplicate expandTilde exists in the codebase

- **WHEN** the codebase is searched for functions named `expandTilde` (case-insensitive)
- **THEN** only `config.ExpandTilde` SHALL exist as an implementation (the kubernetes copy SHALL be removed)

### Requirement: Shared resource utilities live in a dedicated file

The shared utility functions (`gvrFromUnstructured`, `ResourceClient`) SHALL be defined in `internal/kubernetes/resource.go`. This file SHALL contain only functions that are shared across multiple K8s operations. The `kindToResource`, `knownKindResources`, and `heuristicPluralize` functions SHALL also be relocated to this file since they are dependencies of `gvrFromUnstructured`.

#### Scenario: resource.go contains all GVR-related helpers

- **WHEN** the file `internal/kubernetes/resource.go` is inspected
- **THEN** it SHALL contain `gvrFromUnstructured`, `kindToResource`, `knownKindResources`, `heuristicPluralize`, and `ResourceClient`

#### Scenario: apply.go no longer contains GVR or pluralization helpers

- **WHEN** the file `internal/kubernetes/apply.go` is inspected
- **THEN** it SHALL NOT contain `gvrFromObject`, `kindToResource`, `knownKindResources`, or `heuristicPluralize`
