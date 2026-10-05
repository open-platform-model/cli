## Purpose

Defines the stale resource pruning behavior for `opm mod apply`. When a module is updated, resources that existed in the previous inventory but are absent from the current render are automatically pruned. Pruning is gated behind safety checks and can be disabled with `--no-prune`.

## Requirements

### Requirement: Stale resource detection

After rendering and before apply, the system SHALL compute the stale set with the library's `opm/k8s/inventory.StaleSet`: every previous inventory entry that no current entry is the same object as, comparing group, kind, namespace and name only. The component and the API version SHALL NOT count, so an object that moved to another component or another API version of its group is never stale. Resources in the stale set are candidates for pruning. Source: 0012:D7:R1.

#### Scenario: Resource removed from module

- **WHEN** the previous inventory contains entries [A, B, C] and the current render produces [A, B]
- **THEN** entry C SHALL appear in the stale set

#### Scenario: Resource renamed in module

- **WHEN** the previous inventory contains `Service/old-name` and the current render produces `Service/new-name`
- **THEN** `Service/old-name` SHALL appear in the stale set
- **AND** `Service/new-name` SHALL NOT appear in the stale set

#### Scenario: First-time apply has empty stale set

- **WHEN** there is no previous inventory (first-time apply)
- **THEN** the stale set SHALL be empty

#### Scenario: Idempotent re-apply has empty stale set

- **WHEN** the previous inventory entries are identical to the current render entries
- **THEN** the stale set SHALL be empty

#### Scenario: Component renamed without resource change is not stale

- **WHEN** the previous inventory has `Deployment/my-app` under component `web`
- **AND** the current render has `Deployment/my-app` under component `frontend`
- **THEN** `Deployment/my-app` SHALL NOT appear in the stale set
- **AND** the resource SHALL NOT be deleted

#### Scenario: Removal under a renamed component stays stale

- **WHEN** the previous inventory has `Deployment/old-app` under component `web`
- **AND** the current render does not contain `Deployment/old-app` under any component
- **THEN** `Deployment/old-app` SHALL appear in the stale set

### Requirement: Pre-apply existence check on first install

On first-time apply (no previous inventory), the system SHALL check each rendered resource against the cluster. If a resource exists with a `deletionTimestamp` (terminating), or exists without OPM labels (untracked), the apply SHALL fail with a clear error message. This check SHALL be skipped entirely when a previous inventory exists. A caller MAY pass an explicit admission set of resources; a resource in that set SHALL pass the untracked-resource test, and SHALL still fail the terminating-resource test. Only `opm operator install` SHALL pass a non-empty set, holding exactly the existing resources it proved came from an earlier opm-operator release manifest, or that already carry the operator instance's identity; every other caller SHALL pass none. No flag SHALL fill the set. The untracked-resource error SHALL NOT name a flag that would bypass it, since no flag does; it SHALL tell the user to remove or rename the existing resource, or to change the module so it renders a different name. Source: 0012:D8:R6.

#### Scenario: Untracked resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked

#### Scenario: Terminating resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource exists on the cluster with a `deletionTimestamp`
- **THEN** the command SHALL fail with an error indicating the resource is terminating

#### Scenario: Check skipped when inventory exists

- **WHEN** performing a subsequent apply (previous inventory exists)
- **THEN** the pre-apply existence check SHALL be skipped entirely

#### Scenario: Untracked-resource error names no bypass flag

- **WHEN** performing a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the error SHALL NOT mention `--force`
- **AND** the error SHALL tell the user to remove or rename the existing resource, or to change the module to render a different name

#### Scenario: Admitted resource passes the untracked test

- **WHEN** performing a first-time apply of the operator instance from `opm operator install`
- **AND** a rendered resource exists without OPM labels, and install proved it came from an earlier operator release manifest
- **THEN** the check SHALL NOT fail for that resource

#### Scenario: Admitted terminating resource still fails

- **WHEN** performing a first-time apply with an admission set
- **AND** an admitted resource exists with a `deletionTimestamp`
- **THEN** the command SHALL fail with an error indicating the resource is terminating

#### Scenario: Other commands admit nothing

- **WHEN** `opm instance apply` performs a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked

### Requirement: Stale resources pruned after successful apply

