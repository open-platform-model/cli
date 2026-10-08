## MODIFIED Requirements

### Requirement: Pre-apply existence check on first install

On first-time apply (no previous inventory), the system SHALL check each rendered resource against the cluster. If a resource exists with a `deletionTimestamp` (terminating), or exists without OPM labels (untracked), the apply SHALL fail with a clear error message. If a resource cannot be read (the read fails with any error other than NotFound), the apply SHALL fail too: the error SHALL name the resource and carry the read error. Every refusal of this check by `opm instance apply` and `opm module apply` SHALL say that the apply stopped before any change; that holds with `--create-namespace` too, since the check runs before the namespace is created and skips the objects in a namespace that is still missing. `opm operator install` SHALL NOT say so when the check refuses after the install applied its CustomResourceDefinitions. For `opm instance apply` and `opm module apply` the exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure; When this check refuses inside `opm operator install`, as that command's apply guard, the command SHALL exit with the code of its other apply-guard refusals (2), also when the refusal is a failed read. That code SHALL apply only to a read this check itself fails: `opm operator install` reads every object it applies in two earlier checks, so an object that is unreadable from the start never reaches this check and is refused there with the exit code of the read error (capability `operator-lifecycle`, "Every check that can refuse install runs before its first write"). A NotFound answer SHALL mean the resource does not exist, and passes. This check SHALL be skipped entirely when a previous inventory exists. A caller MAY pass an explicit admission set of resources; a resource in that set SHALL pass the untracked-resource test, and SHALL still fail the terminating-resource test and the unreadable-resource test. Only `opm operator install` SHALL pass a non-empty set, holding exactly the existing resources it proved came from an earlier opm-operator release manifest, or that already carry the operator instance's identity; every other caller SHALL pass none. No flag SHALL fill the set. The untracked-resource error SHALL NOT name a flag that would bypass it, since no flag does; it SHALL tell the user to remove or rename the existing resource, or to change the module so it renders a different name. Source: 0012:D8:R6.

#### Scenario: Untracked resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource already exists on the cluster without OPM labels
- **THEN** the command SHALL fail with an error indicating the resource is untracked

#### Scenario: Terminating resource detected on first install

- **WHEN** performing a first-time apply
- **AND** a rendered resource exists on the cluster with a `deletionTimestamp`
- **THEN** the command SHALL fail with an error indicating the resource is terminating

#### Scenario: Unreadable resource refused on first install

- **WHEN** performing a first-time apply
- **AND** the read of a rendered resource fails with Forbidden
- **THEN** the command SHALL fail with an error naming the resource and the read error
- **AND** `opm instance apply` SHALL exit 4
- **AND** no rendered resource SHALL be applied

#### Scenario: Refusal after the namespace was created

- **WHEN** `opm instance apply --create-namespace` performs a first-time apply into a namespace that already exists, an earlier run having created it
- **AND** the read of a rendered resource fails with Forbidden
- **THEN** the namespace SHALL stay as it is
- **AND** no rendered resource SHALL be applied

#### Scenario: Refusal with create-namespace changes nothing

- **WHEN** `opm instance apply --create-namespace` performs a first-time apply into a missing namespace
- **AND** the read of a rendered cluster-scoped resource fails with Forbidden
- **THEN** the namespace SHALL NOT have been created
- **AND** no rendered resource SHALL be applied
- **AND** the error SHALL say that the apply stopped before any change

#### Scenario: Operator install refuses an unreadable resource

- **WHEN** `opm operator install` plans a first install and every read of an object it would apply fails with Forbidden
- **THEN** the command SHALL refuse with exit code 4 and an error naming the resource and the read error
- **AND** the refusal SHALL come from the first of install's reads of that object, before this check runs

#### Scenario: Operator install apply guard fails its own read

- **WHEN** `opm operator install` plans a first install, its two earlier reads of an object it would apply were answered, and this check's read of that object fails with Forbidden
- **THEN** the command SHALL refuse with exit code 2 and an error naming the resource and the read error

#### Scenario: Absent resource passes

- **WHEN** performing a first-time apply
- **AND** the read of a rendered resource answers NotFound
- **THEN** the check SHALL NOT fail for that resource

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
