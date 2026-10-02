## Why

The cli is the last repo in the release order (workspace RELEASING.md, section "Release order"): it ships library as a Go dependency (`go.mod:13`), the opm-operator `install.yaml` as an embedded file pinned by `PinnedOperatorVersion` (`internal/operator/manifest.go:19`, image tag at `internal/operator/dist/install.yaml:1611`), and core plus the opm catalog in the three templates (`templates/{minimal,standard,advanced}/cue.mod/module.cue`). Today nothing stops a release PR from shipping a local or unpublished pin, nothing asks for evidence when the embedded operator moved, and several repo settings files would fight the release cascade the workspace is about to roll out:

- `.github/labels.yml` is synced with `skip-delete: false` (`.github/workflows/labels.yml:22`) and lists none of the labels release-please (`autorelease: pending`, `autorelease: tagged`) and Dependabot (`dependencies`, `go`, `github_actions`) put on PRs. The next edit to `labels.yml` deletes them from the repo.
- Dependabot proposes `github.com/open-platform-model/library` bumps as `build(deps)` (`.github/dependabot.yml:9-17`), a hidden type that never releases; the cascade owns that pin.
- `docs` is a visible changelog section (`release-please-config.json:23`), so a docs-only commit cuts a cli release, which then cascades. The owner has decided to hide `docs` (owner decision 2026-10-01 (RELEASING.md, Pin classes)).
- `.github/workflows/pr-title.yml:3-10` says the squash commit takes the PR title. The repo's `squash_merge_commit_title` is `COMMIT_OR_PR_TITLE` today, so a one-commit PR squashes under its commit subject instead.

This change is the cli's Phase 1 preparation (workspace RELEASING.md, section "Rollout and changes"). It plans the gates and settings files only; the cascade task, receiver and notify jobs come later.

## What Changes

