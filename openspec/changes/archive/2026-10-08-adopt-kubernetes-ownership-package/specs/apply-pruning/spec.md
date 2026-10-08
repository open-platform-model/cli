## ADDED Requirements

### Requirement: Prune asks the ownership verdict for each stale resource

Before the prune deletes a stale resource, it SHALL read the live object and SHALL take the decision to delete it from the delete verdict the CLI shares with the operator. The verdict SHALL be asked with the instance identity stored in the instance's `ModuleInstance` record (`status.instanceUUID`), which is the identity that applied the stale resources, also when the identity of the current render differs from it. The prune SHALL delete the resource only when the live object carries an OPM managed-by label, its `module-instance.opmodel.dev/uuid` label does not name an instance other than the recorded one, and its `opmodel.dev/adopt` annotation does not name an instance other than the recorded one. An object with no UUID label SHALL be judged without the UUID comparison, and so SHALL every object when the record holds no identity. A resource that fails the verdict SHALL NOT be deleted: it SHALL be listed with the status `left behind` and the reason, SHALL NOT count as a failure, and SHALL NOT be kept in the written inventory. The stale set itself SHALL NOT change: it stays every recorded resource the render no longer names. Source: 0012:D4:R1, 0012:D7:R1, 0012:D8:R8.

An instance's identity derives from its module's path without the major version, the instance name and the namespace. The record is found by name and namespace, so within one record the identity changes only when the module's path changes. On the first apply after such a change the prune SHALL still delete the stale resources the instance owned under the recorded identity, and the record SHALL take the rendered identity only in the write that follows the prune.

Each delete SHALL carry a precondition on the UID of the object that was read, so that an object deleted and created again under the same name since the read is not deleted. A delete the API server refuses on that precondition, and a live read that fails with any error other than NotFound, SHALL each count as a failed delete of that resource ("Failed prune keeps the entry and fails the command"). A stale `PersistentVolumeClaim` that is kept SHALL NOT be judged; with `--delete-data` it SHALL be judged like any other stale resource. The dry-run prune preview SHALL stay as it is: it reads no stale resource for a verdict, so it MAY list as `would prune` a resource the real run leaves behind.

#### Scenario: Stale name taken by a user's object

- **WHEN** the stale set holds ConfigMap `old`
- **AND** the live ConfigMap `old` carries no OPM managed-by label
- **THEN** the ConfigMap SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that OPM does not manage it
- **AND** the written inventory SHALL NOT hold it
- **AND** the command SHALL exit 0

#### Scenario: Stale resource now owned by another instance

- **WHEN** the live object of a stale resource carries a `module-instance.opmodel.dev/uuid` that is not the recorded identity
- **THEN** the resource SHALL NOT be deleted and SHALL be listed as `left behind`

#### Scenario: Stale resource adopted by another instance

- **WHEN** the live object of a stale resource carries `opmodel.dev/adopt` with another instance's UUID
- **THEN** the resource SHALL NOT be deleted and SHALL be listed as `left behind`

#### Scenario: Stale object after the instance identity changed

- **WHEN** the module of an instance moved to a new path, so the rendered identity differs from the identity in the record
- **AND** the stale set holds a ConfigMap whose live `module-instance.opmodel.dev/uuid` label is the recorded identity
- **THEN** the ConfigMap SHALL be deleted
- **AND** the written record SHALL hold the rendered identity and SHALL NOT hold the ConfigMap

#### Scenario: Stale object that carries neither identity

- **WHEN** the rendered identity differs from the identity in the record
- **AND** the live object of a stale resource carries a UUID label equal to neither
- **THEN** the resource SHALL NOT be deleted and SHALL be listed as `left behind`

#### Scenario: Owned stale resource is deleted with a UID precondition

- **WHEN** the live object of a stale resource carries an OPM managed-by label and the recorded identity
- **THEN** the delete request SHALL carry a precondition on that object's UID

#### Scenario: Object replaced between the read and the delete

- **WHEN** the API server refuses the delete of a stale resource because the UID precondition does not match
- **THEN** the resource SHALL NOT be reported as pruned
- **AND** its entry SHALL stay in the written inventory
- **AND** the command SHALL exit 1 without the success line

#### Scenario: Live read denied

- **WHEN** the live read of a stale resource fails with Forbidden
- **THEN** the resource SHALL NOT be deleted
- **AND** its entry SHALL stay in the written inventory
- **AND** the command SHALL exit 1 without the success line

#### Scenario: Kept claim is not judged

- **WHEN** the stale set holds a PersistentVolumeClaim and `--delete-data` is not set
- **THEN** the claim SHALL be listed as `kept` and SHALL stay in the written inventory

## MODIFIED Requirements

### Requirement: Apply flow orchestration

The apply flow SHALL follow this sequence: (1) render resources, (2) compute manifest digest, (3) compute change ID, (4) read previous inventory, and stop if the record cannot be read, (5a) compute stale set, (5b) run pre-apply existence check if first install, (6) apply all rendered resources via SSA, (7a) if all applied successfully, read each stale resource and prune those the delete verdict allows, judged with the instance identity the record held when step 4 read it, (7b) skip prune and inventory write if any apply failed, (8) write the inventory record, with every stale entry that step 7a failed to delete kept in it and with the instance identity of this render, (9) exit with an error if step 7a failed to delete a stale resource. No step SHALL filter the stale set after it is computed: the stale set is already component-blind.

#### Scenario: Normal apply with pruning

- **WHEN** a module is applied with changes from a previous apply
- **THEN** the system SHALL render, compute digest and change ID, read inventory, compute stale, apply resources, prune stale, and write inventory in order

#### Scenario: First-time apply

- **WHEN** a module is applied for the first time
- **THEN** the system SHALL render, compute digest and change ID, find no inventory, run pre-apply check, apply resources, skip pruning (empty stale set), and write a new inventory record

#### Scenario: Record write follows a failed prune

- **WHEN** a module is applied and a stale resource fails to delete
- **THEN** the system SHALL still write the inventory record, with the failed entry kept
- **AND** the command SHALL exit with an error after the write

#### Scenario: The record takes the new identity after the prune

- **WHEN** a module is applied whose rendered instance identity differs from the identity in the record
- **THEN** the system SHALL prune the stale resources judged with the identity the record held
- **AND** only then SHALL it write the record with the rendered identity
