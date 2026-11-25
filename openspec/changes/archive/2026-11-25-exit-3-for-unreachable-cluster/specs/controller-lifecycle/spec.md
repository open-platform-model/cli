## MODIFIED Requirements

### Requirement: A failed record delete exits by the cause of the failure

When `opm controller uninstall` deleted every recorded object and the delete of the controller's instance record then fails with an error other than NotFound, the exit code SHALL be 4 when the API server denied the delete (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable or the cluster sent no response, and 1 for any other failure.

#### Scenario: Record delete times out

- **WHEN** `opm controller uninstall` runs and the delete of the instance record fails with a server timeout
- **THEN** the command SHALL exit 3 without reporting the controller uninstalled

#### Scenario: Record delete fails for another reason

- **WHEN** `opm controller uninstall` runs and the delete of the instance record fails with an error that is not a denial, a server timeout, service unavailable or a request with no response
- **THEN** the command SHALL exit 1 without reporting the controller uninstalled

### Requirement: A failed read in the wait for terminating objects exits by the read error

When the wait for terminating objects fails a read with an error other than NotFound, the command SHALL exit with the code of the read error: 4 when the API server denied the read (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable or the cluster sent no response, and 1 for any other failure. An object that install cannot read from the start is therefore refused by the wait for terminating objects, with exit code 4 when the read is denied.

#### Scenario: An object unreadable from the start exits by the read error

- **WHEN** install runs on a cluster with no record of the controller's instance and every read of a ClusterRole the render names fails with Forbidden
- **THEN** the command SHALL exit 4 naming that ClusterRole and the read error, and no object in the cluster has changed
