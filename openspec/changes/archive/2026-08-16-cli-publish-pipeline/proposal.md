# Proposal — cli-publish-pipeline

> Slice `cli-publish-pipeline` of enhancement `0011` (Module and Catalog Publishing). Decisions D1, D2, D4, D6, D12, D15, D16, D18, D21. Depends on `0011:core-identity-package` (done — `#IdentityPackage` ships in core v2) and `0010:cli-coordinate-adoption` (done — the CLI builds on the core-v2 library line).

## Why

Nothing today can publish an OPM artifact from its committed source. The catalog releases through a copy-and-stamp task (rsync `src/` to a build dir, write a `version_override.cue` into the copy, publish the copy — the committed tree and the published bytes are different trees), and the module fleet published through a checksum ledger that never read or wrote the module's own version — measured: `jellyfin` `v2.0.1` and `v2.0.2` both shipped metadata claiming `2.0.0`, giving three non-authoritative answers to "what version is this", and a CLI-deployed instance recorded a version the operator then resolved to a *different artifact*, reconciling it green forever.

Enhancement 0011's answer is one publish pipeline with two entry points — `opm module publish` and `opm catalog publish` — that derives coordinates from the artifact's identity instead of rewriting the artifact to match a coordinate: decode, read `identity/identity.cue`, unify it against core's shipped `#IdentityPackage` (CUE produces the diagnostic — refusal 10), derive path/repo/major/tag by reading and splitting `metadata.modulePath`, run the gates, push. What is published is what is committed (D2). Both CI cutover slices (`catalogs-publish-cutover`, `modules-publish-cutover`) and 0010's two republish migrations wait behind this slice; `cli-catalog-gates` builds its compatibility and member gates on this pipeline.

This slice also lands a check that has never existed anywhere: the version-major/path-major agreement (`#IdentityPackage.VersionMajor`) — core deleted its own assertions of that relation citing publish-side validation (0010 D43/D45), so until this pipeline runs, the relation is enforced by nothing.

## What Changes

- **New command group `opm catalog`** (first in the repo) with `opm catalog publish`; **new `opm module publish`**. Both are thin cobra wrappers over one new `internal/publish` package (Constitution II: commands orchestrate).
- **The pipeline**: load the artifact tree and its `identity/` subpackage (a tree that does not load surfaces the load error — never "identity absent"); unify the identity package against `core.#IdentityPackage` from the kernel's schema cache; verify `metadata.{modulePath,version}` derive from `id.{ModulePath,Version}` (D12); check `cue.mod`'s `module:` line against the declared path (D16); derive `registryRepo`/`major`/`tag` by splitting the declared path; enforce tag == effective declared version (D18) and the module package-name rule (D1); detect `cue.mod/local-module.cue` (D6 — modules may waive the *gate* with `--skip-override-check`, catalogs never; replacements are ignored either way); refuse an already-published tag via a registry lookup (D15 — no skip mode exists); push via CUE's own machinery.
- **Refusals 1–8 and 10** exactly as drafted in 0011 `06-operational.md`: fixed condition → evidence → consequence → action shape, aligned two-value columns naming the file that declares each value, actions as runnable commands, `file:line` positions carried through decode. All evaluable gates run and refusals accumulate into one list (one pass fixes everything); one cause yields one refusal.
- **Plan output = the dry run** (D16: no separate `--check`): every publish prints the resolved plan (kind, declared path, cue.mod path, repo, major, tag and its source, per-field identity state, override gate) before pushing; `--dry-run` stops after the plan. The rendering follows 0011 experiment 02's measured format.
- **`--version` on both commands**: fills an open identity field or asserts a concrete one, never overwrites (refusal 5). The fill writes the working tree (D12) through a new minimal `internal/cueedit` writer implementing D8's schema-fixed-path surgical rewrite — the mechanism `cli-authoring-commands` will reuse for `version set` (implementation here, command surface there; same pattern as the library's `StripProvenance` precedent).
- **`opm module vet` gains the D16 and D18 checks** (coordinate agreement, tag/version agreement, identity conformance) before its values stanza — which requires finally threading `cfg` into vet so its loads respect the resolved registry.
- **Exit codes**: any refusal exits `ExitValidationError` (2, per the status-exit-codes precedent); registry unreachability exits `ExitConnectivityError` (3); success 0. No specific code is promised to sweeps (0011's constraint) — the CLI publishes one artifact per invocation.
- **Deliberately out of scope**: refusals 9 and 11 plus `opm catalog registry check` (`cli-catalog-gates`); `version set` / `mod init` commands (`cli-authoring-commands`); `opm login` (`cli-login` — the push already reads docker config + CUE's `logins.json` through `modconfig`'s shared transport, so CI publishes work today with `docker login`); the CI cutovers; a machine-readable plan format (future, YAGNI).

## Capabilities

### New Capabilities

- `artifact-publishing`: the identity-driven publish pipeline — gates, refusal catalog, plan/dry-run, push, exit codes.

### Modified Capabilities

- `mod-vet`: vet additionally reports coordinate disagreement (D16), tag/version skew (D18), and identity-package non-conformance (D21) before values validation, and becomes registry-aware (loads with the resolved registry).

## Impact

- **SemVer: MINOR** — new commands and a new command group; `opm module vet` gains checks (stricter output for defective modules is the feature, not a break; its flag surface is unchanged).
- **Commands**: new `internal/cmd/catalog/` group + `publish`; new `internal/cmd/module/publish.go`; `internal/cmd/module/vet.go` modified (cfg threading + three checks).
- **Packages**: new `internal/publish` (pipeline, gates, plan rendering), new `internal/cueedit` (the D8 writer), `internal/cmdutil`/`internal/output` additions (aligned two-column disagreement rendering — no helper exists today).
- **New flags** (all defaulted off): `--dry-run` (repo-uniform spelling), `--version <semver>`, `--skip-override-check` (module publish only — D6's name; it waives the gate, never enables overrides). Complexity justified: each flag is named by an accepted decision's refusal action.
- **Dependencies**: `cuelabs.dev/go/oci/ociregistry` moves indirect → direct (in-process registry tests via the public `modregistrytest`, which supports immutable tags and auth — D15/D10-shaped tests run hermetically).
- **Consumers**: `cli-catalog-gates` extends the catalog entry point; both cutover slices point CI at these commands; `cli-authoring-commands` reuses `internal/cueedit`.
- **Known deviations to record in 0011 on landing**: the module package-name rule gets a drafted refusal message here (none exists in 06-operational); the tidiness gate (`cue mod publish`'s `modload.CheckTidy`) is not reproduced — it is internal to the CUE module and the pipeline's full decode already refuses unresolvable dependency trees (the publishable-artifact failure class); extraneous-dep hygiene stays with `cue mod tidy` authoring flow.
