## ADDED Requirements

### Requirement: First install over resources OPM already manages warns

On a first-time apply (no `ModuleInstance` record) that is not a dry run, when one or more rendered resources already exist in the cluster with an OPM managed-by label, `opm instance apply` and `opm module apply` SHALL print one warning on the log stream before any rendered resource is applied. The warning SHALL give the number of such resources and the number of rendered resources, SHALL say that the instance has no `ModuleInstance` record, that the apply updates those resources in place and records them, and that it prunes nothing, and SHALL name the release to apply with first when an older release recorded the instance in a Secret. The warning SHALL NOT change the exit code and SHALL NOT stop the apply. The apply SHALL print no such warning when no rendered resource exists yet, or when the instance has a record.

#### Scenario: Existing managed resources on a first install

- **WHEN** `opm instance apply` runs for an instance with no `ModuleInstance` record
- **AND** two of its three rendered resources already exist with the OPM managed-by label
- **THEN** the command SHALL print one warning that names 2 of 3 resources and the missing record
- **AND** the apply SHALL go on and exit 0 when it succeeds

#### Scenario: Clean first install prints no warning

- **WHEN** `opm instance apply` runs for an instance with no `ModuleInstance` record
- **AND** none of its rendered resources exists
- **THEN** the command SHALL print no such warning

#### Scenario: Instance with a record prints no warning

- **WHEN** `opm instance apply` runs for an instance that has a `ModuleInstance` record
- **THEN** the command SHALL print no such warning
