## Context

The owner chose `squash_merge_commit_message: BLANK` on 2026-10-02 (workspace `RELEASING.md`, section "Owner settings" › "Merge settings"). Under it the squash commit on `main` is the PR title alone, so release-please sees nothing but titles. The cli's contributor documents (`AGENTS.md`, `CONSTITUTION.md`, the constitution copy in `openspec/config.yaml`) and the main spec `release-workflow` still describe the earlier footer-based rules: `BREAKING CHANGE:` footers for breaks, a one-shot `Release-As:` footer for a forced version, and a ban on the `release-as` key. This change rewrites that text. It changes no behavior of any workflow or script; the release workflow never read footers itself, release-please did.

Today's live settings: the owner has not yet applied the Phase 0 settings (RELEASING.md, section "Rollout and changes" › "Phases", row 0). The live cli settings (read by the plan reviewer, 2026-10-04) are `squash_merge_commit_title: COMMIT_OR_PR_TITLE`, `squash_merge_commit_message: COMMIT_MESSAGES`, `allow_merge_commit: true`, `allow_rebase_merge: true`. The wording below must therefore be correct both before and after the settings change, which needs an interim clause (D2).

## Goals / Non-Goals

**Goals:**

- Every cli document that tells a contributor how to mark a break or force a version gives the `BLANK` rule from RELEASING.md.
- The `release-workflow` main spec states the `release-as` key as the one mechanism and keeps every other requirement and scenario unchanged.
- `CONSTITUTION.md` and the `context` copy in `openspec/config.yaml` stay byte-identical.

**Non-Goals:**

- Adding, removing or testing a `release-as` key in `release-please-config.json`.
- A gate that fails while a `release-as` key outlives its release (D4).
- The GA sentence ("`prerelease: false` plus a visible carrier commit per package"): it does not depend on a footer and stays.
- `.github/workflows/pr-title.yml`: its comment (lines 3-13) already states the `BLANK` rule and the interim.
- The `.github` README, `mention-guard.yml`, opm-operator and catalog_opm text, and enhancement 0021's policy text: separate items.

## Decisions

### D1: The spec delta is REMOVED plus ADDED, not MODIFIED

The requirement "Release runs from pushes to main through release-please" has four scenarios. One of them, "Line change forced by a footer", describes the footer mechanism and cannot survive. The OpenSpec CLI (1.12) refuses a MODIFIED delta that drops a main-spec scenario, so the delta:

- REMOVES "Release runs from pushes to main through release-please", with a Reason and a Migration;
- ADDS "Release-please runs on pushes to main", carrying the old text minus the footer clause and the three scenarios that do not involve a footer ("Push without a merged release PR", "Release PR merged", "Next release stays on the beta line"), renamed under the new requirement and with the last one reworded to name the absent key instead of an absent footer;
- ADDS "A version line moves only through a release-as key", with four scenarios: a key forces the line change when a releasable commit is present, a hidden-type key PR alone opens no release PR (D6), the key is removed before any later release PR merges, and a `Release-As:` footer in a PR body or branch commit forces nothing;
- ADDS "A breaking change is marked by a bang in the pull request title", with two scenarios: a `feat!:` title advances `-beta.N` under breaking changes, and a `BREAKING CHANGE:` footer alone marks nothing.

Keeping the trigger, App, draft and outputs text unchanged in the re-added requirement means archive replaces only the release-as clause in substance. OpenSpec applies ADDED requirements last, so archive moves the release-please requirement from the top of `openspec/specs/release-workflow/spec.md` to the end, after "Commit types decide whether a release is cut". That is accepted: requirement order carries no meaning, and reordering by hand at archive time would be an edit the delta does not describe.

The re-added release-please requirement keeps `Source: 0021:D10:R8`, which covers the workflow trigger and draft-first release. The new release-as requirement cites workspace RELEASING.md, section "Owner settings", because the owner decision of 2026-10-02 is its source, not 0021.

### D2: The wording needs an interim clause until the Phase 0 settings land

