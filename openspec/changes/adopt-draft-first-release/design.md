## Context

`.github/workflows/release.yml` runs on every push to `main`. release-please (action v5.0.0, release-please 17.6.0, acting as the opm-release-please App) maintains the release PR; when that PR merges it creates the tag and a published GitHub Release, and two jobs follow: `goreleaser` (archives, `checksums.txt`, `LICENSE` uploaded into the existing release, `release.prerelease: auto`) and `publish-templates` (official template modules to GHCR, unpublished versions only). A `workflow_dispatch` run with a `tag` input re-runs both jobs for a release that exists without its assets (the v1.0.0-alpha.21 incident).

The workspace rule (owner, 2026-10-01): release tags are immutable; a broken release is fixed by releasing the next version. Enforcement is the org tag ruleset `tags-immutable` and, for cli after this change ships one release, GitHub immutable releases. Immutable releases forbid adding, replacing or deleting assets after publish, so the current publish-then-upload order cannot run under them. This change is the cli half of the canon's "draft-first" item; opm-operator carries its own change.

No Go code, command, flag or exit code changes. The config rules about command syntax, flags, exit codes and example output do not apply; the CI-side error messages are specified instead.

## Goals / Non-Goals

**Goals:**

- A release is created as a draft, filled with every asset, and published last, by goreleaser.
- Before any build or publish step, the workflow proves the tag points at the commit release-please released.
- The manual recovery run works on a draft and refuses a published release with a roll-forward instruction.
- Nothing in the workflow moves, deletes or re-creates a tag.

**Non-Goals:**

- Turning on immutable releases for cli, editing rulesets, or touching any existing tag or release (owner actions; gate G-owner).
- Changing what is built, the archive names, the template publish pipeline, or the release line.
- A tag-mutation guard for agents (workspace hook) or the drift ledger (`open-platform-model/.github`); both live outside this repo.

## Decisions

### D1. release-please creates the tag and a draft release

`release-please-config.json` top level gains `"draft": true` and `"force-tag-creation": true`.

- `draft: true` makes `createRelease` pass `draft: true`. A draft GitHub Release has no git tag until it is published, which breaks two things: goreleaser checks out the tag, and release-please's next run looks for the previous release's tag (release-please issue 1650).
- `force-tag-creation: true` makes release-please call `git.createRef refs/tags/<tag>` at the release commit before `createRelease` (release-please 17.2.0+, `src/github.ts` at v17.3.0). The tag therefore exists from the first second, exactly as today, created by the release App, which the `tags-immutable` ruleset allows (it carries no `creation` rule).

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

`draft` stays unset (false). Traced in goreleaser `internal/client/github.go`:

- Without `use_existing_draft`, `findRelease` calls `GetReleaseByTag`, which GitHub answers only for published releases, so goreleaser would create a second release on the same tag. With it, `findDraftRelease` lists releases and matches a draft by **name**. goreleaser's `name_template` defaults to `{{.Tag}}`, and release-please names cli releases after the tag (`v1.0.0-beta.1` through `v1.0.0-beta.3` all carry name == tag), so the draft is found. The config MUST NOT set a `name_template` that departs from the tag.
- `createOrUpdateRelease` keeps the found release a draft while uploading, merges notes per `mode`, and fails fast if the found release is immutable.
- `PublishRelease` sets `draft: false` at the end (and `make_latest: false` for a prerelease), so the release becomes visible only with all assets on it.
- `replace_existing_artifacts` only matters on a re-run against the same draft; a published release never reaches the upload because of D4.

The goreleaser-action keeps `version` at its default (`~> v2`, latest v2), which is above v2.5 (`use_existing_draft`) and v2.18 (preflight abort on an immutable published release).

### D3. verify-release job: tag-SHA assertion and draft state

New script `.github/scripts/verify-release-tag.sh <tag> [expected-sha]`, run by a new `verify-release` job that `needs: release-please` with the same `if:` as today's downstream jobs. It uses `gh` with the job's `GITHUB_TOKEN` (`contents: read`). The job checks out only `.github/scripts` at `github.sha` (the pushed main commit, or the default branch on a manual run), never the tag, so a stale tag cannot supply its own verifier:

1. Resolve the release: `gh release view "$TAG" --json isDraft,targetCommitish`. Absent release: fail.
2. Expected commit: the `expected-sha` argument (release-please's `sha` output, exposed by the `release-please` job as a new output), or on a manual run the release's `targetCommitish`, which release-please sets to the release commit SHA.
3. Actual commit: the tag's peeled commit, `git ls-remote origin "refs/tags/$TAG" "refs/tags/$TAG^{}"` taking the `^{}` line when present (annotated) and the plain line otherwise (release-please's `createRef` makes a lightweight tag). Run as `git ls-remote "https://github.com/$GITHUB_REPOSITORY"`, independent of the sparse checkout. Absent tag: fail.
4. Mismatch: fail with `release tag <tag> points at <actual>, release-please released <expected>; the tag is never moved: cut the next release instead`.
5. Write `draft=<true|false>` and `tag=<tag>` to `$GITHUB_OUTPUT`.

