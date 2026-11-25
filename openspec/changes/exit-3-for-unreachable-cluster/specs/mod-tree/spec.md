## MODIFIED Requirements

### Requirement: Tree fails fast on connectivity errors

The command SHALL fail immediately with a clear error message if the Kubernetes cluster is unreachable or authentication fails, using the exit codes the shared helpers assign: 3 when `cmdutil.NewK8sClient` cannot build a client, the cluster sends no response, or the API server reports itself unavailable, 4 for authentication and RBAC failures, as mapped by `cmdutil.ExitCodeFromK8sError`.

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with code 3 and display a connectivity error message

#### Scenario: Authentication failure

- **WHEN** the kubeconfig credentials are invalid or expired
- **THEN** the command SHALL exit with code 4 and display an authentication error message
