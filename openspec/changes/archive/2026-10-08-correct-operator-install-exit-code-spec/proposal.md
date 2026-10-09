## Why

The spec says that `opm operator install` exits 2 when it cannot read an object it would apply. The code exits 4 for the common case, an object that is unreadable from the start, and the owner decided that the code is right. The spec must say what the command does.

## What Changes

- `apply-pruning`, "Pre-apply existence check on first install": exit code 2 is limited to a read that the check itself fails as install's apply guard. The scenario for an object unreadable from the start states exit code 4, and a second scenario states the guard case.
- `operator-lifecycle`, "Every check that can refuse install runs before its first write": states which of install's three reads of an object refuses, and with which exit code (the code of the read error for the wait for terminating objects and the migration proof, 2 for the apply guard).
- No code, test, flag or exit code changes. The tests that pin this behaviour are already on `main` (`internal/operator/unreadable_test.go`, `TestInstallErrorMapping`).

SemVer class: none (specification only; no release).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `apply-pruning`: the exit code of `opm operator install` for an unreadable object in the existence-check requirement.
- `operator-lifecycle`: the exit codes of install's reads before its first write.

## Impact

- `openspec/specs/apply-pruning/spec.md` and `openspec/specs/operator-lifecycle/spec.md` only.
- No user docs page states this exit code, so none changes.
