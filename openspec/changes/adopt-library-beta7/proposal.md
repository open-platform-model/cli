## Why

Library v1.0.0-beta.7 is released and the release cascade proposed the pin bump, but the bump fails the cli's unit tests: `internal/kubernetes/order_parity_test.go` pins the kind-class weight table on purpose, and beta.7 moves thirteen weights so that the library's order agrees with Flux's staged apply (library ADR-013, decision e5 as amended; 0012:D5:R1). The owner decided that the new order ships as a breaking change. The pinned literals are the reviewed edit that lets the bump land.

## What Changes

- Bump `github.com/open-platform-model/library` from v1.0.0-beta.6 to v1.0.0-beta.7 with `task -x deps:cascade` (the same pin set the cascade branch carries: `go.mod` and `go.sum` only).
- **BREAKING** The order the cli applies, prunes, deletes and prints objects in follows the library's new weights. Lower weight applies first and deletes last:

  | Kind class | Old weight | New weight |
  | --- | --- | --- |
  | CustomResourceDefinition (`apiextensions.k8s.io`) | -100 | -100 |
  | Namespace (core) | 0 | 0 |
  | ClusterRole (`rbac.authorization.k8s.io`) | 5 | 5 |
  | StorageClass | 20 | 6 |
  | PriorityClass, RuntimeClass, IngressClass, GatewayClass, ClusterClass, VolumeSnapshotClass, any kind named `*Class` | 1000 | 6 |
  | A kind named CustomResourceDefinition, Namespace or ClusterRole in another group | -100, 0, 5 | 6 |
  | ClusterRoleBinding | 5 | 7 |
  | ResourceQuota | 1000 | 8 |
  | ServiceAccount, Role, RoleBinding | 10 | 10 |
  | Secret, ConfigMap | 15 | 15 |
  | Service | 50 | 50 |
  | LimitRange | 1000 | 60 |
  | Deployment, StatefulSet | 100 | 100 |
  | CronJob | 110 | 105 |
  | PodDisruptionBudget | 200 | 108 |
  | PersistentVolume, PersistentVolumeClaim | 20 | 1000 |
  | DaemonSet, ReplicaSet | 100 | 1000 |
  | Job | 110 | 1000 |
  | Ingress, NetworkPolicy | 150 | 1000 |
  | HorizontalPodAutoscaler, VerticalPodAutoscaler | 200 | 1000 |
  | Every other kind (custom resources) | 1000 | 1000 |
  | ValidatingWebhookConfiguration, MutatingWebhookConfiguration | 500 | 2000 |

- **BREAKING** (kernel behaviour the cli passes through) A render refuses values that leave a required `#config` value unset, even when no component reads it. Observed in the cli: `opm module build` on a module whose `debugValues` leave a bare `note: string` unset exits 0 on beta.6 and exits 2 on beta.7.
- `order_parity_test.go` pins the new weights and the new sort order; its comment says the literals are the cli's reviewed copy of the library's order and that the retired cli table is history.
- No change to commands, flags, exit codes or output formats. No adoption of `opm/k8s/ownership` or `opm/k8s/lifecycle`.

SemVer class: MAJOR after GA (the order and the required-value refusal change behaviour); during beta it ships as the next `-beta.N` with a `!` in the PR title.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `resource-conversion`: the requirement "Object order comes from the library weight table" promised that every weight of the retired cli table keeps its value. That promise ends; the replacement requirement states the order the cli follows now and that the cli pins it in a test.

## Impact

- `go.mod`, `go.sum`: library pin.
- `internal/kubernetes/order_parity_test.go`: literals, expected sort order, comments, test names.
- `openspec/specs/resource-conversion/spec.md`: one requirement replaced.
- Users: apply, prune and delete order, the `instance tree` order inside a component, and the document order of `module build` and `instance build` change for the kinds in the table. `status` and `diff` do not sort by weight and do not change.
- Other beta.7 changes checked against the cli (design.md): the adopt rule and the lifecycle package have no effect (the cli imports neither package); `*ResolutionError` keeps the message text and the fetch kinds the cli reads.
