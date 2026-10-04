## Why

On 2026-10-02 the owner set the squash message for the release-cascade repos to `BLANK` (workspace `RELEASING.md`, section "Owner settings" › "Merge settings", lines 457-467, the `BLANK` bullet at 460-467). A squash commit then carries only the PR title. Two consequences follow, and RELEASING.md states both:

- a breaking change is a `!` in the PR title (`feat!:`, `fix(deps)!:`); a `BREAKING CHANGE:` footer in the PR body or in a branch commit never reaches `main`;
- a forced version is a `release-as` key in `release-please-config.json`, set by a normal PR and removed by the next PR once that release is cut; a `Release-As:` footer never reaches `main` (also RELEASING.md, section "The cascade" › "Title from diff class", lines 249-250).

The cli's own documents still teach the opposite rule, so a contributor who follows them ships a breaking change with no `!` (release-please files it as an ordinary feature) or forces a version with a footer that silently does nothing:

- `AGENTS.md:144-147` says a line change "travels as a one-shot `Release-As` footer in the carrier's squash commit message, never as a `release-as` key in `release-please-config.json`".
- `AGENTS.md:357` says a beta break lands "only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note the CHANGELOG shows"; `AGENTS.md:358` says "never a `Release-As` to `1.1.0-beta.1` or beyond".
- `CONSTITUTION.md:106-107` and its mirror in `openspec/config.yaml:53-54` (the `context` block) repeat the `BREAKING CHANGE:` footer rule.
- `openspec/config.yaml:225` (apply guidance) gives "because the squash body reaches release-please" as the reason for a commit-body rule; under `BLANK` the squash body is empty.
- `.claude/skills/commit/SKILL.md:26` (tracked) names "a non-obvious breaking change" as a reason for a commit body; under `BLANK` that body never reaches `main`.
- The main spec `openspec/specs/release-workflow/spec.md:10` requires that the configuration "SHALL NOT carry a `release-as` value" and that a line change "SHALL travel as a one-shot `Release-As: <version>` footer"; its scenarios at lines 20-26 ("Line change forced by a footer", "Next release stays on the beta line") are written around that footer.

`.github/workflows/pr-title.yml:3-13` already states the new rule in its comment and stays as it is.

## What Changes

- `AGENTS.md`: the "Release line: beta" note (lines 144-147) says a line change is a `release-as` key in `release-please-config.json`, landed by a normal PR and removed before any later release PR merges; that the key takes effect only with a releasable commit since the last release (so the key PR carries a releasable type or lands after a releasable commit); and that a `Release-As:` footer never reaches `main`. The "Beta promise" bullet (line 357) says a beta break is a `!` in the PR title (and, until the owner merge settings land, in a one-commit PR's commit subject, with squash as the only merge method), the squash title is the CHANGELOG entry, and the migration note goes in the PR body (and the user docs where users need it), with a hand edit of the release PR's `CHANGELOG.md` carrying it into the CHANGELOG. The "Beta skew rule" bullet (line 358) says "never a `release-as` of `1.1.0-beta.1` or beyond".
- `CONSTITUTION.md` (Pre-GA note, lines 104-113) and `openspec/config.yaml` (`context`, lines 51-60): the same wording as the "Beta promise" bullet, kept byte-identical between the two, as today.
- `.claude/skills/commit/SKILL.md:26`: "a non-obvious breaking change" no longer warrants a body; the `!` goes in the subject and PR title and the note in the PR body.
- `openspec/config.yaml:225`: the apply-guidance rule against a body line starting with `word(` stays; its reason changes from "the squash body reaches release-please" to one that holds under both the current and the `BLANK` setting (design.md D3).
- `openspec/specs/release-workflow/spec.md`: the requirement "Release runs from pushes to main through release-please" loses its footer clause and the "Line change forced by a footer" scenario. A new requirement states that a version line moves only through a `release-as` key in a normal PR, removed after the release. Because the OpenSpec CLI refuses a MODIFIED delta that drops a main-spec scenario, the delta is REMOVED plus ADDED under new names (design.md D1). A second new requirement states the `!`-in-the-title rule with its own scenarios.

No Go code, workflow, script, `release-please-config.json` or template changes. This change does not add or remove a `release-as` key; it only documents how one is used.

SemVer: none. Every commit is `docs`, hidden in `release-please-config.json:23`, so no cli release is cut.

## Depends on / gates

- **Workspace RELEASING.md "Owner settings"** (lines 459-467) and "Title from diff class" (lines 248-250) on workspace `main`: already there (owner decision 2026-10-02, entry 19 of the owner selections). This change cites them by section.
- **S0 owner settings** (RELEASING.md, section "Owner settings"): not a merge precondition, but the live cli settings today are `squash_merge_commit_title: COMMIT_OR_PR_TITLE`, `squash_merge_commit_message: COMMIT_MESSAGES`, with merge commits and rebase merges still enabled. A `!` in the PR title reaches `main` only for a multi-commit squash merge; a one-commit PR squashes under its commit subject, and a merge commit or rebase merge carries the branch commits instead. So the new text carries an interim clause: until the owner merge settings land, merge by squash only and give a one-commit PR's commit subject the same `!` (design.md D2). `pr-title.yml` checks only the PR title (it sets no `validateSingleCommit`); only its comment states the one-commit rule.
- **`.github` `add-cascade-resolver`**: not a dependency. This change edits no task.
- **Sibling changes** of the same goal in opm-operator and catalog_opm, and the `.github` README and `mention-guard.yml` comment fixes: independent; no shared files.
- **A gap in RELEASING.md itself**: a `release-as` key opens no release PR when only hidden-type commits (`chore`, `build`, `ci`, `docs`, `test`) landed since the last release, because release-please skips an empty changelog even when the key is set. The old footer escaped that only because the conventionalcommits preset un-hides a commit carrying a `Release-As:` footer; the config key gets no such exemption. This change states the condition in the cli text; RELEASING.md "Merge settings" needs the same sentence (SUPERVISOR follow-up in tasks.md). Not a merge precondition.
- **Enhancement 0021**: `enhancements/0021/policy/01-core-schema.md:8,22,70,74,165,167,169,173-183` still requires the `BREAKING CHANGE:` footer and the `Release-As:` footer and forbids a `release-as` key (167: "MUST NOT be written as a `release-as` key"). The existing requirement cites `0021:D10:R8`. Updating the enhancement belongs to the enhancements repo and the supervisor (SUPERVISOR follow-up in tasks.md); this change does not wait for it, because the owner decision on 2026-10-02 supersedes that policy text.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-workflow`: the release-please requirement drops the `Release-As:` footer clause; a new requirement says a version line moves only through a `release-as` key set by a normal PR, effective only with a releasable commit, and removed after the release; a second says a breaking change is a `!` in the PR title.

## Impact

- Commands: none.
- Files: `AGENTS.md`, `CONSTITUTION.md`, `openspec/config.yaml`, `.claude/skills/commit/SKILL.md`; the archive rewrites one requirement of `openspec/specs/release-workflow/spec.md` and adds two.
- Contributors and agents: a breaking cli change is marked with `!` in the PR title (and, until the owner merge settings land, in a one-commit PR's commit subject); a forced version goes through `release-please-config.json` in a normal PR of a releasable type (or landing after a releasable commit), followed by a PR that removes the key before any later release PR merges.
- Risk: a `release-as` key left in place pins every later release to that version (release-please re-applies it on every run). The new requirement makes its removal a stated step; no gate enforces it (design.md D4).
