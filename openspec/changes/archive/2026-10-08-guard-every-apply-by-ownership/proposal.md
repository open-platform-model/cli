## Why

The cli's apply guard runs only on an instance's first apply, passes any OPM-managed object whichever instance it belongs to, and is skipped once a record exists. A later apply therefore takes over an existing object that a new release of the module happens to name, and a first apply takes over another instance's object. `opm operator install` has the same gap by specification: with a record it runs no guard.

The library holds the rule, `ownership.CanApply` in `opm/k8s/ownership` (library `v1.0.0-beta.7`, already pinned). Enhancement 0012 decided that both frontends ask it for every object on every apply, with a per-object adopt annotation as the only override (0012:D4:R2, 0012:D8).

This change is the apply half of the cli's adoption. The delete half is the change `adopt-kubernetes-ownership-package`, merged as cli#347 and archived; this change builds on its code and its spec text.

## What Changes

| Path | Verdict and identity | What the cli does with the answer |
| --- | --- | --- |
| `opm instance apply`, `opm module apply`: every rendered object, on every apply | `ownership.CanApply`, with the render's identity; `InInventory` from the record | Allowed: apply. `terminating`, `foreign-object`, `other-instance`: refuse the whole apply before any change, exit 1. `adopted-elsewhere`: do not apply the object, warn, drop it from the inventory, go on. |
| The first-install existence check and its warning (cli#333) | the same pass | The check becomes that pass. The warning stays, and counts the existing OPM-managed objects the verdict allowed. |
| `opm operator install`: the check-phase guard, on every install | the same pass, with `Admit` for the objects the migration proved | Every refusal exits 2 with nothing changed. Install also refuses on `adopted-elsewhere`: it needs every object it renders. |
| `opm operator install`: the CRD step, the migration's field-ownership moves, the instance apply | covered by the check-phase guard | They run only after the guard passed, so none writes to an object another instance owns or adopts. |

- **BREAKING** The apply guard runs on every apply, not only the first (0012:D8:R1). An apply that used to take over an existing object, because a record existed or because the object carried another instance's OPM labels, now refuses and exits non-zero. A terminating object refuses every apply (0012:D8:R5).
- **BREAKING** `opm operator install` runs its guard with a record too, and refuses with exit 2 when an object it renders is owned or adopted by another instance.
- **BREAKING** The refusal message is the library's. It no longer says "remove or rename it"; it names the adopt annotation and the UUID to set.
- **Adopt rule, annotation only (0012:D8:R8).** An object is adopted only when its live `opmodel.dev/adopt` annotation holds the instance's UUID. The user sets it, for example with `kubectl annotate`. The cli never sets it and offers no flag for it (0012:D8:R3, 0012:D8:R6); it honours it through the verdict and prints the exact annotation in every ownership refusal.
- **Hand-over.** An object whose adopt annotation names another instance is not applied, is dropped from the inventory the apply records, and is not deleted.
- **Exit codes.** `design.md` holds the table of every case whose code or message changes. No code changes for a case that cli#332 to cli#346 settled.
- **Built on, not replaced.** The fail-safe rules of cli#332 and cli#334 (a refusal before the first write changes nothing), the discovery resolver of cli#342, the claim rule of cli#345 and the typed errors of cli#346 stay.
- No new command and no new flag. A dry run still refuses nothing; a follow-up prints "would refuse" lines.

Not in this change: the delete side (change `adopt-kubernetes-ownership-package`); the deletion protocol; flags; publish; the operator repo; the `--rbac` objects of `opm operator install`, which belong to no instance and stay outside the guard (a follow-up issue decides their ownership).

SemVer: MAJOR after GA (an apply that succeeded now refuses). Before GA it ships as the next beta, as `feat!` in the PR title, with the migration note in the PR body. It is meant for the same beta as the delete half. A dry run can preview an apply that the real run refuses; the migration note says so.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `apply-pruning`: the first-install existence check becomes an ownership guard on every apply; an object adopted by another instance is let go; the apply flow names the guard.
- `apply-preflight-gates`: the gate order names the ownership guard.
- `deploy`: the `--create-namespace` requirement names the ownership guard.
- `operator-lifecycle`: `opm operator install` runs the apply guard on every install and refuses on an object another instance owns or adopts.

## Impact

- Code: `internal/inventory/` (the guard, the admit set), `internal/workflow/apply/` (apply flow, record entries, messages), `internal/operator/` (plan, install), a docs page, one integration script.
- Imports: `opm/k8s/ownership`, which the delete half already imports. No `go.mod` change.
- Stability: whether the `opm/k8s` tier is in the library's v1 promise is still open; the cli takes the dependency either way.
- Order: the delta for `apply-pruning` "Apply flow orchestration" contains the text the delete half gives that requirement. `adopt-kubernetes-ownership-package` is archived, so the main spec holds that text.
- Users: a migration note (text in `design.md`) and a docs page on adopting an existing object.
- Enhancement: implements part of 0012 (`enhancement.yaml`). The refusal of `opm operator install` on an object annotated for another instance needs a revision note on 0012:D8:R8, which words that case as a skip.
