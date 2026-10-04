# Capability: status-exit-codes

## Purpose

`opm instance status` reports its verdict through the exit code so a pipeline can act on it without parsing output: 0 when every tracked resource is healthy, 2 when the command ran but resources are not ready or a tracked resource could not be read, 5 when the instance or its resources cannot be found, 1 for usage errors and a failed read of the `ModuleInstance` record, 3 when no cluster client can be built, and 3 or 4 when the status evaluation itself hits a connectivity or RBAC failure.

## Requirements

### Requirement: Status exits with code 0 when all resources are healthy

The command SHALL exit with code 0 (`ExitSuccess`) when all discovered resources have a healthy status (`Ready`, `Applied`, `Complete` or `Bound`).

#### Scenario: All resources healthy

- **WHEN** the user runs `opm instance status my-app -n prod`
- **AND** all discovered resources are healthy
- **THEN** the command SHALL exit with code 0

### Requirement: Status exits with code 2 when resources are not ready

The command SHALL exit with code 2 (`ExitValidationError`) when the command executes successfully but one or more resources have a health status of `NotReady` or `Unknown` (any aggregate status `IsHealthy` rejects). This enables CI/CD pipelines to distinguish between "command failed" (exit 1) and "resources unhealthy" (exit 2).

#### Scenario: Some resources not ready

- **WHEN** the user runs `opm instance status my-app -n prod`
- **AND** at least one resource has health status `NotReady`
- **THEN** the command SHALL print the status table and exit with code 2

#### Scenario: Exit code 2 in CI pipeline

- **WHEN** a CI pipeline runs `opm instance status my-app -n prod`
- **AND** the Deployment is not yet ready
- **THEN** the pipeline can distinguish this from a connectivity error by checking `$?` equals 2

### Requirement: Status exits with code 5 when no resources are found

The command SHALL exit with code 5 (`ExitNotFound`) when no ModuleInstance record (or no tracked resources) exists for the instance in the given namespace.

#### Scenario: No resources found

- **WHEN** the user runs `opm instance status nonexistent -n prod`
- **AND** no ModuleInstance record or tracked resources exist for it
- **THEN** the command SHALL print a "no resources found" message and exit with code 5

### Requirement: Status exits with code 1 when the check cannot run

The command SHALL exit with code 1 (`ExitGeneralError`) for errors that prevent the status check from running: invalid flags, configuration errors, and a failed read of the instance's `ModuleInstance` record. A failed read of an individual tracked resource during discovery SHALL NOT exit 1: it is an `Unknown` row and the command exits 2.

#### Scenario: Invalid output format is a general error

- **WHEN** the user runs `opm instance status my-app -n prod -o invalid`
- **THEN** the command SHALL exit with code 1 and print `invalid output format "invalid" (valid: table, wide, yaml, json)`

#### Scenario: ModuleInstance record read error

- **WHEN** reading the instance's `ModuleInstance` record fails with an error other than NotFound
- **THEN** the command SHALL exit with code 1

#### Scenario: Tracked resource read error is not a general error

- **WHEN** the `ModuleInstance` record is read
- **AND** reading one tracked resource fails with Forbidden
- **THEN** the command SHALL list that resource as `Unknown` and exit with code 2, not 1 or 4

### Requirement: Status exits with code 3 when no cluster client can be built

The command SHALL exit with code 3 (`ExitConnectivityError`) when no Kubernetes client can be built from the resolved kubeconfig and context (`cmdutil.NewK8sClient`). Errors from the status evaluation itself SHALL map through `cmdutil.ExitCodeFromK8sError` (3 for connectivity, 4 for authentication and RBAC). Per-resource read errors during discovery SHALL NOT be mapped this way.

#### Scenario: Client cannot be built

- **WHEN** the kubeconfig or context cannot produce a Kubernetes client
- **THEN** the command SHALL exit with code 3
