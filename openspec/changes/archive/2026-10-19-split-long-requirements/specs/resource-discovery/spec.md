## REMOVED Requirements

### Requirement: Child resource discovery via ownerReference traversal
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The resource discovery package provides `DiscoverChildren`", "`DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract", "The traversal of `DiscoverChildren` is targeted, not generic", "`DiscoverChildren` skips non-workload parent kinds", "`DiscoverChildren` matches children by ownerReference uid".

### Requirement: Inventory discovery reports tracked resources it could not read
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "An inventory entry lands in exactly one of three groups and none is dropped", "Every command that reads an instance by its inventory tells the user about each unreadable entry", "`opm instance diff` says that orphan detection could not check an unreadable resource", "A warning about an unreadable entry does not change the exit code of `diff`, `tree` or `events`", "`opm instance tree` exits 5 with `no resources found` when it can read none of the tracked resources".

## ADDED Requirements

### Requirement: The resource discovery package provides `DiscoverChildren`

The resource discovery package SHALL provide a `DiscoverChildren` function in `internal/kubernetes/children.go` that, given a set of parent resources, walks ownerReferences downward to find Kubernetes-owned child resources.

#### Scenario: No children exist

- **WHEN** `DiscoverChildren` is called with a Deployment that has no ReplicaSets
- **THEN** it SHALL return an empty result for that parent (not an error)

### Requirement: `DiscoverChildren` returns children as `[]*unstructured.Unstructured` and differs from the tree ownership walk in its return contract

`DiscoverChildren` SHALL return children as `[]*unstructured.Unstructured` for UID extraction (used by the events command to match `event.involvedObject.uid`).

Note: `internal/kubernetes/tree.go` already implements equivalent ownership walking (`walkOwnership` and related helpers) that returns `[]ResourceNode` for tree display. `DiscoverChildren` follows the same traversal patterns with a different return contract: callers need the raw child resources, not rendered display nodes.

#### Scenario: API errors during child listing are non-fatal

- **WHEN** a List call for child resources fails (e.g., RBAC restriction on Pods)
- **THEN** the function SHALL log a warning and continue with other parents
- **AND** it SHALL NOT return an error for the overall operation

### Requirement: The traversal of `DiscoverChildren` is targeted, not generic

The traversal SHALL be targeted, not generic. It SHALL use knowledge of Kubernetes workload hierarchies to make specific queries:

| Parent Kind | Child Kind(s) | Grandchild Kind(s) |
|-------------|---------------|---------------------|
| Deployment | ReplicaSet | Pod |
| StatefulSet | Pod | - |
| DaemonSet | Pod | - |
| Job | Pod | - |
| CronJob | Job | Pod |

(Requirement: The resource discovery package provides `DiscoverChildren`)

#### Scenario: Deployment children discovered

- **WHEN** `DiscoverChildren` is called with a Deployment resource
- **THEN** it SHALL list ReplicaSets in the namespace and return those with an ownerReference pointing to the Deployment's UID
- **AND** it SHALL list Pods in the namespace and return those with an ownerReference pointing to any discovered ReplicaSet's UID

### Requirement: `DiscoverChildren` skips non-workload parent kinds

Non-workload parent kinds (ConfigMap, Secret, Service, etc.) SHALL be skipped: no child traversal is performed for them.
(Requirement: The resource discovery package provides `DiscoverChildren`)

#### Scenario: Non-workload parents skipped

- **WHEN** `DiscoverChildren` is called with a ConfigMap, Secret, Service, or other non-workload resource
- **THEN** it SHALL return no children for that resource

### Requirement: `DiscoverChildren` matches children by ownerReference uid

Child matching SHALL be performed by comparing `ownerReferences[].uid` on candidate children against the parent resource's `metadata.uid`.
(Requirement: The resource discovery package provides `DiscoverChildren`)

#### Scenario: StatefulSet children discovered

- **WHEN** `DiscoverChildren` is called with a StatefulSet resource
- **THEN** it SHALL list Pods in the namespace and return those with an ownerReference pointing to the StatefulSet's UID

### Requirement: An inventory entry lands in exactly one of three groups and none is dropped

When the CLI reads the live objects an instance's inventory tracks (one targeted GET per entry), every entry SHALL land in exactly one of three groups, each kept in inventory order: live (the read returned the object), missing (the read returned NotFound), or unreadable (the read failed with any other error, such as Forbidden or a timeout), the last carrying its read error.

No entry SHALL be dropped.

#### Scenario: A forbidden read is reported as unreadable

- **WHEN** an instance tracks a Deployment and a ConfigMap
- **AND** the read of the ConfigMap fails with Forbidden
- **THEN** discovery SHALL return the Deployment as live
- **AND** SHALL return the ConfigMap as unreadable with the Forbidden error
- **AND** SHALL NOT return the ConfigMap as missing

#### Scenario: NotFound is still missing

- **WHEN** the read of a tracked resource returns NotFound
- **THEN** discovery SHALL return it as missing, not unreadable

### Requirement: Every command that reads an instance by its inventory tells the user about each unreadable entry

Every command that reads an instance this way (`opm instance delete`, `status`, `list`, `diff`, `tree` and `events`) SHALL tell the user about each unreadable entry instead of omitting it silently: `delete`, `status` and `list` as their own requirements state, and `diff`, `tree` and `events` with a warning that names the resource's kind, namespace and name and the read error.
(Requirement: An inventory entry lands in exactly one of three groups and none is dropped)

#### Scenario: Tree warns about an unreadable resource

- **WHEN** the user runs `opm instance tree demo -n apps`
- **AND** one tracked resource cannot be read
- **THEN** the command SHALL print a warning naming that resource and the read error
- **AND** SHALL show the resources it could read

### Requirement: `opm instance diff` says that orphan detection could not check an unreadable resource

For `diff`, the warning SHALL also say that orphan detection could not check those resources.
(Requirement: Every command that reads an instance by its inventory tells the user about each unreadable entry)

#### Scenario: Diff warns that orphan detection is incomplete

- **WHEN** the user runs `opm instance diff` for a deployed instance
- **AND** one tracked resource cannot be read
- **THEN** the command SHALL print a warning naming that resource and the read error
- **AND** SHALL say that orphan detection could not check it

### Requirement: A warning about an unreadable entry does not change the exit code of `diff`, `tree` or `events`

A warning SHALL NOT change the exit code of `diff`, `tree` or `events`: a per-entry read error is not mapped through `cmdutil.ExitCodeFromK8sError`, so a Forbidden read of one tracked resource does not make `tree` exit 4.
(Requirement: Every command that reads an instance by its inventory tells the user about each unreadable entry)

#### Scenario: A forbidden read of one tracked resource does not make tree exit 4

- **WHEN** `opm instance tree` cannot read one tracked resource because the read fails with Forbidden
- **THEN** the warning SHALL NOT change the exit code and `tree` SHALL NOT exit 4

### Requirement: `opm instance tree` exits 5 with `no resources found` when it can read none of the tracked resources

When `tree` can read none of the tracked resources, it SHALL print the warnings and then exit 5 with `no resources found`, as it does for a record that tracks no live resources.
(Requirement: Every command that reads an instance by its inventory tells the user about each unreadable entry)

#### Scenario: Tree with every tracked resource unreadable

- **WHEN** the user runs `opm instance tree demo -n apps`
- **AND** no tracked resource can be read
- **THEN** the command SHALL print one warning per tracked resource naming it and the read error
- **AND** SHALL exit 5 with `no resources found`
