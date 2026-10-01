## Revision

Revision 3, 2026-10-01, after review 2:

- Changed-implies-bumped compares the tree with the **content GHCR holds** at the template's declared version, not with a git base. Revision 2's previous-release tag stopped seeing an unbumped change once a release landed on top of it.
- `publish-templates` now runs **before** goreleaser builds, so a template failure leaves the release a draft.
- No checkout needs full history any more.

design.md Revisions has the details.

## Why

The official templates are meant to be "vetted, gated, published" modules that fail the cli release instead of failing the user (0011:D25). They are not vetted. `.github/scripts/publish-templates.sh` runs three checks before a push: `opm module tidy --check`, the `opm module publish --dry-run` gates (identity, coordinates, namespace, kernel loader shape), and the already-published filter (`publish-templates.sh:48-83`). None of them renders the template. So `opmodel.dev/templates/advanced` v1.0.2 shipped, first in release v1.0.0-alpha.22, with `debugValues` that do not render: its worker and cache components leave the update-strategy `type` an unresolved disjunction. The published artifact still fails today:

```
$ opm module vet .          # GHCR opmodel.dev/templates/advanced v1.0.2, unpacked
unresolved disjunction "RollingUpdate" | "Recreate" | "OnDelete" (type string)
  values.#UpdateStrategySchema
    > update_strategy.cue:37:5
exit 2
```

The same tree passes every current gate (`publish --dry-run` prints `GO`, `tidy --check` passes). Version 1.0.3 fixed the content by hand (commit 2e90c24), and nothing stops the next one. The archived change `adopt-beta-release-line` recorded this as a follow-up (its design.md, Risks).

The same change recorded a second gap: nothing checks that a changed template carries a new version. The PR dry-run accepts "already published" as the only refusal, and the release filter skips any version GHCR holds. So a template whose files change while its identity `Version` stays put lands on `main` and is never published. Three `fix(deps)` commits (e6d9f21, 5d372c0, d5b5c01), pushed straight to `main` without a PR, re-pinned the templates at 1.0.2 this way. Releases v1.0.0-alpha.25 to alpha.27 shipped those trees, and GHCR v1.0.2 still pins core v2.0.0-alpha.10 and catalogs/opm v4.4.0. Today the rule exists only as prose in `AGENTS.md`.

Both gaps break the same invariant: what `opm module init` fetches equals the template tree on `main`, and that tree works. This change closes both. It also stops a template failure from shipping a cli release without its templates.

## What Changes

- **Vet gate.** `publish-templates.sh` runs `opm module vet` on every template tree, in both the PR dry-run and the release run. `opm module vet` validates `debugValues` against `#config` and then renders the module against a platform generated from the template's own pins, as `opm module build` does (`internal/cmd/module/vet.go:40-45`, spec `mod-vet`). It never reads a cluster.
- **Gate everything before publishing anything.** The script gains a gate phase over every template (tidy, vet, publish dry-run). Only when every template passes does the release run push the templates whose versions GHCR lacks. Today the script publishes template by template, so a later template's failure can leave an earlier one published.
- **Changed implies bumped, by content.** When GHCR already holds a template's declared version, the script fetches that published module zip anonymously and verifies its digest. It builds the tree's module zip with `cue mod publish --out`, which runs the same `modzip.CreateFromDir` that `opm module publish` zips with, and compares the two file by file (paths and bytes, comments included). A difference fails, in both modes, before any publish, and prints the diff and `opm module version set`. Equal content passes, and an unpublished version passes and is published. Any fetch or comparison error fails; it never counts as unpublished. The check reads no git history, so it holds however the change reached `main` and however many releases were cut on top of it.
- **Release order.** In `release.yml`, `goreleaser` needs `publish-templates`. After its unchanged draft check, it fails unless the template job succeeded. With draft-first releases (PR 260), a template failure therefore leaves the release a draft: a transient failure is re-run, and a content failure is fixed by the next patch. Neither publishes binaries without templates.
- **Scaffold test over every template.** `tests/e2e/mod_init_test.go` `TestE2E_ModInit_ThenVet` today publishes, scaffolds and vets only `standard`. It becomes a subtest per template, so the re-identified scaffold of each template must vet too.
- `AGENTS.md` (the `templates/` entry, line 128, and the template rule, line 351) and the workflow comments state the bump rule and where it is enforced.

SemVer class: none. Every commit is `ci` or `test`; no product Go code, command, flag, output or exit code changes, and no release is cut by this change (it would be none after GA too). PR title: `ci(templates): gate template publishing on opm module vet`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `pr-workflow`: the `template-gates` job vets every template and refuses a template whose tree differs from the artifact GHCR holds at its declared version.
- `release-workflow`:
  - The `publish-templates` job vets every template, refuses a template whose tree differs from its published artifact, and runs every gate on every template before publishing any.
  - The `goreleaser` job waits for `publish-templates` and refuses to build when it failed.
- `template-modules`: a template SHALL render its own `debugValues`, and its published content SHALL equal its tree at that version.

## Impact

- Files: `.github/scripts/publish-templates.sh`, `.github/workflows/pr.yml` (`template-gates` comment), `.github/workflows/release.yml` (`goreleaser` needs and one step, header runbook, `publish-templates` comment), `tests/e2e/mod_init_test.go`, `AGENTS.md` (two lines), three spec deltas.
- Commands and packages: none.
- Workflow: any change to `templates/*`, by PR or by direct push, must bump each touched template with `opm module version set`, comment-only edits included. If it does not, every following PR's `template-gates` goes red, and every release fails before any publish and stays a draft, until a bump lands.
  - The workspace `task deps:update` re-pins templates without bumping them (root `Taskfile.yml` `deps:update:templates`), so its cli PR fails `template-gates` until the bump is added.
  - design.md R5 describes the workspace fix: bump the patch when a pin moved and GHCR holds the current version. It recommends landing that with the workspace fixtures PR, before or with this change.
- Owner setting, not part of this change: `Template Publish Gates (dry-run)` is advisory today, because `main` requires no status check. Making it required stops a red release PR from merging. A release-time failure then no longer costs a tag (design.md Risks).
- A cli PR that bumps the library can now fail `template-gates` if the new kernel no longer renders an unchanged, published template. That is intended: `opm module init` from that cli would scaffold a broken module.
- Runtime (measured locally):
  - Each script run adds three renders, about 1 to 2 s each, and three small anonymous GHCR downloads; the whole dry-run takes about 10 s.
  - The e2e test gains two subtests, about 4 s each.
  - Release binaries now wait one to two minutes for the template job.
