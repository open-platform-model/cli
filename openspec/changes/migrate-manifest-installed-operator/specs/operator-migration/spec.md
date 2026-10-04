## Purpose

Install's one-time migration of an opm-operator installed from an earlier release's manifest into the CLI-owned operator ModuleInstance. It defines which objects install may adopt or delete, the Deployment recreate, the field ownership of adopted objects, the order of checks and writes, how a stopped migration resumes, and what install reports.

## ADDED Requirements

### Requirement: The proof list covers exactly the objects of earlier release manifests

The CLI SHALL carry a fixed list of every object of every opm-operator release that published an install manifest. For each object the list SHALL hold its kind, its namespace (empty for a cluster-scoped kind), its name and the labels that manifest set on it. For the controller Deployment it SHALL also hold the pod selector that manifest set. The list SHALL be the union over those releases, so it includes an object that an older release installed and a newer one dropped. The list SHALL NOT be a copy of any manifest: it carries no spec, no rules and no image. The migration SHALL NOT adopt or delete any object outside the list.

#### Scenario: An object every manifest release shipped is on the list

- **WHEN** the CLI's proof list is read
- **THEN** it holds `Deployment opm-operator-system/opm-operator-controller-manager` with labels `app.kubernetes.io/name=opm-operator`, `app.kubernetes.io/managed-by=kustomize`, `control-plane=controller-manager` and the selector `app.kubernetes.io/name=opm-operator, control-plane=controller-manager`

#### Scenario: An object only older releases shipped is on the list

- **WHEN** the CLI's proof list is read
- **THEN** it holds `ClusterRole opm-operator-modulerelease-admin-role` and `CustomResourceDefinition releases.releases.opmodel.dev`, which releases v0.5.0 to v0.7.5 shipped and later releases dropped

#### Scenario: A same-named object of another kind is not on the list

- **WHEN** the cluster holds a `ConfigMap opm-operator-system/opm-operator-controller-manager`
- **THEN** the migration neither adopts nor deletes it, and the apply guard treats it as it treats any foreign object

### Requirement: An object is proven only by kind, name, labels and the absence of an instance identity

An existing object SHALL be proven to come from an earlier operator release only when all of the following hold:

- its kind, namespace and name match an entry of the proof list;
- its labels include every label that entry lists (added labels do not disprove it);
- it carries none of the OPM instance identity labels (`module-instance.opmodel.dev/uuid`, `module-instance.opmodel.dev/name`, `module-instance.opmodel.dev/namespace`).

An object that already carries the operator instance's own identity, as after an interrupted migration or a `kubectl apply` of a manifest rendered from the module, SHALL NOT need proof, because it is not foreign. An object that carries another instance's identity SHALL NOT be proven. An object that the CLI cannot read SHALL NOT be proven.

#### Scenario: Object from an opm-cli install is proven

- **WHEN** `ServiceAccount opm-operator-system/opm-operator-controller-manager` exists with the labels the v1.0.0-beta.5 manifest set and no instance identity
- **THEN** it is proven

#### Scenario: Missing listed label disproves the object

- **WHEN** `RoleBinding opm-operator-system/opm-operator-leader-election-rolebinding` exists without the `app.kubernetes.io/name=opm-operator` label that its manifests set
- **THEN** it is not proven

#### Scenario: Another instance's identity disproves the object

- **WHEN** an object on the proof list carries the `module-instance.opmodel.dev/uuid` of an instance other than the operator's
- **THEN** it is not proven, and install refuses before any object changes, naming the object and the instance whose identity it carries

#### Scenario: Unreadable object refuses the install

- **WHEN** the CLI's read of an object on the proof list fails with an error other than NotFound
- **THEN** install refuses before any object changes, naming the object and the error

### Requirement: Install adopts the proven objects the module renders and no others

On the first module install over an operator installed from an earlier manifest, install SHALL admit through the apply guard exactly the proven objects that the module renders. It SHALL record every object the module renders in the operator instance's inventory. The guard SHALL refuse every other existing object the render names that is neither proven nor already this instance's, as it does for any instance. Install SHALL offer no flag or prompt that overrides that refusal. Install SHALL write no annotation and no label in order to admit an object.

#### Scenario: Migration over an opm-cli server-side install

