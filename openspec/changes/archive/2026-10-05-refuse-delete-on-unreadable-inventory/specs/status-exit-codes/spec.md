## REMOVED Requirements

### Requirement: Status exits with code 1 for general errors

**Reason**: Its scenario "Kubernetes API error" said any unexpected API error during resource discovery exits 1. A failed read of one tracked resource is now an `Unknown` row and exits 2 (mod-status, "Status lists unreadable tracked resources as Unknown"). The requirement is re-added below as "Status exits with code 1 when the check cannot run", scoped to flags, configuration and the `ModuleInstance` record read.

**Migration**: A tracked resource whose read failed was dropped silently, so status exited 0 (or 5 if nothing else was readable); the old spec promised 1, which the code never did. It now exits 2 with a warning naming the resource and the error, so a status gate that passed may now fail. Exit 1 still means the status check could not run.

### Requirement: Status preserves existing connectivity exit codes

**Reason**: Its scenario "RBAC denied" said an RBAC denial on resources exits 4. A denied read of a tracked resource is now an `Unknown` row and exits 2. The requirement is re-added below as "Status exits with code 3 when no cluster client can be built", scoped to client setup.

**Migration**: None for client setup failures. A denied read of a tracked resource exits 2 with a warning naming the resource and the Forbidden error. An unreachable cluster, or a denied read of the `ModuleInstance` record, exits 1: the old spec promised 3 or 4, which the code never did. Mapping that read through `cmdutil.ExitCodeFromK8sError` is tracked in cli issue #310.

## ADDED Requirements

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
