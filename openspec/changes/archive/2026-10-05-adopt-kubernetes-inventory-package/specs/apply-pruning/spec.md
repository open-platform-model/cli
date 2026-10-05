## MODIFIED Requirements

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

### Requirement: Apply flow orchestration

The apply flow SHALL follow this sequence: (1) render resources, (2) compute manifest digest, (3) compute change ID, (4) read previous inventory, (5a) compute stale set, (5b) run pre-apply existence check if first install, (6) apply all rendered resources via SSA, (7a) prune stale resources if all applied successfully, (7b) skip prune and inventory write if any apply failed, (8) write the inventory record. No step SHALL filter the stale set after it is computed: the stale set is already component-blind.

#### Scenario: Normal apply with pruning

- **WHEN** a module is applied with changes from a previous apply
- **THEN** the system SHALL render, compute digest and change ID, read inventory, compute stale, apply resources, prune stale, and write inventory in order

#### Scenario: First-time apply

- **WHEN** a module is applied for the first time
- **THEN** the system SHALL render, compute digest and change ID, find no inventory, run pre-apply check, apply resources, skip pruning (empty stale set), and write a new inventory record

## REMOVED Requirements

### Requirement: Component-rename safety check
**Reason**: The stale set is component-blind (0012:D7), so a component rename never puts an object in it and there is nothing for a filter to rescue. The filter and its call are deleted. No prune decision changes.
**Migration**: None. Both of its scenarios hold as scenarios of "Stale resource detection" ("Component renamed without resource change is not stale", "Removal under a renamed component stays stale").
