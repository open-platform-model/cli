## MODIFIED Requirements

### Requirement: Object order comes from the library weight table

Every CLI path that orders Kubernetes objects (apply, delete, prune, the `instance tree` view, the `module build` output, and the controller install and uninstall, which apply and delete through the same paths) SHALL order by the library's kind-class weight, `opm/k8s/object.Weight`. Apply, delete, prune and the tree view SHALL sort through `object.Sort`, ascending for apply and descending for delete and prune, and SHALL keep objects of equal weight in their input order. The `module build` output SHALL break equal weights by namespace, then name, as the `cmd-structure` requirement for its output order states. The CLI SHALL NOT keep its own weight table. Every weight constant, group-version-kind entry and kind entry of the table the CLI applied by before the move SHALL keep its weight in `object.Weight`, so the order of every kind the CLI table held is unchanged; the library's own weight-table guard test covers entries the library adds (0012:D5:R1/R2).

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
