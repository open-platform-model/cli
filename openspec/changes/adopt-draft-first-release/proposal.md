## Why

The owner made release tags immutable across the open-platform-model repos that release (core, library, catalog_opm, cli, opm-operator): no tag under `refs/tags/` is ever moved, deleted or re-created, and a broken release is fixed by releasing the next version. The reason is the docs system: opmodel.dev pins a git ref per site version, and a moved tag silently changes what a published docs version shows. Three platform controls enforce the rule, all owner actions in the browser: the org tag ruleset `tags-immutable` (update, deletion, non-fast-forward, empty bypass list), the org tag ruleset `tags-create-app-only` (only the opm-release-please App may create a tag), and GitHub immutable releases, which lock a published release's tag and assets for good. Because only the release App can create a tag, a stale or hand-made tag cannot exist, so this change carries no tag-commit assertion of its own.

The cli's release flow is not compatible with immutable releases today. release-please creates the tag and publishes the GitHub Release first, then goreleaser uploads the archives into that already-published release. Under immutable releases that upload fails ("you cannot add, replace, or delete assets after a release is published"), and so does the `workflow_dispatch` recovery path, whose whole job is attaching assets to a release that exists without them. GitHub's documented flow is draft, attach assets, publish. This change moves the cli to that flow so the owner can turn immutable releases on for cli after one real release has shipped through it.

The owner also chose a lazy maintenance-branch model: a `release/vX.Y` branch is cut from the newest `vX.Y.*` tag by one automated action, only when a released minor needs a backport (never during beta), is never deleted, and takes backports by PR. The release workflow has to release from such a branch too, and the cli needs the caller that cuts one.

## What Changes

- **release-please creates a draft.** `release-please-config.json` gains `"draft": true` and `"force-tag-creation": true`, so release-please (as the release App, the only identity allowed to create a tag) creates the git tag itself and leaves the GitHub Release as a draft.
- **goreleaser fills and publishes the draft.** `.goreleaser.yml` `release` gains `use_existing_draft: true`, `mode: keep-existing` (release-please's notes stay), `replace_existing_artifacts: true` (a re-run on the draft replaces a half-uploaded asset) and a `make_latest` driven by the branch, so a backport release never becomes the repository's Latest. `draft` stays false, so goreleaser publishes the release as its last step.
- **Manual recovery refuses a published release.** The `goreleaser` job first reads the release's draft state (`gh release view --json isDraft`, with the job's `contents: write` token, the only permission that can see a draft) and fails with a roll-forward message when it is published. A dispatch must also run on the branch the tag was released from.
- **Release branches.** The release workflow also triggers on pushes to `release/**` and runs release-please with `target-branch` set to the pushed branch. A new thin `cut-release-branch` workflow (`workflow_dispatch`, input `X.Y`) calls the shared reusable workflow in `open-platform-model/.github`, which cuts `release/vX.Y` and opens the branch-local config PR. No release branch is created by this change.
- **AGENTS.md** gains one line pointing at the workspace release-tag rule.

Not changed: rulesets, the immutable releases setting, and every existing tag and release (owner actions in the browser).

SemVer class: none. Every commit is `ci` or `chore` and the PR title is `ci(release): ...`, so merging cuts no release on its own.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `release-workflow`: releases are created as drafts with a release-please-created tag, goreleaser attaches to the draft and publishes it last, manual recovery refuses a published release and a tag from another branch, the workflow releases from `release/**` branches without moving Latest, a `cut-release-branch` caller cuts maintenance branches, and the workflow never moves, deletes or re-creates a tag.

## Impact

- Commands: none. No Go code changes; `opm` behavior and the released archives are identical.
- Files: `release-please-config.json`, `.goreleaser.yml`, `.github/workflows/release.yml`, new `.github/workflows/cut-release-branch.yml`, `AGENTS.md`, `openspec/specs/release-workflow/spec.md` (on archive).
- Dependencies: the reusable `open-platform-model/.github` workflow `cut-release-branch.yml`, written in parallel; the caller pins it by commit SHA once it is merged.
- Consumers: a `releases/download/<tag>/...` URL answers 404 for a draft, so a half-finished release is never installable. The workspace `deps:pins:opm-cli` default resolves cli's newest git tag, not its newest published release, so run during the draft window it would pick a tag whose assets are not public yet (design.md D6).
- Risk: the draft-first path depends on release-please and goreleaser behavior proved in `open-platform-model/release-flow-sandbox` (gate G-sandbox). If the first real release stalls, the draft is mutable and the manual recovery run finishes it.
