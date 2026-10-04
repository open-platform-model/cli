## Why

opmodel.dev builds the cli's pages from this repository's git tree: the authored pages under `docs/site/` and the command reference that `internal/cmdref` (through `hack/cmdref`) generates and commits under `docs/site/reference/cli/`. It also reads the cli's pins from git to choose the library, core and operator docs a site version shows (`opmodel.dev/site/scripts/resolve-versions.sh`). docs-kit phase 2 replaces both: the cli publishes a signed docs bundle on each release, carrying its command reference (docs-kit's `cobra` extractor, contract C19) and, in `manifest.json`, the exact versions it pins (C15), which `opm-docs pull` follows to the other three bundles (docs-kit DESIGN decision 10). The cli's bundle is the anchor of a site version.

The owner decided one cutover per repository (docs-kit DESIGN decision 20): the bundle carries the cli's whole `docs/site/` with the generated reference from adoption on. While the site still reads the cli from git, the committed reference stays and is left out of the bundle with the `markdown` source's `exclude`; once the site reads the bundle, the generator, the pages and the exclude go.

docs-kit reads the command tree through a `hack/` program that imports `github.com/open-platform-model/docs-kit/cobradump`, a nested module depending only on cobra and pflag, so the shipped `opm` binary carries no docs code and the cli's module graph gains one module.

Sequence and contracts: docs-kit `docs/orchestration.md` (phase 2, "cli: `publish-cli-bundle`", and gate G2-pins) and `https://github.com/open-platform-model/docs-kit/blob/main/docs/contracts.md` (C5, C6, C9, C12, C14, C15, C19). Until a docs-kit change lands, its `openspec/changes/<change>/design.md` on docs-kit `main` shows the contract.

Delivery: one PR per section (opmodel.dev needs the docs/cli release bundle after section 2)

## What Changes

- **Section 1, adopt (gate G2-cli).** `hack/docskit-dump/main.go` prints the cobra dump (`cobradump.Write(cmd.NewRootCmd(), ...)`) or, with the argument `pins`, the pins of `library`, `core` and `opm-operator`, read where the site reads them today; a test that those pins equal their sources. `go.mod` requires `github.com/open-platform-model/docs-kit/cobradump` and nothing else new. `docs-kit.cue` declares the project `cli` (owning `reference/cli/`, `pins`, a `cobra` source, a `markdown` source excluding `reference/cli/`). `.opm-docs-version`, `.tasks/opm-docs.sh`, the docs tasks (`docs:bundle:check` in `task check`), `.github/workflows/docs.yml` with `setup-go: true`, and `publish-docs` in `release.yml` after `goreleaser` succeeds. `AGENTS.md` gains a "Docs bundles" paragraph. cmdref and its check stay.
- **Section 2, the first anchor bundle (gate G2-pins, owner).** The owner's path (2026-10-03): the siblings backfill exactly the versions the cli pins today (core `v2.0.0-beta.1`, library `v1.0.0-beta.1`, opm-operator `v1.0.0-beta.4`); then release PR cli#276 (`v1.0.0-beta.6`), held until then, merges after a local pull on its head resolves all three pins; confirm anonymously.
- **Section 3, retire cmdref (gates G2-switch and G2-edge, after section 2 is recorded).** G2-edge (docs-kit orchestration) holds once opmodel.dev's `sources-main` job reads the cli's `main` from the `edge` docs bundle; until then it reads the cli's `main` checkout, where every page linking a deleted command page would fail. Delete `internal/cmdref/`, `hack/cmdref/`, `docs/site/reference/cli/`, the `docs:reference` tasks, the "Command Reference (current)" jobs in `ci.yml` and `pr.yml`, and the `exclude`; archive.

## Capabilities

### New Capabilities

- `docs-bundle`: the cli's docs bundle, the dump and pins program, publishing, and the docs-kit pin.

### Modified Capabilities

- `command-reference`: generated into the docs bundle by docs-kit from the dump; nothing is committed; cmdref and its CI jobs are gone.

## Impact

**SemVer: none.** No command, flag, help text or output changes; the `opm` binary is unchanged (the dump program is under `hack/`, not linked into `cmd/opm`). Commits are `ci:`, `docs:` and `test:`; release-please releases none of them. Principle VII: the change replaces about 1,300 lines of generator and tests (`internal/cmdref`, `hack/cmdref`) with a program of about 60 lines and a test.

**Affected:** `hack/docskit-dump/` (new), `hack/cmdref/`, `internal/cmdref/`, `docs/site/reference/cli/`, `go.mod`/`go.sum`, `Taskfile.yml`, `.github/workflows/{ci,pr,release,docs}.yml`, `AGENTS.md`. No command package.

**Downstream consumers:**

| Consumer | What it has to do |
| --- | --- |
| opmodel.dev | `pull-reference-bundles` makes the cli bundle the v1.0 anchor once G2-pins holds; its merge is gate G2-switch. G2-edge holds once `add-edge-build` switches `sources-main` to edge bundles. cmdref's retirement waits for both. After it, opmodel.dev's local explicit-mode builds no longer resolve links into the command reference; `task build:edge`, or `OPM_DOCS_BUNDLES=1` after a pull, is the local check (`add-edge-build` documents it). |
| library, core, opm-operator | Each backfills the version the cli pins (their own sibling changes) before the cli's release; no pin moves. |

**Owner items:** merging docs-kit's release PRs and the `cobradump/v0.1.0` release (gate G2-cli); holding release PR cli#276 until adoption merges and G2-pins holds; the three sibling backfills; the cli release; checking `ghcr.io/open-platform-model/docs/cli` is public on first push; nothing for "Command Reference (current)": no ruleset requires it (design.md D6, owner-confirmed 2026-10-04).
