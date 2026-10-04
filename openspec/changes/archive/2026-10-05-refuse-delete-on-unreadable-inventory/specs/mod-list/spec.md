## ADDED Requirements

### Requirement: List counts unreadable tracked resources as not ready

When a tracked resource of a listed instance cannot be read (its GET fails with an error other than NotFound), `opm instance list` SHALL count it toward the instance's total and not toward its ready count, as it counts a missing resource, and SHALL print one warning for that instance naming it and the number of resources it could not read, and pointing to `opm instance status` for details. The warning SHALL NOT change the command's exit code.

#### Scenario: Unreadable resource makes the instance not ready

- **WHEN** an instance tracks 5 resources, 4 are healthy and the read of the fifth fails with Forbidden
- **THEN** the STATUS column SHALL display `NotReady (4/5)`
- **AND** one warning SHALL name the instance and say that 1 tracked resource could not be read
