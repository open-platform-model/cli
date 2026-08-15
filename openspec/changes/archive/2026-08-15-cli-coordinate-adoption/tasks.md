# Tasks — cli-coordinate-adoption

> Staged so every group before and after the atomic crossing (group 3) is independently green. Group 3 is atomic: the library bump, the wire-shape move, and the fixture crossing must land together — the alpha.13 compile break and the v2 schema both force it. Group 8 is **blocked on the operator `v1.0.0-alpha.9` release** and is the only part that waits on anything. Breaking change: `feat!` with the platform-file format change named in the commit body.

## 1. Pre-steps green on alpha.9 (no library bump yet)

- [x] 1.1 `pkg/module`: rewrite `CanonicalModuleRef` to `(m.ModulePath, ensureVPrefix(m.Version))`; delete `majorVersionTag` + snake fallback; drop `NameSnakeCase`/`DefaultNamespace` fields; rewrite the unit table (verbatim path; bare→prefixed; empty-version cases). NOTE: callers unchanged; `internal/inventory/gates.go`'s private `ensureVPrefix` untouched.
- [x] 1.2 Digest peer pin: unit test on `internal/workflow/apply` `sourceDigest` golden (`sha256("path@version")` for a fixed pair), comment naming `opm-operator/internal/status.ModuleSourceDigest` as the byte-identical peer.
- [x] 1.3 D19 warning: instance path — `output.Warn` when `SourceLocal` (render.go, boolean already computed); module path — `HasLocalModuleReplacement(loader.ModuleRootFrom(absPath))` in `FromModule`. One warning string, tested on both entries (present with a replaceWith, absent without).
- [x] 1.4 Riders: `version.CUESDKVersion` → `v0.17.1` (+ test update); delete `testing/test/`.
- [x] 1.5 `task check:fast` green.

## 2. Fixture reauthoring staged (still v1-pinned, still green)

- [x] 2.1 Reauthor the three `tests/fixtures/valid/*` modules to snake identity in place where v1 tolerates it (package renames, name/leaf agreement) — anything v1 rejects waits for group 3. Prepare-but-don't-switch is acceptable; keep the suite green.

## 3. The atomic crossing (one PR-sized commit series, merged together)

- [x] 3.1 `go.mod`: library → `v1.0.0-alpha.13`; `go mod tidy`.
- [x] 3.2 `internal/platform/spec.go`: delete `wireFilter`; `wireSubscription` gains `Version`; `toInput`/`wireFromInput` straight-copy. `cluster_test.go` `testInput()` moves to `Version`.
- [x] 3.3 `internal/config/schema/platform.cue`: `#SubscriptionFilter` deleted; `version!: string`; registry keys constrained to `=~"@v[0-9]+$"` paths.
- [x] 3.4 `internal/config/templates.go`: platform template → single subscription `"opmodel.dev/catalogs/opm@v2": {version: "2.0.0-alpha.3"}`; doc comments rewritten (pin is load-bearing; peers: `hack/platform.cue`, operator sample). `hack/platform.cue` same commit.
- [x] 3.5 CR seam: `DecodeCRSpec` ignores `filter`, passes empty version through; materialize call site wraps `ErrSubscriptionMissingVersion` with the legacy-CR hint. `EnsureClusterPlatform` refuses writes while the vendored CRD lacks `version` (checked against embedded `install.yaml`), error naming `operator-library-retarget`.
- [x] 3.6 Fixture crossing, all 17 files: cue.mod pins (core `v2.0.0-alpha.4`, catalogs `v2.0.0-alpha.3`), imports (`core@v2`, D49 `…/v1beta1` packages), metadata to v2 identity (snake name == leaf, full `@vN` modulePath). Registry-pulling pins → `podinfo v0.1.4` (`render-parity`, e2e handoff cue.mod, `ssa-ownership` string, `examples/`); `examples/` podinfo-only (hello-web instance removed, README note pointing at `hello_web` for later).
- [x] 3.7 Test updates: `spec_test.go` (filter tests → version tests incl. empty-version refusal via synth), `platform_test.go` (new invalid-shape subject), `config/init_test.go` (single subscription, `version:` literal), `vet_test.go` comments/skip text, `platform-materialize/main.go` comments.
- [x] 3.8 `task check` green; `test:integration` suites that run against GHCR green; pulling tests confirmed still gated (no CI change).

## 4. New-error surfacing

- [x] 4.1 `cmdutil.PrintValidationError`: `errors.As` branch for `*oerrors.UnresolvedDemandsError` — per-demand line (component, key, kind) + alternatives clause; test with a constructed aggregate.
- [x] 4.2 `handoff/verify.go`: `IdentityError` branch — "declared X, fetched by Y" message; not-found keeps the publish hint; test both.

## 5. Spec deltas

- [x] 5.1 Write the five deltas (`platform-resolution`, `config-commands`, `instance-building`, `instance-inventory`, `kernel-render`) — see `specs/`.

## 6. Docs

- [x] 6.1 `cli/CLAUDE.md` env-notes: v2 line references; `README` command examples if they show platform files.

## 7. Record

- [x] 7.1 `enhancements/0010/`: slice → `done`, `openspec_ref: "cli/cli-coordinate-adoption"`, history event; record the declared-coordinate deviation (02-design:213 reinterpretation) and the D19 predicate refinement.
- [x] 7.2 `enhancements/0011/plan.yaml`: add the `cli-publish-pipeline → 0010:cli-coordinate-adoption` compile-dependency edge (with the enhancement-slicing skill loaded).

## 8. Blocked on operator release `v1.0.0-alpha.9` (final tail)

- [x] 8.1 `task operator:sync` + `PinnedOperatorVersion` bump; delete the `EnsureClusterPlatform` refusal + its CRD probe.
- [x] 8.2 `hack/kind-platform.yaml` → v2 shape (`@v2` key, `version:`), same pin as the template.
- [x] 8.3 Un-gate and run `render-parity` + e2e handoff against the republished fleet (`podinfo v0.1.4`; re-pin here if the republish shipped a different version).
- [x] 8.4 Full `task check` + kind flow; append a completing history event in 0010 if 7.1 landed before this tail.
