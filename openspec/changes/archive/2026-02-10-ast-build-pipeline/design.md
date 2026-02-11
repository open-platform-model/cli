## Context

The render pipeline (`internal/build/`) uses a 6-phase architecture (see `render-pipeline-v2/design.md`). Two of those phases have implementation weaknesses:

**Phase 2 (Metadata Extraction)** — `pipeline.extractModuleMetadata()` (`pipeline.go:189`) calls `load.Instances` → `ctx.BuildInstance` → `LookupPath("metadata.name")`. This is a full CUE evaluation just to extract two string literals (`name` and `defaultNamespace`) that are visible in the AST without evaluation.

**Phase 3 (ReleaseBuilder)** — `ReleaseBuilder.Build()` (`release_builder.go:68`) does:

1. `detectPackageName()` — calls `load.Instances` again just to read `inst.PkgName`
2. `generateOverlayCUE()` — uses `fmt.Sprintf` with embedded CUE source to produce the overlay
3. `load.Instances` with overlay → `BuildInstance` — the actual build

That's three `load.Instances` calls (Phase 2 + detectPackageName + overlay build) and one unnecessary `BuildInstance` (Phase 2). The overlay is generated from a format string with `%s` and `%q` substitutions into raw CUE source — invisible to the Go compiler and `go vet`.

The `experiments/ast-pipeline/` validated that:

- AST construction via `ast.NewStruct`, `ast.NewIdent`, etc. produces byte-identical CUE output (Hypothesis 1)
- `inst.PkgName` and AST walk of `inst.Files` extract metadata without evaluation (Hypothesis 2)
- Both are drop-in replacements with no behavioral change

## Goals / Non-Goals

**Goals:**

- Replace `fmt.Sprintf` overlay generation with typed AST construction
- Eliminate the Phase 2 `BuildInstance` call by extracting metadata from AST
- Reduce three `load.Instances` calls to two (AST inspection + overlay build)
- Maintain byte-identical pipeline output for all existing modules

**Non-Goals:**

- Parallel transformer execution (deferred — cross-context FillPath is rejected by CUE)
- Changing the Pipeline interface, RenderResult, or any command-level behavior
- Optimizing `BuildFromValue` (legacy path for test fixtures — no overlay involved)
- Changing the overlay's CUE structure (`#opmReleaseMeta` definition stays the same)

## Decisions

### Decision 1: Replace `generateOverlayCUE` with AST construction

**Context:** `release_builder.go:264` uses `fmt.Sprintf` to embed CUE source text with `%s` (package name) and `%q` (release name, namespace) substitutions. The format string contains CUE interpolation syntax (`"\(fqn):\(name):\(namespace)"`) that looks like Go format verbs but isn't — confusing to read and invisible to the compiler.

**Options considered:**

1. Keep `fmt.Sprintf` — works, but fragile. A stray `%` in a module name would break it. No compile-time validation.
2. `text/template` — slightly better syntax but still string-based. No structural guarantees.
3. CUE AST construction — type-safe, compile-time checked, produces identical output.

**Decision:** Option 3 — AST construction.

**Implementation:** New method `generateOverlayAST(pkgName string, opts ReleaseOptions) *ast.File` builds the overlay as an `*ast.File` using `ast.NewStruct`, `ast.NewIdent`, `ast.NewString`, `ast.Interpolation`, and `ast.NewCall`. The file is formatted to bytes via `format.Node` before passing to `load.Config.Overlay`.

**Key rules from the experiment:**

- Field labels that are referenced from nested scopes (`name`, `namespace`, `fqn`, `version`, `identity`) MUST use `ast.NewIdent` (unquoted identifier labels), not `ast.NewString` (quoted string labels). Quoted labels break CUE scope resolution.
- Label keys with special characters (`"module-release.opmodel.dev/name"`) use `ast.NewString` because they're never referenced.
- `astutil.Resolve(file, errFn)` MUST be called after construction to wire up scope references.
- `ast.Interpolation.Elts` must match parser output format: `"\(` prefix, `):` separators, `)"` suffix as `*ast.BasicLit{Kind: token.STRING}`.

### Decision 2: Merge Phase 2 metadata extraction into ReleaseBuilder

**Context:** Phase 2 (`pipeline.extractModuleMetadata` at `pipeline.go:189`) and Phase 3 (`ReleaseBuilder.detectPackageName` at `release_builder.go:237`) both call `load.Instances` independently. Phase 2 additionally calls `BuildInstance` just to `LookupPath` two string fields.

**Options considered:**

1. Keep separate phases, just change Phase 2 to use AST — eliminates `BuildInstance` but still two separate `load.Instances` calls.
2. Merge into ReleaseBuilder — single `load.Instances` call that extracts package name (`inst.PkgName`), metadata (`ast.Walk` over `inst.Files`), AND builds the overlay. ReleaseBuilder returns the preview metadata alongside the built release.
3. Create a new `ModuleInspector` type — separate concern for AST inspection, used by both pipeline and release builder.

