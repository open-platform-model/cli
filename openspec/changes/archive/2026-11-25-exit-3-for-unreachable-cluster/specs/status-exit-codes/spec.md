## MODIFIED Requirements

### Requirement: Status exits with code 1 when the check cannot run

The command SHALL exit with code 1 (`ExitGeneralError`) for errors that prevent the status check from running: invalid flags, configuration errors, and a failed read of the instance's `ModuleInstance` record that is not a denial or a connectivity failure. A failed read of an individual tracked resource during discovery SHALL NOT exit 1: it is an `Unknown` row and the command exits 2.

#### Scenario: Invalid output format is a general error

- **WHEN** the user runs `opm instance status my-app -n prod -o invalid`
- **THEN** the command SHALL exit with code 1 and print `invalid output format "invalid" (valid: table, wide, yaml, json)`

#### Scenario: ModuleInstance record read error

- **WHEN** reading the instance's `ModuleInstance` record fails with an internal server error
- **THEN** the command SHALL exit with code 1

#### Scenario: Tracked resource read error is not a general error

- **WHEN** the `ModuleInstance` record is read
- **AND** reading one tracked resource fails with Forbidden
- **THEN** the command SHALL list that resource as `Unknown` and exit with code 2, not 1 or 4

### Requirement: Status exits with code 3 when no cluster client can be built

The command SHALL exit with code 3 (`ExitConnectivityError`) when no Kubernetes client can be built from the resolved kubeconfig and context (`cmdutil.NewK8sClient`). A failed read of the `ModuleInstance` record, printed once, and errors from the status evaluation itself SHALL map through `cmdutil.ExitCodeFromK8sError` (3 for connectivity, 4 for authentication and RBAC). Per-resource read errors during discovery SHALL NOT be mapped this way.

#### Scenario: Client cannot be built

- **WHEN** the kubeconfig or context cannot produce a Kubernetes client
- **THEN** the command SHALL exit with code 3

#### Scenario: Cluster unreachable at the record read

- **WHEN** the user runs `opm instance status my-app -n prod` with a kubeconfig whose server refuses the connection
- **THEN** the command SHALL exit with code 3
- **AND** it SHALL print the error once

#### Scenario: Record read denied

- **WHEN** reading the instance's `ModuleInstance` record fails with Forbidden
- **THEN** the command SHALL exit with code 4
