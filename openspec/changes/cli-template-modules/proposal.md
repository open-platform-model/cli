# Proposal — cli-template-modules

> Extends 0011's `cli-authoring-commands` slice (the `mod init` half — the slice lands as two OpenSpec changes; `openspec_ref` will cite both). Depends on `cli-authoring-commands` (the cueedit writers) and on the templates-segment decision in enhancement 0011 (prerequisite, drafted alongside the D9 amendment).

## Why

The builtin templates rotted two schema generations because nothing evaluated them — text-with-placeholders embedded in the binary, exercised only by file-existence tests. Rebuilding them as `.tmpl` files (the previously drafted plan) fixes the content but not the disease: the next schema crossing rots them again, silently.

This change removes the disease's habitat. Templates become **real CUE modules** hosted in the cli repo, published to a reserved registry segment (`opmodel.dev/templates/<name>`) by the cli's own release CI **through `opm module publish`** — dogfooding the pipeline, which means a template that violates any gate fails the cli's release. Rot becomes structurally impossible: templates are vetted, gated, published artifacts whose dependency pins the normal module tooling maintains.

`opm mod init` becomes fetch-and-re-identify: pull the template module, copy its staged source, rewrite its identity to the user's path. The re-identification set is exactly D16's three statements (`cue.mod` `module:` line, identity package, literal self-imports) plus the package-clause rename — the writers `cli-authoring-commands` builds, applied wholesale in the one context where rewriting self-imports is correct (the user owns nothing in the tree yet). Official templates get docker-official-image-style shortcuts: `opm mod init standard@v1` expands the bare word into the reserved segment — safe precisely because the segment is reserved and gate-curated.

## What Changes

- **`cli/templates/{minimal,standard,advanced}/`** — three real modules on the v2 shape (identity subpackage with defaulted version, D12 derivation wiring, D49 catalog imports, `source: {kind: "self"}`), each passing `opm module vet` and `publish --dry-run` in CI. Their identity is `opmodel.dev/templates/<name>@v1`. The set mirrors the historical variants with `simple` renamed `minimal`.
- **Publish-on-release CI job** using the freshly built `opm`: for each template, resolve the declared version and invoke `opm module publish` only when the tag is unpublished (the caller-side sweep filter D15 prescribes — publish itself never skips).
- **The publish gates learn the segment**: `templates/<name>` joins the namespace/kind tables as a module-kind segment (Go gate + the 0011 `target.cue` pins move with the DN). The name `index` is reserved within the segment (one DN line — future listing insurance, unused today).
- **`opm mod init` rewritten** — one coherent command owning scaffold *and* repair (D20):
  - *Scaffold*: `opm mod init <new-module-path> [template]` — fetch the template (`AcquireModuleFromRegistry`; version = newest release within the major, stable-preferred with prerelease fallback via `compat.HighestStable`, its first true caller), copy the staged tree, re-identify (cue.mod line, identity `ModulePath` + `Version` reset to the defaulted `0.1.0` form, self-import rewrite, package-clause rename to the new snake leaf). Metadata needs nothing — it derives.
  - *Shortcuts*: a bare-word template ref (`^[a-z0-9_]+$` before an optional `@`) expands to `opmodel.dev/templates/<word>`; `@vN` = major float, full semver = exact tag; anything with `/` or `.` is a literal path, never expanded. `--from` is the explicit spelling and accepts any published module (clone-and-re-identify as a first-class operation). `-t/--template` survives as an alias.
  - *Interactive form*: `opm mod init standard@v1` (template only) prompts for the module path — D20's "asks for one"; non-TTY without a path refuses.
  - *Repair* (unchanged from the prior draft, relocated here): detect via the pipeline's checks, second confirmation listing every file and current→replacement pair, `--yes` bypass, never invent identity, refuse stranding self-imports.
- **`opm module template list`** — new `template` subgroup; v1 prints the baked table (name, description, default major) that also drives shortcut expansion — one constant, release-coupled to the published set by construction. Registry-resolved versions: deferred, YAGNI.
- **`internal/templates/` + the embed machinery deleted**; the variant spec requirements get deltas: three variants persist (`minimal` replacing `simple`), sourced as published modules instead of embedded text.
- **Offline stance, stated**: init requires the registry once (anonymous GHCR pull; CUE's disk cache makes repeats warm-offline); no embedded fallback tree — that is the rot vector this change exists to remove. First-run works without config via the built-in default registry mapping.

## Capabilities

### New Capabilities

- `template-modules`: the reserved segment, fetch-based init with shortcut expansion and re-identification, `template list`, the CI publish contract.

### Modified Capabilities

- `core` / `core/project-structure`: the three-fixed-template-variant requirements are replaced by the template-modules contract (deltas REMOVE FR-001's variant enumeration).

## Impact

- **SemVer: MINOR** (init's argument grammar was already changing in the drafted authoring plan; no released consumer exists).
- **Ordering**: after `cli-authoring-commands` (writers) and after the enhancements batch lands the templates DN (the gate-table change here cites it). The CI job needs no cutover — it is born on the pipeline.
- **Packages**: new `cli/templates/` trees; `internal/cueedit` gains the self-import-rewrite and package-rename writers; `internal/cmd/module/init.go` rewritten; `internal/cmd/module/template.go` new; `internal/templates/` deleted; `internal/publish` gate tables extended.
- **Tests**: init e2e becomes registry-backed and *stronger* — publish a template into the hermetic registry, init from it, vet + dry-run the result. `TestE2E_ModInit_ThenVet` (long-skipped) revives here in that form.
- **Record on landing**: 0011 history event citing both changes for the `cli-authoring-commands` slice; the `HighestStable` disposition question (catalog-gates task 6.3) resolves — re-document as the float selector with template resolution as its first caller.
