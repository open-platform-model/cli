# Experiment: AST-Based Build Pipeline

## Goal

Prove that loading a CUE module as AST first, then building `cue.Value` on demand, is viable for the build pipeline. Validate what AST is capable of — construction, conversion, manipulation, inspection, and round-tripping — before committing to a refactor.

## Hypotheses

1. **AST Overlay** — Build the release overlay as `*ast.File` instead of `fmt.Sprintf`, format it, confirm it loads correctly
2. **Single Load** — Load once, get `inst.Files` + package name, inject overlay, build to Value — eliminating the double-load in the current pipeline
3. **Parallel Evaluation** — From the same AST files, spin up independent `cue.Context` per goroutine and run FillPath concurrently without panics
4. **Cross-Context FillPath** — Validate whether transformer values from one context can be used with module values from another context (the production pattern), and identify race-free parallel execution strategies

## Background

The current build pipeline (`internal/build/`) works almost entirely with `cue.Value`. This is correct for evaluation, validation, and data extraction. But it has friction points:

- **Overlay generation** uses `fmt.Sprintf` with embedded CUE — fragile, no compile-time safety
- **Double loading** — `extractModuleMetadata` loads the module once to get name/namespace, then `Build` loads again with the overlay
- **Sequential execution** — `cue.Context` isn't thread-safe, so the executor runs transformer jobs one at a time
- **Early information loss** — values are decoded to `map[string]any` for K8s output, losing CUE type info

AST could address all of these. This experiment validates the approach before any production changes.

## Structure

```
experiments/ast-pipeline/
├── README.md                    # This file — plan, findings, decisions
├── ast_basics_test.go           # AST construction, conversion, round-trips
├── ast_manipulation_test.go     # Modifying AST: add/remove/change fields
├── ast_inspection_test.go       # Walking, querying, extracting from AST
├── overlay_test.go              # Hypothesis 1: AST-based overlay generation
├── loader_test.go               # Hypothesis 2: Single-load with AST inspection
├── parallel_test.go             # Hypothesis 3: Parallel eval from shared AST
├── cross_context_test.go        # Hypothesis 4: Cross-context FillPath & parallel strategies
└── testdata/
    └── test-module/             # Self-contained CUE module for testing
        ├── cue.mod/
        │   └── module.cue
        ├── module.cue
        └── values.cue
```

## Test Module

`testdata/test-module/` — A self-contained module with enough structure to exercise all scenarios:

- Package declaration
- Metadata (name, version, fqn, labels, defaultNamespace)
- `#config` / `values` pattern with multiple typed fields
- `#components` with 3 components (web, api, worker)
- At least one component with `#resources` and `#traits`
- Comments (to verify they survive round-trips)

No external imports. Fully self-contained. No registry needed.

---

## Part 1: AST Fundamentals (`ast_basics_test.go`)

Prove the basic mechanics of AST ↔ Value conversion.

### Construction → Value

| Test | What it proves |
|------|----------------|
| `TestAST_StructToValue` | Build `ast.NewStruct(...)`, call `ctx.BuildExpr()`, read fields back via `LookupPath` |
| `TestAST_FileToValue` | Build `*ast.File` with package + fields, call `ctx.BuildFile()`, verify all fields accessible |
| `TestAST_DefinitionsWork` | Build AST with `#config` definition, `BuildFile()`, verify `FillPath` works on the definition |
| `TestAST_NestedStructs` | Build deeply nested AST (`a.b.c.d: "value"`), convert to Value, verify path lookup works |
| `TestAST_ListsAndExpressions` | Build AST with lists, binary expressions (`&`, `|`), unary (`*default`), verify they evaluate correctly |
| `TestAST_CommentsPreserved` | Build AST with doc comments, convert to Value, call `Syntax()`, verify comments survive round-trip |

### Value → AST

| Test | What it proves |
|------|----------------|
| `TestValue_SyntaxReturnsAST` | `CompileString` → Value → `Syntax()` → `ast.Node`, verify it's a usable AST |
| `TestValue_SyntaxFormatsCleanly` | Value → `Syntax()` → `format.Node()` → verify output is valid, parseable CUE |
| `TestValue_SyntaxConcrete` | Value with `Final()`, `Concrete(true)` → `Syntax()` → all values are literals, no constraints |
| `TestValue_SyntaxWithDefinitions` | Module with `#config` → `Syntax()` without `Concrete` → definitions still present in AST |

