## Why

The owner made release tags immutable across the open-platform-model repos that release (core, library, catalog_opm, cli, opm-operator): no tag under `refs/tags/` is ever moved, deleted or re-created, and a broken release is fixed by releasing the next version. The reason is the docs system: opmodel.dev pins a git ref per site version, and a moved tag silently changes what a published docs version shows. Two platform controls enforce the rule: the org tag ruleset `tags-immutable` (update, deletion, non-fast-forward, empty bypass list) and GitHub immutable releases, which lock a published release's tag and assets for good.

The cli's release flow is not compatible with immutable releases today. release-please creates the tag and publishes the GitHub Release first, then goreleaser uploads the archives into that already-published release. Under immutable releases that upload fails ("you cannot add, replace, or delete assets after a release is published"), and so does the `workflow_dispatch` recovery path, whose whole job is attaching assets to a release that exists without them. GitHub's documented flow is draft, attach assets, publish. This change moves the cli to that flow so the owner can turn immutable releases on for cli after one real release has shipped through it.

It also closes a silent failure in release-please itself: with `force-tag-creation`, `createRef` ignores a 422 when the tag already exists and `createRelease` then attaches to whatever commit that stale tag points at. The workflow asserts that the tag points at the commit release-please released before anything is built or published.

## What Changes

- **release-please creates a draft.** `release-please-config.json` gains `"draft": true` and `"force-tag-creation": true`, so release-please creates the git tag itself (a draft release has no tag of its own) and leaves the GitHub Release as a draft.
- **goreleaser fills and publishes the draft.** `.goreleaser.yml` `release` gains `use_existing_draft: true` (attach to release-please's draft instead of creating a second release), `mode: keep-existing` stated explicitly (release-please's notes stay), and `replace_existing_artifacts: true` (a re-run on the draft replaces a half-uploaded asset). `draft` stays false, so goreleaser publishes the release as its last step, with the prerelease flag from the tag.
- **Tag-SHA assertion and draft check before any publish step.** A new `.github/scripts/verify-release-tag.sh` resolves the tag's peeled commit and compares it with release-please's `sha` output (on a manual run, with the release's `target_commitish`), and reports whether the release is still a draft. A new `verify-release` job runs it; `goreleaser` and `publish-templates` need it.
- **Manual recovery refuses a published release.** The `workflow_dispatch` run builds binaries only for a release that is still a draft. For a published release the goreleaser job fails with a message that says to cut the next patch release instead. Template publishing stays allowed on a manual run for a published release, since it never touches the tag or the GitHub Release and only publishes versions GHCR does not hold.
- **AGENTS.md** gains one line pointing at the workspace release-tag rule and stating the roll-forward recovery.

Not changed: the org ruleset, the immutable releases setting and every existing tag and release (owner actions in the browser, outside any repo change). Turning immutable releases on for cli is gate G-owner's follow-up, after this flow has shipped one real release.

SemVer class: none. Every commit is `ci(release)` or `chore`, hidden types, so merging the change cuts no release on its own; the next releasable merge is the first draft-first release.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `release-workflow`: releases are created as drafts with a release-please-created tag, a `verify-release` job asserts the tag commit and draft state before any publish step, goreleaser attaches to the draft and publishes it last, manual recovery refuses a published release, and the workflow never moves, deletes or re-creates a tag.

## Impact

- Commands: none. No Go code changes; `opm` behavior and the released archives are identical.
- Files: `release-please-config.json`, `.goreleaser.yml`, `.github/workflows/release.yml`, new `.github/scripts/verify-release-tag.sh`, `AGENTS.md`, `openspec/specs/release-workflow/spec.md` (on archive).
- Consumers: a `releases/download/<tag>/...` URL answers 404 for a draft, so a half-finished release is never installable; today it is a published release with missing assets for the same window. The workspace `deps:pins:opm-cli` default resolves cli's newest git tag (`.tasks/deps/latest-tag.sh`), not its newest published release, so run during that window it would pick a tag whose assets are not public yet. The window and the hazard exist today too; the task stays as is, and its operator runs it after a release finished (design.md D5).
- Risk: the draft-first path depends on release-please and goreleaser behavior proved in `open-platform-model/release-flow-sandbox` (gate G-sandbox), not on the cli itself. If the first real release stalls, the draft is mutable and the manual recovery run finishes it.
