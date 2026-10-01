## Context

`.github/workflows/release.yml` runs on every push to `main`. release-please (action v5.0.0, release-please 17.6.0, acting as the opm-release-please App through `actions/create-github-app-token`, with no GITHUB_TOKEN fallback) maintains the release PR; when that PR merges it creates the tag and a published GitHub Release, and two jobs follow: `goreleaser` (archives, `checksums.txt`, `LICENSE` uploaded into the existing release, `release.prerelease: auto`) and `publish-templates` (official template modules to GHCR, unpublished versions only). A `workflow_dispatch` run with a `tag` input re-runs both jobs for a release that exists without its assets (the v1.0.0-alpha.21 incident).

The workspace rule (owner, 2026-10-01, revised the same day): release tags are immutable; a broken release is fixed by releasing the next version. Enforcement:

- org ruleset `tags-immutable`: update, deletion, non-fast-forward on every tag, empty bypass list;
- org ruleset `tags-create-app-only`: tag creation only by the opm-release-please App. A stale or hand-made tag therefore cannot exist, and the per-repo tag-commit assertion an earlier draft of this change carried is dropped;
- org ruleset `release-branches`: `refs/heads/release/*` cannot be deleted or force-pushed and takes changes only by PR (squash);
- GitHub immutable releases, for cli only after this change has shipped one real release.

Immutable releases forbid adding, replacing or deleting assets after publish, so the current publish-then-upload order cannot run under them. This change is the cli half of the canon's draft-first item (opm-operator carries its own) plus the cli's share of the release-branch model.

No Go code, command, flag or exit code changes. The config rules about command syntax, flags, exit codes and example output do not apply; the CI-side error messages are specified instead.

## Goals / Non-Goals

**Goals:**

- A release is created as a draft, filled with every asset, and published last, by goreleaser.
- The manual recovery run works on a draft and refuses a published release with a roll-forward instruction.
- The release workflow releases from a `release/vX.Y` branch as well as from `main`, and a backport release never becomes Latest.
- One thin caller cuts a `release/vX.Y` branch through the shared reusable workflow.
- Nothing in the workflow moves, deletes or re-creates a tag.

**Non-Goals:**

- Turning on immutable releases for cli, editing rulesets, or touching any existing tag or release (owner actions; gate G-owner).
- Cutting a release branch now. During beta the cli fixes forward on main; the first `release/v1.0` is cut at GA or when main starts work a released minor must not get.
- The reusable `cut-release-branch` workflow itself, the agent hook and the drift ledger: all live outside this repo.
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
  make_latest: '{{ envOrDefault "MAKE_LATEST" "false" }}'  # "true" on main, "false" on release/** (D3, D4)
  extra_files:
    - glob: LICENSE