- **WHEN** `opm operator install` runs on a cluster whose operator was installed by an earlier `opm operator install` from the v1.0.0-beta.5 manifest
- **THEN** the run succeeds and the operator's ModuleInstance inventory records every object the module renders
- **AND** every adopted object other than the Deployment keeps its `metadata.uid`

#### Scenario: Migration over a client-side kubectl install

- **WHEN** `opm operator install` runs on a cluster whose operator was installed with client-side `kubectl apply -f install.yaml` of an earlier release
- **THEN** the run succeeds, every object the module renders is recorded, and every adopted object other than the Deployment keeps its `metadata.uid`

#### Scenario: Unproven foreign object is refused

- **WHEN** the render names an object that exists on the cluster, is not on the proof list, and carries no OPM `managed-by` label
- **THEN** install refuses before any object changes, naming the object, and offers no override

### Requirement: Adopted objects end under the CLI's field ownership whatever installed them

After a migration, every adopted object SHALL hold only the fields the module renders, whichever tool installed it earlier. A label or annotation that the earlier manifest or its installer set and the module does not render SHALL be gone; this includes `kubectl.kubernetes.io/last-applied-configuration`. The field manager `opm-cli` SHALL own the fields the module renders.

#### Scenario: Earlier labels are dropped after a client-side install

- **WHEN** the migration adopts `Namespace opm-operator-system`, which client-side `kubectl apply` created with `app.kubernetes.io/managed-by=kustomize` and `control-plane=controller-manager`
- **THEN** after install the Namespace carries the module's labels and neither of those two
- **AND** it carries no `kubectl.kubernetes.io/last-applied-configuration` annotation

#### Scenario: Earlier labels are dropped from a CRD after a client-side install

- **WHEN** the migration adopts `CustomResourceDefinition moduleinstances.opmodel.dev`, which client-side `kubectl apply` created
- **THEN** after install it carries no `kubectl.kubernetes.io/last-applied-configuration` annotation, and no field manager other than `opm-cli` and the API server owns a field of it

#### Scenario: Earlier labels are dropped after an opm-cli install

- **WHEN** the migration adopts objects that an earlier `opm operator install` applied as `opm-cli`
- **THEN** after install none of them carries `app.kubernetes.io/managed-by=kustomize`

### Requirement: Install recreates the earlier Deployment once

When the operator's Deployment exists, is proven, and carries the selector of an earlier manifest, install SHALL delete it and wait until it is gone before the instance apply, so that the instance creates it with the module's selector. Install SHALL NOT delete a Deployment that carries the operator instance's identity or the module's selector. As a result, no install between module versions, and no install over a manifest rendered from the module, deletes the Deployment. The migration SHALL NOT touch the workloads of the instances that the operator manages.

#### Scenario: Earlier Deployment is recreated

- **WHEN** install migrates an operator installed from the v1.0.0-beta.5 manifest
- **THEN** the Deployment `opm-operator-controller-manager` is deleted and created again with the module's selector, and install reports it as recreated

#### Scenario: Re-running install after the migration recreates nothing

- **WHEN** install runs again with the same module version after a completed migration
- **THEN** the Deployment keeps its `metadata.uid`

### Requirement: Install deletes the superseded role bindings

Install SHALL delete the earlier manifest's role bindings that the module's bindings replace: `ClusterRoleBinding opm-operator-manager-rolebinding`, `ClusterRoleBinding opm-operator-metrics-auth-rolebinding` and `RoleBinding opm-operator-system/opm-operator-leader-election-rolebinding`. It SHALL delete each one only when it is proven. After a completed migration, no role binding of an earlier manifest SHALL remain in the cluster.

#### Scenario: Three bindings are deleted

- **WHEN** install migrates an operator installed from any manifest release
- **THEN** none of the three bindings exists afterwards, the module's bindings exist, and install reports each deleted binding

### Requirement: The migration deletes only proven objects, and refuses otherwise

The migration SHALL delete an object only when that object meets the proof. When the earlier Deployment or a superseded binding exists and is not proven, install SHALL refuse before any object changes, naming the object and the label or identity that failed the proof. The migration SHALL NOT delete or change any of the following:

- any CustomResourceDefinition, other than the CRD step's own apply;
- any custom resource stored under the operator's CRDs;
- the operator Namespace, other than the instance apply's own apply;
- any object recorded in the inventory of another ModuleInstance.

