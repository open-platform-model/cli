## ADDED Requirements

### Requirement: Prune keeps stale PersistentVolumeClaims unless delete-data is set

The prune of `opm instance apply` and `opm module apply` SHALL NOT delete a stale `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. A kept claim SHALL stay in the written inventory, so that it is stale again on the next apply and an apply with `--delete-data` removes it. Each kept claim SHALL be listed on its own line, naming its kind, namespace and name with the status `kept`, at the informational log level and never as a warning or an error; one more line SHALL say how many claims were kept and name `--delete-data`. A stale claim that is no longer in the cluster (its read answers NotFound) SHALL NOT be kept: it SHALL NOT be listed, and it SHALL leave the written inventory without `--delete-data`, as an already-deleted stale resource does. A stale claim whose read fails in any other way SHALL be kept. A kept claim SHALL NOT fail the apply: the exit code SHALL be 0 and the success line SHALL print. With `--delete-data`, a stale claim SHALL be pruned like any other stale resource and SHALL leave the inventory. `--delete-data` and `--no-prune` SHALL exclude each other: both together SHALL be a usage error. On an operator-managed instance `--delete-data` SHALL have no effect, and the command SHALL print a warning that says so.

#### Scenario: Stale claim is kept and stays in the inventory

- **WHEN** the stale set contains a ConfigMap and a PersistentVolumeClaim
- **AND** `--delete-data` is not set
- **THEN** the ConfigMap SHALL be deleted and the PersistentVolumeClaim SHALL NOT be deleted
- **AND** the written inventory SHALL hold the current entries and the PersistentVolumeClaim
- **AND** the output SHALL list the claim with the status `kept`
- **AND** the command SHALL exit 0

#### Scenario: Stale claim that is already gone

- **WHEN** the stale set contains a PersistentVolumeClaim that is no longer in the cluster
- **AND** `--delete-data` is not set
- **THEN** the output SHALL NOT list that claim as `kept`
- **AND** the written inventory SHALL NOT hold it

#### Scenario: Stale claim that cannot be read

- **WHEN** the read of a stale PersistentVolumeClaim fails with an error other than NotFound
- **AND** `--delete-data` is not set
- **THEN** the claim SHALL be listed as `kept` and SHALL stay in the written inventory

#### Scenario: Delete-data prunes the claim

- **WHEN** the stale set contains a PersistentVolumeClaim
- **AND** `--delete-data` is set
- **THEN** the PersistentVolumeClaim SHALL be deleted
- **AND** the written inventory SHALL NOT hold it

#### Scenario: A later apply with the flag removes a claim kept earlier

- **WHEN** an apply kept a stale PersistentVolumeClaim
- **AND** the next apply of the same render runs with `--delete-data`
- **THEN** the claim SHALL be in the stale set of that apply and SHALL be deleted

#### Scenario: Empty render with force keeps claims

- **WHEN** the render produces 0 resources, `--force` is set and `--delete-data` is not
- **AND** the previous inventory holds a ConfigMap and a PersistentVolumeClaim
- **THEN** the ConfigMap SHALL be pruned and the PersistentVolumeClaim SHALL be kept

#### Scenario: Delete-data with no-prune

- **WHEN** `opm instance apply --no-prune --delete-data` is run
- **THEN** the process SHALL exit 1 and standard error SHALL name both flags

## MODIFIED Requirements

### Requirement: Stale resources pruned after successful apply

After all rendered resources have been successfully applied, stale resources SHALL be deleted in reverse weight order (highest weight first: custom resources before CRDs). Deletion SHALL treat 404 as success (resource already gone). A `Namespace` in the core API group and a `CustomResourceDefinition` in the `apiextensions.k8s.io` group SHALL never be pruned, with no flag to override this: deleting a Namespace deletes everything inside it, and deleting a CRD deletes every custom resource of its kind in the cluster. Every stale resource of a protected kind SHALL be reported as left behind, one line per resource naming its kind, namespace and name with the status `left behind`. A stale `PersistentVolumeClaim` in the core API group SHALL be pruned only with `--delete-data` (see "Prune keeps stale PersistentVolumeClaims unless delete-data is set").

#### Scenario: Stale resources deleted in reverse weight order

- **WHEN** the stale set contains a Deployment (weight 100) and a ConfigMap (weight 15)
- **THEN** the Deployment SHALL be deleted before the ConfigMap

#### Scenario: Already-deleted stale resource

- **WHEN** a stale resource no longer exists on the cluster
- **THEN** the prune operation SHALL treat the 404 as success

#### Scenario: Namespace excluded from pruning

- **WHEN** the stale set contains a Namespace resource
- **THEN** the Namespace SHALL NOT be pruned

#### Scenario: CRD excluded from pruning

- **WHEN** the stale set contains a `CustomResourceDefinition` in the `apiextensions.k8s.io` group
- **THEN** the CRD SHALL NOT be deleted
- **AND** the other stale resources SHALL still be pruned

#### Scenario: Protected stale resources are reported as left behind

- **WHEN** the stale set contains a Namespace and a CRD
- **THEN** the apply output SHALL list both with the status `left behind`
- **AND** the apply SHALL NOT fail because of them

#### Scenario: Same kind name in another group is not protected

- **WHEN** the stale set contains a resource of kind `Namespace` whose API group is not the core group
- **THEN** that resource SHALL be pruned like any other stale resource

### Requirement: Dry-run prune preview lists left-behind resources

When `--dry-run` is set and pruning is enabled, the apply SHALL list the stale resources a real apply would prune, each with the status `would prune`, SHALL list separately each stale resource of a protected kind (core `Namespace`, `apiextensions.k8s.io` `CustomResourceDefinition`) with the status `left behind`, and SHALL list separately each stale core `PersistentVolumeClaim` that a real apply with the same flags would keep, with the status `kept`. The preview SHALL use the same protected-kind rule and the same `--delete-data` rule as the real prune, so the three lists together cover the whole stale set. With `--no-prune`, none of the lists SHALL be printed.

#### Scenario: Preview separates pruned and left-behind resources

- **WHEN** `opm instance apply --dry-run` runs with a stale set holding a ConfigMap, a Namespace and a CRD
- **THEN** the output SHALL list the ConfigMap as `would prune`
- **AND** the output SHALL list the Namespace and the CRD as `left behind`
- **AND** nothing SHALL be deleted

#### Scenario: No-prune dry run lists nothing

- **WHEN** `opm instance apply --dry-run --no-prune` runs with a non-empty stale set
- **THEN** the output SHALL list no `would prune` and no `left behind` lines

#### Scenario: Preview lists a claim as kept

- **WHEN** `opm instance apply --dry-run` runs with a stale set holding a ConfigMap and a PersistentVolumeClaim
- **THEN** the output SHALL list the ConfigMap as `would prune` and the claim as `kept`

#### Scenario: Preview with delete-data lists a claim as pruned

- **WHEN** `opm instance apply --dry-run --delete-data` runs with a stale set holding a PersistentVolumeClaim
- **THEN** the output SHALL list the claim as `would prune`
