## Context

`.github/workflows/release.yml` runs on every push to `main`. release-please (action v5.0.0, release-please 17.6.0, acting as the opm-release-please App through `actions/create-github-app-token`, with no GITHUB_TOKEN fallback) maintains the release PR; when that PR merges it creates the tag and a published GitHub Release, and two jobs follow: `goreleaser` (archives, `checksums.txt`, `LICENSE` uploaded into the existing release, `release.prerelease: auto`) and `publish-templates` (official template modules to GHCR, unpublished versions only). A `workflow_dispatch` run with a `tag` input re-runs both jobs for a release that exists without its assets (the v1.0.0-alpha.21 incident).

The workspace rule (owner, 2026-10-01, revised the same day): release tags are immutable; a broken release is fixed by releasing the next version. Enforcement:

- org ruleset `tags-immutable`: update, deletion, non-fast-forward on every tag, empty bypass list;
- org ruleset `tags-create-app-only`: tag creation only by the opm-release-please App. A stale or hand-made tag therefore cannot exist, and the per-repo tag-commit assertion an earlier draft of this change carried is dropped;
- GitHub immutable releases, for cli only after this change has shipped one real release (0021:D10:R8). The setting was briefly ON for cli on 2026-10-01, ahead of that plan; the owner turned it off and gate G-platform (b) records it OFF.

Immutable releases forbid adding, replacing or deleting assets after publish, so the current publish-then-upload order cannot run under them. The policy is enhancement 0021 D10 (on enhancements PR 74); this change is the cli's share of it, 0021:D10:R1 and 0021:D10:R8, and opm-operator carries its own. It is Phase 1 of the owner's plan: the cli releases only from `main`. Release branches are Phase 2 (see "Phase 2" below) and nothing here prepares for them.

No Go code, command, flag or exit code changes. The config rules about command syntax, flags, exit codes and example output do not apply; the CI-side error messages are specified instead.

## Goals / Non-Goals

**Goals:**

- A release is created as a draft, filled with every asset, and published last, by goreleaser.
- The manual recovery run works on a draft and refuses a published release with a roll-forward instruction.
- Nothing in the workflow moves, deletes or re-creates a tag.

**Non-Goals:**

- Turning on immutable releases for cli, editing rulesets, or touching any existing tag or release (owner actions; gate G-owner).
- Release-branch support of any kind (trigger on `release/**`, `target-branch`, Latest handling for backports, a cut caller, PR checks on `release/**`): Phase 2.
- The agent hook and the drift ledger: both live outside this repo.
- Changing what is built, the archive names, the template publish pipeline, or the release line.

## Decisions

### D1. release-please creates the tag and a draft release

`release-please-config.json` top level gains `"draft": true` and `"force-tag-creation": true`.

- `draft: true` makes `createRelease` pass `draft: true`. A draft GitHub Release has no git tag until it is published, which breaks two things: goreleaser checks out the tag, and release-please's next run looks for the previous release's tag (release-please issue 1650).
- `force-tag-creation: true` makes release-please call `git.createRef refs/tags/<tag>` at the release commit before `createRelease` (release-please 17.2.0+, `src/github.ts` at v17.3.0). The tag exists from the first second, exactly as today, created by the release App, the one identity `tags-create-app-only` allows.

The release-please step keeps the App token with no fallback: under `tags-create-app-only` a GITHUB_TOKEN run could not create the tag at all.

### D2. goreleaser attaches to the draft and publishes it last

`.goreleaser.yml` `release:` becomes:

```yaml
release:
  prerelease: auto
  use_existing_draft: true          # attach to release-please's draft (goreleaser v2.5+)
  mode: keep-existing               # keep release-please's notes (already the default; stated)
  replace_existing_artifacts: true  # a re-run replaces a half-uploaded asset on the draft
  extra_files:
    - glob: LICENSE
```

`draft` stays unset (false). Traced in goreleaser `internal/client/github.go` (v2.18.2):

