# Tasks — cli-publish-pipeline

> Grouped by package, tiny chunks, every group independently green. The pipeline package builds bottom-up (states → gates → plan → push) so each layer is unit-tested before the commands exist. Commit messages: glue majors to paths (bare-`@` ban).

## 1. `internal/cueedit` — the D8 writer (shared with cli-authoring-commands later)

- [x] 1.1 `SetIdentityVersion(dir, version string) error`: locate `Version` in `identity/identity.cue` by the schema-fixed path, surgical AST rewrite preserving comments/field order, rebuild the file; refuse (typed error) when the file does not match `#IdentityPackage`'s shape.
- [x] 1.2 Table tests: literal value, defaulted value (`#VersionType | *"x"` — rewrite the default), open field (insert), absent file (typed refusal), comment/alignment preservation golden.

## 2. `internal/publish` — states and identity

- [x] 2.1 Loader: root package + `./identity` subpackage via `load.Instances` with the resolved registry (no `os.Setenv`); load failure returns the CUE error verbatim; `cue.Value`s retained for `Pos()`.
- [x] 2.2 Identity unification against `core.#IdentityPackage` from the kernel schema cache; CUE error surfaced through the grouped funnel (refusal 10). Test: conformant, renamed field, missing required field, absent package.
- [x] 2.3 `IdentityState` tristate with defaults-are-concrete semantics; effective-version resolution incl. `--version` assert/fill (fill calls `cueedit`); one-cause-one-refusal on the absent path. Tests per design's state matrix.

## 3. `internal/publish` — gates and plan

- [x] 3.1 Derivation gates: `metadata` ↔ `id` agreement (msg 7), `cue.mod` `module:` ↔ declared path (msg 2), no-`cue.mod` (msg 3), `source: {kind:"self"}` preflight (new msg).
- [x] 3.2 Coordinate derivation (split declared path), tag construction, tag/version + tag-major/path-major gate (msg 6), namespace/kind gates (owned domains only — pin the four passing cases from `schemas/target.cue`: vanity, testing, community, core).
- [x] 3.3 Module package-name gate (msg 8 per design's drafted message); catalog entry point skips it.
- [x] 3.4 Override gate: file-presence detection (reuse `pkg/loader` provenance helpers), per-replacement report lines with superseding registry versions, msgs 4a/4b, `--skip-override-check` waives the gate for modules only and changes nothing else. Tests: all four presence/flag states × both kinds + published-deps resolution assertion.
- [x] 3.5 Refusal accumulation (all evaluable gates run; load failure short-circuits) and plan rendering per experiment 02's format; aligned two-value column helper added to `internal/output` with tests.

## 4. `internal/publish` — registry I/O

- [x] 4.1 Already-published lookup via `modregistry.Client.ModuleVersions` (msg 8 of the catalog — refusal 8); unreachable registry → connectivity error (exit 3), distinct from refusal. In-process `modregistrytest` tests (immutable tags).
- [x] 4.2 Push: `modzip.CreateFromDir` → `PutModule` through `modconfig.NewResolver(&Config{CUERegistry: cfg.Registry})`; round-trip test (publish, then resolve the module back and compare identity); authenticated-push test via `modregistrytest.NewServer` + `AuthConfig`.

## 5. Commands

- [x] 5.1 `internal/cmd/catalog/catalog.go` (new group, registered in root.go) + `publish.go`; `internal/cmd/module/publish.go`. Thin: flags (`--version`, `--dry-run`, module-only `--skip-override-check`), delegate to `internal/publish`, exit-code mapping. Constructor tests per house pattern.
- [x] 5.2 E2E: refusal-shape assertions modeled on `vet_output_test.go` (msg 2's aligned columns, msg 4b's flag mention, dry-run GO exit 0 / REFUSED exit 2), against fixture trees.

## 6. `opm module vet` (D16/D18/D21)

- [x] 6.1 Thread `cfg` into vet; loads use the resolved registry (constructor + `runVet` signature; existing tests updated).
- [x] 6.2 Insert the three checks between module load and the values stanza, rendering as `FormatVetCheck` lines; failures through `PrintValidationError` + exit 2. Tests: coordinate drift, derivation drift, non-conformant identity, all on a module with no `debugValues`.

## 7. Specs, docs, validation gates

- [x] 7.1 Spec deltas: new `artifact-publishing` capability; `mod-vet` delta.
- [x] 7.2 `CLAUDE.md` package map + command groups updated (`internal/publish`, `internal/cueedit`, `internal/cmd/catalog/`); `docs/roadmap.md`'s stale `distribution-v1`/oras-go publish paragraph corrected to this pipeline.
- [x] 7.3 `task fmt`, `task vet`, `task lint`, `task test` — all green.

## 8. Record

- [x] 8.1 `enhancements/0011/`: slice → `done` + `openspec_ref` + history event, recording the three deviations (package-name refusal message drafted CLI-side; tidiness gate not reproduced — decode covers the load-failure class; `internal/cueedit` writer landed here for `cli-authoring-commands` to reuse) and the D4 evidence correction (catalog-vs-module `cue vet` asymmetry) so 03-decisions.md's stale sentence is not propagated.
