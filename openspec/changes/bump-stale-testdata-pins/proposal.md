## Why

Five cli test trees still pin core and the opm catalog from the alpha line (core `v2.0.0-alpha.6` to `alpha.11`, opm `v4.0.1` to `v4.4.1`), while everything else the repo maintains is on core `v2.0.0-beta.1` and opm `v4.4.4` (`examples/cue.mod/module.cue`, `hack/platform/cue.mod/module.cue:15-16`). Nothing records whether those old pins are deliberate, so the release cascade (workspace RELEASING.md, section "Pin classes") cannot tell a stale pin it should move from a frozen one it must never touch. Owner decision D7 settles it: list the pins that are old on purpose in a repo-root `.cascade-frozen` with a reason, and bump the rest once as `test(fixtures)`.

The trees went stale by copy, not by intent. `git log` shows each was authored or last touched at an already-old pin: `duplicate-identities` was created on 2026-09-19 (cli #226) pinning core alpha.6 when alpha.13 was current, and the two newest trees (`initvalues`, cli #235; `skip-unprovided`, cli #238, both 2026-09-29) pin alpha.11 and alpha.10. None of them carries a comment, test assertion or commit note that depends on the old version.

## What Changes

- Add a repo-root `.cascade-frozen` (format: workspace RELEASING.md, section "Cascade files") listing the core pins that are old on purpose, each with the reason taken from the test's own comment:
  - `tests/e2e/instance_build_test.go`: the `older-core` platform at core `v2.0.0-alpha.11` (`:247-283`, it exists to be refused as too old) and `collisionCorePin = "v2.0.0-alpha.13"` (`:285-287`, a floor that `seedCollidingPlatform` re-pins up to and never down from, `:316-335`).
  - `internal/cmd/platform/check_test.go`: the too-old platforms at core alpha.6, alpha.9 and alpha.11 (`:727`, `:761`, `:785`), and `collisionCoreVersion = "v2.0.0-alpha.13"` (`:351-357`). This file holds the same two kinds of deliberate pin as the e2e file, and it was missing from the plan's inventory.
- Bump the five stale trees to core `v2.0.0-beta.1`, plus opm `v4.4.4` where the tree pins the catalog. The bump uses `cue mod get` then `cue mod tidy`, never a hand edit:
  - `tests/fixtures/valid/simple-module`, `tests/fixtures/valid/module-with-debug-values` (core alpha.6, no catalog)
  - `internal/instinit/testdata/initvalues` (core alpha.11, opm 4.4.1)
  - `internal/workflow/render/testdata/skip-unprovided` (core alpha.10, opm 4.4.0)
  - `tests/e2e/testdata/duplicate-identities` (core alpha.6, opm 4.0.1)
  - `tests/integration/module-apply/testdata` (core alpha.6, opm 4.0.1)

  None of the trees pins `opmodel.dev/catalogs/k8s@v1`, so k8s is not bumped.
- A tree whose test breaks after the bump is fixed in the same section. If the breakage shows the old pin was deliberate after all, the tree is reverted and added to `.cascade-frozen` with the reason.
- Add a test-fixture-lineage requirement: an old pin in a test input is either current or listed in `.cascade-frozen`.

Out of scope: `examples/`, `tests/e2e/testdata/operator-owned` and the podinfo fixture, which are already current and belong to the cascade's own class (trap T3 below). Also out: Go string literals that are never resolved from a registry, such as parser and fake-lister inputs in `internal/modref`, `internal/platform`, `internal/cmd/platform/pull_test.go`, and the golden string in `internal/instinit/render_test.go:25,45`.

Release class: none. Every commit is `test(fixtures)`, so release-please cuts no release from this change. No command, flag or package behaviour changes, and after GA it would still release nothing.

Depends on / gates:
- Depends on: nothing. The change is independent of the other Phase 1 changes. `.cascade-frozen` follows the format fixed in workspace RELEASING.md, section "Cascade files" (branch `docs/release-cascade`), and needs only that format to be stable, not the workspace PR merged.
- Gates: cli `add-deps-cascade-task` (B4) reads `.cascade-frozen`, and its first run must find these five trees current. Otherwise the first cascade PR would also carry this one-off catch-up. cli `join-release-cascade` (C*) comes later still.
- Not affected: cli `prepare-release-cascade`, cli `add-embedded-operator-e2e-job`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `test-fixture-lineage`: adds a requirement that a test input's core or catalog pin is either the current published release or listed with a reason in `.cascade-frozen`.

## Impact

- Files: `.cascade-frozen` (new), and six `cue.mod/module.cue` files under the five trees. No Go code changes unless a bumped tree's test breaks.
- Tests exercised: `internal/cmd/module` (`vet_test.go:47`, `eval_test.go:212`), `internal/workflow/render` (`module_test.go:258,288,316`, `skip_test.go:30-41`), `internal/instinit` (`values_test.go:20`), the e2e test `tests/e2e/duplicate_identities_test.go:25`, and the cluster programs `tests/integration/module-apply` (`main.go:59`) and `tests/integration/skip-unprovided` (`main.go:55`). Both cluster programs run only through local `task test:integration` (`Taskfile.yml:88-115`); CI's integration job does not run them (`.github/workflows/pr.yml:202-209`).
- No user-visible impact. Nothing shipped pins these trees.
