## Why

The CLI pins `library v1.0.0-alpha.26` and cannot build against `library` main. Two breaking library changes have landed there since:

1. **`one-api-tier`** (merged, unreleased) folded the loader and synth helpers into one API tier. `opm/helper/loader/file` and `opm/helper/synth` are gone, the raw value tier (`LoadModulePackage`, `LoadPlatformPackage`, `LoadInstancePackage`, `Kernel.NewModuleFromValue`, `Kernel.NewPlatformFromValue`) is gone, no acquire verb takes a per-call `LoadOptions`, and the shape-gate sentinels moved to `opm/errors`. Eleven CLI files import one of the two deleted helper packages; `go build ./...` against `../library` fails at the import line before any type error is reachable.
2. **`cue-owned-verdicts`** (in flight in `library`) makes the render build own every verdict. `RenderResult.Warnings` is removed, `RenderDiagnostics.Unmatched` / `.Unify` / `.OverSubscribed` change element type, and five `opm/errors` types are renamed or reshaped (`UnifyError` -> `UnifyRefusal`, `MatchResult` -> `CandidateVerdict`, `OverSubscribedContractError` -> the `OverSubscribedContract` row plus a `*OverSubscribedContractsError` aggregate, `UnmatchedComponentsError.Components` becomes `[]UnmatchedComponent`, `TransformError.ComponentName`/`.TransformerFQN` become `.Component`/`.Transformer`).

3. **`move-compat-to-cli`** (archived in `library` on 2026-09-08) deleted `opm/compat`. Its only callers are this repo: the publish gate reads `ParseLevel` and `CheckAtLevel` (`internal/publish/compat.go:133,287`) and template version selection reads `HighestStable` (`internal/scaffold/scaffold.go:109`). The comparator ships here now, so a comparator change is one CLI PR instead of a library release plus a re-pin.
4. **`kernel-owns-no-build-context`** (planned in `library`, slice 6a of the kernel surface cut) takes the `cue.Context` out of the Kernel: every verb builds in a context it creates and releases, `Kernel.CueContext()` is removed, `kernel.Source` carries bytes instead of a compiled value, and `schema.Cache.Get()` takes no argument. Three CLI lines call the removed accessor: `internal/cmd/module/vet.go:169` and `internal/cmdutil/publish.go:53` fetch the schema through it, and `internal/cmdutil/publish.go:71` hands it to the publish pipeline as `Options.Context`, the context every catalog package is built in (`internal/publish/load.go:83`, `gates.go:71,384`) so it can unify with `IdentitySchema`. That context has to be the schema value's own (`schemaVal.Context()`), which is what the value already carries. Two tests build values through the accessor. No CLI code reads `Source.Value`; every source comes from `LoadSourceFromFile`.

All four land in the same library alpha, and the CLI cannot compile between them: the acquire-surface migration and the verdict migration touch the same four files under `internal/workflow/render/`. Splitting them would leave a red tree at the boundary, which is why this is one change per repo rather than three. Slice 5 joins for the same reason plus a second one: `internal/scaffold/scaffold.go` is edited by both the acquire-surface migration and the comparator adoption.

The second change also hands the CLI a job it already wanted. `internal/cmdutil/output.go` reconstructs a `*UnresolvedDemandsError` from diagnostics rows purely to reach its own formatter, and `internal/workflow/render/validation.go` re-derives the unmatched line the kernel had already worded. With the kernel out of the prose business (library Principle IV), the CLI owns the wording outright: one formatter over rows, no aggregate reconstruction, and an unmatched component can finally say WHICH candidate was refused and why, because the candidate matrix now arrives on the diagnostics instead of only on the refusal.

## What Changes

**Acquire surface (`one-api-tier`):**

- **BREAKING (internal)** every `loaderfile.LoadOptions{Registry: ...}` argument is deleted: `AcquirePlatformFromDir(ctx, dir)`, `AcquireInstanceFromDir(ctx, dir, values ...Source)`, `AcquireModuleFromDir(ctx, dir)`. The registry mapping is already supplied once through `kernel.WithRegistry` in `NewKernel`, so these are pure deletions.
- `kernel.WithValues(sources...)` becomes the variadic trailing `Source` arguments of `AcquireInstanceFromDir`.
- `LoadModulePackage` + `NewModuleFromValue` becomes `AcquireModuleFromDir`; `LoadPlatformPackage` + `NewPlatformFromValue` becomes `AcquirePlatformFromDir`. A caller that wanted the raw value reads the returned artifact's `Package` field.
- `internal/publish/kernel_gate.go` calls the free function `loaderfile.LoadModulePackage`; it becomes a `Kernel.AcquireModuleFromDir` call on a kernel the gate constructs or receives.
- `loaderfile.ErrWrongKind` / `ErrInvalidPackage` / `ErrMissingRequiredField` become `liberrors.ErrWrongKind` / `ErrInvalidPackage` / `ErrMissingRequiredField`.
- Both integration programs under `tests/integration/` migrate with the same edits.

**Verdicts (`cue-owned-verdicts`):**

