## Why

The root help of `opm` promises exit 3 when the cluster cannot be reached. The cluster commands do not keep that promise. `cmdutil.NewK8sClient` sends no request, so the first request of a command is where an unreachable cluster fails. client-go reports that failure as a `*url.Error` (a refused connection, a failed DNS lookup, a dial timeout, a TLS failure), not as an API status. `cmdutil.ExitCodeFromK8sError` gives 3 only for the API statuses ServerTimeout and ServiceUnavailable, so every such failure exits 1. `opm instance status`, `tree`, `events` and `delete` do not map the error of their record read at all: they exit 1 also when the API server denies the read.

A pipeline cannot tell "the cluster is down" (3) and "access is denied" (4) from other failures (1). Issue 310 asks for this.

## What Changes

- `cmdutil.ExitCodeFromK8sError` gives 3 when the server sent no response: a `net.Error` or `context.DeadlineExceeded` in the error chain. The caller's own cancellation (`context.Canceled`, for example Ctrl-C) stays 1. The API statuses keep their codes.
- The private copy of that function in `internal/workflow/apply` is removed. Apply uses the shared function, so `opm instance apply` and `opm module apply` follow the same rule.
- `query.ResolveInventory` maps a failed read of the `ModuleInstance` record through the shared function: 4 when it is denied, 3 when the cluster cannot be reached, 1 for other failures. The error is printed once; today it is printed twice.
- The help of `opm instance status` no longer says that a failed record read exits 1.

Every cluster command that exits 1 today for an unreachable cluster exits 3 after this change. Commands that use the shared function: `opm instance status`, `tree`, `events`, `delete`, `list`, `diff`, `apply`, `opm module apply`, `opm controller install` and `uninstall`. The root help already documents exit 3 for this case, so the change makes the code keep the documented contract.

Affected packages: `internal/cmdutil`, `internal/workflow/query`, `internal/workflow/apply`, `internal/cmd/instance`, `tests/e2e`.

Release class: PATCH after GA, and `beta.N+1` in beta. It is not marked breaking. Exit 3 for an unreachable cluster is the contract that the root help documents, and the change brings the code in line with it. A script that tests for exit 1 on an unreachable cluster sees 3; the PR title states that change.

Commit type and PR title for the implementing PR: `fix(cmd): exit 3 when the cluster is unreachable and 4 when a cluster read is denied`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cmdutil`: adds the requirement for `ExitCodeFromK8sError`, including the "no response" class.
- `status-exit-codes`: a failed record read exits 3, 4 or 1 by its cause, not always 1.
- `mod-tree`: exit 3 covers a cluster that sends no response.
- `apply-pruning`: a failed record read exits 3 also when the cluster sends no response.
- `deploy`: a failed record delete exits 3 also when the cluster sends no response.
- `controller-lifecycle`: a failed record delete and a failed read in the wait for terminating objects exit 3 also when the cluster sends no response.
- `k8s-helpers`: a failed discovery request exits 3 also when the cluster sends no response.
- `inst-commands`: a diff read failure is in the connectivity class also when the cluster sends no response.

## Enhancement

None.

## Impact

- Code: one new branch in one function, one removed copy, one changed error return in `ResolveInventory`, one help line.
- Tests: new rows in the table test of `ExitCodeFromK8sError`, reactor tests on `ResolveInventory`, one e2e test that needs no cluster.
- Users: a changed exit code for an unreachable cluster in every cluster command, and for a denied record read in `status`, `tree`, `events` and `delete`.
