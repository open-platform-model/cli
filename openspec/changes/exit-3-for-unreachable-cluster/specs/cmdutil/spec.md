## ADDED Requirements

### Requirement: ExitCodeFromK8sError maps a Kubernetes error to the exit code of its cause

`cmdutil.ExitCodeFromK8sError` SHALL return the exit code of the cause of a Kubernetes error, read from the typed error chain and never from the error text, also when the error is wrapped with `%w`: 5 (`ExitNotFound`) for NotFound, 4 (`ExitPermissionDenied`) for Forbidden or Unauthorized, 3 (`ExitConnectivityError`) for a connectivity failure, and 1 (`ExitGeneralError`) for any other error.

#### Scenario: Wrapped denial keeps its code

- **WHEN** a Forbidden error is wrapped with `fmt.Errorf("...: %w", err)`
- **THEN** `ExitCodeFromK8sError` SHALL return 4

#### Scenario: API statuses keep their codes

- **WHEN** the error is NotFound, Unauthorized, ServiceUnavailable, ServerTimeout or BadRequest
- **THEN** `ExitCodeFromK8sError` SHALL return 5, 4, 3, 3 and 1 in that order

### Requirement: A request with no response is a connectivity failure and a cancellation is not

A connectivity failure SHALL be a server timeout, service unavailable, or a request to which the cluster sent no response: an error chain that holds a `net.Error` or `context.DeadlineExceeded`, such as a refused connection, a host name that does not resolve, a timeout or a failed TLS handshake. A chain that holds `context.Canceled` SHALL NOT be a connectivity failure, also when it holds a `net.Error`.

#### Scenario: Refused connection is a connectivity failure

- **WHEN** a request fails because the connection to the API server is refused
- **THEN** `ExitCodeFromK8sError` SHALL return 3

#### Scenario: Deadline is a connectivity failure

- **WHEN** a request fails with `context.DeadlineExceeded` in its error chain
- **THEN** `ExitCodeFromK8sError` SHALL return 3

#### Scenario: Cancellation is not a connectivity failure

- **WHEN** a request fails with a `*url.Error` that wraps `context.Canceled`
- **THEN** `ExitCodeFromK8sError` SHALL return 1
