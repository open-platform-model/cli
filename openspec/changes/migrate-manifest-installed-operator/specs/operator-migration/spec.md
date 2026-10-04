## Purpose

Install's one-time migration of an opm-operator installed from an earlier release's manifest into the CLI-owned operator ModuleInstance. It defines which objects install may adopt or delete, the Deployment recreate, the field ownership of rendered objects, the order of checks and writes, how a stopped migration resumes, and what install reports.

## ADDED Requirements

### Requirement: The proof list covers exactly the objects of earlier release manifests

The CLI SHALL carry a fixed list of every object of every opm-operator operator release, a release tagged `v<semver>`, that published an install manifest. The list SHALL NOT be built from any release of the operator module (tagged `opm_operator-vX.Y.Z`), even one that attaches an install manifest. For each object the list SHALL hold its kind, its namespace (empty for a cluster-scoped kind), its name and the labels that manifest set on it. For the controller Deployment it SHALL also hold the pod selector that manifest set. The list SHALL be the union over those releases, so it includes an object that an older release installed and a newer one dropped. The list SHALL NOT be a copy of any manifest: it carries no spec, no rules and no image. The migration SHALL NOT adopt or delete any object outside the list.

#### Scenario: An object every manifest release shipped is on the list

- **WHEN** the CLI's proof list is read
- **THEN** it holds `Deployment opm-operator-system/opm-operator-controller-manager` with labels `app.kubernetes.io/name=opm-operator`, `app.kubernetes.io/managed-by=kustomize`, `control-plane=controller-manager` and the selector `app.kubernetes.io/name=opm-operator, control-plane=controller-manager`

#### Scenario: An object only older releases shipped is on the list

- **WHEN** the CLI's proof list is read
- **THEN** it holds `ClusterRole opm-operator-modulerelease-admin-role` and `CustomResourceDefinition releases.releases.opmodel.dev`, which releases v0.5.0 to v0.7.5 shipped and later releases dropped

#### Scenario: A module release's manifest is not a source of the list

- **WHEN** the opm-operator repository holds a release tagged `opm_operator-v0.1.0` that attaches an `install.yaml`
- **THEN** no entry of the proof list comes from that manifest, and an object it created, carrying the operator instance's identity, is treated as the instance's own

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

On the first module install over an operator installed from an earlier manifest, install SHALL admit through the apply guard exactly the proven objects that the module renders. It SHALL record every object the module renders in the operator instance's inventory. The guard SHALL refuse every other existing object the render names that is neither proven nor already this instance's, as it does for any instance. Install SHALL offer no flag or prompt that overrides that refusal. The proven objects SHALL pass both the guard in install's check phase and the guard of the instance apply, so a manifest-installed cluster is not refused at either.

#### Scenario: Migration over an opm-cli server-side install

- **WHEN** `opm operator install` runs on a cluster whose operator was installed by an earlier `opm operator install` from the v1.0.0-beta.5 manifest
- **THEN** the run succeeds and the operator's ModuleInstance inventory records every object the module renders
- **AND** every adopted object other than the Deployment keeps its `metadata.uid`

#### Scenario: Migration over a client-side kubectl install

- **WHEN** `opm operator install` runs on a cluster whose operator was installed with client-side `kubectl apply -f install.yaml` of an earlier release
- **THEN** the run succeeds, every object the module renders is recorded, and every adopted object other than the Deployment keeps its `metadata.uid`

#### Scenario: The check-phase guard admits proven objects

- **WHEN** install's check phase runs on a cluster whose `Namespace opm-operator-system` and the other objects of the v1.0.0-beta.5 manifest carry `app.kubernetes.io/managed-by=kustomize` and no instance identity
- **THEN** the check phase passes, and the guard refuses none of them

#### Scenario: Unproven foreign object is refused

- **WHEN** the render names an object that exists on the cluster, is not on the proof list, and carries no OPM `managed-by` label
- **THEN** install refuses before any object changes, naming the object, and offers no override