- **G1 release-pin gate** (workspace RELEASING.md, section "Gates"). A new script `.github/scripts/release-pin-check.sh`, run locally as `task deps:release-check`, fails on: a `replace` directive in `go.mod`; an OPM Go pin (`github.com/open-platform-model/*` in `go.mod`) that is a pseudo-version or whose tag does not exist upstream; a `-0.dev.` CUE pin in a template `cue.mod/module.cue`; a tracked `cue.mod/local-module.cue` anywhere; and `PinnedOperatorVersion` differing from the operator image tag in `internal/operator/dist/install.yaml` (or that image line missing or duplicated). These are the failure kinds of the shared G1 rule; the cli adds none of its own. It runs as a step inside the existing `lint` job of both `pr.yml` and `ci.yml`, only when `${{ github.head_ref || github.ref_name }}` starts with `release-please--`.
- **G4 operator-embed evidence** (interim). A new workflow `.github/workflows/release-evidence.yml` with one always-running job (check name `G4 operator-embed evidence`), whose logic lives in a new script `.github/scripts/release-evidence.sh`. On a release-please PR whose `PinnedOperatorVersion` differs from the one at the last cli tag (the version in the base branch's `.release-please-manifest.json`), it fails unless the PR carries the label `e2e-verified`. Its messages say it is interim until `add-embedded-operator-e2e-job` lands.
- **Labels.** `.github/labels.yml` gains `deps-cascade`, `deps-cascade:conflict`, `deps-cascade:hold`, `deps-cascade:breaking`, `need-human-review`, `e2e-verified`, and the bot-managed labels `autorelease: pending`, `autorelease: tagged`, `dependencies`, `go`, `github_actions` with their live colors, so the sync deletes none of them.
- **Dependabot.** The `gomod` entry ignores `github.com/open-platform-model/*`, and its prefix comment stops claiming release-please drops `deps`.
- **Docs hidden.** Following owner decision 2026-10-01 (RELEASING.md, Pin classes), `release-please-config.json` hides the `docs` section; `refactor` stays visible. `AGENTS.md:348` is updated to match, and says that edits under `templates/` are never typed `docs` (a template edit must bump and publish the template).
- **pr-title.yml comment** states that the squash title is the PR title only once the owner sets `squash_merge_commit_title` to `PR_TITLE` (workspace RELEASING.md, section "Owner settings").
- **Archive.** The change is archived on this branch (`openspec archive`), so the archive rides the implementing PR; nothing is pushed to `main` (owner decision 2026-10-01 (RELEASING.md, Owner settings)).

Not changed: the embedded operator pin (moved to `v1.0.0-beta.4` by cli PR 269 and released in `v1.0.0-beta.5` while this change was planned), the cascade receiver and notify jobs (`join-release-cascade`), the cascade task (`add-deps-cascade-task`), rulesets and repo settings (owner actions), and any tag or release.

SemVer class: none. Every commit is `ci` or `chore`, and the PR title is `ci(release): prepare the cli for the release cascade`. No Go code changes; `opm` behaves the same.

## Depends on / gates

- **workspace `docs/release-cascade`** (RELEASING.md, AGENTS.md, commit-skill amendment): must be on workspace `main` before this PR merges, because the artifacts and the new comments cite RELEASING.md sections. The workspace commit skill (`.claude/skills/commit/SKILL.md:28-29`) still lists `docs` as releasing in cli; that amendment must ride the workspace branch (it is not on it at `895a439`), and gate G-workspace checks it.
- **S0 owner settings** (RELEASING.md, section "Owner settings"): not a merge precondition. Until the cli ruleset on `main` requires the `Lint` checks and the check `G4 operator-embed evidence`, G1 and G4 are advisory. RELEASING.md "Owner settings" › "Rulesets on main" must name `G4 operator-embed evidence` as a cli required check until `add-embedded-operator-e2e-job` retires it (flagged to the workspace item; gate G-workspace checks it). The pr-title comment becomes fully true only after `PR_TITLE` is set.
- **`add-embedded-operator-e2e-job`** (cli, Phase 1): independent of this change. G4 retires only when all of these hold: `add-embedded-operator-e2e-job` has merged, its e2e check has passed on at least one cli release PR, and the owner has made that check required (owner-approved 2026-10-02; RELEASING.md, Gates). Retiring G4 is its own later cli change, working name `retire-g4-operator-embed-evidence`. It removes a whole workflow, not a step; design.md D6 lists everything it removes.
- **`bump-stale-testdata-pins`** (cli, Phase 1): independent; neither touches the other's files.
- **Operator catch-up (cli PR 269, `fix(deps): embed opm-operator v1.0.0-beta.4`)**: already on `main` and released in `v1.0.0-beta.5`; this branch was cut before it (base `f3569b24`) and touches neither `internal/operator/manifest.go` nor `install.yaml`, so the squash merge carries no conflict. G4 on the next release PR compares against `v1.0.0-beta.5`, which embeds `v1.0.0-beta.4`. Any later operator move fails G4 until a human runs `task test:e2e` and adds `e2e-verified` to the release PR; that is the intended behavior.
- **Label values** (workspace item): RELEASING.md "The cascade" › Labels must carry each cascade label's colour and description (design.md D4 proposes them), and the cli copies them, so every repo uses the same values. The `add-release-cascade-workflows` (X1) receiver must not create labels in the cli, where `.github/labels.yml` owns them.
- **opmodel.dev `build-docs-from-branch-head`** (opmodel.dev branch `feat/build-docs-from-branch-head`): Depends on: opmodel.dev change `build-docs-from-branch-head` merged before section 4's commit (the docs-hiding section) merges. Until then opmodel.dev builds cli docs from the newest cli tag, and library and opm-operator docs from what that tag pins, so a docs-only fix in this repo reaches opmodel.dev only with the next release (owner decision 2026-10-02 (RELEASING.md, Rollout and changes); design.md D5).
- **Later phases** depend on this change: `add-deps-cascade-task` (B4) uses the labels; `join-release-cascade` (C4) requires G1 and G4 in place; `require-pin-freshness-gate` (R4) adds G2 to the same ruleset.

## Capabilities

### New Capabilities

- `release-gates`: G1 (release PRs refuse local, unpublished or mismatched pins) and G4 (a moved embedded operator on a release PR needs the `e2e-verified` label).
- `repo-automation`: the label sync keeps bot-managed labels and declares the cascade labels; Dependabot never proposes OPM Go modules.

### Modified Capabilities

- `release-workflow`: a new requirement fixing which commit types cut a release, with `docs` now hidden.

## Impact

- Commands: none.
- Files: `.github/scripts/release-pin-check.sh` (new), `.github/scripts/release-evidence.sh` (new), `.github/workflows/release-evidence.yml` (new), `.github/workflows/{pr,ci,pr-title}.yml`, `.github/labels.yml`, `.github/dependabot.yml`, `release-please-config.json`, `Taskfile.yml`, `AGENTS.md`; the archive adds `openspec/specs/release-gates/spec.md` and `openspec/specs/repo-automation/spec.md` and extends `openspec/specs/release-workflow/spec.md`.
- Consumers: a docs-only cli commit no longer releases. Until opmodel.dev's `build-docs-from-branch-head` merges, opmodel.dev builds cli docs from the newest cli tag and library and opm-operator docs from what that tag pins (`opmodel.dev/site/versions.conf` header; core and catalog_opm docs come from their release branch or `main`), so a docs-only fix in this repo reaches opmodel.dev only with the next release (design.md D5). That change gates section 4.
- Templates: a `docs`-typed edit under `templates/` would still bump the template (AGENTS.md Template rule) but publish nothing until the next releasing commit, so template edits are never typed `docs` (design.md D5).
- Risk: G4's label is not tied to a commit; a label added before the embedded operator moves again still passes (design.md D3).
- Risk: if section 4 merges before opmodel.dev's `build-docs-from-branch-head`, a docs-only fix in this repo reaches opmodel.dev only with the next release.
