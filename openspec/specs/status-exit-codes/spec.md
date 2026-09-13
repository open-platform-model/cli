# Capability: status-exit-codes

## Purpose

`opm instance status` reports its verdict through the exit code so a pipeline can act on it without parsing output: 0 when every tracked resource is healthy, 2 when the command ran but resources are not ready, 5 when the instance or its resources cannot be found, 1 for usage and unexpected errors, and 3 or 4 for cluster connectivity and RBAC failures.

## Requirements

### Requirement: Status exits with code 0 when all resources are healthy

The command SHALL exit with code 0 (`ExitSuccess`) when all discovered resources have a health status of `Ready` or `Complete`.

#### Scenario: All resources healthy

- **WHEN** the user runs `opm instance status my-app -n prod`
- **AND** all discovered resources are healthy
- **THEN** the command SHALL exit with code 0

### Requirement: Status exits with code 2 when resources are not ready

The command SHALL exit with code 2 (`ExitValidationError`) when the command executes successfully but one or more resources have a health status of `NotReady` or `Unknown` (any aggregate status other than `Ready` or `Complete`). This enables CI/CD pipelines to distinguish between "command failed" (exit 1) and "resources unhealthy" (exit 2).

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

### Requirement: Status exits with code 1 for general errors

The command SHALL exit with code 1 (`ExitGeneralError`) for errors that prevent the status check from completing, such as invalid flags, configuration errors, or unexpected API failures.

#### Scenario: Invalid output format

- **WHEN** the user runs `opm instance status my-app -n prod -o invalid`
- **THEN** the command SHALL exit with code 1 and print `invalid output format "invalid" (valid: table, wide, yaml, json)`

#### Scenario: Kubernetes API error

- **WHEN** the Kubernetes API returns an unexpected error during resource discovery
- **THEN** the command SHALL exit with code 1

### Requirement: Status preserves existing connectivity exit codes

The command SHALL continue to use exit code 3 (`ExitConnectivityError`) for cluster connectivity failures and exit code 4 (`ExitPermissionDenied`) for RBAC errors. These are handled by the `cmdutil.ExitCodeFromK8sError` function and are not changed.

#### Scenario: Cluster unreachable

- **WHEN** the cluster is unreachable
- **THEN** the command SHALL exit with code 3

#### Scenario: RBAC denied

- **WHEN** the user lacks permissions to list resources
- **THEN** the command SHALL exit with code 4
