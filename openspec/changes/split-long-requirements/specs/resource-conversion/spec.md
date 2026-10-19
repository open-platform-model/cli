## REMOVED Requirements

### Requirement: Object order follows the library weight table
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every CLI path orders objects by the library's kind-class weight", "Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order", "The `module build` and `instance build` output breaks equal weights by namespace, then name", "Library weight classes: CustomResourceDefinition to ClusterRole", "Library weight classes: StorageClass to RoleBinding", "Library weight classes: Secret to the webhook configurations", "The CLI pins the library weight order in a unit test of its own".

### Requirement: One export feeds the apply objects and the shared render digest
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The CLI exports a render's compiled objects once", "The applied objects and the render digest come from the same exported set", "An export failure exits with the general error code and names the failing resource".

## ADDED Requirements

### Requirement: Every CLI path orders objects by the library's kind-class weight

Every CLI path that orders Kubernetes objects (apply, delete, prune, the `instance tree` view, the `module build` and `instance build` output, and the operator install and uninstall, which apply and delete through the same paths) SHALL order by the library's kind-class weight, `opm/k8s/object.Weight`.

The CLI SHALL NOT keep its own weight table.

#### Scenario: A CLI path orders by the library weight

- **WHEN** a CLI path orders Kubernetes objects: apply, delete, prune, the `instance tree` view, the `module build` and `instance build` output, or the operator install and uninstall
- **THEN** it SHALL order by the library's kind-class weight, `opm/k8s/object.Weight`

### Requirement: Apply, delete, prune and the tree view sort through `object.Sort` and keep objects of equal weight in input order

Apply, delete, prune and the tree view SHALL sort through `object.Sort`, ascending for apply and descending for delete and prune, and SHALL keep objects of equal weight in their input order.

#### Scenario: Ascending order for apply

- **WHEN** the sort runs ascending over a Deployment, a CustomResourceDefinition and a ConfigMap
- **THEN** the result is CustomResourceDefinition, ConfigMap, Deployment

#### Scenario: Descending order for delete

- **WHEN** the sort runs descending over the same three objects
- **THEN** the result is Deployment, ConfigMap, CustomResourceDefinition

#### Scenario: Equal weights keep input order

- **WHEN** the sort runs over two ConfigMaps `b` then `a`
- **THEN** the result is `b` then `a`

### Requirement: The `module build` and `instance build` output breaks equal weights by namespace, then name

The `module build` and `instance build` output SHALL break equal weights by namespace, then name, as the `cmd-structure` requirement for its output order states.

#### Scenario: Build output breaks equal weights by namespace, then name

- **WHEN** the `module build` or `instance build` output holds objects of equal weight
- **THEN** the output SHALL break equal weights by namespace, then name

### Requirement: Library weight classes: CustomResourceDefinition to ClusterRole

The order follows Flux's staged apply (enhancement 0012, "Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

Ascending, the classes SHALL be: CustomResourceDefinition of `apiextensions.k8s.io`; core Namespace; ClusterRole of `rbac.authorization.k8s.io`.
(Requirement: Library weight classes: StorageClass to RoleBinding)

#### Scenario: A CustomResourceDefinition applies first

- **WHEN** the sort runs ascending over a Deployment, a CustomResourceDefinition and a ConfigMap
- **THEN** the result is CustomResourceDefinition, ConfigMap, Deployment

### Requirement: Library weight classes: StorageClass to RoleBinding

Ascending, the classes SHALL be: StorageClass and every kind whose name ends in `Class`, with any kind named CustomResourceDefinition, Namespace or ClusterRole in another group; ClusterRoleBinding; ResourceQuota; ServiceAccount, Role and RoleBinding.
(Requirements: Library weight classes: CustomResourceDefinition to ClusterRole; Library weight classes: Secret to the webhook configurations)

#### Scenario: A class kind applies before the bindings and the namespaced objects

- **WHEN** the sort runs ascending over a ConfigMap, a ClusterRoleBinding, a StorageClass and a ClusterRole
- **THEN** the result is ClusterRole, StorageClass, ClusterRoleBinding, ConfigMap

### Requirement: Library weight classes: Secret to the webhook configurations

Ascending, the classes SHALL be: Secret and ConfigMap; Service; LimitRange; Deployment and StatefulSet; CronJob; PodDisruptionBudget; every other kind, which includes PersistentVolume, PersistentVolumeClaim, DaemonSet, ReplicaSet, Job, Ingress, NetworkPolicy, the autoscalers and custom resources; and last ValidatingWebhookConfiguration and MutatingWebhookConfiguration.
(Requirement: Library weight classes: StorageClass to RoleBinding)

#### Scenario: A claim and a Job apply after the workloads

- **WHEN** the sort runs ascending over a PersistentVolumeClaim, a Job, a Deployment and a Service
- **THEN** the result is Service, Deployment, PersistentVolumeClaim, Job

#### Scenario: Webhook configurations apply last and delete first

- **WHEN** the sort runs over a ValidatingWebhookConfiguration, a custom resource of an unknown kind and a Deployment
- **THEN** ascending the result is Deployment, the custom resource, the webhook configuration, and descending it is the reverse

### Requirement: The CLI pins the library weight order in a unit test of its own

The CLI SHALL pin this order in a unit test of its own, holding every weight constant, every table row and both sort directions as literals, so that a library release that moves a weight fails on the pin bump and the order change is a reviewed edit in the CLI.

The library's own weight-table guard test covers rows the library adds.
(Requirement: Every CLI path orders objects by the library's kind-class weight)

#### Scenario: A moved weight fails the CLI's order test

- **WHEN** the CLI is built against a library whose `object.Weight` returns another weight for a constant, a table row or a fallback the CLI's order test pins
- **THEN** the order test fails and names the entry

### Requirement: The CLI exports a render's compiled objects once

The CLI SHALL convert a render's compiled objects with the library's `opm/k8s/object`: it SHALL wrap them with `object.Resources` and export them with exactly one `object.Export` call per render.

#### Scenario: The digest and the apply objects come from one export

- **WHEN** `opm instance apply` or `opm module apply` renders a module
- **THEN** the render digest and the objects passed to apply both come from one `object.Export` over the render's compiled objects

### Requirement: The applied objects and the render digest come from the same exported set

The objects the CLI applies SHALL be the exported objects, in render order, and the render digest SHALL be the library's `opm/k8s/inventory.RenderDigest` of the same exported set, so no compiled object is exported from CUE twice and the CLI keeps no render digest algorithm of its own (enhancement 0012, "Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value").
(Requirement: The CLI exports a render's compiled objects once)

#### Scenario: The render digest is the library's

- **WHEN** the CLI digests an exported object set
- **THEN** the digest equals `inventory.RenderDigest` of that set

### Requirement: An export failure exits with the general error code and names the failing resource

An export failure SHALL exit with the general error code and name the failing resource.
(Requirement: The CLI exports a render's compiled objects once)

#### Scenario: A failed export names the resource

- **WHEN** the export of a render's compiled objects fails
- **THEN** the CLI SHALL exit with the general error code and name the failing resource
