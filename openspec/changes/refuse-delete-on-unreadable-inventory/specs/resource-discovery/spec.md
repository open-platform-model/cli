## ADDED Requirements

### Requirement: Inventory discovery reports tracked resources it could not read

When the CLI reads the live objects an instance's inventory tracks (one targeted GET per entry), every entry SHALL land in exactly one of three groups, each kept in inventory order: live (the read returned the object), missing (the read returned NotFound), or unreadable (the read failed with any other error, such as Forbidden or a timeout), the last carrying its read error. No entry SHALL be dropped. Every command that reads an instance this way (`opm instance delete`, `status`, `list`, `diff`, `tree` and `events`) SHALL tell the user about each unreadable entry instead of omitting it silently: `delete`, `status` and `list` as their own requirements state, and `diff`, `tree` and `events` with a warning that names the resource's kind, namespace and name and the read error. For `diff`, the warning SHALL also say that orphan detection could not check those resources. A warning SHALL NOT change the exit code of `diff`, `tree` or `events`.

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

#### Scenario: Diff warns that orphan detection is incomplete

- **WHEN** the user runs `opm instance diff` for a deployed instance
- **AND** one tracked resource cannot be read
- **THEN** the command SHALL print a warning naming that resource and the read error
- **AND** SHALL say that orphan detection could not check it
