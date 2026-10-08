## 1. Instance delete keeps claims

- [x] 1.1 Write the failing tests first: `kubernetes.Delete` keeps a tracked claim by default (read or unreadable), deletes it with `DeleteData`, and does not keep a `PersistentVolumeClaim` of another group; `executeInstanceDelete` deletes the record, exits 0, prints the `kubectl` line and no warning; the dry run lists the claim as kept; the prompt names the claims with `--delete-data`; the flag reaches the delete; an operator-managed instance warns
- [x] 1.2 `internal/kubernetes`: `IsDataClaim` in `protected.go`; `DeleteOptions.DeleteData` and `DeleteResult.Kept` in `delete.go`; `internal/output`: status `kept`
- [x] 1.3 `internal/workflow/apply/delete.go`: `DeleteRequest.DeleteData`, kept lines at INFO
- [x] 1.4 `internal/cmd/instance/delete.go`: `--delete-data`, record read before the prompt, prompt text, closing output, help text
- [x] 1.5 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `feat(cmd)!: keep PersistentVolumeClaims on instance delete unless --delete-data`

## 2. Prune keeps stale claims

- [x] 2.1 Write the failing tests first: a stale claim is not deleted, stays in the written inventory, is listed as kept at INFO, and the apply exits 0 with its success line; with `DeleteData` it is deleted and leaves the inventory; a later apply with the flag removes a claim kept earlier; the dry-run preview lists it as kept, or as would prune with the flag; `--delete-data` with `--no-prune` is a usage error on both apply commands; all three commands offer the flag; an operator-managed apply warns
- [x] 2.2 `internal/inventory/stale.go`: `SplitDataClaims`
- [x] 2.3 `internal/workflow/apply`: `Options.DeleteData`, the split, the preview, the record entries, the thin-editor note
- [x] 2.4 `internal/cmd/instance/apply.go` and `internal/cmd/module/apply.go`: `--delete-data`, exclusion with `--no-prune`, help text
- [x] 2.5 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check`, `task test:unit` and `task cascade:wiring:check` green, then commit `feat(cmd)!: keep stale PersistentVolumeClaims on prune unless --delete-data`

## 3. Docs and migration note

- [ ] 3.1 Add `docs/site/diagnostics/kept-volume-claims.md`: what is kept, how to delete, the record handling, StatefulSet claims, operator-managed instances, and the migration note for the changed default
- [ ] 3.2 Update the `README.md` command notes
- [ ] 3.3 `task docs:bundle:check` and `task openspec:check` green, then commit `docs: describe kept PersistentVolumeClaims and the --delete-data flag`
