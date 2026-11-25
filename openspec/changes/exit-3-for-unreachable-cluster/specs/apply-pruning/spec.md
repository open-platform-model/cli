## MODIFIED Requirements

### Requirement: A failed record read exits by the answer of the API server

When the read of the instance's `ModuleInstance` record fails with any error other than NotFound, `opm instance apply` and `opm module apply` SHALL stop with an error. The exit code SHALL be 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable or the cluster sent no response, and 1 for any other failure. A NotFound answer SHALL still mean that no record exists.

#### Scenario: Read denied

- **WHEN** `opm instance apply` runs and the `ModuleInstance` read fails with Forbidden
- **THEN** the command SHALL exit 4
- **AND** no rendered resource SHALL be applied

#### Scenario: Missing record is still a first install

- **WHEN** `opm instance apply` runs and the `ModuleInstance` read answers NotFound
- **THEN** the apply SHALL proceed as a first-time apply
