## Revision

2026-10-01, after review. Changed-implies-bumped now compares each template against the **previous release tag**, not the merge-base, and runs in **both** modes. The release run fails before any publish when a template changed since the previous release and GHCR already holds its version. Reason: cli `main` takes direct pushes (the `Protected` ruleset is disabled and no status check is required), and the three commits that left GHCR v1.0.2 stale (e6d9f21, 5d372c0, d5b5c01) reached `main` without a PR. A PR-only, merge-base check would have caught none of them. Also changed: the release checkout gets `fetch-depth: 0`; every verification run is isolated behind an `OPM_BIN` shim that refuses real publishes, and a publish-mode success case was added; the Risks text no longer calls the release PR's check a safeguard on its own; `AGENTS.md` lines 128 and 351 change together; the existing e2e scaffold test is cited and extended (section 3). The rewritten D3, R4 and Risks below supersede the first draft.

## Context

`.github/scripts/publish-templates.sh` is the one gate list for the official templates, run by two jobs:

- PR CI, `pr.yml` `template-gates` (lines 98-114): builds `opm` from the PR, runs the script with `--dry-run`. A dry run whose only refusal is "already holds" passes (script lines 69-82).
- Release, `release.yml` `publish-templates` (lines 146-181): checks out the release tag (shallow), builds `opm` from it, runs the script without flags. Per template: skip if GHCR holds the tag (lines 61-65), else `tidy --check`, then `opm module publish` (lines 66-68).

The gates inside `opm module publish` cover identity, coordinates, namespace, package name, override and kernel loader shape (`internal/publish/gates.go`). None of them evaluates `debugValues` or renders. `opm module vet` does both: it checks values against `#config`, then renders the synthesized instance exactly as `opm module build` does against a platform generated from the module's own pins, without a cluster (`internal/cmd/module/vet.go:40-45`, `:84-87`, `:168-195`; spec `mod-vet`, requirement "mod vet command validates module without generating manifests").

`vet` has rendered only since commit 874bce8 (2026-09-29, first in v1.0.0-alpha.23). Before that it stopped after the `#config` check. This is why the gate could not have been "just vet" earlier, and why it is meaningful now: the release job always builds `opm` from the tag it publishes, so the vet it runs is the vet that ships.

One test already renders a template: `tests/e2e/mod_init_test.go:123-182`, `TestE2E_ModInit_ThenVet`, publishes `standard` to an in-process registry, scaffolds from it with `opm mod init` and requires the scaffold to pass `opm module vet` and a publish dry-run. It covers only `standard`, and it vets the re-identified scaffold rather than the template tree. Those are complementary: the script vets what is published, the test vets what a user gets (the `template-modules` requirement that a fresh scaffold passes vet). Section 3 extends the test to all three templates.

How changes reach cli `main`, measured 2026-10-01: `gh api repos/open-platform-model/cli/rules/branches/main` returns only the mention-guard workflow rule; the `Protected` ruleset (PR requirement) is disabled, branch protection is absent, and no status check is required. Direct pushes land, and a red PR check does not block a merge.

No Go code, command, flag or exit code changes. The config rules about command syntax, flags and example output do not apply; the script's messages are specified instead.

## Goals / Non-Goals

**Goals:**

- A template whose `debugValues` does not render under the cli being built cannot publish, and fails the PR that introduces it.
- No template is pushed unless every template passed every gate in the same run.
- A template whose files changed since the previous release, at a version GHCR already holds, fails the PR check and fails the release run before anything is pushed, however the change reached `main`.

**Non-Goals:**

- A render gate inside `opm module publish` for every module (see R2).
- Making the workspace `deps:update:templates` task bump template versions. That is a workspace-repo change; R5 describes it and recommends where it lands.
- Making `Template Publish Gates (dry-run)` a required check, or enabling the `Protected` ruleset. Both are owner settings, outside any change.
- Content comparison between the tree and the GHCR artifact (see R4).
- Republishing or retracting `advanced` v1.0.2. Published versions are immutable; `opm module init` resolves the newest stable version (v1.0.3), so v1.0.2 is only reached by an explicit `advanced@1.0.2` pin.

