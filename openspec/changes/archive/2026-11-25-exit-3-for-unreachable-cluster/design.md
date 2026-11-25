## Context

`internal/exit` defines exit 3 as a connectivity failure and 4 as a denied request. The root help of `opm` promises `3  connectivity error (cluster or registry unreachable)`. The registry paths keep that promise: the library's `Classify` reads a `net.Error` as unreachable, and `internal/cuemod.IsConnectivityError` excludes the caller's cancellation. The platform commands keep it too: `ErrClusterRead` exits 3. The Kubernetes paths do not keep it:

- `cmdutil.NewK8sClient` builds a client and sends no request. The first request of a command is where an unreachable cluster fails.
- `cmdutil.ExitCodeFromK8sError` gives 3 only for the API statuses ServerTimeout and ServiceUnavailable.
- `internal/workflow/apply` has a private copy of that function, `exitCodeFromK8sError`, with the same branches.
- `query.ResolveInventory` (`status`, `tree`, `events`, `delete`) returns exit 1 for any failed record read, also a denied one. It logs the error and returns an `ExitError` without `Printed`, so `internal/cmd/run.go` prints the error a second time.

## Goals / Non-Goals

**Goals:**

- One rule for every cluster command: no response from the cluster is exit 3.
- A failed record read in the four inventory commands exits by its cause (3, 4 or 1) and prints once.

**Non-Goals:**

- Probing the server when the client is built. The first request still finds the failure.
- A per-entry read error during discovery. It stays an unreadable entry and changes no exit code (capability `resource-discovery`).
- The other exit-code functions (`deleteFailureExitCode`, the diff `failureExitCode`, `platform check`). They call the shared function or have their own contract, and they get the new rule through it where they call it.

## Research & Decisions

### What client-go returns for an unreachable cluster

**Context**: Issue 310 proposes to route the record-read error through `ExitCodeFromK8sError`. That gives 3 only if the error is an API status.
**Explored**: A standalone probe with the cli's client-go version: a GET to a refused port, a host name that does not resolve and an address that does not answer. Evidence: `evidence/01-client-go-unreachable-error/`.
**Options considered**:
1. Route the record read through the function, and change nothing else. Pro: smallest diff. Con: the probe shows that every unreachable case is a `*url.Error`, so the function returns 1, and exit 3 never fires.
2. Add the transport check in `ResolveInventory` only. Pro: exit 3 for the four inventory commands. Con: `list`, `diff` and `apply` keep exiting 1, so one CLI has two rules.
3. Add the transport check to the shared function. Pro: one rule for every cluster command, the same as the registry paths. Con: more commands change exit code.
**Decision**: Option 3 (owner).
**Rationale**: The probe shows that option 1 does not meet the issue's goal. Option 3 matches the rule that the library and the platform commands already use.

### What counts as no response

**Context**: A `*url.Error` is a `net.Error` for every round-trip failure, also when the caller canceled the context.
**Options considered**:
1. Any `net.Error`. Con: Ctrl-C exits 3.
2. `net.Error` or `context.DeadlineExceeded`, but never when the chain holds `context.Canceled`. Pro: the same rule as `internal/cuemod.IsConnectivityError` for the registry.
**Decision**: Option 2. A failed TLS handshake is a `*url.Error` too and exits 3, the same as the library's registry rule (assumption).

### Release class

**Decision**: `fix(cmd):`, with no `!` (owner). The root help documents exit 3 for an unreachable cluster, and the change makes the code keep it. The PR title names the change, so the CHANGELOG line says it.

### Where the tests go

**Decision** (owner): the rule is tested in the table test of `ExitCodeFromK8sError`. The wiring is tested by reactor tests on `ResolveInventory` and by one e2e test that runs `opm instance status` against a kubeconfig whose server is `https://127.0.0.1:1`. That e2e test needs no kind cluster.

## Design

