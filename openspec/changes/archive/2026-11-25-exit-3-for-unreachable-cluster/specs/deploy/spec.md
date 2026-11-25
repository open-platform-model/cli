## MODIFIED Requirements

### Requirement: A failed record delete exits by the answer of the API server

When `opm instance delete` deletes a CLI-owned instance, every tracked resource was deleted, and the delete of the `ModuleInstance` record then fails with an error other than NotFound, the exit code SHALL be 4 when the API server denied the delete (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable or the cluster sent no response, and 1 for any other failure. A NotFound answer SHALL count as deleted.

#### Scenario: Record delete fails with a server error

- **WHEN** the delete of the `ModuleInstance` record fails with an internal server error
- **THEN** the command SHALL exit 1 without a success line