After all rendered resources have been successfully applied, stale resources SHALL be deleted in reverse weight order (highest weight first — custom resources before CRDs). Deletion SHALL treat 404 as success (resource already gone). A `Namespace` in the core API group and a `CustomResourceDefinition` in the `apiextensions.k8s.io` group SHALL never be pruned, with no flag to override this: deleting a Namespace deletes everything inside it, and deleting a CRD deletes every custom resource of its kind in the cluster. Every stale resource of a protected kind SHALL be reported as left behind, one line per resource naming its kind, namespace and name with the status `left behind`.

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

### Requirement: No prune and no inventory write on apply failure

If any resource fails to apply, the system SHALL NOT prune stale resources and SHALL NOT write the inventory record. The inventory SHALL remain at the previous state, allowing a retry to converge naturally.

#### Scenario: Partial apply failure skips prune

- **WHEN** 3 of 5 resources apply successfully and 2 fail
- **THEN** no stale resources SHALL be pruned
- **AND** the inventory record SHALL NOT be written
- **AND** the command SHALL exit with an error

### Requirement: Empty render safety gate

If the current render produces zero resources and the previous inventory is non-empty, the apply SHALL fail with an error unless `--force` is provided. This prevents accidental deletion of all resources due to a misconfigured module.

#### Scenario: Empty render without --force

- **WHEN** the render produces 0 resources
- **AND** the previous inventory has 5 entries
- **AND** `--force` is not set
- **THEN** the command SHALL fail with an error message indicating that all resources would be pruned

#### Scenario: Empty render with --force

- **WHEN** the render produces 0 resources
- **AND** the previous inventory has 5 entries
- **AND** `--force` is set
- **THEN** all 5 previous resources SHALL be pruned after apply

### Requirement: --no-prune flag skips pruning

The `--no-prune` flag SHALL skip the pruning step entirely. Stale resources SHALL remain on the cluster. The inventory record SHALL still be written with the current resource set.

#### Scenario: No-prune leaves stale resources

- **WHEN** the stale set contains 2 resources
- **AND** `--no-prune` is set
- **THEN** no stale resources SHALL be deleted
- **AND** the inventory SHALL be written with the current entries

### Requirement: --max-history flag caps change history

The `--max-history` flag SHALL control the maximum number of change entries retained in the inventory. The default SHALL be 10. After writing a new change, entries exceeding the limit SHALL be pruned from the tail of the index.

#### Scenario: Default max-history

- **WHEN** `--max-history` is not specified
- **THEN** the maximum history SHALL be 10

#### Scenario: Custom max-history

- **WHEN** the user specifies `--max-history=5`
- **THEN** the inventory SHALL retain at most 5 change entries

### Requirement: Apply flow orchestration

The apply flow SHALL follow this sequence: (1) render resources, (2) compute manifest digest, (3) compute change ID, (4) read previous inventory, (5a) compute stale set, (5b) run pre-apply existence check if first install, (6) apply all rendered resources via SSA, (7a) prune stale resources if all applied successfully, (7b) skip prune and inventory write if any apply failed, (8) write the inventory record. No step SHALL filter the stale set after it is computed: the stale set is already component-blind.

#### Scenario: Normal apply with pruning

- **WHEN** a module is applied with changes from a previous apply
- **THEN** the system SHALL render, compute digest and change ID, read inventory, compute stale, apply resources, prune stale, and write inventory in order

#### Scenario: First-time apply

- **WHEN** a module is applied for the first time
- **THEN** the system SHALL render, compute digest and change ID, find no inventory, run pre-apply check, apply resources, skip pruning (empty stale set), and write a new inventory record

### Requirement: Dry-run prune preview lists left-behind resources

When `--dry-run` is set and pruning is enabled, the apply SHALL list the stale resources a real apply would prune, each with the status `would prune`, and SHALL list separately each stale resource of a protected kind (core `Namespace`, `apiextensions.k8s.io` `CustomResourceDefinition`) with the status `left behind`. The preview SHALL use the same protected-kind rule as the real prune, so the two lists together cover the whole stale set. With `--no-prune`, neither list SHALL be printed.

#### Scenario: Preview separates pruned and left-behind resources

- **WHEN** `opm instance apply --dry-run` runs with a stale set holding a ConfigMap, a Namespace and a CRD
- **THEN** the output SHALL list the ConfigMap as `would prune`
- **AND** the output SHALL list the Namespace and the CRD as `left behind`
- **AND** nothing SHALL be deleted

#### Scenario: No-prune dry run lists nothing

- **WHEN** `opm instance apply --dry-run --no-prune` runs with a non-empty stale set
- **THEN** the output SHALL list no `would prune` and no `left behind` lines
