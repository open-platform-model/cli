## 1. Remove the command and its package

- [x] 1.1 Delete `internal/workflow/handoff/` (handoff.go, verify.go and their tests) and `internal/cmd/instance/handoff.go`; drop the `AddCommand` line in `internal/cmd/instance/instance.go` and `TestNewInstanceHandoffCmd*` in `instance_test.go`.
- [x] 1.2 Delete `internal/inventory/drift.go` and `drift_test.go`; `go build ./... && go vet ./...` clean.

## 2. Reword shared code that describes itself in handoff terms

- [x] 2.1 `internal/inventory/digest.go`, `record.go`, `cr.go`, `ownership.go`, `patch_test.go`, `internal/workflow/render/types.go`, `pkg/module/module.go`: rewrite the comments per design (digest kept for a future transfer, annotation consumed by the thin editor, resolver consumed by apply and delete).
- [x] 2.2 `tests/integration/ssa-ownership/main.go`: reword the three handoff comments to describe the SSA properties in ownership-model terms; keep the checks.

## 3. e2e suite

- [x] 3.1 Rename `tests/e2e/instance_handoff_test.go` and `tests/e2e/testdata/handoff/` to their operator-owned names; rename the `handoff*` helpers and constants; update the fixture comment and, in the workspace root, the `.tasks/deps/fixtures.sh` pin and path comment, the `deps:pins:fixtures` description in `.tasks/deps.yml`, and the `tests/e2e/testdata/handoff` mention in the root `CLAUDE.md`.
- [x] 3.2 Delete `TestE2E_Handoff_Adoption`, `TestE2E_Handoff_DigestGate`, `TestE2E_Handoff_PreconditionRefusals` and the helpers only they used (`podUIDs`, `inventoryEntryIDs`, any other the linter reports).
- [x] 3.3 Add the operator-owned setup helper (apply CLI-owned, `kubectl patch` `spec.owner: operator`, wait for the operator's reconcile of the new generation) and use it in `TestE2E_ThinEditor_ValuesRoundTrip` and `TestE2E_Delete_OperatorOwnedDelegates` in place of the handoff invocation.
- [x] 3.4 Reword the applier-precondition comments and failure text that say "handoff" (`requireReconcilingOperator`, `requireOperatorApplierGrant`) to "operator-owned"; the behavior is unchanged.

## 4. Documentation and dev-cluster comments

- [x] 4.1 `README.md`: remove the `instance handoff` row and the "Graduating an instance to the operator" section; reframe "CLI-managed vs operator-managed instances" per design (CLI creates CLI-managed only; operator-managed instances are created outside the CLI; delete/prune material stays); remove the handoff line from the command examples.
- [x] 4.2 `QUICKSTART.md`: remove "Graduating to the Operator".
- [x] 4.3 `CLAUDE.md` dev-cluster note, `Taskfile.yml` `cluster:operator` summary, `hack/kind-operator-rbac.yaml` header, `hack/kind-platform.yaml` comment: replace "handoff" wording with "operator-owned e2e tests".
- [x] 4.4 `grep -ri handoff` over the repo excluding `CHANGELOG.md`, `docs/rfc/`, `docs/comparisons/` (generic English use of the word, not the command), `openspec/changes/archive/`, `openspec/specs/instance-handoff/` (removed at archive) returns nothing.

## 5. Validation

- [x] 5.1 `task lint` and `task test:unit` pass.
- [x] 5.2 Against `kind-opm-dev` prepared by `task cluster:operator`, `task test:e2e` passes, including the two re-based operator-owned tests.