### Round-trip

| Test | What it proves |
|------|----------------|
| `TestRoundTrip_ASTToValueToAST` | Build AST → `BuildFile` → Value → `Syntax()` → compare structure to original |
| `TestRoundTrip_ValueToASTToValue` | `CompileString` → Value → `Syntax()` → `BuildFile` on that AST → second Value → verify fields match |
| `TestRoundTrip_FormatParseIdentity` | Build AST → `format.Node()` → `parser.ParseFile()` → compare AST structure |

---

## Part 2: AST Manipulation (`ast_manipulation_test.go`)

Prove we can modify AST structure before evaluation.

### Adding things

| Test | What it proves |
|------|----------------|
| `TestManipulate_AddField` | Load module AST, append a new `*ast.Field` to file's Decls, build to Value, verify new field exists |
| `TestManipulate_AddDefinition` | Load module AST, add `#newDef: { ... }` field, build, verify definition is accessible and constrains |
| `TestManipulate_AddComponent` | Load module AST, add a new component to `#components` struct, build, verify it shows up in iteration |
| `TestManipulate_AddImport` | Build AST, add import via `ast.NewImport()`, use `astutil.Sanitize()`, format, verify valid CUE |
| `TestManipulate_InjectOverlayDecls` | Take module's AST files, append overlay declarations (not a separate file), build, verify overlay fields resolve |

### Modifying things

| Test | What it proves |
|------|----------------|
| `TestManipulate_ChangeFieldValue` | Load AST, use `astutil.Apply` to find a field and replace its value, build, verify new value |
| `TestManipulate_ChangeLabel` | Rename a field label via `astutil.Apply`, build, verify old name gone, new name present |
| `TestManipulate_ReplaceStruct` | Replace entire struct value of a component with a new one, build, verify |

### Removing things

| Test | What it proves |
|------|----------------|
| `TestManipulate_DeleteField` | Use `astutil.Apply` with `cursor.Delete()`, build, verify field is gone |
| `TestManipulate_DeleteComponent` | Remove a component from `#components`, build, verify fewer components |

### Composing things

| Test | What it proves |
|------|----------------|
| `TestManipulate_MergeTwoFiles` | Take decls from two `*ast.File`s, combine into one, build, verify unified result |
| `TestManipulate_OverlayAsASTFile` | Build overlay as `*ast.File`, inject via `load.Config.Overlay` using `format.Node()` output, verify identical to string approach |

---

## Part 3: AST Inspection (`ast_inspection_test.go`)

Prove we can extract useful information from AST without building a Value.

| Test | What it proves |
|------|----------------|
| `TestInspect_WalkFindAllFields` | `ast.Walk` over a file, collect all top-level field names, verify complete list |
| `TestInspect_FindDefinitions` | Walk AST, identify all `#definition` fields (labels starting with `#`), list them |
| `TestInspect_FindImports` | Use `file.ImportSpecs()` to list all imports from AST |
| `TestInspect_ExtractPackageName` | Get package name from `*ast.File` without building Value (inspect `Package` decl) |
| `TestInspect_ExtractMetadataField` | Walk AST to find `metadata.name` string literal value — no evaluation needed for static values |
| `TestInspect_FindComments` | Walk AST, extract all doc comments, verify they're accessible |
| `TestInspect_IdentifyConfigPattern` | Detect whether a module uses the `#config` / `values` pattern by inspecting AST structure |

---

## Hypothesis 1: AST Overlay (`overlay_test.go`)

Proves the specific overlay generation use case — replacing `fmt.Sprintf` with type-safe AST construction.

| Test | What it proves |
|------|----------------|
| `TestOverlayAST_FormatsToValidCUE` | Build overlay as `*ast.File`, `format.Node()`, parse back, no errors |
| `TestOverlayAST_MatchesStringTemplate` | Compare AST-generated overlay against current `fmt.Sprintf` output — semantically equivalent after evaluation |
| `TestOverlayAST_LoadsWithModule` | Inject AST overlay via `load.Config.Overlay`, build to Value, verify `#opmReleaseMeta` fields |
| `TestOverlayAST_InterpolationExpr` | Build the `uuid.SHA1(...)` call expression as AST, verify it evaluates to the same UUID as the string template version |

