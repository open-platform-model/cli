## Why

Since the ownership guard runs on every apply (cli#349), a real apply stops when a rendered object belongs to someone else. A dry run does not run the guard, so `--dry-run` can report success for an apply the real run refuses, and CI that gates on a dry run passes a change that cannot be applied. The prune preview has the same gap: it lists as `would prune` an object the real prune leaves in the cluster. A real apply whose every rendered object another instance adopted exits 0 with an empty inventory and no line that says what happened.

## What Changes

- A dry run of `opm instance apply` and `opm module apply` runs the same read-only ownership guard as the real run. It prints one `would refuse` line per object the real run refuses, with the reason and the owner, then stops where the real run stops and exits 1. A rendered object it cannot read fails the dry run with the code of the real run (4, 3 or 1).
- A dry run prints one `would skip` line per object the real run lets go because another instance adopted it, and does not send that object to the server.
- The prune preview of a dry run reads each stale object and asks the delete verdict of the real prune. An object the verdict leaves in place is listed as `would keep` (not managed by OPM, or of another instance) or `would let go` (another instance is adopting it) and no longer as `would prune`. A stale object that is gone is not listed. A stale object that cannot be read is listed as `cannot check`, and the dry run exits as the real apply does after a failed prune.
- A real apply that lets go of every rendered object prints one closing line that says that nothing was applied and why. Exit 0 stays. The dry run prints the matching line.
- The docs page "Adopt an existing object" says what a dry run shows, in place of what it does not show.
- `opm operator install` has no `--dry-run` flag, so nothing changes there.

A dry run that exited 0 before can now exit non-zero. That happens only where the real apply exits non-zero with the same inputs. No flag, command or real-apply exit code changes.

SemVer: PATCH after GA (a dry run that reported a wrong result is corrected). Before GA it ships in the next `1.0.0-beta.N`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `apply-pruning`: the dry run runs the ownership guard and previews its refusals and let-go objects; the prune preview asks the delete verdict; an apply that lets go of everything says so. Two requirements are removed and added again under new names, because each loses a scenario that no longer holds.
- `apply-preflight-gates`: the dry-run exemption covers the cluster gates and the status-RBAC gate, not the ownership guard.
- `deploy`: one reference to a renamed `apply-pruning` requirement.
- `operator-lifecycle`: one reference to a renamed `apply-pruning` requirement.

## Impact

- `internal/workflow/apply/apply.go`: the dry-run path of `Execute`, the prune preview, the closing lines.
- `internal/inventory/stale.go`: a read-only run of the prune's deletion plan.
- `internal/output/styles.go`: the new status words.
- Tests in `internal/workflow/apply` and `internal/inventory`; three tests that pinned "a dry run refuses nothing" are replaced.
- `docs/site/diagnostics/adopt-an-existing-object.md`.
- No change to `opm instance delete`, `opm operator install`, `opm operator uninstall`, the operator-managed dry-run preview, or any flag.