| Mechanism | Today (`COMMIT_OR_PR_TITLE`, `COMMIT_MESSAGES`, merge and rebase merges allowed) | After Phase 0 (`PR_TITLE`, `BLANK`, squash only) |
| --- | --- | --- |
| `release-as` key in `release-please-config.json` | forces the version, once a releasable commit is on `main` since the last release (D6) | same |
| `!` in the PR title | reaches `main` only for a multi-commit squash merge; a one-commit squash takes its commit subject, and a merge commit or rebase merge carries the branch commits | reaches `main` for every PR |
| `BREAKING CHANGE:` / `Release-As:` footer in a commit | reaches `main` | never reaches `main` |

So text that says only "`!` in the PR title" is wrong today for one-commit PRs and for any merge that is not a squash. `pr-title.yml` does not cover the gap: it sets no `validateSingleCommit`, so it checks the PR title alone; only its comment (lines 7-8) states the one-commit rule.

Two fixes were open: an interim clause in the text, or a SUPERVISOR gate requiring the cli's `squash_merge_commit_title` to be `PR_TITLE` before merge. The gate would hold this docs change behind an owner action that is still pending, while the old text keeps teaching the footer. The interim clause is chosen: `AGENTS.md` (both bullets) and the `CONSTITUTION.md`/`openspec/config.yaml` copy say "until the owner merge settings land, merge by squash only and give a one-commit PR's commit subject the same `!`". The clause is dead text once Phase 0 lands and can be dropped then; following it is correct on both sides. The new text says a footer "never reaches `main` under the `BLANK` squash message", which is accurate and names the setting.

### D3: The commit-body rule in the apply guidance stays, with a new reason

`openspec/config.yaml:225` forbids a commit body line starting with `word(` "because the squash body reaches release-please". Under `BLANK` that reason is false, but the rule is still worth keeping: until Phase 0 the squash body is the branch commit messages, and a body line such as `fix(cli): ...` is then parsed by release-please as a separate commit. The new reason: "because a squash message other than `BLANK` copies commit bodies into `main`, where release-please parses such a line as a commit". The rule text itself is unchanged, so agents behave the same.

### D4: No gate for a stale `release-as` key

A `release-as` key that stays after its release pins every later release to that version; release-please then proposes the same version again. A G1-style check could refuse a release PR whose version equals the key while the tag already exists. The requirement asks for removal before any later release PR merges, not by the very next PR: a Dependabot or cascade PR may well merge first, and what matters is that the key is gone before release-please's next proposal is accepted. It is not added here:

- the mechanism has not been used under the new rule yet, and a stale key shows itself on the next release PR, which proposes the version already released; every release PR is merged by a human (owner selection 7), who sees it. What release-please and the release workflow do if such a PR merged anyway is not verified here;
- adding a gate changes `release-gates` and a script, which this docs change does not otherwise touch.

The requirement makes the removal PR an explicit step. A gate can follow as its own change if the owner wants one (Open Questions).

### D5: Where a migration note goes

The old text made the `BREAKING CHANGE:` footer "the migration note the CHANGELOG shows". Under `BLANK` the CHANGELOG entry is the PR title alone. The new text says the migration note goes in the PR body, and in the user docs (`docs/site/`) when users need it to upgrade. This is a statement of fact under the owner's setting, not a new policy; whether the CHANGELOG should still carry migration notes (for example by a human editing the release PR's `CHANGELOG.md`) is an owner question (Open Questions).

### D6: A release-as key needs a releasable commit beside it

release-please 17.11.2 (`strategies/base`, the `changelogEmpty` check after `buildNewVersion`) skips a release with "No user facing commits found" when every commit since the last release is of a hidden type, even when `releaseAs` is set. The old `Release-As:` footer escaped this only because the conventionalcommits preset un-hides a commit that carries the footer (`conventional-changelog-conventionalcommits/writer-opts.js`); the config key gets no such exemption. A PR that only edits `release-please-config.json` would naturally be typed `chore`, `build` or `ci`, which are hidden (`release-please-config.json:23-28`), so it would open nothing. The text therefore says the key takes effect only with a releasable commit since the last release: the key PR carries a releasable type itself, or lands with or after a `fix:`/`feat:` commit in the same release. Workspace RELEASING.md "Merge settings" (lines 460-467) has the same gap; that is a SUPERVISOR follow-up, not a cli edit.

