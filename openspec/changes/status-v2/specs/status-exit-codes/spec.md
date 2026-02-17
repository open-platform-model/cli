## ADDED Requirements

### Requirement: Status exits with code 0 when all resources are healthy

The command SHALL exit with code 0 (`ExitSuccess`) when all discovered resources have a health status of `Ready` or `Complete`.

#### Scenario: All resources healthy

- **WHEN** the user runs `opm mod status --release-name my-app -n prod`
- **AND** all discovered resources are healthy
- **THEN** the command SHALL exit with code 0

### Requirement: Status exits with code 2 when resources are not ready

The command SHALL exit with code 2 (`ExitValidationError`) when the command executes successfully but one or more resources have a health status of `NotReady` or `Unknown`. This enables CI/CD pipelines to distinguish between "command failed" (exit 1) and "resources unhealthy" (exit 2).

#### Scenario: Some resources not ready

- **WHEN** the user runs `opm mod status --release-name my-app -n prod`
- **AND** at least one resource has health status `NotReady`
- **THEN** the command SHALL print the status table and exit with code 2

#### Scenario: Exit code 2 in CI pipeline

- **WHEN** a CI pipeline runs `opm mod status --release-name my-app -n prod`
- **AND** the Deployment is not yet ready
- **THEN** the pipeline can distinguish this from a connectivity error by checking `$?` equals 2

### Requirement: Status exits with code 5 when no resources are found

The command SHALL exit with code 5 (`ExitNotFound`) when no resources match the release selector in the given namespace. This replaces the current behavior of exiting with code 1 for this case.

#### Scenario: No resources found

- **WHEN** the user runs `opm mod status --release-name nonexistent -n prod`
- **AND** no resources match the selector
- **THEN** the command SHALL print a "no resources found" message and exit with code 5

#### Scenario: Ignore not found overrides to exit 0

- **WHEN** the user runs `opm mod status --release-name nonexistent -n prod --ignore-not-found`
- **AND** no resources match the selector
- **THEN** the command SHALL print "no resources found (ignored)" and exit with code 0

### Requirement: Status exits with code 1 for general errors

The command SHALL exit with code 1 (`ExitGeneralError`) for errors that prevent the status check from completing, such as invalid flags, configuration errors, or unexpected API failures. This is unchanged from current behavior.

#### Scenario: Invalid output format

- **WHEN** the user runs `opm mod status --release-name my-app -n prod -o invalid`
- **THEN** the command SHALL exit with code 1 with an error message indicating the invalid format

#### Scenario: Kubernetes API error

- **WHEN** the Kubernetes API returns an unexpected error during resource discovery
- **THEN** the command SHALL exit with code 1

### Requirement: Status preserves existing connectivity exit codes

The command SHALL continue to use exit code 3 (`ExitConnectivityError`) for cluster connectivity failures and exit code 4 (`ExitPermissionDenied`) for RBAC errors. These are handled by the existing `exitCodeFromK8sError` function and are not changed.

#### Scenario: Cluster unreachable

- **WHEN** the cluster is unreachable
- **THEN** the command SHALL exit with code 3

#### Scenario: RBAC denied

- **WHEN** the user lacks permissions to list resources
- **THEN** the command SHALL exit with code 4
