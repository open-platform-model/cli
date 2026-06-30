## Purpose

Defines how OPM CLI commands discover and select resources in a Kubernetes cluster. Primary discovery uses the inventory Secret (targeted GET calls per tracked resource). Label-based discovery via `DiscoverResources()` is retained for commands that still require it (e.g., delete fallback). This covers the `delete` and `status` commands that operate on existing deployed resources.

## Requirements

### Requirement: Selector mutual exclusivity

Commands that discover resources (`delete`, `status`) MUST accept exactly one selector type per invocation.

#### Scenario: Both --name and --instance-id provided

- **WHEN** user provides both `--name` and `--instance-id` flags
- **THEN** command exits with error: `"--name and --instance-id are mutually exclusive"`

#### Scenario: Neither --name nor --instance-id provided

- **WHEN** user provides neither `--name` nor `--instance-id` flag
- **THEN** command exits with error: `"either --name or --instance-id is required"`

#### Scenario: Only --name provided

- **WHEN** user provides `--name` flag (and `--namespace`)
- **THEN** command uses name+namespace label selector

#### Scenario: Only --instance-id provided

- **WHEN** user provides `--instance-id` flag (and `--namespace`)
- **THEN** command uses instance-id label selector

---

### Requirement: Namespace defaults to config

The `--namespace`/`-n` flag SHALL be optional for commands that discover resources (`delete`, `status`). When omitted, namespace SHALL be resolved using the precedence: `--namespace` flag → `OPM_NAMESPACE` environment variable → `kubernetes.namespace` in `~/.opm/config.cue` → `"default"`.

#### Scenario: Namespace omitted uses config default

- **WHEN** the user runs `opm mod delete --instance-name my-app` without `-n`
- **AND** the config file sets `kubernetes: namespace: "staging"`
- **THEN** the command SHALL operate in the `staging` namespace

#### Scenario: Namespace omitted falls back to default

- **WHEN** the user runs `opm mod status --instance-name my-app` without `-n`
- **AND** no config or env sets a namespace
- **THEN** the command SHALL operate in the `default` namespace

---

### Requirement: Status command supports --instance-id

The `status` command MUST support `--instance-id` flag with same semantics as `delete`.

#### Scenario: Status with --instance-id

- **WHEN** user runs `opm mod status --instance-id <uuid> --namespace bar`
- **THEN** status displays resources matching the instance-id label selector

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
