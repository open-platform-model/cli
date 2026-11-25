# Build report: exit-3-for-unreachable-cluster

Base: main

## Slices

- S1, the connectivity rule in the shared function
  - `a27252d0` test(cmd): cover exit codes for a cluster request with no response
  - `31f24273` fix(cmd): exit 3 when a cluster request gets no response
- S2, the record read in ResolveInventory
  - `165e2815` test(cmd): cover the exit code and single print of a failed record read
  - `9222ae1f` fix(cmd): exit 4 when the instance record read is denied and print the error once

## Assumptions

- S1 test writer. Which error does the deadline row use? A bare `fmt.Errorf("...: %w", context.DeadlineExceeded)`, with no `*url.Error`.
- S1 test writer. Do the cancel rows cover more than the scenario? Yes: one extra row with a `*url.Error` around a `*net.OpError` around `context.Canceled`, for the "also when it holds a `net.Error`" clause.
- S1 test writer. Does a refused dial that a caller wraps with `%w` also give 3? Yes: one extra row.
- S1 test writer. The spec does not say which failure kinds get their own rows. So name resolution, TLS and other timeouts have no rows. They take the same `net.Error` path as the refused dial.
- S1 session. Can the red check pass when some rows already pass? Yes. The guard rows (wrapped Forbidden, ServerTimeout, the other API statuses, both cancel rows) pass before the code change, because task 1.1 asks for them. Only the rows for new behaviour (refused dial, wrapped refused dial, DeadlineExceeded) must be red, and they were, on assertions.
- S1 code writer. Can `workflow/apply` import `cmdutil`? Yes. The design says so, and there is no import cycle.
- S1 code writer. Is `noResponse` exported? No. It is private, as in the design.
- S2 test writer. Which selector tests the uuid path? `InstanceID` only, so the `list` branch runs.
- S2 test writer. Which internal server error? `apierrors.NewInternalError`, a 500 status.
- S2 test writer. How does the e2e test reach the record read? It passes `--context unreachable` with `--kubeconfig`. Without the context flag, the command fails when it builds the client, which is a false pass.
- S2 test writer. How does the e2e test count "printed once"? It counts `connection refused` on stderr. The count was 2 before the fix.
- S2 test writer. What does the e2e kubeconfig hold? `insecure-skip-tls-verify: true` and a dummy token, so only the refused dial can fail.
- S2 code writer. Does a caller print the error again now that `Printed` is set? No. All four callers only return it.
- S2 code writer. Do the help lines for exits 3 and 4 change? No. They already matched the design. Only line 1 changed.

## Bugs, fix slices and leads

- Bugs fixed: none. Fix slices: none.
- Lead: `tests/integration/apply-staging` fails on `main` without this change. Its pre-clean step waits for an object of a kind that the cluster does not serve, and `kubernetes.WaitAbsent` fails with "kind StagingTest ... is not served by the cluster".
- Lead: `tests/integration/module-apply` waits 15 seconds for a pruned Deployment to be deleted. It failed once in the S1 run and passed on the rerun.
- Lead (found by the S2 test writer, not in the specs): a `--context` that does not exist exits 3, but its error is printed twice.
- Lead: the S2 code commit title has 82 characters, so it breaks the 72-character subject limit. The title came from the commit task in `tasks.md`. The flow never amends, so the commit stays.

## Disputed tests

None.

## Failed checks

- S1, full test command, round 1: `task test` stopped in `test:integration` at `apply-staging`, a failure that is on `main` too. So the 12 programs after it and e2e did not run. The build stopped for an owner decision (below).
- S1, full test command, round 2 (unit tests, 13 integration programs, e2e): `integration/module-apply` timed out waiting for a deletion. The program passed alone on the rerun. No code was changed or restored.
- S2: no check failed.

## Stops

- S1, step (g): `task test` cannot pass, because `apply-staging` fails on `main`. Owner answer A: each slice runs `task test:unit`, every integration program except `apply-staging`, and `task test:e2e`, and this report lists `apply-staging` as a lead. Both slices passed that suite.

## Commands

- Test command: `task test`, with `tests/integration/apply-staging` left out (owner answer A).
- Project rules: `## Commit rules` and `## Release rules` (from `RULES.md`, the guide's `Rules file:`), `## Text rules` (from `AGENTS.md`).
- Check command: `task check:commit`, passed before both code commits.
