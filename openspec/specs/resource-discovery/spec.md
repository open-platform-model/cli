## Purpose

Defines how OPM CLI commands discover and select resources in a Kubernetes cluster. Primary discovery reads the `ModuleInstance` CR (direct GET by name, or CR list matched on `status.instanceUUID`). There is no label-based fallback: an instance without a `ModuleInstance` record is reported as not found. This covers the `delete`, `status`, `tree` and `events` commands that operate on existing deployed resources.

## Requirements

### Requirement: Selector resolution from the positional argument

Commands that discover resources (`delete`, `status`, `tree`, `events`) MUST take exactly one positional `<file|name|uuid>` argument per invocation and resolve it through `cmdutil.ResolveInstanceTarget`. A UUID argument resolves by listing `ModuleInstance` CRs in the namespace and matching `status.instanceUUID`; a name argument resolves by a direct `ModuleInstance` GET; an instance-file argument yields the instance name (and namespace) from the file.

#### Scenario: Missing argument

- **WHEN** the user provides no positional argument
- **THEN** the command exits with a usage error

#### Scenario: Name argument

- **WHEN** the user provides an instance name (and `--namespace`)
- **THEN** the command resolves the instance by a direct `ModuleInstance` GET by name in the namespace

#### Scenario: UUID argument

- **WHEN** the user provides an instance UUID (and `--namespace`)
- **THEN** the command resolves the instance by listing `ModuleInstance` CRs in the namespace and matching `status.instanceUUID`

#### Scenario: File argument

- **WHEN** the user provides a path to an instance file
- **THEN** the command reads the instance name and namespace from the file's metadata, with `--namespace` taking precedence when set

---

### Requirement: Namespace defaults to config

The `--namespace`/`-n` flag SHALL be optional for commands that discover resources (`delete`, `status`). When omitted, namespace SHALL be resolved using the precedence: `--namespace` flag → `OPM_NAMESPACE` environment variable → `kubernetes.namespace` in `~/.opm/config.cue` → `"default"`.

#### Scenario: Namespace omitted uses config default

- **WHEN** the user runs `opm instance delete my-app` without `-n`
- **AND** the config file sets `kubernetes: namespace: "staging"`
- **THEN** the command SHALL operate in the `staging` namespace

#### Scenario: Namespace omitted falls back to default

- **WHEN** the user runs `opm instance status my-app` without `-n`
- **AND** no config or env sets a namespace
- **THEN** the command SHALL operate in the `default` namespace

---

### Requirement: Status command supports UUID identifiers

The `status` command MUST accept an instance UUID as its positional argument with the same semantics as `delete`.

#### Scenario: Status with a UUID argument

- **WHEN** user runs `opm instance status <uuid> --namespace bar`
- **THEN** status resolves the `ModuleInstance` whose `status.instanceUUID` matches and displays its resources

---

### Requirement: Child resource discovery via ownerReference traversal

The resource discovery package SHALL provide a `DiscoverChildren` function in `internal/kubernetes/children.go` that, given a set of parent resources, walks ownerReferences downward to find Kubernetes-owned child resources. It returns children as `[]*unstructured.Unstructured` for UID extraction (used by the events command to match `event.involvedObject.uid`).

Note: `internal/kubernetes/tree.go` already implements equivalent ownership walking (`walkOwnership` and related helpers) that returns `[]ResourceNode` for tree display. `DiscoverChildren` follows the same traversal patterns with a different return contract — callers need the raw child resources, not rendered display nodes.

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