## Decisions

### D1. Vet every template, in both modes, through the shared script

The gate phase runs `"$opm" module vet "./$dir"` on every `templates/*/` tree, whether or not GHCR already holds its version, in both `--dry-run` and publish mode. A template that is already published and unchanged is still vetted, because the cli under test may no longer render it, and `opm module init` from that cli would scaffold a broken module.

The vet runs through the script, not as a separate workflow step, so the PR job and the release job keep one gate list (the `pr.yml:99-101` comment's promise).

### D2. Two phases: gate all, then publish

```
prev := newest v* tag reachable from HEAD^                  # D3
gate phase (both modes), for each templates/<t>/:
  version   := cue eval ./identity -e Version               # unchanged
  tidy      := opm module tidy --check ./<dir>               # existing gate, moved here
  vet       := opm module vet ./<dir>                        # new
  dryrun    := opm module publish --dry-run ./<dir>          # now also in publish mode
     GO                                  -> pass
     only refusal "already holds"        -> D3
     any other refusal                   -> fail
  record failure, continue to next template (report all, not first)
any failure -> print "gates failed for: <list>; nothing published", exit 1
--dry-run   -> exit 0
publish phase (release only), for each template:
  GHCR holds v<version> -> skip (unchanged filter, 0011:D15)
  else                  -> opm module publish ./<dir>
```

`opm module publish` re-runs its own gates during the push. That duplicate is intentional and cheap: the dry-run in the gate phase is what makes "every gate before any push" true across templates.

### D3. Changed implies bumped, against the previous release tag, both modes

`prev` is `git describe --tags --abbrev=0 --match 'v[0-9]*' HEAD^`, the newest release tag reachable from `HEAD^`:

- PR job: `HEAD` is the pull request's merge commit and `HEAD^` the base branch tip, so `prev` is the last release on `main`, including one whose release commit is the tip itself.
- Release job: `HEAD` is the release tag's commit, so `HEAD^` skips that tag and `prev` is the release before it.

No tag reachable (a shallow checkout or a clone without tags) fails the run with a hint naming `fetch-depth: 0`.

When a template's dry-run refuses only for "already holds" (GHCR holds `v<version>`):

1. `git diff --quiet "$prev" HEAD -- <dir>` is clean: the tree is what was released. Pass.
2. Publish mode only: the declared `Version` at `prev` (read from `git show "$prev:<dir>identity/identity.cue"` and `cue eval`, empty when the template had no identity there) is lower than the current one under `sort -V`. This release raised the version, and an earlier attempt of the same release already pushed it: a re-run after a partial publish. Pass, and the publish phase skips it.
3. Otherwise fail:

```
==> minimal: changed since v1.0.0-beta.4 but v1.0.3 is already published.
    Published versions are immutable: run 'opm module version set <semver> ./templates/minimal/'.
```

Exception 2 is publish-mode only and requires a raise, so a PR that moves a template to an older published version (sideways, 1.0.3 to 1.0.2) fails in both modes (cases I and I-publish below).

Why the tag and not the merge-base: the release tag is the last state GHCR could have been published from, so "changed since `prev`" is exactly "differs from what the last release saw". It catches a change pushed straight to `main` on the next PR's check (every PR goes red until it is fixed) and in the release run before any push. A change and its revert between two releases is not a change (case H). Two PRs that both bump to the same version, with a release cut between their merges, are caught too: the second PR's content differs from the tag that published that version.

Workflow wiring: `pr.yml` `template-gates` and `release.yml` `publish-templates` both check out with `fetch-depth: 0` (actions/checkout then fetches every tag). The script needs no `BASE_REF`.

### D4. Messages and exit codes

The script exits 0 when every gate passes (and, in publish mode, every needed push succeeds) and 1 otherwise. New lines, beside `opm module vet`'s own diagnostics:

- `==> previous release tag: <prev>`
- `==> <t>: 'opm module vet' failed; a template must render its own debugValues`
- `==> <t>: v<version> already published and unchanged since <prev>`
- `==> <t>: v<version> already published by an earlier run of this release`
- the D3 failure message
- `==> no release tag reachable from HEAD^; check out full history with tags (fetch-depth: 0)`
- `==> gates failed for: <t> [<t> ...]; nothing published`

## Research & Decisions

### Reproduction

**Context**: the gate must fail on the content that shipped as advanced v1.0.2 and pass on `main`.

**Explored** (scratch copies only, `opm` built from `origin/main` 437bebd, canonical GHCR mapping, empty HOME):

| Tree | `opm module vet` | `publish --dry-run` |
| --- | --- | --- |
| `main` minimal, standard, advanced (v1.0.3) | exit 0 (1, 2, 8 resources) | already holds only |
| advanced at d5b5c01 (v1.0.2, last pre-fix) | exit 2, unresolved disjunction | not run |
| advanced at v1.0.0-alpha.22 (the release that first published v1.0.2) | exit 2, same | not run |
| GHCR `opmodel.dev/templates/advanced` v1.0.2, unpacked | exit 2, same | not run |
| `main` advanced with the two `updateStrategy: type: "RollingUpdate"` lines removed, `version set 1.99.0` | exit 2, same | `GO — pushing ... v1.99.0` |

The last row is the defect in one line: the current gates say GO for a template that does not render. The GHCR v1.0.2 `components.cue` is byte-identical to the alpha.22 tree.

With `opm` built from v1.0.0-alpha.22 itself, `vet` passes all three alpha.22 templates: that vet did not render yet (874bce8, above).

**Prototype of D2/D3** (revised script, shellcheck clean), run in a scratch clone of this worktree with its real tags (`prev` = v1.0.0-beta.4 on the case branches). Every run: `env -i`, empty `HOME`, `DOCKER_CONFIG` and `XDG_CONFIG_HOME`, asserted to hold no credential file before each run, and `OPM_BIN` pointing at a shim that execs the real `opm` for everything except `module publish` without `--dry-run`, which it logs as `WOULD PUBLISH <dir>` and exits 0. No registry was written; no local registry container was used.

| Case | Mode | Result |
| --- | --- | --- |
| A: unchanged | both | exit 0; all three "already published and unchanged since v1.0.0-beta.4"; publish: no WOULD PUBLISH |
| B: advanced fix removed, bumped to 1.0.4 | both | exit 1, vet fails advanced, `gates failed for: advanced; nothing published`; publish: no WOULD PUBLISH |
| C: minimal pin catalogs/opm v4.4.4 to v4.4.3, no bump | both | exit 1, D3 message names minimal; publish: no WOULD PUBLISH |
| D: C plus `version set 1.0.4` | dry-run | exit 0, minimal `GO` at v1.0.4 |
| D | publish | exit 0, exactly `WOULD PUBLISH ./templates/minimal/`; advanced and standard "already published; skipped" |
| E: comment-only edit to standard, no bump | dry-run | exit 1, D3 names standard |
| F: unrelated file outside `templates/` | dry-run | exit 0 |
| G: C, then an unrelated commit on top (a direct push followed by an innocent PR) | dry-run | exit 1, D3 names minimal |
| H: C, then its revert | dry-run | exit 0 |
| I: C plus `version set 1.0.2` (older, published) | both | exit 1, D3 names minimal |
| J: tag v1.0.0-beta.1 checked out (re-run of the release that raised every template to 1.0.3) | publish | exit 0, each "already published by an earlier run of this release", all skipped |
| K: tag v1.0.0-alpha.27 (prev alpha.26) | publish | exit 1 before any push: vet fails advanced, D3 names minimal and standard |
| L: tag v1.0.0-alpha.25 (prev alpha.24) | publish | exit 1 before any push, same three |

K and L are real releases that shipped re-pinned templates under the already-published v1.0.2. Under this script both release runs fail before publishing anything.

The extended e2e test (section 3) was prototyped in the same clone: all three subtests pass on `main` (about 10 s together, no cluster, in-process registry), and the advanced subtest fails with the unresolved disjunction on case B's template tree.

### R1. vet versus build

**Options considered**:
1. `opm module vet`: renders as build does, prints one validation line per object, no manifests, exit 2 on failure. Its render is a spec requirement (`mod-vet`).
2. `opm module build > /dev/null`: same verdict (vet.go:40-41), but its diagnostics are less author-facing and it emits manifests to discard.
3. Only the e2e scaffold test (`TestE2E_ModInit_ThenVet`), extended to every template.

**Decision**: vet in the script, and also extend the e2e test (section 3). **Rationale**: vet is the command the spec defines as the author's pass/fail verdict. The e2e test alone does not gate the release job, which runs only the script, and it checks the re-identified scaffold rather than the tree that is published; the script alone does not check the scaffold a user receives. Both are cheap.

### R2. Where the render gate lives

**Options considered**:
1. In `publish-templates.sh` (chosen).
2. A new gate inside `opm module publish` (render `debugValues`, refuse on failure) for every module publisher.
3. A separate workflow step in `pr.yml` and `release.yml`.

**Decision**: option 1. **Rationale**: option 2 changes `opm module publish` for every publisher (modules fleets, fixtures that deliberately do not render), a product decision that belongs in an enhancement amending 0011, not in a CI fix. Option 3 splits one gate list across two YAML files, and the release path would drift from the PR path.

### R3. Where changed-implies-bumped lives

**Options considered**:
1. Inline in `publish-templates.sh` (chosen).
2. Run `FIXTURES_DIR=templates hack/fixtures.sh check`.
3. Defer to a follow-up change.

**Decision**: option 1. **Rationale**: option 2 would make `hack/fixtures.sh`, which must stay byte-identical with opm-operator's copy (workspace `task fixtures:lint`), carry template-only gates (tidy, vet) and a different base (release tag, not merge-base); its policy text also assumes the testing domain. Option 3 leaves the exact failure behind GHCR v1.0.2 (stale pins) open while the same file and jobs are already being edited.

### R4. What the bump check compares against

**Options considered**:
1. The merge-base of `BASE_REF` and `HEAD`, PR job only (the first draft).
2. The previous release tag, both modes (chosen).
3. The GHCR artifact's content.
4. Auto-bump unpublished changes at release time.
5. Option 2 plus a run of the script from `ci.yml` on every push to `main`.

**Decision**: option 2. **Rationale**: option 1 only sees changes that arrive through a PR, and cli `main` takes direct pushes; the three commits behind stale GHCR v1.0.2 were direct pushes, so option 1 would have caught none of them. It also misreads a change and its revert as a change. Option 2 compares with the last state GHCR could have been published from, needs only git history, and fails the release run before any push (cases K, L). Option 3 must reproduce CUE's module-zip file selection to compare reliably, which is more code than the defect warrants, and option 2 gives the same verdict for every case measured. Option 4 publishes a version no commit declares, against the identity package as the single source of the version. Option 5 would flag a bad direct push minutes earlier, but option 2 already turns the next PR and the release PR red and blocks the release run; the extra job is not worth a third trigger for the same script.

### R5. The workspace `deps:update:templates` task (recommendation, workspace repo)

Root `Taskfile.yml:118-150` re-pins each `cli/templates/*/cue.mod/module.cue` with `cue mod get` and `cue mod tidy` and never touches the template's version. After this change, every cli PR produced by `task deps:update` that moves a template pin fails `template-gates` until someone runs `opm module version set` by hand.

**Recommendation**: the task bumps the patch version of every template whose `module.cue` it changed, but only when GHCR already holds the template's declared version. If the declared version is still unpublished (bumped earlier in the same release cycle), that pending version already covers the new pins and a second bump would skip a version for nothing. Shape, mirroring `.tasks/deps/fixtures.sh` lines 21-54 (`bump_deps` reports whether anything moved, then `opm module version set` on the next patch):

```
moved := module.cue differs after cue mod get + tidy
if moved and GHCR holds v<Version>:      # same anonymous manifest probe as publish-templates.sh
  opm module version set <Version with patch+1> .
```

It needs `opm` on `PATH`, which `deps:pins:fixtures` already requires. Move the loop into `.tasks/deps/templates.sh` beside `fixtures.sh` and share the patch-increment helper. Update the `deps:update` row in the workspace `AGENTS.md` to say it also bumps changed templates. The cli commit stays `fix(deps)`, since template pins are shipped.

**Where it lands**: as its own commit in the workspace PR that reworks `.tasks/deps/fixtures.sh` (the fixtures plan): same directory, same bump idiom, one shared helper. The latest-tag PR (`.tasks/deps/latest-tag.sh`) is unrelated in substance. If the fixtures PR stalls, ship it alone. Either way it should merge before, or together with, this cli change, so the first `task deps:update` after it does not produce a red cli PR.

### R6. Comment-only edits need a bump

The gate fails a comment-only template edit at a published version (case E). That is intended: `opm module init` copies comments into users' modules, so a comment change is a content change of the artifact, the fixtures rule is the same, and a50cd73 bumped standard 1.0.1 to 1.0.2 for comments alone. Excluding comments would need a CUE-aware diff, which is R4 option 3's cost.

### R7. enhancement.yaml

None. The change fulfils the "vetted" intent of 0011:D25, but 0011 is archived with status `delivered`; the `template-modules` delta cites D25 as its source. No new decision is implemented.

## Risks / Trade-offs

- [Release-time failure] `publish-templates` runs alongside `goreleaser`, not before it, so a gate failure at release leaves the cli binaries published and no template pushed (the two-phase script guarantees none is pushed). The release PR runs the same gates on the same tree first, but that check is advisory: no status check is required on `main`, and release PRs are merged by the owner or an agent who may not read it. The safeguard therefore depends on someone reading the release PR's checks. **Recommendation (owner setting, not part of this change)**: make `Template Publish Gates (dry-run)` a required status check on `main`. Recovery after a failed release run: bump the template, cut the next patch.
- [Direct pushes] A required check binds PRs only. A direct push that changes a template without a bump still lands; this change makes every following PR and the release run fail until a bump lands, rather than letting the stale version ship silently.
- [Vet stops rendering] The gate's strength rests on `opm module vet` rendering. The `mod-vet` spec requires it; a change that drops the render would have to change that spec.
- [Friction] Every template edit needs a bump, comment-only ones included (R6), and `task deps:update` cli PRs need a manual `version set` until R5 lands. A bad direct push turns unrelated PRs red until fixed (case G); that is the point, but it is noisy.
- [Re-run exception] Publish mode accepts a changed, already-published template when its version rose since the previous release, which is what a re-run after a partial publish looks like. On a linear `main` a raised version can only have been pushed by an attempt of the same release, so this admits nothing else; `sort -V` orders the plain `X.Y.Z` template versions correctly but would misorder SemVer prereleases, which templates do not use.
- [Manual PR-workflow run] `pr.yml` also has `workflow_dispatch`; there `HEAD` is a branch tip, not a merge commit, and `prev` is the newest tag below its parent. The check stays meaningful but compares one commit further back.
- [Network] vet resolves core and catalogs from GHCR, as the dry-run already does.

## Migration Plan

None. The current templates (all at v1.0.3) pass every new gate unchanged (case A).
