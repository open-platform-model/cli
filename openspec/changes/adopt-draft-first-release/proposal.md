## Why

The owner made release tags immutable across the open-platform-model repos that release (core, library, catalog_opm, cli, opm-operator): no tag under `refs/tags/` is ever moved, deleted or re-created, and a broken release is fixed by releasing the next version. The reason is the docs system: opmodel.dev pins a git ref per site version, and a moved tag silently changes what a published docs version shows. Three platform controls enforce the rule, all owner actions in the browser: the org tag ruleset `tags-immutable` (update, deletion, non-fast-forward, empty bypass list), the org tag ruleset `tags-create-app-only` (only the opm-release-please App may create a tag), and GitHub immutable releases, which lock a published release's tag and assets for good. Because only the release App can create a tag, a stale or hand-made tag cannot exist, so this change carries no tag-commit assertion of its own. The rule is enhancement 0021 D10 ("Release tags are immutable", on enhancements PR 74); this change carries the cli's share of it, 0021:D10:R1 and 0021:D10:R8 (see `enhancement.yaml`).

The cli's release flow is not compatible with immutable releases today. release-please creates the tag and publishes the GitHub Release first, then goreleaser uploads the archives into that already-published release. Under immutable releases that upload fails ("you cannot add, replace, or delete assets after a release is published"), and so does the `workflow_dispatch` recovery path, whose whole job is attaching assets to a release that exists without them. GitHub's documented flow is draft, attach assets, publish. This change moves the cli to that flow so the owner can turn immutable releases on for cli after one real release has shipped through it.

This is Phase 1 of the owner's plan. Release branches (`release/vX.Y`, cut lazily, never deleted, backports by PR) are Phase 2, before GA: the cli releases only from `main` here, and nothing in this change prepares a release branch. design.md "Phase 2" records what that later change must cover.

## What Changes

- **release-please creates a draft.** `release-please-config.json` gains `"draft": true` and `"force-tag-creation": true`, so release-please (as the release App, the only identity allowed to create a tag) creates the git tag itself and leaves the GitHub Release as a draft.
- **goreleaser fills and publishes the draft.** `.goreleaser.yml` `release` gains `use_existing_draft: true`, `mode: keep-existing` (release-please's notes stay), and `replace_existing_artifacts: true` (a re-run on the draft replaces a half-uploaded asset). `draft` stays false, so goreleaser publishes the release as its last step.
- **Manual recovery refuses a published release.** The `goreleaser` job first reads the release's draft state (`gh release view --json isDraft`, with the job's `contents: write` token, the only permission that can see a draft). A draft proceeds; a published release fails with a roll-forward message; a tag with no GitHub Release fails with an instruction to re-run the release-please job; any other read error fails as itself and is never taken for "published". A per-tag concurrency group keeps two runs off one draft.
- **goreleaser is pinned.** The goreleaser-action `version` input is pinned to the version the sandbox proved instead of floating on `~> v2`.
- **AGENTS.md** gains one line pointing at the workspace release-tag rule.

Not changed: rulesets, the immutable releases setting, and every existing tag and release (owner actions in the browser).

SemVer class: none. Every commit is `ci` or `chore` and the PR title is `ci(release): cut releases as drafts`. The repo squash-merges with the PR title as the commit title, so the title keeps a hidden type and the squash body carries no releasable conventional line; merging then cuts no release on its own.

Delivery: sections 1 to 4 in one PR; section 5 (verify the first draft-first release, archive) in a second PR after gate G-first-release.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `release-workflow`: releases are created as drafts with a release-please-created tag, goreleaser attaches to the draft and publishes it last, manual recovery refuses a published release, and the workflow never moves, deletes or re-creates a tag.

## Impact

- Commands: none. No Go code changes; `opm` behavior and the released archives are identical.
- Files: `release-please-config.json`, `.goreleaser.yml`, `.github/workflows/release.yml`, `AGENTS.md`, `openspec/specs/release-workflow/spec.md` (on archive).
- Dependencies: none outside this repo. The org tag rulesets and the immutable-releases setting are owner actions (gate G-platform).
- Consumers: a `releases/download/<tag>/...` URL answers 404 for a draft, so a half-finished release is never installable. The workspace `deps:pins:opm-cli` default resolves cli's newest git tag, not its newest published release, so run during the draft window it would pick a tag whose assets are not public yet (design.md D4).
- Risk: the draft-first path depends on release-please and goreleaser behavior proved in `open-platform-model/release-flow-sandbox` (gate G-sandbox). If the first real release stalls, the draft is mutable and the manual recovery run finishes it.
- Risk: immutable releases were briefly ON for cli on 2026-10-01, ahead of the plan; the owner turned them off (G-platform (b)). They must stay OFF until task 5.1 is green, or the first draft-first release locks at publish with no fix in place.