---

## Hypothesis 2: Single Load (`loader_test.go`)

Proves the double-load can be eliminated.

| Test | What it proves |
|------|----------------|
| `TestSingleLoad_PackageNameFromInstance` | `inst.PkgName` is available without separate load |
| `TestSingleLoad_FilesAvailable` | `inst.Files` contains `[]*ast.File` with all source files |
| `TestSingleLoad_InspectThenBuild` | Load once → inspect AST for package name → build overlay AST → load again with overlay → build Value. Same result as current double-load. |
| `TestSingleLoad_ASTInspectVsValueLookup` | Compare metadata extracted from AST walk vs metadata extracted from `Value.LookupPath` — both find the same name/version |

---

## Hypothesis 3: Parallel Evaluation (`parallel_test.go`)

Proves concurrent transformer execution is possible via AST sharing.

| Test | What it proves |
|------|----------------|
| `TestParallel_SharedASTIndependentContexts` | Load module → get `inst`. Spawn N goroutines each doing `cuecontext.New()` + `ctx.BuildInstance(inst)`. No panics. |
| `TestParallel_FillPathConcurrent` | Each goroutine does `BuildInstance` + `FillPath` on its own Value. No `adt.Vertex` panics. |
| `TestParallel_TransformerSimulation` | Simulate executor: shared AST, 3 transformer jobs, 3 goroutines. Each builds own context, injects `#component` + `#context`, extracts output. |
| `TestParallel_ResultsMatchSequential` | Run same jobs sequentially vs parallel. Assert identical resources (order-independent). |
| `TestParallel_RebuildFromFiles` | If `BuildInstance` can't be shared: re-parse from `inst.Files` in each goroutine. Test this fallback path. |

---

## Hypothesis 4: Cross-Context FillPath (`cross_context_test.go`)

Tests whether the production parallelization pattern is viable: transformer values from one `*cue.Context` used with module values from another. Also validates race-free parallel execution strategies.

| Test | What it proves |
|------|----------------|
| `TestCrossContext_FillPathPanics` | CUE **rejects** `FillPath` across different contexts — panics with "values are not from the same runtime". Each goroutine must build both transformer and module in the same context. |
| `TestCrossContext_SharedBuildInstanceRace` | Documents that the shared `build.Instance` approach (Hypothesis 3) has data races detectable with `-race`. The tests produce correct results but are not race-free. |
| `TestCrossContext_Strategy_FormatAndReparse` | **Race-free** parallel approach: serialize `inst.Files` to bytes once, each goroutine re-parses from bytes + compiles transformer in its own context. Full production-like FillPath sequence. |
| `TestCrossContext_Strategy_ReloadPerGoroutine` | Most robust approach: each goroutine does its own `load.Instances` + `BuildInstance`. Works with external CUE imports. No shared state whatsoever. |
| `TestCrossContext_Strategy_SequentialVsParallelEquivalence` | Verifies parallel (reload) and sequential strategies produce byte-identical output for the same inputs. |

## What we're NOT testing

- Replacing the entire pipeline (proof-of-concept only)
- Provider loading (orthogonal concern)
- Matcher logic (unchanged by AST approach)
- K8s output normalization (downstream of evaluation)
- Performance benchmarks (premature — prove correctness first)

## Questions to Answer

| Question | Where answered |
|----------|----------------|
| Can AST be constructed type-safely and produce valid CUE? | `ast_basics_test.go` |
| Do comments/docs survive AST → Value → AST? | `ast_basics_test.go` (round-trip) |
| Can we add/remove/modify fields before evaluation? | `ast_manipulation_test.go` |
| Can we extract metadata without evaluating? | `ast_inspection_test.go` |
| Is AST overlay viable as string template replacement? | `overlay_test.go` |
| Can we eliminate the double-load? | `loader_test.go` |
| Can we parallelize transformer execution via AST sharing? | `parallel_test.go` |
| Can `build.Instance` be reused across contexts, or do we need to re-parse? | `parallel_test.go` |
| Can FillPath work across different CUE contexts? | `cross_context_test.go` |
| Is shared `build.Instance` actually race-free? | `cross_context_test.go` |
| What parallel execution strategies are viable and race-free? | `cross_context_test.go` |
| What are the gotchas and limitations? | All — documented as found |

## Findings