### Requirement: Labels and annotations of an earlier client-side or opm-cli apply do not survive install

After a full install, a label or annotation on a rendered object that an earlier client-side `kubectl apply` (field manager `kubectl-client-side-apply`) or an earlier `opm-cli` apply set, and that the module does not render, SHALL be gone; this includes `kubectl.kubernetes.io/last-applied-configuration`. This SHALL hold for adopted objects and for objects that already carry the operator instance's identity alike. The field manager `opm-cli` SHALL own the fields the module renders. Install SHALL NOT move or remove fields owned by any other field manager.

#### Scenario: Earlier labels are dropped after a client-side install

- **WHEN** the migration adopts `Namespace opm-operator-system`, which client-side `kubectl apply` created with `app.kubernetes.io/managed-by=kustomize` and `control-plane=controller-manager`
- **THEN** after install the Namespace carries the module's labels and neither of those two
- **AND** it carries no `kubectl.kubernetes.io/last-applied-configuration` annotation

#### Scenario: Earlier labels are dropped from a CRD after a client-side install

- **WHEN** the migration adopts `CustomResourceDefinition moduleinstances.opmodel.dev`, which client-side `kubectl apply` created
- **THEN** after install it carries no `kubectl.kubernetes.io/last-applied-configuration` annotation, and the field manager `kubectl-client-side-apply` owns no field of it

#### Scenario: Fields of another manager are left alone

- **WHEN** an adopted object holds a label owned by a field manager other than `kubectl-client-side-apply` and `opm-cli`
- **THEN** after install that label and its manager's ownership are unchanged

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

Install SHALL delete the earlier manifest's role bindings that the module's bindings replace: `ClusterRoleBinding opm-operator-manager-rolebinding`, `ClusterRoleBinding opm-operator-metrics-auth-rolebinding` and `RoleBinding opm-operator-system/opm-operator-leader-election-rolebinding`. It SHALL delete each one only when it is proven and the render holds a binding of the same kind and namespace whose `roleRef` equals the live binding's `roleRef`. When a proven superseded binding has no such replacement in the render, install SHALL refuse before any object changes, naming the binding and its `roleRef`. After a completed migration, no role binding of an earlier manifest SHALL remain in the cluster.

#### Scenario: Three bindings are deleted

- **WHEN** install migrates an operator installed from any manifest release
- **THEN** none of the three bindings exists afterwards, the module's bindings exist, and install reports each deleted binding with the rendered binding that replaces it

#### Scenario: Binding without a replacement refuses the install

- **WHEN** the render holds no `ClusterRoleBinding` whose `roleRef` names `ClusterRole opm-operator-manager-role`, and the proven `ClusterRoleBinding opm-operator-manager-rolebinding` exists
- **THEN** install refuses before any object changes, naming the binding and its `roleRef`

### Requirement: The migration deletes only proven objects, and refuses otherwise

The migration SHALL delete an object only when that object meets the proof. When an object the migration would adopt, recreate or delete (a rendered object on the proof list, the earlier Deployment, or a superseded binding) exists and is not proven, install SHALL refuse before any object changes, naming the object and the label or identity that failed the proof. An unproven object on the proof list that the module does not render and the migration does not delete SHALL NOT refuse the install; install SHALL leave it unchanged. The migration SHALL NOT delete or change any of the following:

- any CustomResourceDefinition, other than the CRD step's own apply;
- any custom resource stored under the operator's CRDs;
- the operator Namespace, other than the instance apply's own apply;
- any object recorded in the inventory of another ModuleInstance.

#### Scenario: Unproven binding refuses the install

- **WHEN** `ClusterRoleBinding opm-operator-manager-rolebinding` exists but carries the `module-instance.opmodel.dev/uuid` of another instance
- **THEN** install refuses before any object changes and names the binding