## Research & Decisions

### Which files carry the old rule

**Context**: The change must catch every stale passage and nothing else.
**Explored**: `grep -rn -i -E "release-as|BREAKING CHANGE|footer|COMMIT_MESSAGES|squash"` over `*.md`, `*.yml`, `*.yaml`, `*.json`, `*.sh`, `*.go`, excluding archived changes and `CHANGELOG.md`.
**Options considered**:
1. Fix only `AGENTS.md` and the spec - leaves `CONSTITUTION.md` and its `openspec/config.yaml` copy teaching the footer.
2. Fix every hit that states the rule - five files, all text.
**Decision**: Option 2. The hits are `AGENTS.md:144-147,357,358`, `CONSTITUTION.md:106-107`, `openspec/config.yaml:53-54,225`, `openspec/specs/release-workflow/spec.md:10,20-26`, and `.claude/skills/commit/SKILL.md:26` (tracked; "a non-obvious breaking change" as a reason for a commit body, which `BLANK` drops; the plan review found it, the first grep missed it because it says "breaking change" in lower case). That file is also a drifted copy of the workspace commit skill; re-syncing it is out of scope and belongs to the workspace owner of that skill. `.github/workflows/pr-title.yml:3-13` is already correct. The remaining hits (`CONSTITUTION.md:102`, `AGENTS.md:18` "Generated-with footers", `QUICKSTART.md:350`, `adr/004`, `docs/roadmap.md:147`, `.github/labels.yml:136`) are unrelated uses of "footer" or "breaking change".
**Rationale**: Agents read `openspec/config.yaml` and `AGENTS.md`; humans read `CONSTITUTION.md`. All three must agree.

### MODIFIED versus REMOVED plus ADDED

**Context**: The footer scenario must go.
**Explored**: The workspace note that OpenSpec 1.12 refuses a MODIFIED delta that drops a main-spec scenario.
**Options considered**:
1. MODIFIED keeping the scenario name "Line change forced by a footer" with new text - a misleading name in the main spec forever.
2. REMOVED plus ADDED under new names - clean names, one extra Reason/Migration block.
**Decision**: Option 2 (D1).
**Rationale**: Scenario names are what readers scan; one must not say "footer" when the mechanism is a key.

## Risks / Trade-offs

- [A stale `release-as` key pins later releases] → the new requirement names the removal PR as a step; D4 defers a gate.
- [Enhancement 0021 policy text still says the opposite] → SUPERVISOR follow-up in tasks.md; the cli text cites RELEASING.md, which carries the owner decision.
- [Migration notes leave the CHANGELOG] → D5; owner question below.
- [A hidden-type key PR forces nothing] → D6; the text names the releasable-commit condition.
- [The interim clause outlives Phase 0] → harmless (it describes a superset of the final rule); drop it in a later docs PR.

## Open Questions

- Enhancement 0021 (`enhancements/0021/policy/01-core-schema.md:8,22,70,74,165,167,169,173-183`) still requires a `BREAKING CHANGE:` footer and a `Release-As:` footer and forbids a `release-as` key (167), including for a prerelease-type flip and a post-GA major crossing. Reconciling it with `BLANK` is a SUPERVISOR follow-up in tasks.md; whether the crossing procedure becomes "the `module:` line edit and a `release-as` key in one releasable PR" is the owner's call there.
- Should a breaking change's migration note still reach the CHANGELOG, for example by editing the release PR's `CHANGELOG.md` before merging it? (owner; D5)
- Should a gate refuse a release PR while a `release-as` key names an already-tagged version? (owner; D4)
