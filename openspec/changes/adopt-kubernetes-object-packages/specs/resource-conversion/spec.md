## REMOVED Requirements

### Requirement: Resource wraps cue.Value with provenance
**Reason**: `pkg/core.Resource` is deleted. The library's `opm/k8s/object.Resource` is the same wrapper, field for field (0012:D1).
**Migration**: Use `object.Resource`, built from the kernel output with `object.NewResource` or `object.Resources`.

### Requirement: Resource provides accessor methods
**Reason**: The accessors live on `object.Resource` in the library, unchanged.
**Migration**: Call the same methods (`Kind`, `Name`, `Namespace`, `APIVersion`, `GVK`, `Labels`, `Annotations`) on `object.Resource`.

### Requirement: Resource provides conversion methods
**Reason**: The conversions live on `object.Resource` in the library, and `object.Export` replaces the pair of calls the cli made. `MarshalYAML` and `ToMap` are not in `pkg/core` at all.
**Migration**: Use `object.Export` to get the JSON and the unstructured object from one CUE export; `object.Resource.MarshalJSON` and `ToUnstructured` remain for a single object.

### Requirement: Label constants in pkg/core and GVK weights in pkg/resourceorder
**Reason**: Both packages are deleted; the labels are the library's `opm/k8s/labels` and the weights are `opm/k8s/object.Weight`.
**Migration**: See the requirements "Label keys come from the library" and "Object order comes from the library weight table".

## ADDED Requirements

### Requirement: Rendered objects convert through the library's single export

The CLI SHALL convert a render's compiled objects with the library's `opm/k8s/object`: it SHALL wrap them with `object.Resources` and export them with exactly one `object.Export` call per render. The render digest SHALL be computed from the exported JSON and the objects the CLI applies SHALL be the exported objects, so no compiled object is exported from CUE twice. The render digest SHALL keep its algorithm: sort by group (the `apiVersion` up to its last `/`), kind, namespace and name, then hash each object's CUE-export JSON in that order, so the same render yields the same digest it yielded before this conversion. An export failure SHALL exit with the general error code and name the failing resource.

#### Scenario: The digest and the apply objects come from one export

- **WHEN** `opm instance apply` or `opm module apply` renders a module
- **THEN** the render digest and the objects passed to apply both come from one `object.Export` over the render's compiled objects

#### Scenario: The render digest does not move

- **WHEN** the CLI digests the three-object test set (a Deployment, a Service and a ConfigMap in namespace `ns`)
- **THEN** the digest equals the value recorded before the conversion moved to the library

### Requirement: Label keys come from the library

The CLI SHALL read every OPM label key and managed-by value, and decide whether a managed-by value is an OPM runtime, through the library's `opm/k8s/labels` (`ManagedBy`, `ManagedByCLI`, `ManagedByController`, `ManagedByLegacy`, `Component`, `ComponentName`, `ModuleInstanceName`, `ModuleInstanceNamespace`, `ModuleInstanceUUID`, `IsOPMManagedBy`), and SHALL NOT keep a copy of them (0012:D1).

#### Scenario: Label values are unchanged

- **WHEN** the CLI writes its inventory record or checks a live object's managed-by label
- **THEN** it uses the keys and values `app.kubernetes.io/managed-by`, `opm-cli`, `opm-controller`, `open-platform-model`, `opmodel.dev/component`, `component.opmodel.dev/name` and `module-instance.opmodel.dev/{name,namespace,uuid}`, as before

### Requirement: Object order comes from the library weight table

Every CLI path that orders Kubernetes objects (apply, delete, prune, the `instance tree` view, the `module build` output, and the operator install and uninstall, which apply and delete through the same paths) SHALL order by the library's kind-class weight, `opm/k8s/object.Weight`. Apply, delete, prune and the tree view SHALL sort through `object.Sort`, ascending for apply and descending for delete and prune, and SHALL keep objects of equal weight in their input order. The `module build` output SHALL break equal weights by namespace, then name, as the `cmd-structure` requirement for its output order states. The CLI SHALL NOT keep its own weight table. Every weight constant, group-version-kind entry and kind entry of the table the CLI applied by before the move SHALL keep its weight in `object.Weight`, so the order of every kind the CLI table held is unchanged; the library's own weight-table guard test covers entries the library adds (0012:D5:R1/R2).

#### Scenario: Ascending order for apply

- **WHEN** the sort runs ascending over a Deployment, a CustomResourceDefinition and a ConfigMap
- **THEN** the result is CustomResourceDefinition, ConfigMap, Deployment

#### Scenario: Descending order for delete

- **WHEN** the sort runs descending over the same three objects
- **THEN** the result is Deployment, ConfigMap, CustomResourceDefinition

#### Scenario: Equal weights keep input order

- **WHEN** the sort runs over two ConfigMaps `b` then `a`
- **THEN** the result is `b` then `a`

#### Scenario: The library table equals the table the CLI applied by

- **WHEN** the CLI's order test checks every weight constant, every group-version-kind entry, every kind entry (through a group and version the group-version-kind entries do not hold) and both fallbacks of the table the CLI carried before the move
- **THEN** `object.Weight` returns the same weight for each
