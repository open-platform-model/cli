## Revision

2026-10-01, after review: changed-implies-bumped compares against the previous release tag in both the PR check and the release run (cli `main` takes direct pushes, and the three commits behind stale v1.0.2 were direct pushes); the release run fails before any publish on a changed, already-published template; the e2e scaffold test is extended to all three templates (section 3); `AGENTS.md` lines 128 and 351 change together. design.md Revision has the full list.

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

The same change recorded a second gap: nothing checks that a changed template carries a new version. The PR dry-run accepts "already published" as the only refusal, and the release filter skips any version GHCR holds, so a template whose files change while its identity `Version` stays put lands on `main` and is never published. Three `fix(deps)` commits (e6d9f21, 5d372c0, d5b5c01), pushed straight to `main` without a PR, re-pinned the templates at 1.0.2 this way, and GHCR v1.0.2 still pins core v2.0.0-alpha.10 and catalogs/opm v4.4.0 while `main` moved on. Today the rule exists only as prose in `AGENTS.md`. Both gaps break the same invariant: what `opm module init` fetches equals the template tree on `main`, and that tree works. This change closes both.

## What Changes

- **Vet gate.** `publish-templates.sh` runs `opm module vet` on every template tree, in both the PR dry-run and the release run. `opm module vet` validates `debugValues` against `#config` and then renders the module against a platform generated from the template's own pins, as `opm module build` does (`internal/cmd/module/vet.go:40-45`, spec `mod-vet`). It never reads a cluster.
- **Gate everything before publishing anything.** The script gains a gate phase over every template (tidy, vet, publish dry-run), and only when every template passes does the release run push the templates whose versions GHCR lacks. Today the script publishes template by template, so a later template's failure can leave an earlier one published.
- **Changed implies bumped, against the previous release tag.** In both modes, a template whose directory differs from the previous release tag (the newest `v*` tag reachable from `HEAD^`) and whose declared version GHCR already holds fails, naming `opm module version set`. In the release run this fails before anything is pushed; the one exception is a re-run of a release that raised the template's version and already pushed it. Comparing against the tag, not a merge-base, catches changes that reached `main` without a PR: the next PR's check and the release run both fail. This is the rule `hack/fixtures.sh check` enforces for fixtures, with a release tag as the base. `template-gates` and `publish-templates` both check out with `fetch-depth: 0`.
- **Scaffold test over every template.** `tests/e2e/mod_init_test.go` `TestE2E_ModInit_ThenVet` today publishes, scaffolds and vets only `standard`; it becomes a subtest per template, so the re-identified scaffold of each template must vet too.
- `AGENTS.md` (the `templates/` entry, line 128, and the template rule, line 351) and the workflow comments state the bump rule and where it is enforced.

SemVer class: none. Every commit is `ci` or `test`; no product Go code, command, flag, output or exit code changes, and no release is cut by this change (it would be none after GA too). PR title: `ci(templates): gate template publishing on opm module vet`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `pr-workflow`: the `template-gates` job checks out full history, vets every template and refuses a template changed since the previous release tag whose version is already published.
- `release-workflow`: the `publish-templates` job checks out full history, vets every template, refuses a template changed since the previous release at an already-published version, and runs every gate on every template before publishing any.
- `template-modules`: a template SHALL render its own `debugValues`, and its published content SHALL equal its tree at that version.

## Impact

- Files: `.github/scripts/publish-templates.sh`, `.github/workflows/pr.yml` (`template-gates` job), `.github/workflows/release.yml` (`publish-templates` checkout depth and comment), `tests/e2e/mod_init_test.go`, `AGENTS.md` (two lines), three spec deltas.
- Commands and packages: none.
- Workflow: any change to `templates/*`, by PR or direct push, must bump each touched template with `opm module version set`, comment-only edits included. A change that does not turns every following PR's `template-gates` red and fails the release run before any publish. The workspace `task deps:update` re-pins templates without bumping them (root `Taskfile.yml` `deps:update:templates`), so its cli PR fails `template-gates` until the bump is added. design.md R5 describes the workspace fix (bump the patch when a pin moved and GHCR holds the current version) and recommends landing it with the workspace fixtures PR, before or with this change.
- Owner setting, not part of this change: `Template Publish Gates (dry-run)` is advisory today, because `main` requires no status check. Making it required is what turns a red release PR into a blocked release (design.md Risks).
- A cli PR that bumps the library can now fail `template-gates` if the new kernel no longer renders an unchanged, published template. That is intended: `opm module init` from that cli would scaffold a broken module.
- Runtime: three extra renders in each script run, about 1 to 2 s each, and two more e2e subtests, about 4 s each (measured locally).
