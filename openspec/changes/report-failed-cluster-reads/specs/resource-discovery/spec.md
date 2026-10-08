## MODIFIED Requirements

### Requirement: Inventory discovery reports tracked resources it could not read

When the CLI reads the live objects an instance's inventory tracks (one targeted GET per entry), every entry SHALL land in exactly one of three groups, each kept in inventory order: live (the read returned the object), missing (the read returned NotFound), or unreadable (the read failed with any other error, such as Forbidden or a timeout), the last carrying its read error. No entry SHALL be dropped. Every command that reads an instance this way (`opm instance delete`, `status`, `list`, `diff`, `tree` and `events`) SHALL tell the user about each unreadable entry instead of omitting it silently: `delete`, `status`, `list` and `diff` as their own requirements state, and `tree` and `events` with a warning that names the resource's kind, namespace and name and the read error. For `diff`, the output SHALL also say that orphan detection could not check those resources, and the command SHALL exit non-zero as its own requirement states. A warning SHALL NOT change the exit code of `tree` or `events`: a per-entry read error is not mapped through `cmdutil.ExitCodeFromK8sError`, so a Forbidden read of one tracked resource does not make `tree` exit 4. When `tree` can read none of the tracked resources, it SHALL print the warnings and then exit 5 with `no resources found`, as it does for a record that tracks no live resources.

#### Scenario: A forbidden read is reported as unreadable

- **WHEN** an instance tracks a Deployment and a ConfigMap
- **AND** the read of the ConfigMap fails with Forbidden
- **THEN** discovery SHALL return the Deployment as live
- **AND** SHALL return the ConfigMap as unreadable with the Forbidden error
- **AND** SHALL NOT return the ConfigMap as missing

#### Scenario: NotFound is still missing

- **WHEN** the read of a tracked resource returns NotFound
- **THEN** discovery SHALL return it as missing, not unreadable

#### Scenario: Tree warns about an unreadable resource

- **WHEN** the user runs `opm instance tree demo -n apps`
- **AND** one tracked resource cannot be read
- **THEN** the command SHALL print a warning naming that resource and the read error
- **AND** SHALL show the resources it could read

#### Scenario: Tree with every tracked resource unreadable

- **WHEN** the user runs `opm instance tree demo -n apps`
- **AND** no tracked resource can be read
- **THEN** the command SHALL print one warning per tracked resource naming it and the read error
- **AND** SHALL exit 5 with `no resources found`

#### Scenario: Diff warns that orphan detection is incomplete

- **WHEN** the user runs `opm instance diff` for a deployed instance
- **AND** one tracked resource cannot be read
- **THEN** the command SHALL name that resource and the read error
- **AND** SHALL say that orphan detection could not check it
- **AND** SHALL exit non-zero without printing `No differences found`