**Decision:** Option 2 — merge into ReleaseBuilder.

**Rationale:** The metadata preview is only used to resolve `--name` and `--namespace` defaults before the overlay build. Moving the initial `load.Instances` call into `ReleaseBuilder.Build` means:

- One `load.Instances` call (no overlay) → extract `inst.PkgName` + AST walk for metadata
- One `load.Instances` call (with overlay) → `BuildInstance` → rest of the pipeline
- `Build` returns both the `BuiltRelease` and the resolved name/namespace
- `pipeline.extractModuleMetadata` and `ReleaseBuilder.detectPackageName` are removed

**Change to Build signature:**

```go
// Before:
func (b *ReleaseBuilder) Build(modulePath string, opts ReleaseOptions, valuesFiles []string) (*BuiltRelease, error)

// After:
func (b *ReleaseBuilder) Build(modulePath string, opts ReleaseOptions, valuesFiles []string) (*BuiltRelease, error)
```

The signature stays the same. The change is internal: `ReleaseOptions.Name` and `Namespace` may now be empty, and `Build` resolves them from AST before constructing the overlay. The pipeline passes through raw flag values and `Build` applies defaults from the module metadata.

Actually — this couples name/namespace resolution into the release builder. The pipeline currently does:

```go
releaseName := opts.Name
if releaseName == "" {
    releaseName = moduleMeta.name
}
namespace := p.resolveNamespace(opts.Namespace, moduleMeta.defaultNamespace)
```

Moving this into `Build` means the builder needs to know the resolution logic. Cleaner approach: `Build` does the initial AST-only load and returns the preview metadata if name/namespace are empty, letting the caller handle resolution. But that adds complexity.

**Revised approach:** Add a new method `InspectModule(modulePath string) (*moduleMetadataPreview, error)` on `ReleaseBuilder` that does a single `load.Instances` (no overlay, no `BuildInstance`) and returns `{name, defaultNamespace, pkgName}` via AST inspection. The pipeline calls this instead of `extractModuleMetadata`. Then `Build` reuses `pkgName` (passed via `ReleaseOptions` or cached) so it doesn't need `detectPackageName` anymore.

```go
type ReleaseOptions struct {
    Name      string
    Namespace string
    PkgName   string // Set by InspectModule, used to skip detectPackageName
}
```

This keeps name/namespace resolution in the pipeline (where it belongs) and eliminates `detectPackageName` + `extractModuleMetadata` as separate load paths.

### Decision 3: AST walk strategy for metadata extraction

**Context:** The AST walk needs to find `metadata.name` and `metadata.defaultNamespace` as string literals in the module files. The experiment (`TestSingleLoad_ASTInspectVsValueLookup`) proved this works for static string literals.

**Options considered:**

1. Walk all `inst.Files` looking for nested `metadata` → `name` / `defaultNamespace` field patterns — handles any file layout.
2. Only walk the first file (`module.cue` by convention) — faster but fragile if metadata is in a different file.
3. Use a visitor pattern with `astutil.Apply` — more structured but heavier for a two-field extraction.

**Decision:** Option 1 — walk all files, stop at first match.

**Limitation:** This only works for static string literals (`name: "my-module"`). If `metadata.name` is a reference or expression, the AST walk will miss it and return empty. The pipeline must handle this gracefully — if AST inspection returns empty, fall back to `BuildInstance` + `LookupPath`. In practice, all OPM modules use string literals for `metadata.name` and `metadata.defaultNamespace`, so the fallback should never trigger.

## Risks / Trade-offs

**[Risk] AST walk misses computed metadata** → If a module uses `metadata: name: someExpression` instead of a string literal, the AST walk returns empty. Mitigation: Fall back to `BuildInstance` + `LookupPath` when AST inspection returns empty name. This preserves correctness at the cost of the extra evaluation.

**[Risk] `ast.Interpolation` format fragility** → The interleaved `Elts` format (`"\(`, ident, `):\(`, ...) matches what the CUE parser produces today. A CUE SDK update could change the parser's output format. Mitigation: The `TestOverlayAST_InterpolationExpr` test in the experiment verifies round-trip identity UUID matching. Port this test to the production test suite.

**[Risk] `astutil.Resolve` scope wiring** → Without the resolve call, identifier references in nested structs fail silently (CUE treats them as new fields instead of references). Mitigation: Always call `astutil.Resolve` after AST construction. The error callback can log warnings for unresolved references (expected for `metadata.*` which is external to the overlay).

**[Trade-off] More code for overlay generation** → The AST construction is ~60 lines vs ~15 lines for `fmt.Sprintf`. The AST version is more verbose but structurally validated by the Go compiler. Worth the trade-off for a critical code path that computes release identity.

**[Trade-off] `PkgName` field on ReleaseOptions** → Adds a field that callers shouldn't set directly (it's populated by `InspectModule`). Mitigation: Document it as internal. Alternatively, cache it on the `ReleaseBuilder` struct, but that adds mutable state.
