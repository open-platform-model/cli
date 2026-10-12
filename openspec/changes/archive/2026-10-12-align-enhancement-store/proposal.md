# Proposal

## Why

The enhancements move to an OpenSpec store (`opm-enhancements`). Decision numbers do not
survive that move: an entry is renumbered, split or retired, and a citation by decision number
then points at nothing. Every citation in cli becomes an enhancement id beside a reason in words.
No citation is a decision number, and none is a lone id. The old `enhancement.yaml` link between
a change and an enhancement is replaced by one `Delivers` line in the proposal.

## What Changes

- Rewrite every enhancement citation in comments, test messages, docs, live specs and archived
  changes by `enh cite-rewrite`, then clear its worklist by hand. A decision number of an
  enhancement in archived text becomes the enhancement id in words; a decision number of the
  archived change's own design stays. The repo is clean under `enh cite-lint`.
- Delete the 21 archived `enhancement.yaml` files.
- `openspec/config.yaml`: add `references:` (the `opm-enhancements` store), the `Delivers` rule
  for `rules.proposal`, a `context` line to read an entry, and the new comment rule. Remove the
  `NNNN:Dn` bullets and every `enhancement.yaml` and `task enhancements:delivery:*` rule.
- `AGENTS.md` says the same.
- Behaviour does not change: comments, rule text, archived text and test messages only. No help
  text or error text changes.
- Exclusions: `tests/fixtures/` is published to GHCR, and an edit forces a fixture version bump and
  a republish. `CHANGELOG.md` is release-please output. `.claude/**` belongs to the owner's
  `openspec update`. `hack/fixtures.sh` stays byte-identical to the opm-operator copy.

Release class: PATCH after GA, and no release at all, because every commit is `docs:`, `test:`
or `chore:`. Beta ships nothing for it.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
None. Live specs change in wording only (ids and decision numbers become words). No requirement
changes, so the change carries no spec delta and `.openspec.yaml` sets `skip_specs: true`. The
one requirement heading that carried a citation, in `kernel-render`, drops it and keeps its name
without the suffix; the citation moves into the requirement body.

## Enhancement

None.

## Impact

- Packages: comments in about 40 packages under `internal/`, `pkg/`, `hack/`, `tests/`.
- Tests: seven assertion messages carry a citation and change with their tests.
- Docs: `AGENTS.md`, `README.md`, four ADR banners, `docs/site/reference/registry-namespaces.md`
  (published text; the meaning stays).
- Out: `tests/fixtures/**`, `CHANGELOG.md`, `.claude/**`. The two skill patch blocks in `.claude/skills/`
  still ask for `enhancement.yaml`; the owner removes them with `openspec update`.
