## MODIFIED Requirements

### Requirement: Create-namespace creates the namespace only after every check passed

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists, and SHALL create a missing namespace only after every check of the apply that can refuse has passed: the cluster gates, the status-permission check, the empty-render guard and the ownership guard (capability `apply-pruning`, "Ownership guard on every apply"). The namespace SHALL be created before the first rendered resource is applied. An apply that one of these checks refuses SHALL NOT create the namespace and SHALL leave the cluster unchanged.

While the namespace is missing, the apply SHALL treat it as holding nothing and SHALL NOT send a read into it: it SHALL NOT read the `ModuleInstance` record, so the apply is a first install, and the ownership guard SHALL skip every rendered object in that namespace. Rendered cluster-scoped objects and objects in other namespaces SHALL be checked as usual. The status-permission check SHALL run unchanged; it asks about the namespace by name and does not need it to exist.

A failure of the first read (whether the namespace exists) SHALL stop the apply with the exit code of its cause, before any other step. A failure of the create SHALL stop the apply with the exit code of its cause, before any rendered resource is applied. When the namespace turns out to exist at the create although the first read found it missing, the apply SHALL stop with exit code 1 before any change and SHALL tell the user to run the command again, because nothing inside that namespace was checked. A dry run SHALL create nothing and SHALL report first that the namespace would be created.

#### Scenario: A refused first install creates no namespace

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **AND** the read of a rendered cluster-scoped resource fails with Forbidden
- **THEN** the command SHALL exit 4
- **AND** the namespace SHALL NOT exist afterwards
- **AND** no rendered resource SHALL be applied and no `ModuleInstance` SHALL be written

#### Scenario: An untracked cluster-scoped resource refuses before the namespace is created

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **AND** a rendered cluster-scoped resource already exists without OPM labels
- **THEN** the command SHALL fail naming that resource
- **AND** the namespace SHALL NOT exist afterwards

#### Scenario: A failed cluster gate creates no namespace

- **WHEN** `opm instance apply --create-namespace` runs against a cluster without the `ModuleInstance` CustomResourceDefinition
- **THEN** the command SHALL exit 2
- **AND** the namespace SHALL NOT exist afterwards

#### Scenario: A denied status permission creates no namespace

- **WHEN** `opm instance apply --create-namespace` runs and the caller may not patch `moduleinstances/status` in the instance namespace
- **THEN** the command SHALL exit 4
- **AND** the namespace SHALL NOT exist afterwards

#### Scenario: A missing namespace is a first install and is not read

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist and no check refuses
- **THEN** the command SHALL send no read of a `ModuleInstance` or of a rendered object into that namespace before it creates it
- **AND** the namespace SHALL be created before the first rendered resource is applied
- **AND** the `ModuleInstance` record SHALL be written at revision 1

#### Scenario: A caller without cluster-wide read is not refused by the missing namespace

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **AND** the API server would answer Forbidden to the caller's read of a `ModuleInstance` or of a ConfigMap in a namespace that does not exist
- **THEN** the apply SHALL NOT fail on such a read

#### Scenario: An existing namespace is left alone

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace exists
- **THEN** no namespace SHALL be created and the apply SHALL proceed as without the flag

#### Scenario: The namespace read fails

- **WHEN** `opm instance apply --create-namespace` runs and the read of the namespace fails with Forbidden
- **THEN** the command SHALL exit 4 before any other call to the cluster

#### Scenario: The namespace create fails

- **WHEN** `opm instance apply --create-namespace` runs, every check passes and the create of the namespace fails with Forbidden
- **THEN** the command SHALL exit 4
- **AND** no rendered resource SHALL be applied and no `ModuleInstance` SHALL be written

#### Scenario: The namespace appears during the checks

- **WHEN** `opm instance apply --create-namespace` found the namespace missing and another actor creates it before the apply does
- **THEN** the command SHALL exit 1, say that the namespace was created by someone else, and tell the user to run the command again
- **AND** no rendered resource SHALL be applied
