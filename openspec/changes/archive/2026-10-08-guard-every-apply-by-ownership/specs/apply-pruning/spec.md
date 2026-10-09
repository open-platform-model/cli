## ADDED Requirements

### Requirement: Ownership guard on every apply

On every apply in CLI-executor mode, the first as well as every later one, `opm instance apply` and `opm module apply` SHALL read each rendered resource from the cluster and SHALL take the decision to apply it from the ownership verdict the CLI shares with the operator, asked with the instance identity of the render. A resource that does not exist SHALL pass. An existing resource that the instance's `ModuleInstance` record does not list SHALL be refused when it carries no OPM managed-by label, or when its `module-instance.opmodel.dev/uuid` label names another instance. An existing resource with a `deletionTimestamp` SHALL be refused whether or not the record lists it. When any resource is refused, the apply SHALL fail before its first write, the namespace of `--create-namespace` included, SHALL report every refused resource by kind, namespace and name with the reason, SHALL say that the apply stopped before any change, and SHALL exit 1. Source: 0012:D4:R2, 0012:D8:R1, 0012:D8:R5.

When the instance has no record and a resource is refused because it carries another instance's UUID, the refusal SHALL add one line that says what to do when the resources are the instance's own under an earlier identity (its module path, its name or its namespace changed, its record was deleted, or opm v1.0.0-alpha.1 or older recorded it in a Secret): annotate each resource as the refusal shows, with nothing to remove first.

The one override SHALL be the adopt annotation: an existing resource whose `opmodel.dev/adopt` annotation holds this instance's UUID SHALL pass the ownership refusals, SHALL be applied and SHALL be recorded in the instance's inventory. The annotation SHALL NOT override the refusal of a terminating resource. The CLI SHALL NOT set the annotation and SHALL offer no flag that sets it or that bypasses the guard. Every ownership refusal SHALL name the annotation and the UUID to set, and SHALL NOT name a flag. Source: 0012:D8:R2, 0012:D8:R3, 0012:D8:R6.

If a rendered resource cannot be read (the read fails with any error other than NotFound), the apply SHALL fail before its first write: the error SHALL name the resource and carry the read error, and the exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer, or a kind the cluster does not serve yet, SHALL mean the resource does not exist. With `--create-namespace` and a missing namespace, the guard SHALL skip the rendered resources in that namespace.

A dry run SHALL refuse nothing because of the guard: it SHALL NOT fail and SHALL NOT print a refusal for a resource the real run would refuse.

A caller MAY pass an explicit admission set of resources; a resource in that set SHALL pass the refusal of a resource OPM does not manage when it carries no UUID label or this instance's, and SHALL still fail every other test. Only `opm operator install` SHALL pass a non-empty set, holding exactly the existing resources it proved came from an earlier opm-operator release manifest, or that already carry the operator instance's identity; every other caller SHALL pass none. No flag SHALL fill the set. When the guard refuses in the check phase of `opm operator install`, the exit code SHALL be that command's apply-guard code (capability `operator-lifecycle`). When it refuses inside that command's instance apply, after the install's writes, the refusal SHALL NOT say that nothing was changed. Source: 0012:D8:R6.

#### Scenario: Foreign resource refused on a first apply

- **WHEN** `opm instance apply` runs for an instance with no record
- **AND** a rendered resource already exists on the cluster without an OPM managed-by label
- **THEN** the command SHALL exit 1 before any change, naming the resource
- **AND** the error SHALL name the annotation `opmodel.dev/adopt` and the instance's UUID
- **AND** the error SHALL NOT mention `--force`

#### Scenario: Foreign resource refused on a later apply

- **WHEN** `opm instance apply` runs for an instance that has a record
- **AND** the render names a resource the record does not list, which exists on the cluster without an OPM managed-by label
- **THEN** the command SHALL exit 1 and SHALL apply no rendered resource

#### Scenario: Resource of another instance refused

