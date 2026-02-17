## ADDED Requirements

### Requirement: Child resource discovery via ownerReference traversal

The resource discovery package SHALL provide a `DiscoverChildren` function that, given a set of parent resources, walks ownerReferences downward to find Kubernetes-owned child resources. This is the inverse of the existing `ExcludeOwned` filter — instead of filtering out owned resources, it finds children of known parents.

The traversal SHALL be targeted, not generic. It SHALL use knowledge of Kubernetes workload hierarchies to make specific queries:

| Parent Kind | Child Kind(s) | Grandchild Kind(s) |
|-------------|---------------|---------------------|
| Deployment | ReplicaSet | Pod |
| StatefulSet | Pod | - |
| DaemonSet | Pod | - |
| Job | Pod | - |
| CronJob | Job | Pod |

Non-workload parent kinds (ConfigMap, Secret, Service, etc.) SHALL be skipped — no child traversal is performed for them.

Child matching SHALL be performed by comparing `ownerReferences[].uid` on candidate children against the parent resource's `metadata.uid`.

#### Scenario: Deployment children discovered

- **WHEN** `DiscoverChildren` is called with a Deployment resource
- **THEN** it SHALL list ReplicaSets in the namespace and return those with an ownerReference pointing to the Deployment's UID
- **AND** it SHALL list Pods in the namespace and return those with an ownerReference pointing to any discovered ReplicaSet's UID

#### Scenario: StatefulSet children discovered

- **WHEN** `DiscoverChildren` is called with a StatefulSet resource
- **THEN** it SHALL list Pods in the namespace and return those with an ownerReference pointing to the StatefulSet's UID

#### Scenario: Non-workload parents skipped

- **WHEN** `DiscoverChildren` is called with a ConfigMap, Secret, Service, or other non-workload resource
- **THEN** it SHALL return no children for that resource

#### Scenario: No children exist

- **WHEN** `DiscoverChildren` is called with a Deployment that has no ReplicaSets
- **THEN** it SHALL return an empty result for that parent (not an error)

#### Scenario: API errors during child listing are non-fatal

- **WHEN** a List call for child resources fails (e.g., RBAC restriction on Pods)
- **THEN** the function SHALL log a warning and continue with other parents
- **AND** it SHALL NOT return an error for the overall operation
