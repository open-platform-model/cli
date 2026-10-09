## REMOVED Requirements

### Requirement: Ownership guard on every apply and dry run
**Reason**: Its admission set existed only for `opm operator install` to take over an operator installed from an earlier release manifest (0012:D8:R6, withdrawn). The requirement is restated without the admission set as "Ownership guard judges every apply and dry run"; three scenarios go with it ("Admitted resource passes", "Admitted terminating resource still fails", "Other commands admit nothing"), and the last of them is already stated by "Foreign resource refused on a first apply".
**Migration**: None for `opm instance apply` and `opm module apply`: no caller but `opm operator install` passed a set. For `opm operator install`, see capability `operator-lifecycle`.

## ADDED Requirements

### Requirement: Ownership guard judges every apply and dry run

On every apply in CLI-executor mode, the first as well as every later one, `opm instance apply` and `opm module apply` SHALL read each rendered resource from the cluster and SHALL take the decision to apply it from the ownership verdict the CLI shares with the operator, asked with the instance identity of the render. A resource that does not exist SHALL pass. An existing resource that the instance's `ModuleInstance` record does not list SHALL be refused when it carries no OPM managed-by label, or when its `module-instance.opmodel.dev/uuid` label names another instance. An existing resource with a `deletionTimestamp` SHALL be refused whether or not the record lists it. When any resource is refused, the apply SHALL fail before its first write, the namespace of `--create-namespace` included, SHALL report every refused resource by kind, namespace and name with the reason, SHALL say that the apply stopped before any change, and SHALL exit 1. Source: 0012:D4:R2, 0012:D8:R1, 0012:D8:R5.

When the instance has no record and a resource is refused because it carries another instance's UUID, the refusal SHALL add one line that says what to do when the resources are the instance's own under an earlier identity (its module path, its name or its namespace changed, its record was deleted, or opm v1.0.0-alpha.1 or older recorded it in a Secret): annotate each resource as the refusal shows, with nothing to remove first.

The one override SHALL be the adopt annotation: an existing resource whose `opmodel.dev/adopt` annotation holds this instance's UUID SHALL pass the ownership refusals, SHALL be applied and SHALL be recorded in the instance's inventory. The annotation SHALL NOT override the refusal of a terminating resource. The CLI SHALL NOT set the annotation and SHALL offer no flag that sets it or that bypasses the guard. Every ownership refusal SHALL name the annotation and the UUID to set, and SHALL NOT name a flag. Source: 0012:D8:R2, 0012:D8:R3.

If a rendered resource cannot be read (the read fails with any error other than NotFound), the apply SHALL fail before its first write: the error SHALL name the resource and carry the read error, and the exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer, or a kind the cluster does not serve yet, SHALL mean the resource does not exist. With `--create-namespace` and a missing namespace, the guard SHALL skip the rendered resources in that namespace.

A dry run SHALL run the same guard, with the same reads, and the same verdict as the real run, and SHALL write nothing. For each resource the real run would refuse, the dry run SHALL print one line that names the resource by kind, namespace and name with the status `would refuse` and the reason of the verdict, which names the owning or adopting instance where there is one and the adopt annotation to set where one lifts the refusal. The dry run SHALL then stop where the real run stops: it SHALL send no rendered resource to the server, SHALL print no prune preview, and SHALL exit 1, the code of the real refusal, with an error that gives the number of refused resources, says that a real apply would be refused, and says that the dry run changed nothing. On a first apply it SHALL add the same earlier-identity line as the real refusal. When a rendered resource cannot be read, the dry run SHALL fail with the same error and the same exit code (4, 3 or 1) as the real run. A dry run of an operator-managed instance SHALL NOT run the guard: the operator applies that instance.

The guard SHALL take no admission set: no caller, `opm operator install` included, SHALL lift a refusal for a resource by a proof of where the resource came from. The record of the instance and the adopt annotation SHALL be the only two things that let an existing resource pass. When the guard refuses in the check phase of `opm operator install`, the exit code SHALL be that command's apply-guard code (capability `operator-lifecycle`). When it refuses inside that command's instance apply, after the install's writes, the refusal SHALL NOT say that nothing was changed.

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

#### Scenario: Dry run previews a refusal and exits 1

- **WHEN** `opm instance apply --dry-run` runs and a rendered resource exists without an OPM managed-by label
- **THEN** the output SHALL list that resource as `would refuse` with the reason and the adopt annotation to set
- **AND** the command SHALL exit 1
- **AND** no rendered resource SHALL be sent to the server and no record SHALL be written

#### Scenario: Dry run names the owner of a refused resource

- **WHEN** `opm module apply --dry-run` runs and a rendered resource the record does not list carries another instance's UUID label
- **THEN** the `would refuse` line SHALL name that instance's UUID
- **AND** the command SHALL exit 1

#### Scenario: Dry run lists every refused resource

- **WHEN** `opm instance apply --dry-run` runs and two rendered resources exist without an OPM managed-by label
- **THEN** the output SHALL hold one `would refuse` line for each

#### Scenario: Dry run with an unreadable resource

- **WHEN** `opm instance apply --dry-run` runs and the read of a rendered resource fails with Forbidden
- **THEN** the command SHALL exit 4 with an error naming the resource and the read error

#### Scenario: Dry run with nothing to refuse

- **WHEN** `opm instance apply --dry-run` runs and every existing rendered resource is the instance's own
- **THEN** the output SHALL hold no `would refuse` line and the command SHALL exit 0 when nothing else fails

#### Scenario: Operator install lifts no refusal

- **WHEN** `opm operator install` runs on a cluster with no record of the operator's instance
- **AND** a rendered resource exists without an OPM managed-by label and without an adopt annotation
- **THEN** the guard SHALL refuse that resource as it does for `opm instance apply`