#### Scenario: Unproven binding refuses the install

- **WHEN** `ClusterRoleBinding opm-operator-manager-rolebinding` exists but carries the `module-instance.opmodel.dev/uuid` of another instance
- **THEN** install refuses before any object changes and names the binding

#### Scenario: Custom resources and managed workloads are untouched

- **WHEN** install migrates an operator that manages ModuleInstances with deployed workloads
- **THEN** every ModuleInstance, ModulePackage, Platform and TransformerRegistration keeps its `metadata.uid` and `metadata.generation`
- **AND** every workload object recorded in those instances' inventories keeps its `metadata.uid` and `metadata.resourceVersion`

### Requirement: Every refusing check runs before the migration's first write

Install SHALL compute and check the whole migration, including the proof of every object it will adopt or delete, together with its other refusing checks and before its first write to the cluster. Only after all of them pass SHALL install write, in this order:

1. the field-ownership moves of adopted objects;
2. the CRD step;
3. the migration's deletes;
4. the instance apply.

A migration install that refuses SHALL leave the earlier operator running and every object of the earlier manifest unchanged.

#### Scenario: Refusal after proof failure changes nothing

- **WHEN** install refuses because one object fails the proof
- **THEN** every object of the earlier manifest, the four CRDs included, keeps its `metadata.resourceVersion`, and the earlier operator's pod keeps running

#### Scenario: Deletes follow the CRD step

- **WHEN** a migration install passes every check
- **THEN** the CRDs are applied and served before the earlier Deployment or any binding is deleted, and both deletes happen before the operator instance is applied

### Requirement: Re-running install completes a migration that stopped partway

When a migration stops after its first write, a re-run of install SHALL complete it:

- it SHALL create the Deployment if it is missing;
- it SHALL delete every superseded binding still present;
- it SHALL record every object the module renders;
- it SHALL NOT refuse any object that the interrupted run already adopted or applied.

#### Scenario: Instance apply failed after the Deployment was deleted

- **WHEN** a migration run deleted the earlier Deployment and the bindings, then failed before the instance record was written, and `opm operator install` runs again
- **THEN** the second run succeeds, creates the Deployment, and records every object the module renders

#### Scenario: Run interrupted between the deletes

- **WHEN** a migration run stopped after deleting one binding, and `opm operator install` runs again
- **THEN** the second run deletes the bindings still present, recreates the Deployment, and completes the install

### Requirement: An operator applied from a module-rendered manifest needs no migration step

When an operator was installed with `kubectl apply` of a manifest rendered from the operator module, install SHALL take it into the instance with no step by the user on any object, with no Deployment recreate and with no delete.

#### Scenario: Install over a module-rendered kubectl install

- **WHEN** `opm operator install` runs on a cluster whose operator was applied with `kubectl apply` of a manifest rendered from the same module version
- **THEN** install succeeds, records every object, deletes nothing, and every object keeps its `metadata.uid`

### Requirement: Install reports what the migration did and left

Beside its ordinary per-object lines, a migration install SHALL report each adopted object, the recreated Deployment and each deleted binding. It SHALL also report each object of an earlier manifest that is present on the cluster, that the module does not render and that the migration does not delete; install SHALL leave such an object in place. A refused migration SHALL name every object that failed the proof, not only the first. An install with nothing to migrate SHALL print no migration lines.

#### Scenario: Leftover object of an older release is named and kept

- **WHEN** install migrates a cluster that still holds `ClusterRole opm-operator-modulerelease-viewer-role` from a v0.7 install
- **THEN** install names it as left in place, and afterwards it still exists with the same `metadata.resourceVersion`

#### Scenario: Refusal names every unproven object

- **WHEN** two objects on the proof list fail the proof
- **THEN** install's refusal names both of them

### Requirement: The CRDs-only form performs no migration

`opm operator install --crds-only` SHALL NOT delete or recreate any object of an earlier manifest, and SHALL NOT move the field ownership of any such object other than the CRDs it applies.

#### Scenario: CRDs-only over a manifest install

- **WHEN** `opm operator install --crds-only` runs on a cluster whose operator was installed from an earlier manifest
- **THEN** the earlier Deployment and the three bindings keep their `metadata.uid` and `metadata.resourceVersion`