- Without `use_existing_draft`, `findRelease` calls `GetReleaseByTag`, which GitHub answers only for published releases, so goreleaser would create a second release on the same tag. With it, `findDraftRelease` lists releases and matches a draft by **name**. goreleaser's `name_template` defaults to `{{.Tag}}`. release-please names the release `v<version>` (`src/strategies/base.ts` at v17.6.0, lines 154 and 708-712: `include-v-in-release-name` defaults to true and the cli sets `include-component-in-tag: false`), which equals the tag. The config MUST NOT set a `name_template` that departs from the tag.
- `createOrUpdateRelease` keeps the found release a draft while uploading, merges notes per `mode`, and fails fast if the found release is immutable.
- `PublishRelease` sets `draft: false` at the end. It forces `make_latest` to `false` for a prerelease and otherwise sends nothing, so GitHub's default (`true`) applies. With releases only from `main` that default is right; `make_latest` stays unset until Phase 2 adds backport releases.
- `replace_existing_artifacts` only matters on a re-run against the same draft; a published release never reaches the upload because of D3.
- The config MUST NOT set `replace_existing_draft`: combined with `draft` it deletes the existing draft release, which the never-mutate requirement forbids.

The goreleaser-action `version` input is pinned to an exact release, `v2.18.2` or the version G-sandbox recorded, instead of the action's default `~> v2`. The flow depends on version-specific client code (the draft match by name, the fallback to `CreateRelease` when no draft matches, the `make_latest` handling in `PublishRelease`), and under immutable releases a regression that publishes a second release for a tag is permanent. The pin is at least v2.5 (`use_existing_draft`) and v2.18 (preflight abort on an immutable published release), and is bumped deliberately, by a commit that names the sandbox or release run proving the new version.

### D3. The goreleaser job checks the draft state first

The `goreleaser` job's first step, before any checkout or build, on every run (push and dispatch):

**Draft check.** `gh release view "$TAG" --json isDraft --jq .isDraft` with `GH_TOKEN: ${{ github.token }}` and `GH_REPO: ${{ github.repository }}` (the step runs before the checkout, so `gh` has no git remote to infer the repository from). The job already holds `contents: write`, and it needs it here: GitHub lists draft releases only to callers with push access, so with `contents: read` `gh release view` answers "release not found" for every draft and every release would stall. The step distinguishes three outcomes and never reads a failed lookup as "published":

- **Draft** (`true`): proceed.
- **Published** (`false`): fail with

  ```text
  release <tag> is already published; its assets can no longer change and its tag never moves.
  Cut the next patch release instead.
  ```

- **No release for the tag** (`gh` exits non-zero with "release not found"): fail with

  ```text
  no GitHub Release exists for <tag>; re-run the release-please job of the run that merged
  the release PR (the release PR is still labelled autorelease: pending).
  ```

  This state is new with `force-tag-creation`: release-please calls `createRef` before `createRelease`, so a `createRelease` failure leaves the tag without a release. Re-running the release-please job finishes it: the merged release PR is still `autorelease: pending`, the 422 on `createRef` for the existing tag is swallowed, and the draft is created (traced in release-please `src/github.ts` at v17.3.0; not exercised in the sandbox).
- **Any other `gh` failure** (network, auth, rate limit): fail with `gh`'s own error and the tag, not with either message above.

The result is captured into a variable under `set -euo pipefail` and matched with an explicit `case` (`true`, `false`, anything else fails), so an empty or unexpected answer cannot fall through to a build.

On a push run the release is always a draft at this point, so the step bites on a dispatch, or on a config regression that stops creating drafts. It is a step failure, so a dispatch on a published release is red, not silently green.
A dispatch runs on `main` (`gh workflow run release.yml -f tag=<tag>`); goreleaser checks out the tag, so the build does not depend on the dispatch ref.

`publish-templates` keeps today's condition and checks out `needs.release-please.outputs.tag_name || inputs.tag`. It does not check the draft state: it touches neither the tag nor the GitHub Release, and `publish-templates.sh` only publishes versions GHCR does not hold, so a recovery run for a published release that lost its templates is still valid. This departs from the canon's wording ("recovery refuses a tag whose release is already published"), which this change reads as covering the GitHub Release's assets; it is listed as an owner decision in tasks.md.

