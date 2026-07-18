# Design: cli-kernel-adoption

## Context

Enhancement 0006 slice C2 (see `enhancements/0006/`, decisions D9, D11, D12, D17, D21, D22, D39; 0002 carryover). C1 already moved inventory to the `ModuleInstance` CR and bumped CUE to v0.17.1 (D36), so this slice starts from a CLI on the final CUE line with the CR backend in place. The operator has rendered through the `library` kernel since 0001 landed: `PlatformReconciler` maps the Platform CR spec → `synth.PlatformInput` → `SynthesizePlatform` → `Materialize` into an in-process store, and `KernelModuleRenderer` acquires the module, synthesizes the instance, and compiles it against the materialized platform. This slice makes the CLI the second consumer of exactly that path, and deletes everything the old CLI pipeline needed that the kernel does not: the `providers` config block, `~/.opm/cue.mod`, the two-phase config loader, `pkg/render`, `pkg/provider`, and the match/synth halves of `pkg/loader`.

Constraints inherited from the enhancement:

- **D13**: the CLI imports `library` only — never `opm-operator` (no controller-runtime/Flux). Cluster objects (`Platform`, `ModuleInstance`) are read/written as `unstructured`.
- **D14**: single user, no compat window — breaking changes land without deprecation machinery.
- **D30**: the accepted→implemented gate is the render-digest parity experiment (CLI render ≡ operator render for the same instance against the same Platform spec). D36 precondition: `library` and `opm-operator` must be on the same CUE line before the experiment is meaningful.
- **D17**: no non-`handoff` path may require cluster access — a local platform must be able to drive every render.

## Goals / Non-Goals

**Goals:**

- One render implementation in the OPM ecosystem: CLI and operator both call the kernel; digest parity is structural.
- Platform resolution by precedence with visible provenance; `~/.opm/platform.cue` as the zero-flag local default (D39).
- `~/.opm` reduced to two import-free data files; config loading single-pass.
- Retire the last `#ModuleRelease` production reference (0002 carryover) — synthetic builds emit `kind: "ModuleInstance"`.

**Non-Goals:**

- Apply-engine unification (D10 — the CLI keeps its own SSA apply; only render/match unifies).
- `opm instance handoff` (slice C3).
- Reverse handoff, rollback, `Release`/`BundleRelease` support (out of 0006 scope).
- Changing the kernel itself — the CLI consumes whatever 0001's kernel ships; kernel gaps found here are filed against `library`, not worked around with CLI-side render logic.

## Decisions

Numbered LD*n* (local decisions) to avoid colliding with the enhancement's D*n*.

### LD1: One Kernel per invocation, constructed at workflow entry

The CLI constructs a single `kernel.Kernel` per command invocation (registry string from resolved config) and threads it through the workflow. `GlobalConfig.CueContext` is deleted; packages that need a `*cue.Context` receive the kernel (or its `CueContext()`). Rationale: the kernel owns the one-Cache-per-process invariant and the schema cache; a CLI invocation is one short-lived process, so process == invocation. Alternative — a lazily-initialized global — rejected: hidden state, harder tests, no startup cost worth avoiding.

### LD2: CLI entry points map onto kernel entry points 1:1