```

`draft` stays unset (false). Traced in goreleaser `internal/client/github.go` (v2.18.2):

- Without `use_existing_draft`, `findRelease` calls `GetReleaseByTag`, which GitHub answers only for published releases, so goreleaser would create a second release on the same tag. With it, `findDraftRelease` lists releases and matches a draft by **name**. goreleaser's `name_template` defaults to `{{.Tag}}`. release-please names the release `v<version>` (`src/strategies/base.ts` at v17.6.0, lines 154 and 708-712: `include-v-in-release-name` defaults to true and the cli sets `include-component-in-tag: false`), which equals the tag. The config MUST NOT set a `name_template` that departs from the tag.
- `createOrUpdateRelease` keeps the found release a draft while uploading, merges notes per `mode`, and fails fast if the found release is immutable.
- `PublishRelease` sets `draft: false` at the end. It templates `make_latest`, forces it to `false` for a prerelease, and sends nothing when the template renders empty, in which case GitHub's default (`true`) applies. That default is why `make_latest` is set explicitly: a backport `v1.0.1` published after `v1.1.0` would otherwise take Latest.
- `replace_existing_artifacts` only matters on a re-run against the same draft; a published release never reaches the upload because of D3.
- The config MUST NOT set `replace_existing_draft`: combined with `draft` it deletes the existing draft release, which the never-mutate requirement forbids.

The goreleaser-action keeps `version` at its default (`~> v2`), which is above v2.5 (`use_existing_draft`) and v2.18 (preflight abort on an immutable published release).

### D3. The goreleaser job checks the draft state and the branch first

The `goreleaser` job's first steps, before any build, on every run (push and dispatch):

1. **Draft check.** `gh release view "$TAG" --json isDraft --jq .isDraft` with `GH_TOKEN: ${{ github.token }}`. The job already holds `contents: write`, and it needs it here: GitHub lists draft releases only to callers with push access, so with `contents: read` `gh release view` answers "release not found" for every draft and every release would stall. Not `true`: fail with

   ```text
   release <tag> is already published; its assets can no longer change and its tag never moves.
   Cut the next patch release instead.
   ```

   On a push run the release is always a draft at this point, so the step bites on a dispatch, or on a config regression that stops creating drafts. It is a step failure, so a dispatch on a published release is red, not silently green.
2. **Branch check.** `git merge-base --is-ancestor "$TAG" "$GITHUB_SHA"` on the full-history checkout. Fails when a dispatch runs on a branch that does not contain the tag (for example a `release/v1.0` tag dispatched from `main`), which would otherwise publish it with the wrong `MAKE_LATEST`. On a push run it always holds.

The job sets `MAKE_LATEST` to `true` when `github.ref_name` is `main` and `false` otherwise. A dispatch therefore runs on the branch the release came from (`gh workflow run release.yml --ref release/v1.0 -f tag=v1.0.1`).

`publish-templates` keeps today's condition and checks out `needs.release-please.outputs.tag_name || inputs.tag`. It does not check the draft state: it touches neither the tag nor the GitHub Release, and `publish-templates.sh` only publishes versions GHCR does not hold, so a recovery run for a published release that lost its templates is still valid. This departs from the canon's wording ("recovery refuses a tag whose release is already published"), which this change reads as covering the GitHub Release's assets; it is listed as an owner decision in tasks.md.

**Concurrency.** The `goreleaser` job carries `concurrency: { group: goreleaser-<tag>, cancel-in-progress: false }`. Without it a dispatch racing a push run, or two dispatches, run two goreleaser jobs on one draft: with `replace_existing_artifacts` each deletes the other's uploads, and the one finishing second no longer finds the draft by name and calls `CreateRelease` for the tag. The group is per tag, not workflow-wide, because GitHub keeps only one pending run per group and a workflow-wide group could drop a release-please run on main.

Recovery runbook, written as the dispatch input description and the workflow header comment:

- Draft with missing or partial assets: re-run the failed jobs, or dispatch with the tag on the branch the tag was released from.
- Published release that is wrong or incomplete: cut the next patch; never re-tag, never delete.
- A second, empty draft for the same tag (a release-please re-run, U6): the owner removes it in the browser; the workflow never deletes a release.

### D4. Release branches: trigger, target branch and Latest

- `on.push.branches` becomes `[main, 'release/**']`.
- The release-please step passes `target-branch: ${{ github.ref_name }}`, so on `release/v1.0` it reads that branch's `release-please-config.json` and manifest, opens its release PR into that branch, and tags from it. The PR on a release branch is opened by the release App, so its CI starts on its own.
- A release branch's config is changed by the PR the cut action opens (D5): `versioning: always-bump-patch` and `prerelease: false` for package `.`, so a stray `feat` backport never claims a minor main needs. That PR merges before any backport PR.
- `MAKE_LATEST` (D3) keeps a backport from becoming Latest. A prerelease is never Latest regardless.
- `publish-templates` runs for a backport release as it does on main. A template version bumped on a release branch must not collide with a different tree at the same version on main; templates are bumped on main and only backported as a fix, so this holds by practice, and the GHCR never-overwrite gate refuses a collision loudly.

### D5. cut-release-branch caller

New `.github/workflows/cut-release-branch.yml`:

```yaml
name: Cut release branch
on:
  workflow_dispatch:
    inputs:
      minor:
        description: Released minor to branch, X.Y (for example 1.0); cuts release/vX.Y from the newest vX.Y.* tag
        required: true
        type: string
permissions: {}
jobs:
  cut:
    uses: open-platform-model/.github/.github/workflows/cut-release-branch.yml@<merged commit SHA>
    with:
      tag_prefix: v
      minor: ${{ inputs.minor }}
      package: .
    secrets: inherit
    permissions:
      contents: write
      pull-requests: write
