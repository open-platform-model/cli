## MODIFIED Requirements

### Requirement: Discovery uses preferred API resources

The CLI SHALL use `ServerPreferredResources()` instead of `ServerGroupsAndResources()` when discovering API resources on the cluster.

#### Scenario: Only preferred version discovered for each resource type

- **WHEN** discovery enumerates API resources
- **THEN** only the preferred version of each resource type SHALL be queried
- **AND** deprecated API versions (like `v1/Endpoints`) SHALL NOT be enumerated when a preferred alternative exists

#### Scenario: Discovery handles unavailable API groups gracefully

- **WHEN** some API groups fail to respond during discovery
- **THEN** the CLI SHALL continue with available groups
- **AND** the CLI SHALL log a warning indicating which groups were unavailable
- **AND** the warning SHALL include the underlying error for diagnostic purposes
