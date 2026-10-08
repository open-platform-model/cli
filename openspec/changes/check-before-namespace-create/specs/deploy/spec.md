## ADDED Requirements

### Requirement: Create-namespace creates the namespace only after every check passed

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists, and SHALL create a missing namespace only after every check of the apply that can refuse has passed: the cluster gates, the read of the `ModuleInstance` record, the ownership decision, the status-permission check, the empty-render guard and the first-install existence check. The namespace SHALL be created before the first rendered resource is applied. An apply that one of these checks refuses SHALL NOT create the namespace and SHALL leave the cluster unchanged. A check that reads inside the namespace while it is still missing SHALL treat the namespace as holding nothing: no `ModuleInstance` record, so the apply is a first install, and no resource, so the existence check passes for every object in it. A failure of the first read (whether the namespace exists) SHALL stop the apply with the exit code of its cause, before any other step. A successful apply SHALL end with the same cluster state as before this requirement: the namespace exists and every rendered resource is applied; an apply that renders nothing and has no record SHALL still create the namespace. A dry run SHALL create nothing and SHALL report first that the namespace would be created.

#### Scenario: A refused first install creates no namespace

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **AND** the read of a rendered resource fails with Forbidden
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

#### Scenario: A missing namespace reads as a first install

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist and no check refuses
- **THEN** the apply SHALL proceed as a first-time apply
- **AND** the namespace SHALL be created before the first rendered resource is applied
- **AND** the `ModuleInstance` record SHALL be written at revision 1

#### Scenario: An existing namespace is left alone

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace exists
- **THEN** no namespace SHALL be created and the apply SHALL proceed as without the flag

#### Scenario: Nothing to apply still creates the namespace

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist, the render holds no resource and no record exists
- **THEN** the command SHALL report that there is nothing to apply, create the namespace and exit 0