- **BREAKING (internal)** `render.Result.Warnings` is filled by a CLI formatter over `out.Diagnostics.UnhandledTraits` and the `Newer` rows of `out.Diagnostics.ResolvedVersions`, instead of being copied from the deleted `RenderResult.Warnings`. `Result.Warnings`, `HasWarnings()` and every reader (`output_internal.go`, `cmd/instance/diff.go`) keep their shape; only the producer changes.
- `cmdutil.FormatUnresolvedDemands` takes `[]liberrors.UnresolvedDemand` instead of `*liberrors.UnresolvedDemandsError`; `PrintValidationError`'s `errors.As` branch reads `demandsErr.Demands` and passes the rows.
- `formatRenderDiagnostics` iterates `d.Unmatched` as rows (`.Component` plus its `.Candidates`) rather than as bare strings, and formats the over-subscription rows from `d.OverSubscribed` directly.
- The over-subscription `errors.As` target becomes `*liberrors.OverSubscribedContractsError` (a pointer aggregate carrying every row), replacing the value-typed `OverSubscribedContractError` that was joined into the gate once per row.
- `liberrors.TransformError` field reads become `.Component` / `.Transformer`.

**Catalog compatibility (`move-compat-to-cli`):**

- `internal/compat/compat.go` and `level.go` arrive verbatim from the library, package name unchanged; the package doc's consumer list drops library-matching, which no longer exists. `compat_test.go` and `level_test.go` arrive with them, including the core-grammar parity pin (`TestAPIVersionPatternCoreParity`).
- `internal/publish/compat.go` imports `github.com/open-platform-model/cli/internal/compat`; nothing else about the gate changes.
- `HighestStable` becomes an unexported `highestStable` in `internal/scaffold`, its only caller, carrying `predecessor.go`'s doc comment (which explains why the float selector is not the gate's predecessor rule) and the four cases of `predecessor_test.go`. `internal/scaffold/scaffold.go` drops its `compat` import.
- `go.mod` promotes `github.com/Masterminds/semver/v3` from an indirect to a direct requirement.

**Kernel context (`kernel-owns-no-build-context`):**

- `internal/cmd/module/vet.go` and `internal/cmdutil/publish.go` fetch the schema with `k.SchemaCache().Get()` and take the CUE context from the value (`schemaVal.Context()`); `publish.Options.Context` receives that context, so the pipeline keeps building catalog packages where the identity and gate schemas live. No behaviour changes: the same schema, the same unifications, one context per invocation as before.
- `internal/workflow/render/module_test.go` and `internal/publish/realtree_test.go` build their values with `cuecontext.New()` or the schema value's context.

**Dependency:** `go.mod` moves `github.com/open-platform-model/library` to the alpha carrying all four changes. That alpha does not exist yet, so this change is blocked on the `library` release.

## Capabilities

### New Capabilities

- `catalog-compatibility`: the comparison walk, level ladder and predecessor selection the library removed with `move-compat-to-cli`, restated against `cli/internal/compat`. Semantics, signatures and tests are unchanged; only the owning repo moves.

### Modified Capabilities

- `kernel-render`: the entry-point requirement drops the removed raw value tier and the per-call load options, naming the acquire verbs and the variadic values arguments instead; the "all renders go through the library kernel" requirement stops promising that the kernel's own warning strings reach the user and states that the CLI words the two advisory facts from the diagnostics rows.
- `cmdutil`: `ShowRenderOutput`'s warning obligation is restated against CLI-formatted advisory rows rather than kernel-produced strings, and the render-error formatting requirement names the reshaped typed causes and the per-candidate verdicts an unmatched component now carries.

## Impact

**SemVer:** PATCH on the CLI's own surface. Every changed symbol, including the arriving comparator, is under `internal/`; `pkg/errors` and `pkg/loader` are untouched, and no command, flag, output format or exit code changes. The library dependency bump is a MAJOR bump of a pre-GA dependency, absorbed here.

**Affected packages (11 files import a deleted helper package, 6 more read a reshaped verdict, 4 arrive from the library):**

- `internal/workflow/render/` — `kernel.go` (platform acquire), `render.go` (instance acquire, `Result.Warnings` producer), `module.go` (module acquire + synth), `validation.go` (`formatRenderDiagnostics`), `types.go` (the `Warnings` doc comment), `output_internal.go` (reader, unchanged behavior).
- `internal/config/platform.go` — platform acquire plus the two sentinel comparisons.
- `internal/scaffold/scaffold.go`, `internal/scaffold/repair.go` — module acquire.
- `internal/publish/kernel_gate.go` — the free-function load becomes a kernel acquire; four sentinel comparisons.
- `internal/cmdutil/output.go` — `FormatUnresolvedDemands` signature, the `errors.As` branch.
- `internal/cmd/instance/diff.go` — reads `Result.Warnings`; unchanged if the producer keeps filling it.
- `tests/integration/platform-build/main.go`, `tests/integration/render-parity/main.go`.
- `internal/compat/` — four files added (`compat.go`, `level.go`, `compat_test.go`, `level_test.go`), copied from the library.
- `internal/scaffold/predecessor.go`, `predecessor_test.go` — added; `scaffold.go:109` calls `highestStable` and drops the `compat` import.
- `internal/publish/compat.go` — one import line.
- `internal/cmd/module/vet.go`, `internal/cmdutil/publish.go` — the three `CueContext()` calls become the schema value's context; `internal/workflow/render/module_test.go`, `internal/publish/realtree_test.go` — test-side context construction.

**Tests:** `internal/cmd/module/verbose_output_test.go` constructs a `Result` with `Warnings`; unchanged. Any test asserting on the kernel's exact warning wording moves to the CLI's wording. The render-parity integration program must keep producing identical digests: it renders, and this change does not touch what is rendered.

**Blocked on:** a `library` release carrying `one-api-tier`, `cue-owned-verdicts`, `move-compat-to-cli` and `kernel-owns-no-build-context`. Until then the work is verifiable only against a local `replace` directive, which is how the tasks stage it.

**Sibling change:** `opm-operator` carries the same migration for its own consumer surface, as a separate change in that repo.
