## Why

The official templates are meant to be "vetted, gated, published" modules that fail the cli release instead of failing the user (0011:D25). They are not vetted. `.github/scripts/publish-templates.sh` runs three checks before a push: `opm module tidy --check`, the `opm module publish --dry-run` gates (identity, coordinates, namespace, kernel loader shape), and the already-published filter (`publish-templates.sh:48-83`). None of them renders the template. So `opmodel.dev/templates/advanced` v1.0.2 shipped, first in release v1.0.0-alpha.22, with debugValues that do not render: its worker and cache components leave the update-strategy `type` an unresolved disjunction. The published artifact still fails today:

```
$ opm module vet .          # GHCR opmodel.dev/templates/advanced v1.0.2, unpacked
unresolved disjunction "RollingUpdate" | "Recreate" | "OnDelete" (type string)
  values.#UpdateStrategySchema
    > update_strategy.cue:37:5
exit 2
```

and the same tree passes every current gate (`publish --dry-run` prints `GO`, `tidy --check` passes). 1.0.3 fixed the content by hand (commit 2e90c24); nothing stops the next one. The archived change `adopt-beta-release-line` recorded this as a follow-up (its design.md, Risks).

The same change recorded a second gap: the PR dry-run accepts "already published" as the only refusal, so a template whose files change while its identity `Version` stays put passes the PR, merges, and is never published (the release filter skips the held version). Three `fix(deps)` commits (e6d9f21, 5d372c0, d5b5c01) re-pinned the templates at 1.0.2 this way, and GHCR v1.0.2 still pins core v2.0.0-alpha.10 and catalogs/opm v4.4.0 while `main` moved on. Today the rule exists only as prose in `AGENTS.md`. Both gaps break the same invariant: what `opm module init` fetches equals the template tree on `main`, and that tree works. This change closes both.

## What Changes

- **Vet gate.** `publish-templates.sh` runs `opm module vet` on every template tree, in both the PR dry-run and the release run. `opm module vet` validates `debugValues` against `#config` and then renders the module against a platform generated from the template's own pins, as `opm module build` does (`internal/cmd/module/vet.go:40-45`, spec `mod-vet`). It never reads a cluster.
- **Gate everything before publishing anything.** The script gains a gate phase over every template (tidy, vet, publish dry-run), and only when every template passes does the release run push the templates whose versions GHCR lacks. Today the script publishes template by template, so a later template's failure can leave an earlier one published.
- **Changed implies bumped (PR only).** With `BASE_REF` set, a template whose directory differs from the merge-base of `BASE_REF` and `HEAD` and whose declared version GHCR already holds fails the PR, naming `opm module version set`. This is the rule `hack/fixtures.sh check` already enforces for fixtures. The `template-gates` job checks out with `fetch-depth: 0` and sets `BASE_REF`.
- `AGENTS.md` (the `templates/` line) and the workflow comments say the PR gate enforces the bump.

SemVer class: none. Every commit is `ci` or `chore`; no Go code, command, flag, output or exit code changes, and no release is cut by this change. PR title: `ci(templates): gate template publishing on opm module vet`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `pr-workflow`: the `template-gates` job also vets every template and refuses a changed template whose version is already published.
- `release-workflow`: the `publish-templates` job vets every template and runs every gate on every template before publishing any.
- `template-modules`: a template SHALL render its own `debugValues`, and its published content SHALL equal its tree at that version.

## Impact

- Files: `.github/scripts/publish-templates.sh`, `.github/workflows/pr.yml` (`template-gates` job), `.github/workflows/release.yml` (comment only), `AGENTS.md` (one line), three spec deltas.
- Commands and packages: none.
- Workflow: any PR that edits `templates/*` must bump each touched template with `opm module version set`, comment-only edits included. The workspace `task deps:update` re-pins templates without bumping them (root `Taskfile.yml` `deps:update:templates`), so its cli PR now fails `template-gates` until the bump is added. Making that task bump the version too is a workspace follow-up, not part of this change.
- A cli PR that bumps the library can now fail `template-gates` if the new kernel no longer renders an unchanged, published template. That is intended: `opm module init` from that cli would scaffold a broken module.
- Runtime: three extra renders, about 1 to 2 s each (measured locally).
