## Why

`opm instance delete` and the stale-resource prune of `opm instance apply` and `opm module apply` delete a PersistentVolumeClaim like any other tracked object, and the data on the volume goes with it. Nothing guards it: only Namespaces and CustomResourceDefinitions are protected (`internal/kubernetes/protected.go`). One renamed volume in a module, or one delete of the wrong instance, loses data with no way back. The owner decided to protect claims by default before v1.0.0, because changing the default after GA would be a breaking change.

## What Changes

- **BREAKING** `opm instance delete` of a CLI-owned instance keeps every tracked PersistentVolumeClaim. It lists each kept claim, deletes the other resources and the `ModuleInstance` record, exits 0, and prints the `kubectl` command that deletes each claim.
- **BREAKING** The prune of `opm instance apply` and `opm module apply` keeps a stale PersistentVolumeClaim. The claim stays in the inventory, so a later apply with the flag can remove it. The apply exits 0 and lists each kept claim.
- A new flag `--delete-data` (default false) on the three commands deletes the claims as before. With it, the confirmation prompt of `instance delete` names every claim that will be deleted.
- The message about kept claims is informational (level INFO, status `kept`), not a warning.
- `opm instance delete` reads the instance record before it prompts, so that the prompt can name the claims. An instance that does not exist is now reported before the prompt, not after it.
- `--delete-data` together with `--no-prune` is a usage error.
- On an operator-managed instance the flag has no effect, and the command says so: the operator decides what it removes.
- Help text, a docs page with the migration note, and the README say all of this.

SemVer: MAJOR after GA (a changed default of two commands). Before GA it ships as the next beta.N, with `!` in the PR title.

Complexity added (Principle VII): one flag and one kind test. The flag is the smallest explicit way to ask for data deletion; reusing `--force` would give that flag a second meaning, and `--yes` must stay "skip the prompt" only.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deploy`: instance delete keeps PersistentVolumeClaims unless `--delete-data`; the prompt names claims that will be deleted.
- `apply-pruning`: prune keeps stale PersistentVolumeClaims in the cluster and in the inventory unless `--delete-data`; the dry-run preview lists them.
- `flag-conventions`: `--delete-data` is the one flag that allows deletion of claims; `--force` and `--yes` never do.

## Impact

- `internal/kubernetes`: `protected.go` (claim test), `delete.go` (option, kept list).
- `internal/inventory`: `stale.go` (split of the stale set).
- `internal/workflow/apply`: `apply.go` (prune, preview, record), `delete.go` (option, report lines), `thineditor.go` (note).
- `internal/cmd/instance`: `delete.go`, `apply.go`; `internal/cmd/module`: `apply.go`.
- `internal/output`: status `kept`.
- Docs: `docs/site/diagnostics/kept-volume-claims.md`, `README.md`.
- Not touched: the operator's delete and prune, `opm operator uninstall` (the operator module renders no claim), PersistentVolumes, Secrets and every other kind, claims a StatefulSet creates from `volumeClaimTemplates` (never tracked), `--no-prune` semantics, `go.mod`.
