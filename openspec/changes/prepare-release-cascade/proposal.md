## Why

The cli is the last repo in the release order (workspace RELEASING.md, section "Release order"): it ships library as a Go dependency (`go.mod:13`), the opm-operator `install.yaml` as an embedded file pinned by `PinnedOperatorVersion` (`internal/operator/manifest.go:19`, image tag at `internal/operator/dist/install.yaml:1611`), and core plus the opm catalog in the three templates (`templates/{minimal,standard,advanced}/cue.mod/module.cue`). Today nothing stops a release PR from shipping a local or unpublished pin, nothing asks for evidence when the embedded operator moved, and several repo settings files would fight the release cascade the workspace is about to roll out:

- `.github/labels.yml` is synced with `skip-delete: false` (`.github/workflows/labels.yml:22`) and lists none of the labels release-please (`autorelease: pending`, `autorelease: tagged`) and Dependabot (`dependencies`, `go`, `github_actions`) put on PRs. The next edit to `labels.yml` deletes them from the repo.
- Dependabot proposes `github.com/open-platform-model/library` bumps as `build(deps)` (`.github/dependabot.yml:9-17`), a hidden type that never releases; the cascade owns that pin.
- `docs` is a visible changelog section (`release-please-config.json:26`), so a docs-only commit cuts a cli release, which then cascades. Owner decision D15 hides it.
- `.github/workflows/pr-title.yml:3-10` says the squash commit takes the PR title. The repo's `squash_merge_commit_title` is `COMMIT_OR_PR_TITLE` today, so a one-commit PR squashes under its commit subject instead.

This change is the cli's Phase 1 preparation (workspace RELEASING.md, section "Rollout and changes"). It plans the gates and settings files only; the cascade task, receiver and notify jobs come later.

## What Changes

- **G1 release-pin gate** (workspace RELEASING.md, section "Gates"). A new script `.github/scripts/release-pin-check.sh`, run locally as `task deps:release-check`, fails on: a `replace` directive in `go.mod`; a tracked `go.work`; an OPM Go pin (`github.com/open-platform-model/*` in `go.mod`) that is a pseudo-version or whose tag does not exist upstream; a `-0.dev.` CUE pin in a template `cue.mod/module.cue`; a tracked `cue.mod/local-module.cue` anywhere; and `PinnedOperatorVersion` differing from the operator image tag in `internal/operator/dist/install.yaml`. It runs as a step inside the existing `lint` job of both `pr.yml` and `ci.yml`, only when `${{ github.head_ref || github.ref_name }}` starts with `release-please--`.
- **G4 operator-embed evidence** (interim). A new workflow `.github/workflows/release-evidence.yml` with one always-running job. On a release-please PR whose `PinnedOperatorVersion` differs from the one at the last cli tag (the version in the base branch's `.release-please-manifest.json`), it fails unless the PR carries the label `e2e-verified`. Its messages say it is interim until `add-embedded-operator-e2e-job` lands.
- **Labels.** `.github/labels.yml` gains `deps-cascade`, `deps-cascade:conflict`, `deps-cascade:hold`, `deps-cascade:breaking`, `need-human-review`, `e2e-verified`, and the bot-managed labels `autorelease: pending`, `autorelease: tagged`, `dependencies`, `go`, `github_actions` with their live colors, so the sync deletes none of them.
- **Dependabot.** The `gomod` entry ignores `github.com/open-platform-model/*`, and its prefix comment stops claiming release-please drops `deps`.
- **D15.** `release-please-config.json` hides the `docs` section; `refactor` stays visible. `AGENTS.md:348` is updated to match.
- **pr-title.yml comment** states that the squash title is the PR title only once the owner sets `squash_merge_commit_title` to `PR_TITLE` (workspace RELEASING.md, section "Owner settings").

Not changed: `operator:sync` to beta.3 (a separate `fix(deps)` catch-up), the cascade receiver and notify jobs (`join-release-cascade`), the cascade task (`add-deps-cascade-task`), rulesets and repo settings (owner actions), and any tag or release.

SemVer class: none. Every commit is `ci` or `chore`, and the PR title is `ci(release): prepare the cli for the release cascade`. No Go code changes; `opm` behaves the same.

## Depends on / gates

- **workspace `docs/release-cascade`** (RELEASING.md, AGENTS.md, commit-skill amendment): must be on workspace `main` before this PR merges, because the artifacts and the new comments cite RELEASING.md sections. The workspace commit skill (`.claude/skills/commit/SKILL.md:28-29`) still lists `docs` as releasing in cli; that amendment rides the workspace branch.
- **S0 owner settings** (RELEASING.md, section "Owner settings"): not a merge precondition. Until the cli ruleset requires the `Lint` checks and the G4 job, G1 and G4 are advisory. The pr-title comment becomes fully true only after `PR_TITLE` is set.
- **`add-embedded-operator-e2e-job`** (cli, Phase 1): independent of this change; when it lands, a follow-up retires G4.
- **`bump-stale-testdata-pins`** (cli, Phase 1): independent; neither touches the other's files.
- **Out-of-scope `operator:sync` beta.3 catch-up**: independent. If it merges after this change, the next release PR fails G4 until a human runs `task test:e2e` and adds `e2e-verified`; that is the intended behavior.
- **Later phases** depend on this change: `add-deps-cascade-task` (B4) uses the labels; `join-release-cascade` (C4) requires G1 and G4 in place; `require-pin-freshness-gate` (R4) adds G2 to the same ruleset.

## Capabilities

### New Capabilities

- `release-gates`: G1 (release PRs refuse local, unpublished or mismatched pins) and G4 (a moved embedded operator on a release PR needs the `e2e-verified` label).
- `repo-automation`: the label sync keeps bot-managed labels and declares the cascade labels; Dependabot never proposes OPM Go modules.

### Modified Capabilities

- `release-workflow`: a new requirement fixing which commit types cut a release, with `docs` now hidden.

## Impact

- Commands: none.
- Files: `.github/scripts/release-pin-check.sh` (new), `.github/workflows/release-evidence.yml` (new), `.github/workflows/{pr,ci,pr-title}.yml`, `.github/labels.yml`, `.github/dependabot.yml`, `release-please-config.json`, `Taskfile.yml`, `AGENTS.md`.
- Consumers: a docs-only cli commit no longer releases. opmodel.dev builds cli docs from the newest cli tag and library, core and opm-operator docs from what that tag pins (`opmodel.dev/site/versions.conf` header), so a docs-only fix in those repos reaches the published site only with the next releasing cli commit (design.md D5).
- Risk: G4's label is not tied to a commit; a label added before the embedded operator moves again still passes (design.md D3).