```

The reusable workflow creates `release/v<minor>` from the highest `v<minor>.*` tag and opens a PR into it that sets `always-bump-patch` and `prerelease: false` for the package and makes sure the release workflow's trigger covers the branch (it already does, by D4). The caller adds nothing else: one implementation for all five repos. Input names and required secrets are taken from the reusable workflow as merged; the names above are the agreed interface (`tag_prefix`, `minor`, package path) and task 3.5 re-checks them against the merged file before committing.

### D6. Consumers and the draft window

Between release-please and goreleaser's publish the git tag exists while the release is a draft. `releases/download/<tag>/...` answers 404 for a draft, so nothing can install a half-built release; today the same window shows a published release with missing assets. The workspace `deps:pins:opm-cli` default resolves the newest git tag, so it is run after a release finished, as today. Changing that helper is workspace tooling, outside this repo.

## Research & Decisions

### Draft-first versus leaving cli out of immutable releases

**Context**: immutable releases block the current upload-after-publish order.
**Explored**: critic review (`tagres/critic.md` C4, C5, C6), release-please `src/github.ts` and `src/manifest.ts` at v17.3.0, goreleaser `internal/client/github.go` at v2.18.2, GitHub immutable releases docs and GA changelog.
**Options considered**:
1. Draft-first with `force-tag-creation` (this change): GitHub's recommended flow; supported config keys in both tools; recovery still works on drafts.
2. Keep cli out of immutable releases: no work, but a published cli release's assets stay replaceable.
3. goreleaser creates the release itself (release-please `skip-github-release`): loses release-please's notes and its tag-anchoring for the next release PR, and goreleaser runs with GITHUB_TOKEN, which `tags-create-app-only` forbids to create a tag.
**Decision**: option 1.
**Rationale**: the only option that runs under immutable releases without giving up release-please's notes or the recovery path.

### No tag-commit assertion

An earlier draft added a `verify-release` job comparing the tag's peeled commit with release-please's `sha`. It guarded against `createRef` swallowing a 422 for a pre-existing tag at another commit. Under `tags-create-app-only` no identity but the release App can create a tag, and the App creates only the release tag, so that tag cannot pre-exist. The job, its script and its permissions problem (it needed `contents: write` to see drafts) are gone; the draft check moved into the `goreleaser` job, which already holds that permission.

### Unverified assumptions (gate G-sandbox, recorded in section 1)

- U1. release-please-action sets `releases_created`, `tag_name` and `sha` for a draft release.
- U2. The next release PR anchors on the force-created tag while the previous release is still a draft, and after it is published (no duplicate changelog, correct next version).
- U3. goreleaser with `use_existing_draft` finds release-please's draft by name, keeps its notes, uploads, and publishes it with the prerelease flag set and Latest unchanged; with `MAKE_LATEST=false` a non-prerelease is published without becoming Latest.
- U4. The flow passes under `tags-immutable` and `tags-create-app-only`, and with immutable releases on (asset upload to a draft allowed, publish locks it).
- U5. A dispatch on a draft finishes it; a dispatch on a published release fails at the D3 draft check; a dispatch on the wrong branch fails at the D3 branch check.
- U6. A release-please re-run against a merged release PR still labelled `autorelease: pending`, whose draft exists, either recognises the existing release or creates a second draft (recorded either way; a second draft triggers the D3 runbook line).
- U7. On a `release/vX.Y` branch with `always-bump-patch` and `prerelease: false`, release-please with `target-branch` opens the release PR into that branch, proposes `vX.Y.(Z+1)` for a `feat` or `fix`, and tags from that branch.

If the sandbox contradicts any of these, section 1 stops the change and this design is revised before section 2.

## Risks / Trade-offs

- [release-please or goreleaser behavior differs from the traced source] → G-sandbox proves U1 to U7 on `open-platform-model/release-flow-sandbox`, running this change's exact config, before the cli changes.
- [A draft is left behind when goreleaser fails] → it is mutable and invisible; the failed run's jobs are re-run, or the manual recovery finishes it. The next release PR does not wait on it (U2).
- [The first real draft-first release fails in a way the sandbox missed] → immutable releases are not on for cli yet, so the release stays fixable in place; the owner enables the setting only after a clean release.
- [A future `name_template` breaks the draft match] → D2 forbids it; goreleaser would then create a second release for the tag, which task 5.1's "exactly one release" check and U3 catch.
- [The reusable workflow's interface lands differently] → task 3.5 aligns the caller with the merged file; the caller is a few lines.
