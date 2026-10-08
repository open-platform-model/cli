## REMOVED Requirements

### Requirement: Object order comes from the library weight table

**Reason**: Library v1.0.0-beta.7 aligns the weight table with Flux's staged apply order, so the promise that every weight of the retired CLI table keeps its value no longer holds.

**Migration**: The requirement "Object order follows the library weight table" replaces it. A user who relied on the old position of a kind reads the new order there.

## ADDED Requirements

### Requirement: Object order follows the library weight table

Every CLI path that orders Kubernetes objects (apply, delete, prune, the `instance tree` view, the `module build` output, and the operator install and uninstall, which apply and delete through the same paths) SHALL order by the library's kind-class weight, `opm/k8s/object.Weight`. Apply, delete, prune and the tree view SHALL sort through `object.Sort`, ascending for apply and descending for delete and prune, and SHALL keep objects of equal weight in their input order. The `module build` output SHALL break equal weights by namespace, then name, as the `cmd-structure` requirement for its output order states. The CLI SHALL NOT keep its own weight table.

The order follows Flux's staged apply (0012:D5:R1). Ascending, the classes are: CustomResourceDefinition of `apiextensions.k8s.io`; core Namespace; ClusterRole of `rbac.authorization.k8s.io`; StorageClass and every kind whose name ends in `Class`, with any kind named CustomResourceDefinition, Namespace or ClusterRole in another group; ClusterRoleBinding; ResourceQuota; ServiceAccount, Role and RoleBinding; Secret and ConfigMap; Service; LimitRange; Deployment and StatefulSet; CronJob; PodDisruptionBudget; every other kind, which includes PersistentVolume, PersistentVolumeClaim, DaemonSet, ReplicaSet, Job, Ingress, NetworkPolicy, the autoscalers and custom resources; and last ValidatingWebhookConfiguration and MutatingWebhookConfiguration.

The CLI SHALL pin this order in a unit test of its own, holding every weight constant, every table row and both sort directions as literals, so that a library release that moves a weight fails on the pin bump and the order change is a reviewed edit in the CLI. The library's own weight-table guard test covers rows the library adds.

#### Scenario: Ascending order for apply

- **WHEN** the sort runs ascending over a Deployment, a CustomResourceDefinition and a ConfigMap
- **THEN** the result is CustomResourceDefinition, ConfigMap, Deployment

#### Scenario: Descending order for delete

- **WHEN** the sort runs descending over the same three objects
- **THEN** the result is Deployment, ConfigMap, CustomResourceDefinition

#### Scenario: Equal weights keep input order

- **WHEN** the sort runs over two ConfigMaps `b` then `a`
- **THEN** the result is `b` then `a`

#### Scenario: A claim and a Job apply after the workloads

- **WHEN** the sort runs ascending over a PersistentVolumeClaim, a Job, a Deployment and a Service
- **THEN** the result is Service, Deployment, PersistentVolumeClaim, Job

#### Scenario: Webhook configurations apply last and delete first

- **WHEN** the sort runs over a ValidatingWebhookConfiguration, a custom resource of an unknown kind and a Deployment
- **THEN** ascending the result is Deployment, the custom resource, the webhook configuration, and descending it is the reverse

#### Scenario: A class kind applies before the bindings and the namespaced objects

- **WHEN** the sort runs ascending over a ConfigMap, a ClusterRoleBinding, a StorageClass and a ClusterRole
- **THEN** the result is ClusterRole, StorageClass, ClusterRoleBinding, ConfigMap

#### Scenario: A moved weight fails the CLI's order test

- **WHEN** the CLI is built against a library whose `object.Weight` returns another weight for a constant, a table row or a fallback the CLI's order test pins
- **THEN** the order test fails and names the entry