All 50 tests pass (13 basics + 12 manipulation + 7 inspection + 4 overlay + 4 loader + 5 parallel + 5 cross-context).

### AST Fundamentals — All Confirmed

- **Construction works end-to-end**: `ast.NewStruct`, `ast.NewString`, `ast.NewLit`, `ast.NewBool`, `ast.NewList`, `ast.NewCall` all produce nodes that `ctx.BuildExpr()` and `ctx.BuildFile()` evaluate correctly.
- **Definitions work**: AST-built `#config` definitions constrain and `FillPath` resolves through them.
- **Comments survive round-trips**: `ast.AddComment` → `format.Node` → `parser.ParseFile` preserves doc comments.
- **Value → AST works**: `val.Syntax()` returns usable AST nodes. With `cue.Final(), cue.Concrete(true)`, defaults resolve to literals. Without those options, definitions/constraints are preserved.
- **Round-trips are stable**: AST → Value → Syntax() → format → parse → format produces identical bytes (idempotent).

### Key Gotcha: `ast.NewStruct` Label Types

**Critical discovery.** When `ast.NewStruct` receives a Go `string` argument as a label, it creates a `*ast.BasicLit` (quoted string label like `"name"`). CUE treats quoted labels differently from identifier labels — **quoted labels are not visible for scope resolution from nested structs.**

To create labels that support reference resolution (e.g., `name` inside `labels` resolving to `name` in the parent struct), you must pass `*ast.Field` entries with `ast.NewIdent("name")` labels instead of string `"name"`.

This distinction doesn't matter for flat structs but breaks nested scoping — exactly the pattern used in `#opmReleaseMeta.labels`.

### Key Gotcha: `ast.Interpolation` Element Format

String fragments in `ast.Interpolation.Elts` must include quote characters and interpolation delimiters. The format matches what the parser produces:

```
"\(fqn):\(name):\(namespace)"
→ Elts: ["\"\\(", fqn, "):\\(", name, "):\\(", namespace, ")\""]
```

Each even-indexed element is a `*ast.BasicLit{Kind: token.STRING}` containing the raw string with escape sequences. Getting this wrong produces a plain string instead of an interpolation expression.

### Key Gotcha: Scope Resolution with `astutil.Resolve`

Programmatically constructed AST nodes don't have `Ident.Scope` and `Ident.Node` wired up. The parser does this automatically via `astutil.Resolve`. For AST that contains cross-scope references (like labels referencing parent struct fields), you must call `astutil.Resolve(file, errFn)` after construction. This is a no-op for already-resolved nodes and safe to call unconditionally.

### Hypothesis 1: AST Overlay — CONFIRMED

The `generateOverlayAST` function produces byte-identical CUE output compared to the `fmt.Sprintf` approach when both are formatted. Both produce the same `#opmReleaseMeta` fields, the same UUID identity, and load identically with the test module.

**Benefits over string templates:**

- Type-safe: compiler catches structural errors
- No string quoting bugs (e.g., values with special chars)
- Interpolation expressions built from typed nodes, not embedded raw CUE
- `astutil.Resolve` catches reference errors early

### Hypothesis 2: Single Load — CONFIRMED

`load.Instances` returns `inst.PkgName` and `inst.Files` (`[]*ast.File`) without evaluation. Package name extraction from `inst.PkgName` matches what you'd get from building a Value and looking up metadata. AST inspection (walking `*ast.Field` nodes) extracts `metadata.name` and `metadata.version` as string literals — matching `Value.LookupPath` results exactly.

**This eliminates the double-load.** The current pipeline loads once for metadata, builds the overlay, then loads again. With AST inspection, we load once, inspect the AST for metadata, construct the overlay AST, and inject it via `load.Config.Overlay` for a single `BuildInstance` call.

### Hypothesis 3: Parallel Evaluation — PARTIALLY CONFIRMED ⚠️

The `build.Instance` returned by `load.Instances` produces correct results when shared across goroutines — each goroutine creates its own `cuecontext.New()` and calls `ctx.BuildInstance(inst)` independently. **No panics, correct output.** However, **the race detector reveals data races** in CUE's `resolveFile` (`runtime/resolve.go:59,151,154`) during concurrent `BuildInstance` calls on the same `*build.Instance`.

**Key findings:**