**Concurrency.** The `goreleaser` job carries `concurrency: { group: goreleaser-<tag>, cancel-in-progress: false }`. Without it a dispatch racing a push run, or two dispatches, run two goreleaser jobs on one draft: with `replace_existing_artifacts` each deletes the other's uploads, and the one finishing second no longer finds the draft by name and calls `CreateRelease` for the tag. The group is per tag, not workflow-wide, because GitHub keeps only one pending run per group and a workflow-wide group could drop a release-please run on main.

Recovery runbook, written as the dispatch input description and the workflow header comment:

- Draft with missing or partial assets: re-run the failed jobs, or dispatch with the tag.
- Published release that is wrong or incomplete: cut the next patch; never re-tag, never delete.
- Tag with no GitHub Release: re-run the release-please job of the run that merged the release PR; a dispatch cannot fix it.
- A second, empty draft for the same tag (a release-please re-run, U6): the owner removes it in the browser; the workflow never deletes a release.
- A draft whose tag does not exist (the `goreleaser` checkout fails on the tag): release-please ran with a token that may not create tags and swallowed the refusal. Restore the App token, set the release PR back to `autorelease: pending` and re-run the release-please job (the U6 procedure); the owner removes the tagless draft in the browser.

### D4. Consumers and the draft window

Between release-please and goreleaser's publish the git tag exists while the release is a draft. `releases/download/<tag>/...` answers 404 for a draft, so nothing can install a half-built release; today the same window shows a published release with missing assets. The workspace `deps:pins:opm-cli` default resolves the newest git tag, so it is run after a release finished, as today. Changing that helper is workspace tooling, outside this repo.

## Research & Decisions

### Draft-first versus leaving cli out of immutable releases

**Context**: immutable releases block the current upload-after-publish order.
**Explored**: enhancement 0021 `research/immutable-tags.md` (platform facts, verified 2026-10-01), release-please `src/github.ts` and `src/manifest.ts` at v17.3.0, goreleaser `internal/client/github.go` at v2.18.2, GitHub immutable releases docs and GA changelog.
**Options considered**:
1. Draft-first with `force-tag-creation` (this change): GitHub's recommended flow; supported config keys in both tools; recovery still works on drafts.
2. Keep cli out of immutable releases: no work, but a published cli release's assets stay replaceable.
3. goreleaser creates the release itself (release-please `skip-github-release`): loses release-please's notes and its tag-anchoring for the next release PR, and goreleaser runs with GITHUB_TOKEN, which `tags-create-app-only` forbids to create a tag.
**Decision**: option 1.
**Rationale**: the only option that runs under immutable releases without giving up release-please's notes or the recovery path.

### No tag-commit assertion

An earlier draft added a `verify-release` job comparing the tag's peeled commit with release-please's `sha`. It guarded against `createRef` swallowing a 422 for a pre-existing tag at another commit. Under `tags-create-app-only` no identity but the release App can create a tag, and in Phase 1 the App creates release tags only from `main`'s single version line, so a release tag cannot pre-exist. This holds only while one branch releases: Phase 2 re-opens it (two branches proposing the same version, see "Phase 2"). The job, its script and its permissions problem (it needed `contents: write` to see drafts) are gone; the draft check moved into the `goreleaser` job, which already holds that permission.

### Assumptions (gate G-sandbox; observed results under "Sandbox findings")

A first sandbox pass (2026-10-01, `open-platform-model/release-flow-sandbox`, release-please-action v5.0.0, goreleaser v2.18.2) already observed parts of U1, U2, U3 and U5, but with `GITHUB_TOKEN` (the App was not available to the sandbox), with goreleaser in the operator shape (`draft: true` plus a separate publish job), and with no tag ruleset or immutable releases: run 36842600439 (U1: `releases_created`, `tag_name` and `sha` set for a draft, tag present at the release commit; U3: goreleaser attached to the one draft and kept its notes), release PR 4 of that sandbox (U2: the next release PR anchored on the forced tag while the previous release was a draft and after it was published), run 36843234784 (recovery into a draft failed with `422 already_exists` without `replace_existing_artifacts`, which is why D2 sets it), run 36844356743 (recovery with `replace_existing_artifacts` succeeded), run 36843387529 (U5: a dispatch on a published release was refused). G-sandbox still requires every assumption to be re-run on the cli's exact shape below.

