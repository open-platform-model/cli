## Why

Five cli test trees still pin core and the opm catalog from the alpha line (core `v2.0.0-alpha.6` to `alpha.11`, opm `v4.0.1` to `v4.4.1`), while everything else the repo maintains is on core `v2.0.0-beta.1` and opm `v4.4.4` (`examples/cue.mod/module.cue`, `hack/platform/cue.mod/module.cue:12-16`). Nothing records whether those old pins are deliberate, so the release cascade (workspace RELEASING.md, section "Pin classes") cannot tell a stale pin it should move from a frozen one it must never touch. An owner decision 2026-10-01 (RELEASING.md, "Pin classes") settles it: list the pins that are old on purpose in a repo-root `.cascade-frozen` with a reason, and bump the rest once as `test(fixtures)`.

The trees went stale because nothing moves them, not because a test needs the old version. `git log` shows when each was authored: `duplicate-identities` was created on 2026-09-19 (cli #226) pinning core alpha.6, a month-old copy, when alpha.10 was current (tagged 2026-09-18). The two newest trees were both created on 2026-09-29: `initvalues` (cli #235) pinned alpha.11, which was tagged that same day, so it was current when created; `skip-unprovided` (cli #238) pinned alpha.10. Every tree has since fallen behind as core moved to alpha.13 and then beta.1. None of them carries a comment, test assertion or commit note that depends on the old version.

## What Changes

- Add a repo-root `.cascade-frozen` (format: workspace RELEASING.md, section "Cascade files") listing the core and catalog pins that are old on purpose, each with the reason taken from the test's own comment or assertion:
  - `tests/e2e/instance_build_test.go`, frozen pins `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4`: the `older-core` platform at core `v2.0.0-alpha.11` (`:247-283`, it exists to be refused as too old), `collisionCorePin = "v2.0.0-alpha.13"` (`:285-287`, a floor that `seedCollidingPlatform` re-pins up to and never down from, `:316-335`), and `olderCatalogPin = "v4.0.0"` (`:74-77`, written into a GHCR-resolved platform by `seedSkewPlatform`, `:123-135`, and asserted at `:213` and `:244`; it must stay older than the catalog `examples` requires so the platform shows catalog version skew).
  - `internal/cmd/platform/check_test.go`, frozen pin `opmodel.dev/core@v2`: the too-old platforms at core alpha.6, alpha.9 and alpha.11 (`:727`, `:761`, `:785`), and `collisionCoreVersion = "v2.0.0-alpha.13"` (`:351-357`). This file holds the same two kinds of deliberate core pin as the e2e file, and the pin inventory behind workspace RELEASING.md, section "Pin classes", missed it.
  - `internal/instinit/render_test.go`, frozen pin `opmodel.dev/core@v2`: `TestRender_Golden` feeds core `v2.0.0-alpha.10` into `Render` (`:25`) and asserts the exact `module.cue` text that carries it (`:45`). The literal is expected output in a golden, never resolved from a registry, so moving it would change nothing the test proves.
- Bump the five stale trees to core `v2.0.0-beta.1`, plus opm `v4.4.4` where the tree pins the catalog. The bump uses `cue mod get` then `cue mod tidy`, never a hand edit:
  - `tests/fixtures/valid/simple-module`, `tests/fixtures/valid/module-with-debug-values` (core alpha.6, no catalog)
  - `internal/instinit/testdata/initvalues` (core alpha.11, opm 4.4.1)
  - `internal/workflow/render/testdata/skip-unprovided` (core alpha.10, opm 4.4.0)
  - `tests/e2e/testdata/duplicate-identities` (core alpha.6, opm 4.0.1)
  - `tests/integration/module-apply/testdata` (core alpha.6, opm 4.0.1)

  None of the trees pins `opmodel.dev/catalogs/k8s@v1`, so k8s is not bumped.
- A tree whose test breaks after the bump is fixed in the same section. If the breakage shows the old pin was deliberate after all, the tree is reverted and added to `.cascade-frozen` with the reason.
- Add a test-fixture-lineage requirement: an old core or catalog pin in a registry-resolved test input is either current or listed in `.cascade-frozen`.
- Correct the existing test-fixture-lineage requirement "Maintained fixtures track the current schema line", which still names `opmodel.dev/catalogs/opm@v2` as the current catalog line; it is `@v4`.
- Archive the change on this branch in its own last section, so the archive and the synced main spec ride the implementing PR; nothing is pushed to `main` (owner decision 2026-10-01 (RELEASING.md, "Owner settings")).

Out of scope: `examples/`, `tests/e2e/testdata/operator-owned` and the podinfo fixture, which are already current and belong to the cascade's own class. Re-pinning those touches the GHCR default-registry trap described in design.md (Context, "trap T3"). Also out: Go string literals that are never resolved from a registry, such as parser and fake-lister inputs in `internal/modref`, `internal/platform`, and `internal/cmd/platform/pull_test.go`.

Release class: none. Every commit is `test(fixtures)` or `chore(openspec)`, so release-please cuts no release from this change. No command, flag or package behaviour changes, and after GA it would still release nothing.

Depends on / gates:
- Depends on: nothing. The change is independent of the other Phase 1 changes. `.cascade-frozen` follows the format fixed in workspace RELEASING.md, section "Cascade files" (branch `docs/release-cascade`), and needs only that format to be stable, not the workspace PR merged.
- Gates cli `add-deps-cascade-task`. That change reads `.cascade-frozen`, and its first run must find these five trees current; otherwise the first cascade PR would also carry this one-off catch-up. It SHALL also include these six files in its test-class set, so the trees are moved on every upstream release and do not go stale again:
  - `tests/fixtures/valid/simple-module/cue.mod/module.cue`
  - `tests/fixtures/valid/module-with-debug-values/cue.mod/module.cue`
  - `internal/instinit/testdata/initvalues/cue.mod/module.cue`
  - `internal/workflow/render/testdata/skip-unprovided/cue.mod/module.cue`
  - `tests/e2e/testdata/duplicate-identities/cue.mod/module.cue`
  - `tests/integration/module-apply/testdata/cue.mod/module.cue`
- Requires of workspace RELEASING.md (branch `docs/release-cascade`): sections "Pin classes" and "What each repo's task moves" list those six files in the cli test class, the "Pin classes" cli frozen row names `olderCatalogPin`, `internal/cmd/platform/check_test.go` and the `internal/instinit/render_test.go` golden beside the two `instance_build_test.go` core pins, and section "Rollout and changes" lists this change as a dependency of cli `add-deps-cascade-task`.
- cli `join-release-cascade` comes after `add-deps-cascade-task`.
- Not affected: cli `prepare-release-cascade`, cli `add-embedded-operator-e2e-job`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `test-fixture-lineage`: adds a requirement that a registry-resolved test input's core or catalog pin is either the current published release or listed with a reason in `.cascade-frozen`, and corrects the catalog line named in "Maintained fixtures track the current schema line" from `@v2` to `@v4`.

## Impact

- Files: `.cascade-frozen` (new), six `cue.mod/module.cue` files under the five trees, and the archived change plus the synced `openspec/specs/test-fixture-lineage/spec.md`. No Go code changes unless a bumped tree's test breaks.
- Tests exercised: `internal/cmd/module` (`vet_test.go:47`, `eval_test.go:212`), `internal/workflow/render` (`module_test.go:258,288,316`, `skip_test.go:30-41`), `internal/instinit` (`values_test.go:20`), the e2e tests `tests/e2e/duplicate_identities_test.go:25`, `tests/e2e/skip_unprovided_test.go:27` (`TestE2E_ModBuild_SkipUnprovided`, reads the skip-unprovided tree) and `tests/e2e/vet_output_test.go:155` (`TestE2E_ModuleVet_OpenDebugValuesRefusedAtSynthesis`, copies `tests/fixtures/valid/simple-module`), and the cluster programs `tests/integration/module-apply` (`main.go:59`) and `tests/integration/skip-unprovided` (`main.go:55`). Both cluster programs run only through local `task test:integration` (`Taskfile.yml:88-115`); CI's integration job does not run them (`.github/workflows/pr.yml:202-209`).
- No user-visible impact. Nothing shipped pins these trees.