- `opm instance apply/build/diff <file.cue>` (instance file): `LoadInstancePackage`/`LoadSourceFromFile` → `ProcessModuleInstance`.
- `opm module apply/build <dir>` (module package): `LoadModulePackage` → `SynthesizeInstance` (values resolved CLI-side into a `cue.Value`, mirroring `KernelModuleRenderer`'s values handling).
- Registry references: `AcquireModuleFromRegistry` → `SynthesizeInstance` — the operator's own acquisition path.
- All paths then run `Validate` → `Match` → `Plan` → `Compile` → `Finalize` against the materialized platform.

Deleted: `pkg/render` (all of it), `pkg/provider`, `pkg/loader`'s `provider.go`/`synth.go` and match plumbing, `pkg/module.ParseModuleInstance`-era processing that duplicates `ProcessModuleInstance`. `internal/workflow/render` keeps its role (options, output shaping, value-file resolution) but its pipeline core becomes kernel calls. Kept: `pkg/resourceorder`, `internal/kubernetes` (SSA apply), `pkg/inventory` + `internal/inventory` (D31 — local inventory logic on the CR store).

### LD3: New `internal/platform` package owns resolution

`Resolve(ctx, opts) (*materialize.MaterializedPlatform, PlatformSource, error)` implements D21 precedence:

1. `--platform <file>` → local file (highest, explicit).
2. Cluster `Platform` CR spec, read as `unstructured` via the dynamic client (only for cluster-facing commands: `apply`, `diff`, `delete`-with-render needs none).
3. Local default `~/.opm/platform.cue`.

Fallback from 2→3 **warns** and every command reports the resolved source (provenance banner, mirroring `config.ResolvedField`'s Source pattern). Offline commands (`build`, `render`) skip step 2 entirely — they never touch the cluster (D21). All three sources decode to `synth.PlatformInput` → `SynthesizePlatform` → `Materialize`; no `LoadPlatformPackage` path for the default (authored CUE-module platforms remain reachable by pointing `--platform` at a file inside a CUE module only if that file is data-shaped; full authored-`#Platform` module loading is out of scope here).

### LD4: `~/.opm/platform.cue` is the CR-spec projection, validated by an embedded schema

File shape (data-only, no imports): `name`, `type`, `registry: {<catalog-path>: {enable?, filter?: {range?, allow?, deny?}}}` — mirroring the operator's `PlatformSpec`/`Subscription` wire shape, which itself projects core `#Platform`. A new embedded projection schema in `internal/config/schema/` validates it exactly like `config.cue` (CompileBytes + unify + concrete check). The same decode function consumes the cluster CR's `spec` (unstructured map) and the `--platform` file, so one mapping covers all three sources. Rationale: the local file and the cluster CR become the same document in two locations — D12's write-if-absent is file-contents wrapped in `apiVersion`/`kind`/`metadata`, and the file is one `kubectl apply` from being the cluster Platform.

### LD5: Config loading becomes single-pass; registry resolution is ordinary field precedence

With no imports in `~/.opm`, nothing needs `CUE_REGISTRY` before parsing. Delete `BootstrapRegistry`, `configHasProviders`, `extractProviders`, and the phase-2 registry env dance; `Load` parses `config.cue` once, validates against the (shrunk) schema, and resolves registry by flag > env > config like every other field. Schema drops `providers` and `cacheDir`. `GlobalConfig` drops `Providers` and `CueContext`. `opm config init` writes `config.cue` + `platform.cue`, creates no `cue.mod`, runs no `cue mod tidy`; `DefaultModuleTemplate` is deleted. `opm config vet` validates both files against their embedded schemas.

### LD6: Solo-cluster Platform write-if-absent is create-only (D22)

On cluster-facing apply, when the cluster has no `Platform` CR and resolution fell back to the local default, the CLI creates the singleton `cluster` Platform from the resolved local spec via a plain dynamic-client `Create` (field manager `opm-cli`), treating `AlreadyExists` as success-noop. Never SSA (create-or-update would clobber), never an update. RBAC denial on create degrades to a warning — the render already succeeded against the local platform (D17).

### LD7: Digest computation — mirror the operator's, upgrade `lastAppliedSourceDigest`

The render digest stays the manifest digest C1 ships, now computed over kernel-finalized output; the implementation MUST match the operator's computation (same canonical serialization) — verified by the D30 parity check, which lands here as an integration test comparing a CLI render digest against an operator-computed digest for the same fixture instance + Platform spec (kind-gated like the existing e2e programs). `lastAppliedSourceDigest` upgrades from C1's module-reference identity stopgap to the kernel-derived module content digest, matching the operator's field semantics.

### LD8: Seeded subscription ranges are explicit and prerelease-tolerant

`opm config init` seeds `platform.cue` with `opmodel.dev/catalogs/opm: {filter: range: ">=1.0.0-0 <2.0.0-0"}` and `opmodel.dev/catalogs/kubernetes: {filter: range: ">=1.1.0-0 <2.0.0-0"}` (Masterminds semantics: the `-0` bounds admit prereleases, cap below v2). When the catalogs cut stable, the template tightens to plain major-bounded ranges — a template edit, not a schema change.

### LD9: `cue-binary-integration` is not a dependency and should be withdrawn

Its motivation (running `cue mod tidy` for `~/.opm`) is deleted by D39. This change does not build on it; recommend withdrawing or re-scoping it separately.

## Risks / Trade-offs

- [Kernel match semantics differ from the CLI pipeline's → existing modules render differently] → The kernel is the contract (0001); differences are correctness fixes by definition, but each must be *visible*: port the CLI's render golden/fixture tests onto the kernel path before deleting `pkg/render`, and diff their output as part of the task sequence, not after.
- [Materialize performs registry I/O on every render → offline `build` breaks without a reachable registry] → Same exposure the operator has; the CUE module cache serves repeat pulls. Document that `build` needs registry access for catalog resolution (the old pipeline needed it for provider imports — not a regression).
- [D30 parity experiment blocked by evaluator skew] → D36 precondition tracked in 0006: `library`/`opm-operator` on the same CUE line. The parity test is kind+registry-gated and reports skew explicitly rather than failing obscurely.
- [Non-admin cannot read cluster `Platform` → warn-fallback renders against a different platform than the cluster's, diff may show drift] → Accepted by D21/OQ12: warn + provenance banner instead of refusal (refusal would break D17 accessibility).
- [Users with an existing `~/.opm` (providers block, cue.mod) hit validation errors] → `opm config vet`/load errors name the removed fields and point at `opm config init`; one-time migration, no window (D14).
- [Big-bang slice] → Mitigated by strict task ordering (below): config simplification and platform resolution land as separate reviewable phases before the pipeline swap; each phase keeps `task check` green.

## Migration Plan

1. Phase A (config, D39): shrink schema + loader, retire templates/cue.mod/tidy (and the now-dead `internal/cuetidy`). *Implemented deviation:* `GlobalConfig.Providers`/`CueContext` stay as documented legacy-shim fields (never populated) rather than being dropped — dropping them in A would force rewiring every render-path consumer twice; Phase C deletes fields + consumers together.
2. Phase B (platform): `internal/platform` + `platform.cue` schema + `--platform` flag + provenance reporting + write-if-absent.
3. Phase C (kernel): add `library` dep, rewire render/build/diff/apply through the kernel, delete `pkg/render`/`pkg/provider`/loader match+synth, retire `--provider`.
4. Phase D (parity + cleanup): source-digest upgrade, D30 parity integration test, docs, 0006 history event.

Rollback: git revert (no persisted-data migration in this slice — the CR schema is untouched).

## Open Questions

None blocking. Two items resolve during implementation with clear owners: the exact operator digest-serialization to mirror (read `opm-operator`'s computation at LD7 task time), and whether `internal/workflow/render`'s value-file resolution can reuse kernel `Source`/`ValidateModuleValuesDetailed` wholesale (preferred) or needs a thin adapter.