- U1. release-please-action sets `releases_created`, `tag_name` and `sha` for a draft release.
- U2. The next release PR anchors on the force-created tag while the previous release is still a draft, and after it is published (no duplicate changelog, correct next version).
- U3. goreleaser with `use_existing_draft` finds release-please's draft by name, keeps its notes, uploads, and publishes it with the prerelease flag set and Latest unchanged.
- U4. The flow passes under `tags-immutable` and `tags-create-app-only`, and with immutable releases on (asset upload to a draft allowed, publish locks it).
- U5. A dispatch on a draft finishes it; a dispatch on a published release fails at the D3 draft check.
- U6. A release-please re-run against a merged release PR still labelled `autorelease: pending`, whose draft exists, either recognises the existing release or creates a second draft (recorded either way; a second draft triggers the D3 runbook line).

If the sandbox contradicts any of these, section 1 stops the change and this design is revised before section 2.

### Sandbox findings

Second pass, 2026-10-01, `open-platform-model/release-flow-sandbox` PR 6 (merged as 09fd181): the cli shape exactly as task 3.2 and D2, release-please acting as the opm-release-please App with no fallback, under `tags-immutable` and `tags-create-app-only` with immutable releases ON. Versions: release-please-action v5.0.0 (release-please 17.6.0), goreleaser v2.18.2 through goreleaser-action v7.2.3, create-github-app-token v3.2.0. Every assumption held; nothing in D1 to D3 changes.

- U1 holds. Run 36862828370: `releases_created=true`, `tag_name=v1.0.0-beta.3`, `sha=841701a`, `draft=true`; right after the job the tag `refs/tags/v1.0.0-beta.3` pointed at `sha` while the release was a draft. Same on runs 36863184964 and 36863653282.
- U2 holds. Release PR 12 was opened while v1.0.0-beta.4 was a draft and PR 14 while v1.0.0-beta.5 was a draft; each proposed the next beta with only the new fix and a compare link from the previous tag, and PR 14 was unchanged after beta.5 was published.
- U3 holds. Run 36862828370: the draft check printed `release v1.0.0-beta.3 is a draft; building`; goreleaser attached to release id 400952000 (no second release), uploaded the archive, `checksums.txt` and `LICENSE`, kept release-please's notes and published it with `isPrerelease: true`; `releases/latest` answered 404 before and after, so Latest did not move.
- U4 holds. The whole flow ran green under both tag rulesets with immutable releases ON; uploads and replacements on the draft succeeded and every published release reports `isImmutable: true`. Tag creation as GITHUB_TOKEN was refused (`422 Reference update failed`, run 36862651882); every release tag was created by the App.
- U5 holds. v1.0.0-beta.5 was held as a draft by an injected failure (run 36863653282); dispatch 36863859196 finished it and replaced a stale `checksums.txt` (`replace_existing_artifacts`). Dispatches 36862966138 and 36863863732 on published releases failed at the draft check with the D3 published message, later steps skipped. Dispatch 36863002685 on a tag with no release failed with the D3 no-release message. Two back-to-back dispatches ran one after the other in the per-tag concurrency group.
- U6: a second draft IS created. Run 36863184964 attempt 2 (release PR relabelled `autorelease: pending`, release-please job re-run): release-please swallowed the 422 on `createRef` and created a second, empty draft with the same name. The D3 runbook line stays.

Consequences adopted in this change:

- GitHub's `::error::` annotation holds one line, so the draft check emits one `::error::` line and prints the full two-line D3 message as plain lines below it (the sandbox shape).
- release-please 17.6.0 swallows every 422 from `createRef`, not only "already exists"; a ruleset refusal is also a 422. A non-App token would therefore produce a draft with no tag, and the `goreleaser` job would fail at the tag checkout with an unclear error. The release-please step keeps the App token with no GITHUB_TOKEN fallback as a hard rule, and the runbook gains the case.
- goreleaser's `previous_tag` takes any tag (run 36862828370 picked the non-release tag `probe-orphan-1`). It is harmless with `mode: keep-existing`, which discards goreleaser's changelog; the cli keeps release-please's notes and does not switch to goreleaser's changelog.

Considered and not adopted: failing the draft check when the tag has more than one release (as the opm-operator guard does). After U6 both drafts are empty when goreleaser runs, so whichever it matches by name ships a correct release and the leftover is the runbook's owner removal; the count would add an unproven step to the sandbox-proven job.