#### Scenario: Unproven leftover does not block the install

- **WHEN** `ClusterRole opm-operator-modulerelease-viewer-role` exists without the labels its manifests set, and every object the migration adopts, recreates or deletes is proven
- **THEN** install migrates, and that ClusterRole keeps its `metadata.resourceVersion`

#### Scenario: Custom resources and managed workloads are untouched

- **WHEN** install migrates an operator that manages ModuleInstances with deployed workloads
- **THEN** every ModuleInstance, ModulePackage, Platform and TransformerRegistration keeps its `metadata.uid` and `metadata.generation`
- **AND** every workload object recorded in those instances' inventories keeps its `metadata.uid` and `metadata.resourceVersion`

### Requirement: Every refusing check runs before the migration's first write

Install SHALL compute and check the whole migration, including the proof of every object it will adopt or delete, together with its other refusing checks, before the apply guard of its check phase and before its first write to the cluster. Only after all of them pass SHALL install write, in this order:

1. the CRD step;
2. the field-ownership moves;
3. the delete of the earlier Deployment, then the deletes of the superseded bindings;
4. the instance apply.

A migration install that refuses SHALL leave the earlier operator running and every object of the earlier manifest unchanged.

#### Scenario: Refusal after proof failure changes nothing

- **WHEN** install refuses because one object fails the proof
- **THEN** every object of the earlier manifest, the four CRDs included, keeps its `metadata.resourceVersion`, and the earlier operator's pod keeps running

#### Scenario: Deletes follow the CRD step

- **WHEN** a migration install passes every check
- **THEN** the CRDs are applied and served before the earlier Deployment or any binding is deleted, the Deployment is deleted before the bindings, and every delete happens before the operator instance is applied

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

Beside its ordinary per-object lines, an install that adopts, recreates or deletes an object SHALL report each adopted object, the recreated Deployment and each deleted binding. That install SHALL also report each proven object of an earlier manifest that is present on the cluster, that the module does not render and that the migration does not delete; install SHALL leave such an object in place. A refused migration SHALL name every object that blocks it, not only the first. An install that adopts, recreates and deletes nothing SHALL print no migration lines, even when objects of an earlier manifest remain in place.

#### Scenario: Leftover object of an older release is named and kept

- **WHEN** install migrates a cluster that still holds `ClusterRole opm-operator-modulerelease-viewer-role` from a v0.7 install
- **THEN** install names it as left in place, and afterwards it still exists with the same `metadata.resourceVersion`

#### Scenario: Install after a completed migration prints no migration lines

- **WHEN** install runs again after a completed migration on a cluster that still holds `ClusterRole opm-operator-modulerelease-viewer-role`
- **THEN** install prints no migration lines

#### Scenario: Refusal names every unproven object

- **WHEN** two objects on the proof list fail the proof
- **THEN** install's refusal names both of them

### Requirement: The CRDs-only form admits proven CRDs and makes no other migration write

`opm operator install --crds-only` SHALL prove the CRDs it applies against the proof list and SHALL admit the proven ones through its guard, so it does not refuse on a cluster whose operator was installed from an earlier manifest. It SHALL refuse an unproven existing CRD as the full install does. It SHALL NOT delete or recreate any object, SHALL NOT move the field ownership of any object, and SHALL print no migration lines.

#### Scenario: CRDs-only over a manifest install

- **WHEN** `opm operator install --crds-only` runs on a cluster whose operator was installed from an earlier manifest
- **THEN** it applies the four CRDs and succeeds
- **AND** the earlier Deployment and the three bindings keep their `metadata.uid` and `metadata.resourceVersion`

#### Scenario: Full install after CRDs-only completes the migration

- **WHEN** `opm operator install --crds-only` ran on a manifest-installed cluster, and a full `opm operator install` of the same version runs next
- **THEN** the full install migrates the remaining objects and records every object the module renders, the four CRDs included
