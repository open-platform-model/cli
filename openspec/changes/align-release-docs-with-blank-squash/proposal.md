## Why

On 2026-10-02 the owner set the squash message for the release-cascade repos to `BLANK` (workspace `RELEASING.md`, section "Owner settings" › "Merge settings", lines 459-467). A squash commit then carries only the PR title. Two consequences follow, and RELEASING.md states both:

- a breaking change is a `!` in the PR title (`feat!:`, `fix(deps)!:`); a `BREAKING CHANGE:` footer in the PR body or in a branch commit never reaches `main`;
- a forced version is a `release-as` key in `release-please-config.json`, set by a normal PR and removed by the next PR once that release is cut; a `Release-As:` footer never reaches `main` (also RELEASING.md, section "The cascade" › "Title from diff class", lines 248-250).

The cli's own documents still teach the opposite rule, so a contributor who follows them ships a breaking change with no `!` (release-please files it as an ordinary feature) or forces a version with a footer that silently does nothing:

- `AGENTS.md:144-147` says a line change "travels as a one-shot `Release-As` footer in the carrier's squash commit message, never as a `release-as` key in `release-please-config.json`".
- `AGENTS.md:357` says a beta break lands "only as a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note the CHANGELOG shows"; `AGENTS.md:358` says "never a `Release-As` to `1.1.0-beta.1` or beyond".
- `CONSTITUTION.md:106-107` and its mirror in `openspec/config.yaml:53-54` (the `context` block) repeat the `BREAKING CHANGE:` footer rule.
- `openspec/config.yaml:225` (apply guidance) gives "because the squash body reaches release-please" as the reason for a commit-body rule; under `BLANK` the squash body is empty.
- The main spec `openspec/specs/release-workflow/spec.md:10` requires that the configuration "SHALL NOT carry a `release-as` value" and that a line change "SHALL travel as a one-shot `Release-As: <version>` footer"; its scenarios at lines 20-26 ("Line change forced by a footer", "Next release stays on the beta line") are written around that footer.

`.github/workflows/pr-title.yml:3-13` already states the new rule and stays as it is.

## What Changes

- `AGENTS.md`: the "Release line: beta" note (lines 144-147) says a line change is a `release-as` key in `release-please-config.json`, landed by a normal PR and removed by the next PR once that release is cut, and that a `Release-As:` footer never reaches `main`. The "Beta promise" bullet (line 357) says a beta break is a `!` in the PR title, the PR title is the CHANGELOG entry, and the migration note goes in the PR body (and the user docs where users need it). The "Beta skew rule" bullet (line 358) says "never a `release-as` of `1.1.0-beta.1` or beyond".
- `CONSTITUTION.md` (Pre-GA note, lines 104-113) and `openspec/config.yaml` (`context`, lines 51-60): the same wording as the "Beta promise" bullet, kept byte-identical between the two, as today.
- `openspec/config.yaml:225`: the apply-guidance rule against a body line starting with `word(` stays; its reason changes from "the squash body reaches release-please" to one that holds under both the current and the `BLANK` setting (design.md D3).
- `openspec/specs/release-workflow/spec.md`: the requirement "Release runs from pushes to main through release-please" loses its footer clause and the "Line change forced by a footer" scenario. A new requirement states that a version line moves only through a `release-as` key in a normal PR, removed after the release. Because the OpenSpec CLI refuses a MODIFIED delta that drops a main-spec scenario, the delta is REMOVED plus ADDED under new names (design.md D1).

No Go code, workflow, script, `release-please-config.json` or template changes. This change does not add or remove a `release-as` key; it only documents how one is used.

SemVer: none. Every commit is `docs`, hidden in `release-please-config.json:23`, so no cli release is cut.

## Depends on / gates

- **Workspace RELEASING.md "Owner settings"** (lines 459-467) and "Title from diff class" (lines 248-250) on workspace `main`: already there (owner decision 2026-10-02, entry 19 of the owner selections). This change cites them by section.
- **S0 owner settings** (RELEASING.md, section "Owner settings"): not a merge precondition. The new wording is correct under today's settings too: a `release-as` key forces a version regardless of the squash message, and a `!` in the PR title reaches `main` whenever the PR title is the squash title (every multi-commit PR today; every PR once `squash_merge_commit_title` is `PR_TITLE`). `pr-title.yml` already makes the commit subject of a one-commit PR conventional for the interim (design.md D2).
- **`.github` `add-cascade-resolver`**: not a dependency. This change edits no task.
- **Sibling changes** of the same goal in opm-operator and catalog_opm, and the `.github` README and `mention-guard.yml` comment fixes: independent; no shared files.
- **Enhancement 0021**: `enhancements/0021/policy/01-core-schema.md:8,22,70,74` still requires the `BREAKING CHANGE:` footer and forbids a `release-as` key. The existing requirement cites `0021:D10:R8`. Updating the enhancement belongs to the enhancements repo and the supervisor (Open Questions in design.md); this change does not wait for it, because the owner decision on 2026-10-02 supersedes that policy text.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-workflow`: the release-please requirement drops the `Release-As:` footer clause, and a new requirement says a version line moves only through a `release-as` key set by a normal PR and removed after the release.

## Impact

- Commands: none.
- Files: `AGENTS.md`, `CONSTITUTION.md`, `openspec/config.yaml`; the archive rewrites one requirement of `openspec/specs/release-workflow/spec.md` and adds one.
- Contributors and agents: a breaking cli change is marked with `!` in the PR title; a forced version goes through `release-please-config.json` in a normal PR, followed by a PR that removes the key once the release is cut.
- Risk: a `release-as` key left in place pins every later release to that version (release-please re-applies it on every run). The new requirement makes its removal a stated step; no gate enforces it (design.md D4).
