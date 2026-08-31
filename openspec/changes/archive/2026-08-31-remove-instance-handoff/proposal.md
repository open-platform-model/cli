## Why

`opm instance handoff` flips a CLI-owned instance to operator ownership irreversibly without checking that the operator's effective applier identity can apply the instance's inventory. On a stock `opm operator install` (no `--default-service-account`, and the CLI never writes `spec.serviceAccountName`) the flip succeeds and the instance strands operator-owned and unreconcilable. Rather than patch the gate chain, the command is removed now and the feature will be redesigned later, on top of the ownership model, which stays.

## What Changes

- **BREAKING**: `opm instance handoff` is removed, together with its precondition chain, strict-registry verification render, ownership flip, and post-flip verdict. The CLI no longer offers any path from CLI ownership to operator ownership.
- The ownership model is unchanged: `spec.owner` semantics, the thin-editor apply path and the operator-owned delete path stay exactly as specified. The consequence is stated explicitly in the docs: the CLI creates CLI-owned instances only; operator-owned instances are created outside the CLI (kubectl, GitOps, the operator's own fixtures) and the CLI edits them.
- `status.lastAppliedRenderDigest` and the `module-instance.opmodel.dev/source: local` provenance annotation keep being written on every apply. The digest is kept so a future ownership transfer has a recorded value to verify against; the annotation is still consumed by the thin-editor path, which refuses a local-bytes module.
- The inventory drift description helper, used only by the handoff verdict, is removed.
- The two operator-owned e2e tests (thin-editor values round-trip, operator-owned delete) that today reach operator ownership by running `handoff` obtain it by patching `spec.owner` directly instead. The three handoff e2e tests and their handoff-only helpers are removed; the test file and its fixture directory are renamed so no `handoff` name survives.
- Documentation is updated in this same change: README (command table, the "Graduating an instance to the operator" section, and the "CLI-managed vs operator-managed instances" section reframed around instances created outside the CLI), QUICKSTART, `CLAUDE.md`, the dev-cluster comments in `Taskfile.yml` and `hack/`, and code comments that describe the digest, the provenance annotation, and the SSA-ownership integration proof in terms of handoff.
- The applier-grant e2e precondition stays: the surviving operator-owned tests still need the operator to apply workloads. Its spec wording drops the "hands off" clause.

Not in scope: any replacement path from CLI to operator ownership (no `--owner` flag on `instance apply`); changes to the operator; the `CHANGELOG.md`, RFC 0007 and archived OpenSpec changes, which remain history.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `instance-handoff`: all five requirements removed (the capability is retired).
- `inventory-ownership`: the single-branch-point requirement no longer lists `handoff` among the consumers of the ownership resolver.
- `instance-inventory`: the render provenance annotation requirement names its consumer as the thin-editor path rather than the handoff pre-gate; the spec-write-contents requirement names later coordinate verification as operator acquire and any future ownership transfer rather than handoff verification.
- `e2e-cluster-preconditions`: the applier-grant requirement is renamed ("before operator-owned tests") and scoped to tests that rely on the operator applying workloads; the "hands off" wording is removed.

## Impact

- Commands: `instance handoff` removed. `instance apply`, `instance delete` unchanged in behavior.
- Packages removed: `internal/workflow/handoff`, `internal/cmd/instance/handoff.go`, `internal/inventory/drift.go`.
- Packages touched for comments only: `internal/inventory` (`cr.go`, `digest.go`, `record.go`, `ownership.go`), `internal/workflow/render/types.go`, `pkg/module/module.go`, `tests/integration/ssa-ownership`.
- Tests: `internal/cmd/instance` (two command tests), `internal/inventory` (`drift_test.go`), `tests/e2e` (three tests removed, two re-based, file and fixture renamed). The fixture path is named in three workspace-root files that move with the rename: the `.tasks/deps/fixtures.sh` pin and path comment, the `deps:pins:fixtures` description in `.tasks/deps.yml`, and the root `CLAUDE.md`.
- SemVer: MAJOR (a command is removed, constitution VI). The repo is on the `1.0.0-alpha` prerelease line (`versioning: prerelease`, `prerelease-type: alpha`), so the release stays inside that line; the commit carries a `BREAKING CHANGE` footer so the changelog says so.
- No enhancement is implemented by this change; enhancement 0006 remains the record of what was built.