## Risks / Trade-offs

- [release-please or goreleaser behavior differs from the traced source] → G-sandbox proves U1 to U6 on `open-platform-model/release-flow-sandbox`, running this change's exact config, before the cli changes.
- [A draft is left behind when goreleaser fails] → it is mutable and invisible; the failed run's jobs are re-run, or the manual recovery finishes it. The next release PR does not wait on it (U2).
- [The first real draft-first release fails in a way the sandbox missed] → immutable releases are off for cli (gate G-platform), so the release stays fixable in place; the owner enables the setting only after a clean release.
- [Immutable releases are turned back on for cli before task 5.1 is green] → the first draft-first release would lock at publish with no fix in place. G-platform (b) recorded the setting OFF on 2026-10-01 (it had been on briefly, ahead of the plan); re-read it before merging this PR.
- [A future `name_template` breaks the draft match] → D2 forbids it; goreleaser would then create a second release for the tag, which task 5.1's "exactly one release" check and U3 catch.
- [Phase 2 inherits this flow] → the draft-first pieces here are branch-agnostic except the main-only trigger and the unset `make_latest`; Phase 2 changes exactly those (see below).

## Phase 2 (before GA, separate change)

Release-branch support was designed in an earlier revision of this change and removed on the owner's Phase 1/Phase 2 split. The Phase 2 change must cover, at least:

- **Trigger and target.** `on.push.branches: [main, 'release/**']`; release-please `target-branch: ${{ github.ref_name }}`; the branch-local config (`versioning: always-bump-patch`, `prerelease: false`) lands by the cut PR before any backport.
- **Latest.** `release.make_latest: '{{ envOrDefault "MAKE_LATEST" "false" }}'` with `MAKE_LATEST=true` only for a run on `main`, so a backport `v1.0.1` published after `v1.1.0` does not take Latest.
- **Dispatch branch check.** `git merge-base --is-ancestor "$TAG" "$GITHUB_SHA"` in the `goreleaser` job, so a dispatch runs on the branch the tag came from and gets the right `MAKE_LATEST`.
- **Cut caller.** A thin `cut-release-branch.yml` (`workflow_dispatch`, input `minor`) calling the shared `open-platform-model/.github` reusable workflow pinned by SHA, top-level `permissions: {}`, job `contents: write` and `pull-requests: write`; input names aligned with the merged reusable workflow.
- **Version-line collision.** With two branches releasing, main and `release/vX.Y` can both propose the same patch; the App creates both tags, so `tags-create-app-only` does not prevent it. release-please swallows the 422 on `createRef`, fails with `DuplicateReleaseError` and relabels the PR `autorelease: tagged`; the losing branch's manifest then anchors on a commit outside its history (unbounded changelogs, a burned version). The owner's version-line rule answers it: `release/vX.Y` is cut only when main's next release is `X.(Y+1).0` or higher, and after the cut main never releases an `X.Y.*` version. Phase 2 must enforce it (the cut refuses otherwise; optionally a check in the release-please job on main) and prove the collision case in the sandbox.
- **PR checks on backports.** `pr.yml` triggers only on `pull_request` into `main`, so template-gates, fixtures and integration never run for a PR into `release/**`. Phase 2 sets `pull_request.branches: [main, 'release/**']`; whether the release-branches ruleset requires those checks is an owner decision.
- **Cut from the newest stable tag.** At GA `v1.0.*` holds 31 or more prereleases, and `git tag --sort=-v:refname`, `sort -V` and a lexical sort all pick `v1.0.0-beta.3` over `v1.0.0`; a mis-cut branch cannot be deleted. The reusable workflow must select the highest non-prerelease `vX.Y.Z` and refuse when none exists; the spec scenario must include prerelease tags of that minor.
- **Sandbox.** Prove release-please on a release branch with `target-branch` and `always-bump-patch` (the old U7), a cut from a tag made before the change, and the main-versus-branch collision case.
- **Templates.** A template version bumped on a release branch must not collide with a different tree at the same version on main; templates are bumped on main and only backported as a fix, and the GHCR never-overwrite gate refuses a collision loudly.