**Why**: `createRef` swallows a 422 for an existing tag and `createRelease` then attaches to that tag whatever `target_commitish` says, so a stale tag at the wrong commit would otherwise be built and published silently.

Data flow:

```text
push to main ─► release-please ──(releases_created, tag_name, sha)──► verify-release ──(tag, draft)──┬─► goreleaser (draft only) ─► publish draft
dispatch(tag) ────────────────────────────────────────────────────────► verify-release                └─► publish-templates (GHCR)
```

### D4. Manual recovery is draft-only for binaries

`goreleaser` `needs: [release-please, verify-release]` and runs when `verify-release` succeeded. Its first step fails when `needs.verify-release.outputs.draft != 'true'`, with:

```text
release <tag> is already published; its assets can no longer change and its tag never moves.
Cut the next patch release instead.
```

On a push run the release is always a draft at this point (release-please just created it), so the step only bites on a manual run. It is a step failure, not a skipped job, so a dispatch on a published release is red, not silently green.

`publish-templates` `needs: [release-please, verify-release]` but ignores `draft`: it touches neither the tag nor the GitHub Release, and `publish-templates.sh` only publishes versions GHCR does not hold, so a recovery run for a published release that lost its templates is still valid. Both jobs check out `needs.verify-release.outputs.tag` (the verified tag), under `always() && needs.verify-release.result == 'success'`, so a skipped release-please on a dispatch does not skip them and a failed verification always does.

Recovery runbook, written as the dispatch input description and the workflow header comment:

- Draft with missing or partial assets: re-run the failed jobs, or dispatch with the tag.
- Published release that is wrong or incomplete: cut the next patch; never re-tag, never delete.

### D5. Consumers and the draft window

Between release-please and goreleaser's publish the git tag exists while the release is a draft. `releases/download/<tag>/...` answers 404 for a draft, so nothing can install a half-built release; today the same window shows a published release with missing assets. The workspace `deps:pins:opm-cli` default resolves the newest git tag, so it is run after a release finished, as today. Changing that helper is workspace tooling, outside this repo.

## Research & Decisions

### Draft-first versus leaving cli out of immutable releases

**Context**: immutable releases block the current upload-after-publish order.
**Explored**: critic review (`tagres/critic.md` C4, C5, C6), release-please `src/github.ts` and `src/manifest.ts` at v17.3.0, goreleaser `internal/client/github.go` at main, GitHub immutable releases docs and GA changelog.
**Options considered**:
1. Draft-first with `force-tag-creation` (this change): GitHub's recommended flow; supported config keys in both tools; recovery still works on drafts.
2. Keep cli out of immutable releases: no work, but the ruleset alone leaves an org owner able to edit it, and the tag of a published cli release stays re-pointable after a ruleset edit.
3. goreleaser creates the release itself (release-please `skip-github-release`): loses release-please's notes and its tag-anchoring for the next release PR.
**Decision**: option 1.
**Rationale**: the only option that runs under immutable releases without giving up release-please's notes or the recovery path.

### Where the tag-SHA assertion runs

**Options considered**:
1. A step inside the `release-please` job: cannot cover the manual run, where that job is skipped.
2. A step duplicated in `goreleaser` and `publish-templates`: two copies, two places to drift.
3. A separate `verify-release` job both need (chosen): one implementation for push and dispatch, and its outputs carry the draft state D4 needs.

### Unverified assumptions (gate G-sandbox, recorded in section 1)

- U1. release-please-action sets `releases_created`, `tag_name` and `sha` for a draft release.
- U2. The next release PR anchors on the force-created tag while the previous release is still a draft, and after it is published (no duplicate changelog, correct next version).
- U3. goreleaser with `use_existing_draft` finds release-please's draft by name, keeps its notes, uploads, and publishes it with the prerelease flag set and Latest unchanged.
- U4. The flow passes under the `tags-immutable` ruleset and with immutable releases on (asset upload to a draft allowed, publish locks it).
- U5. A dispatch on a draft finishes it; a dispatch on a published release fails at the D4 step.

If the sandbox contradicts any of these, section 1 stops the change and this design is revised before section 2.

## Risks / Trade-offs

- [release-please or goreleaser behavior differs from the traced source] → G-sandbox proves U1 to U5 on `open-platform-model/release-flow-sandbox` before the cli changes.
- [A draft is left behind when goreleaser fails] → it is mutable and invisible; the failed run's jobs are re-run, or the manual recovery finishes it. The release PR for the next version does not wait on it (U2).
- [The first real draft-first release fails in a way the sandbox missed] → immutable releases are not on for cli yet (G-owner follow-up), so the release stays fixable in place; the owner enables the setting only after a clean release.
- [A future `name_template` breaks the draft match] → D2 forbids it; goreleaser would then create a second release and fail on the duplicate tag, loudly.
- [The verify job adds about 10 s to a release run] → accepted.
