# Proposal

## Why

The enhancements move to an OpenSpec store (`opm-enhancements`). Decision numbers do not
survive that move: an entry is renumbered, split or retired, and a citation such as `0011:D9`
then points at nothing. Every citation in cli becomes an enhancement id beside a reason in words.
No citation is a decision number, and none is a lone id. The old `enhancement.yaml` link between
a change and an enhancement is replaced by one `Delivers` line in the proposal.

## What Changes

- Rewrite every enhancement citation in comments, test messages, help text, docs, live specs and
  archived changes by `enh cite-rewrite`, then clear its worklist by hand. The repo is clean
  under `enh cite-lint`.
- Delete the 21 archived `enhancement.yaml` files.
- `openspec/config.yaml`: add `references:` (the `opm-enhancements` store), the `Delivers` rule
  for `rules.proposal`, a `context` line to read an entry, and the new comment rule. Remove the
  `NNNN:Dn` bullets and every `enhancement.yaml` and `task enhancements:delivery:*` rule.
- `AGENTS.md` says the same.
- Behaviour does not change: comments, rule text, archived text, test messages and generated
  descriptions only. Published fixtures under `tests/fixtures/` are not edited (an edit forces a
  fixture version bump and a republish).

Release class: PATCH after GA, and no release at all, because every commit is `docs:`, `test:`
or `chore:`. Beta ships nothing for it.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
None. Live specs change in wording only (ids and decision numbers become words). No requirement
changes, so the change carries no spec delta and `.openspec.yaml` sets `skip_specs: true`.

## Enhancement

None.

## Impact

- Packages: comments in about 40 packages under `internal/`, `pkg/`, `hack/`, `tests/`.
- Tests: seven assertion messages carry a citation and change with their tests.
- Docs: `AGENTS.md`, `README.md`, four ADR banners, `docs/site/reference/registry-namespaces.md`
  (published text; the meaning stays).
- Out: `tests/fixtures/**`, `CHANGELOG.md`, `.claude/**`. The skill patch blocks in `.claude/skills/`
  are listed in the report for the owner's `openspec update`.