- **WHEN** `opm instance apply` runs
- **AND** a rendered resource the record does not list exists with an OPM managed-by label and the `module-instance.opmodel.dev/uuid` of another instance
- **THEN** the command SHALL exit 1 before any change, naming the resource and the other instance's UUID

#### Scenario: Recorded resource is applied whatever its UUID label

- **WHEN** `opm instance apply` runs for an instance whose record lists a rendered resource
- **AND** the live resource carries a UUID label that differs from the instance's and no adopt annotation
- **THEN** the guard SHALL NOT refuse it and the resource SHALL be applied

#### Scenario: Terminating resource refused on every apply

- **WHEN** `opm instance apply` runs for an instance whose record lists a rendered resource
- **AND** that resource exists with a `deletionTimestamp`
- **THEN** the command SHALL exit 1 before any change with an error saying the resource is being deleted

#### Scenario: Adopt annotation lets the instance take a resource

- **WHEN** `opm instance apply` runs and a rendered resource exists without an OPM managed-by label
- **AND** its `opmodel.dev/adopt` annotation holds the instance's UUID
- **THEN** the resource SHALL be applied and recorded in the instance's inventory

#### Scenario: Adopt annotation does not lift the terminating refusal

- **WHEN** a rendered resource has a `deletionTimestamp` and an `opmodel.dev/adopt` annotation holding the instance's UUID
- **THEN** the command SHALL exit 1 before any change

#### Scenario: First install over the instance's own resources under an earlier identity

- **WHEN** `opm instance apply` runs for an instance with no record
- **AND** every rendered resource exists with an OPM managed-by label and one other UUID
- **THEN** the command SHALL exit 1 before any change, naming each resource and the annotation to set
- **AND** the output SHALL say that nothing has to be removed first when the resources are the instance's own

#### Scenario: Every refused resource is reported

- **WHEN** two rendered resources exist without an OPM managed-by label
- **THEN** the output SHALL name both before the command exits 1

#### Scenario: Unreadable resource refused on a later apply

- **WHEN** `opm instance apply` runs for an instance that has a record
- **AND** the read of a rendered resource fails with Forbidden
- **THEN** the command SHALL exit 4 with an error naming the resource and the read error
- **AND** no rendered resource SHALL be applied

#### Scenario: Refusal with create-namespace changes nothing

- **WHEN** `opm instance apply --create-namespace` performs a first-time apply into a missing namespace
- **AND** a rendered cluster-scoped resource exists without an OPM managed-by label
- **THEN** the namespace SHALL NOT have been created
- **AND** the error SHALL say that the apply stopped before any change

#### Scenario: Absent resource passes

- **WHEN** the read of a rendered resource answers NotFound
- **THEN** the guard SHALL NOT refuse that resource

#### Scenario: Dry run refuses nothing

- **WHEN** `opm instance apply --dry-run` runs and a rendered resource exists without an OPM managed-by label
- **THEN** the dry run SHALL NOT fail because of that resource

#### Scenario: Admitted resource passes

- **WHEN** `opm operator install` applies the operator instance
- **AND** a rendered resource exists without OPM labels, and install proved it came from an earlier operator release manifest
- **THEN** the guard SHALL NOT refuse that resource

#### Scenario: Admitted terminating resource still fails

- **WHEN** an admitted resource exists with a `deletionTimestamp`
- **THEN** the guard SHALL refuse it

#### Scenario: Other commands admit nothing

- **WHEN** `opm instance apply` performs a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error naming the resource

### Requirement: Resource adopted by another instance is let go

When the instance's UUID is known and the live `opmodel.dev/adopt` annotation of a rendered resource names another instance, the apply SHALL NOT apply that resource and SHALL NOT fail because of it. This SHALL hold for a resource the instance's record lists, and for an OPM-managed resource the record does not list that carries no UUID label, this instance's UUID or the UUID its annotation names. The apply SHALL print one warning that names the resource and the adopting instance and says how to take the resource back, SHALL apply the other rendered resources, and SHALL write the inventory without that resource. The apply SHALL NOT prune or otherwise delete it. `opm operator install` is the exception: it refuses on such a resource (capability `operator-lifecycle`). Source: 0012:D8:R8, 0012:D7:R1.