- `BuildInstance` from the same `inst` produces correct results in N goroutines simultaneously
- `FillPath` on independently-built Values works concurrently
- Transformer simulation (3 goroutines building → filling → extracting) produces results identical to sequential execution
- **⚠️ `-race` detector reports data races** — `resolveFile` mutates fields on the shared `*build.Instance` during `BuildInstance`. The tests pass without `-race` but fail with it.
- The fallback path (re-parse from `inst.Files` via `format.Node` + `load` per goroutine) is **race-free and necessary** for correct parallel execution

**Implication:** The shared `build.Instance` approach is not safe for production use with `-race`. Use format+reparse or reload-per-goroutine strategies instead (see Hypothesis 4).

### Hypothesis 4: Cross-Context FillPath — REJECTED; Viable Strategies Identified

Three critical findings for the production parallelization story:

**Finding 1: Cross-context FillPath is rejected.** CUE explicitly checks and panics with `"values are not from the same runtime"` when `FillPath` is called with values from different `*cue.Context` instances. This means you cannot pre-load transformers in a shared provider context and inject module values from per-goroutine contexts. Each goroutine must build BOTH transformer and module in the same context.

**Finding 2: Shared `build.Instance` has data races.** See Hypothesis 3 correction above.

**Finding 3: Two viable race-free strategies exist:**

| Strategy | How it works | Pro | Con |
|----------|-------------|-----|-----|
| **Format + Reparse** | Serialize `inst.Files` to `[]byte` once (single-threaded). Each goroutine re-parses from bytes, `BuildFile` + `Unify`, compiles transformer in same context. | Fast (no disk I/O per goroutine). Shares only immutable bytes. | Doesn't handle external CUE imports (BuildFile doesn't resolve imports). |
| **Reload per goroutine** | Each goroutine does its own `load.Instances` + `BuildInstance`. Fully independent. | Works with any module including external imports. Trivially correct. | Re-reads from disk and re-resolves CUE deps per goroutine. |

Both strategies produce output identical to sequential execution (verified by `TestCrossContext_Strategy_SequentialVsParallelEquivalence`).

**For production OPM modules** (which import `opmodel.dev/core@v0`), the **reload-per-goroutine** strategy is required because `BuildFile` cannot resolve external imports. The format+reparse strategy works only for self-contained modules without registry imports.

### `Value.Default()` API

`Value.Default()` returns `(Value, bool)`, not `(Value, error)`. The bool indicates whether a default value exists. This differs from most CUE API methods that return `(T, error)`.

## Decisions

### Proceed with AST-based refactor

Hypotheses 1 and 2 are fully confirmed. Hypothesis 3 is partially confirmed (correct results, but shared `build.Instance` has races). Hypothesis 4 reveals cross-context FillPath is rejected and identifies two viable race-free parallel strategies.

### Recommended refactor sequence

1. **Overlay generation** (lowest risk, highest clarity gain): Replace `fmt.Sprintf` in `release_builder.go` with `generateOverlayAST`-style typed construction. Add `astutil.Resolve` call. This is a drop-in replacement — same bytes in, same bytes out.

2. **Single load** (medium risk, eliminates redundant work): Refactor the load path to call `load.Instances` once, inspect `inst.PkgName` and AST for metadata, then build with overlay. Removes `extractModuleMetadata` as a separate load.

3. **Parallel execution** (highest impact, most design work): Each goroutine must do its own `load.Instances` + `cuecontext.New()` + `BuildInstance` AND compile transformers in the same context (cross-context FillPath is rejected). The reload-per-goroutine strategy is required for modules with external CUE imports. Transformer CUE source must be serialized at provider-load time (the render-pipeline-v2 design notes that `Syntax()` panics on complex transformer values — this remains an open risk that needs a spike on production transformers).

### Rules for AST construction

- Always use `ast.NewIdent("fieldName")` for labels that need scope visibility — never pass raw strings to `ast.NewStruct` for those fields.
- Quoted string labels (`ast.NewString(...)`) are fine for keys that contain special characters (e.g., `"module-release.opmodel.dev/name"`) since those are never referenced.
- Call `astutil.Resolve(file, errFn)` on any programmatically-built `*ast.File` that contains cross-scope identifier references.
- For `ast.Interpolation`, match the parser's element format exactly: include `"\(` and `)"` delimiters in the `BasicLit` values.