```go
// internal/cmdutil/k8s.go
func ExitCodeFromK8sError(err error) int {
	switch {
	case apierrors.IsNotFound(err):
		return opmexit.ExitNotFound
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return opmexit.ExitPermissionDenied
	case apierrors.IsServerTimeout(err), apierrors.IsServiceUnavailable(err), noResponse(err):
		return opmexit.ExitConnectivityError
	default:
		return opmexit.ExitGeneralError
	}
}

// noResponse reports whether the request got no answer from the cluster.
func noResponse(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded)
}
```

The `apierrors.Is*` functions already read a wrapped chain (`errors.As` on `APIStatus`), so a wrapped denial keeps 4.

```go
// internal/workflow/query/status.go, in ResolveInventory
if invErr != nil {
	instanceLog.Error("reading inventory", "error", invErr)
	return nil, nil, nil, nil, &opmexit.ExitError{
		Code:    cmdutil.ExitCodeFromK8sError(invErr),
		Err:     fmt.Errorf("reading inventory: %w", invErr),
		Printed: true,
	}
}
```

`internal/workflow/apply` deletes `exitCodeFromK8sError` and calls `cmdutil.ExitCodeFromK8sError` at each of its call sites. `internal/workflow/query` already imports `cmdutil`, so `workflow/apply` may import it too.

Command syntax, flags and output do not change. Exit codes after the change, for a failed record read in `opm instance status|tree|events|delete <file|name|uuid> [flags]`:

| Cause | Before | After |
| --- | --- | --- |
| Refused connection, DNS failure, timeout, TLS failure | 1 | 3 |
| ServerTimeout, ServiceUnavailable | 1 | 3 |
| Forbidden, Unauthorized | 1 | 4 |
| NotFound | 5 | 5 |
| Any other error, Ctrl-C | 1 | 1 |

For every other caller of the shared function, only the first row changes (1 to 3).

Example error output (once, not twice):

```text
ERROR my-app reading inventory error="getting ModuleInstance \"my-app\": Get \"https://127.0.0.1:1/apis/...\": dial tcp 127.0.0.1:1: connect: connection refused"
```

The help of `opm instance status` changes its exit code list:

```text
  1  usage error, or the instance record could not be read for another reason
  3  the cluster could not be reached
  4  permission denied
```

## Slices

### S1: The connectivity rule in the shared function

- Delivers: `ExitCodeFromK8sError` gives 3 for a request with no response and 1 for a cancellation; apply uses the shared function.
- Seam: the table test of `cmdutil.ExitCodeFromK8sError` (`internal/cmdutil/exit_test.go`).
- Depends on: none.
- Done when: the table test passes with rows for a refused dial (`*url.Error`), a wrapped `context.DeadlineExceeded`, a `*url.Error` that wraps `context.Canceled` and a `%w`-wrapped Forbidden; `internal/workflow/apply` has no private copy; `task test:unit` is green.

### S2: The record read in ResolveInventory

- Delivers: a failed record read in `status`, `tree`, `events` and `delete` exits 3, 4 or 1 by its cause and prints once; the `status` help lists the codes as they are.
- Seam: reactor tests on `query.ResolveInventory` (`internal/workflow/query/inventory_test.go`), and one e2e test of `opm instance status` against a server that refuses the connection (`tests/e2e/`).
- Depends on: S1.
- Done when: for the name path and the uuid path, a Forbidden record read gives code 4, a `*url.Error` gives 3 and an internal server error gives 1, each with `Printed` set; the e2e test sees exit 3 and the error once on stderr; `task test:unit` and that e2e test are green.

## Risks / Trade-offs

- A script that tests for exit 1 when the cluster is down sees 3. The PR title says so. The documented contract was 3.
- A `net.Error` that does not come from the cluster request (a local I/O error in the chain) also gives 3. Mitigation: the function is called only on errors from Kubernetes requests.
- The main spec `status-exit-codes` has a Purpose paragraph that says a failed record read exits 1. A delta cannot change it. Assumption: whoever archives this change corrects that sentence in the synced spec.