#### Scenario: Recorded resource handed to another instance

- **WHEN** `opm instance apply` runs for an instance whose record lists ConfigMap `settings`
- **AND** the live ConfigMap carries `opmodel.dev/adopt` with another instance's UUID
- **THEN** the ConfigMap SHALL NOT be applied and SHALL NOT be deleted
- **AND** the output SHALL warn that the instance drops it from its inventory
- **AND** the written inventory SHALL NOT hold the ConfigMap
- **AND** the command SHALL exit 0 when the other resources apply

#### Scenario: The instance does not take the resource back

- **WHEN** a later apply of the same instance renders ConfigMap `settings` again
- **AND** the live ConfigMap carries the adopting instance's UUID in both its adopt annotation and its UUID label
- **THEN** the ConfigMap SHALL NOT be applied, and the command SHALL NOT fail because of it

#### Scenario: Annotating the resource back reverses the hand-over

- **WHEN** the adopt annotation of ConfigMap `settings` is set to the first instance's UUID
- **AND** the first instance is applied
- **THEN** the ConfigMap SHALL be applied and recorded in its inventory

## MODIFIED Requirements

### Requirement: Apply flow orchestration

The apply flow SHALL follow this sequence: (1) render resources, (2) compute manifest digest, (3) compute change ID, (4) read previous inventory, and stop if the record cannot be read, (5a) compute stale set, (5b) run the ownership guard over every rendered resource, on a first install and on every later apply, and stop before any write if it refuses, (6) apply via SSA every rendered resource the guard did not let go, (7a) if all applied successfully, read each stale resource and prune those the delete verdict allows, judged with the instance identity the record held when step 4 read it, (7b) skip prune and inventory write if any apply failed, (8) write the inventory record, with every stale entry that step 7a failed to delete kept in it, without any resource the guard let go, and with the instance identity of this render, (9) exit with an error if step 7a failed to delete a stale resource. No step SHALL filter the stale set after it is computed: the stale set is already component-blind.

#### Scenario: Normal apply with pruning

- **WHEN** a module is applied with changes from a previous apply
- **THEN** the system SHALL render, compute digest and change ID, read inventory, compute stale, run the ownership guard, apply resources, prune stale, and write inventory in order

#### Scenario: First-time apply

- **WHEN** a module is applied for the first time
- **THEN** the system SHALL render, compute digest and change ID, find no inventory, run the ownership guard, apply resources, skip pruning (empty stale set), and write a new inventory record

#### Scenario: Record write follows a failed prune

- **WHEN** a module is applied and a stale resource fails to delete
- **THEN** the system SHALL still write the inventory record, with the failed entry kept
- **AND** the command SHALL exit with an error after the write

#### Scenario: The record takes the new identity after the prune

- **WHEN** a module is applied whose rendered instance identity differs from the identity in the record
- **THEN** the system SHALL prune the stale resources judged with the identity the record held
- **AND** only then SHALL it write the record with the rendered identity

#### Scenario: The guard runs on a later apply

- **WHEN** a module is applied for an instance that has a record
- **THEN** the ownership guard SHALL run before any rendered resource is applied

## REMOVED Requirements

### Requirement: Pre-apply existence check on first install

**Reason**: The check ran only on an instance's first apply and passed every OPM-managed resource, whichever instance owned it. The guard now runs on every apply and takes its decision from the ownership verdict shared with the operator (0012:D8:R1).

**Migration**: See "Ownership guard on every apply". A resource the old check refused is still refused, with exit code 1; the message now names the adopt annotation in place of "remove or rename". An apply that used to take over an existing resource on a later apply, or a resource of another instance on a first apply, now refuses: annotate the resource with `opmodel.dev/adopt=<instance UUID>` to adopt it, or remove it from the module or the cluster.
