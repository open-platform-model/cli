## Context

The owner chose `squash_merge_commit_message: BLANK` on 2026-10-02 (workspace `RELEASING.md`, section "Owner settings" › "Merge settings"). Under it the squash commit on `main` is the PR title alone, so release-please sees nothing but titles. The cli's contributor documents (`AGENTS.md`, `CONSTITUTION.md`, the constitution copy in `openspec/config.yaml`) and the main spec `release-workflow` still describe the earlier footer-based rules: `BREAKING CHANGE:` footers for breaks, a one-shot `Release-As:` footer for a forced version, and a ban on the `release-as` key. This change rewrites that text. It changes no behavior of any workflow or script; the release workflow never read footers itself, release-please did.

Today's live settings: the owner has not yet applied the Phase 0 settings (RELEASING.md, section "Rollout and changes" › "Phases", row 0). The cli's `squash_merge_commit_title` was `COMMIT_OR_PR_TITLE` when `prepare-release-cascade` was planned (that change's proposal), and the squash message was `COMMIT_MESSAGES`. The wording below must therefore be correct both before and after the settings change.

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
- ADDS "A version line moves only through a release-as key", with three scenarios: a key forces the line change, a key left in place pins later releases (the reason it must be removed), and a `Release-As:` footer in a PR body or branch commit forces nothing.

Keeping the trigger, App, draft and outputs text unchanged in the re-added requirement means archive replaces only the release-as clause in substance.

The re-added release-please requirement keeps `Source: 0021:D10:R8`, which covers the workflow trigger and draft-first release. The new release-as requirement cites workspace RELEASING.md, section "Owner settings", because the owner decision of 2026-10-02 is its source, not 0021.

### D2: The wording holds before and after the Phase 0 settings

| Mechanism | Today (`COMMIT_OR_PR_TITLE`, `COMMIT_MESSAGES`) | After Phase 0 (`PR_TITLE`, `BLANK`) |
| --- | --- | --- |
| `release-as` key in `release-please-config.json` | forces the version | forces the version |
| `!` in the PR title | reaches `main` for a multi-commit PR; a one-commit PR squashes under its commit subject, which `pr-title.yml` also requires to be conventional | reaches `main` for every PR |
| `BREAKING CHANGE:` / `Release-As:` footer in a commit | reaches `main` | never reaches `main` |

So the new text does not need an "until the settings change" clause: following it is correct on both sides of Phase 0. The one interim caveat (the commit subject of a one-commit PR) is already documented in `pr-title.yml` and needs no copy in `AGENTS.md`. The new text says a footer "never reaches `main` under the `BLANK` squash message", which is accurate and names the setting.

### D3: The commit-body rule in the apply guidance stays, with a new reason

`openspec/config.yaml:225` forbids a commit body line starting with `word(` "because the squash body reaches release-please". Under `BLANK` that reason is false, but the rule is still worth keeping: until Phase 0 the squash body is the branch commit messages, and a body line such as `fix(cli): ...` is then parsed by release-please as a separate commit. The new reason: "because a squash message other than `BLANK` copies commit bodies into `main`, where release-please parses such a line as a commit". The rule text itself is unchanged, so agents behave the same.

### D4: No gate for a stale `release-as` key

A `release-as` key that stays after its release pins every later release to that version; release-please then proposes the same version again. A G1-style check could refuse a release PR whose version equals the key while the tag already exists. It is not added here:

- the mechanism has not been used under the new rule yet, and a stale key shows itself on the next release PR, which proposes the version already released; every release PR is merged by a human (owner selection 7), who sees it. What release-please and the release workflow do if such a PR merged anyway is not verified here;
- adding a gate changes `release-gates` and a script, which this docs change does not otherwise touch.

The requirement makes the removal PR an explicit step. A gate can follow as its own change if the owner wants one (Open Questions).

### D5: Where a migration note goes

The old text made the `BREAKING CHANGE:` footer "the migration note the CHANGELOG shows". Under `BLANK` the CHANGELOG entry is the PR title alone. The new text says the migration note goes in the PR body, and in the user docs (`docs/site/`) when users need it to upgrade. This is a statement of fact under the owner's setting, not a new policy; whether the CHANGELOG should still carry migration notes (for example by a human editing the release PR's `CHANGELOG.md`) is an owner question (Open Questions).

## Research & Decisions

### Which files carry the old rule

**Context**: The change must catch every stale passage and nothing else.
**Explored**: `grep -rn -i -E "release-as|BREAKING CHANGE|footer|COMMIT_MESSAGES|squash"` over `*.md`, `*.yml`, `*.yaml`, `*.json`, `*.sh`, `*.go`, excluding archived changes and `CHANGELOG.md`.
**Options considered**:
1. Fix only `AGENTS.md` and the spec - leaves `CONSTITUTION.md` and its `openspec/config.yaml` copy teaching the footer.
2. Fix every hit that states the rule - five files, all text.
**Decision**: Option 2. The hits are `AGENTS.md:144-147,357,358`, `CONSTITUTION.md:106-107`, `openspec/config.yaml:53-54,225`, `openspec/specs/release-workflow/spec.md:10,20-26`. `.github/workflows/pr-title.yml:3-13` is already correct. The remaining hits (`CONSTITUTION.md:102`, `AGENTS.md:18` "Generated-with footers", `QUICKSTART.md:350`, `adr/004`, `docs/roadmap.md:147`, `.github/labels.yml:136`) are unrelated uses of "footer" or "breaking change".
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
- [Enhancement 0021 policy text still says the opposite] → flagged to the supervisor; the cli text cites RELEASING.md, which carries the owner decision.
- [Migration notes leave the CHANGELOG] → D5; owner question below.

## Open Questions

- Enhancement 0021 (`enhancements/0021/policy/01-core-schema.md:8,22,70,74`) still requires a `BREAKING CHANGE:` footer and forbids a `release-as` key, including for a post-GA major crossing ("a `Release-As:` footer in the same commit"). Who updates it, and should the crossing procedure become "the `module:` line edit and a `release-as` key in one PR"? (supervisor or owner)
- Should a breaking change's migration note still reach the CHANGELOG, for example by editing the release PR's `CHANGELOG.md` before merging it? (owner; D5)
- Should a gate refuse a release PR while a `release-as` key names an already-tagged version? (owner; D4)
