## ADDED Requirements

### Requirement: Status lists unreadable tracked resources as Unknown

When a tracked resource cannot be read (its GET fails with an error other than NotFound), `opm instance status` SHALL print a warning naming the resource's kind, namespace and name and the read error, and SHALL list the resource with health `Unknown`. An `Unknown` resource is not healthy: the instance's aggregate status SHALL be `NotReady`, the summary SHALL count it as not ready, and the command SHALL exit 2. An instance whose tracked resources all failed to read SHALL be reported this way, not as not found.

#### Scenario: Forbidden resource makes the instance not ready

- **WHEN** the user runs `opm instance status demo -n apps`
- **AND** every tracked resource is healthy except a ConfigMap whose read fails with Forbidden
- **THEN** the output SHALL list the ConfigMap with status `Unknown`
- **AND** a warning SHALL name the ConfigMap and the Forbidden error
- **AND** the aggregate status SHALL be `NotReady`
- **AND** the command SHALL exit 2

#### Scenario: Every read fails

- **WHEN** no tracked resource of an instance can be read
- **THEN** the command SHALL list each with status `Unknown`
- **AND** SHALL exit 2, not 5
