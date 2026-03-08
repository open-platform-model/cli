## Context

The `experiments/factory/` prototype validated a simplified rendering architecture over 6 months of iteration. The current CLI rendering system spans 5 packages (`internal/builder`, `internal/pipeline`, `internal/loader`, `internal/core/transformer`, `internal/core/provider`) with Go-side component matching, a separate build phase, eager K8s type conversion, and no bundle support. The factory consolidated this into 2 packages (`pkg/engine`, `pkg/loader`) with CUE-native matching, unified loading, lazy `cue.Value` resources, and full bundle support. The factory also moved all types to `pkg/` for external reusability and proved that auto-secrets can be handled entirely in CUE.

This is a big-bang internal refactor. External CLI behavior (commands, flags, output format) does not change. The 15 technical debt items tracked in `experiments/factory/DEBT.md` MUST be fixed during promotion.

**Stakeholders**: Module authors (CLI users), platform operators, downstream tooling that may import `pkg/` types.

**Constraints**:

- CUE context (`*cue.Context`) is not goroutine-safe — all CUE operations MUST be sequential
- `internal/kubernetes/` and `internal/inventory/` MUST NOT import CUE packages — clean boundary at `internal/cmdutil/`
- Existing CLI commands (`mod build`, `mod apply`, `mod diff`, `mod vet`, `mod status`, `mod delete`) MUST produce identical output for valid modules
- All existing tests MUST pass after adaptation (test behavior, not implementation)

## Goals / Non-Goals

**Goals:**

- Replace the 5-package rendering system with factory's 2-package engine+loader architecture
- Move all shared types from `internal/core/*` to `pkg/*` for external reusability
- Implement CUE-native matching via `#MatchPlan` (eliminate Go-side component matching)
- Implement validation gates (Module Gate, Bundle Gate) with structured error output
- Change `Resource` type from `*unstructured.Unstructured` to `cue.Value` with conversion methods
- Eliminate the `Component` Go type (CUE-native matching doesn't need it)
- Eliminate the `Pipeline` interface (direct engine calls)
- Eliminate Go-side auto-secrets injection (CUE `#AutoSecrets` handles it)
- Fix all 15 DEBT.md items during promotion
- Establish clean CUE boundary: nothing below `internal/cmdutil/` imports CUE packages

**Non-Goals:**

- Adding bundle CLI commands (`opm bundle build`, etc.) — separate change
- Changing CLI command syntax, flags, or user-facing output format
- Refactoring `internal/kubernetes/` or `internal/inventory/` internals beyond import path changes and signature adaptations
- Adding new features to the rendering engine beyond what the factory proved
- Changing the CUE definitions in `v1alpha1/` (they stay as-is from the factory)

## Decisions

### Decision 1: Resource type wraps `cue.Value` with conversion methods

**Alternatives considered:**

1. Keep `*unstructured.Unstructured` — forces eager conversion, couples core type to K8s
2. Wrap `cue.Value` with lazy conversion — runtime-agnostic, converts on demand
3. Interface-based with multiple implementations — over-engineered for this use case

**Decision:** Option 2. `pkg/core.Resource` wraps `cue.Value` with accessor and conversion methods.

**Rationale:** The factory proved that keeping `cue.Value` until the last possible moment eliminates an entire class of conversion bugs and makes the core type runtime-agnostic. K8s consumers call `ToUnstructured()` at the boundary.

```go
// pkg/core/resource.go
type Resource struct {
    Value       cue.Value
    Release     string
    Component   string
    Transformer string
}

// Accessors — read from cue.Value, no conversion
func (r *Resource) Kind() string
func (r *Resource) Name() string
func (r *Resource) Namespace() string
func (r *Resource) APIVersion() string
func (r *Resource) GVK() schema.GroupVersionKind
func (r *Resource) Labels() map[string]string
func (r *Resource) Annotations() map[string]string

// Conversion — lazy, on demand
func (r *Resource) MarshalJSON() ([]byte, error)
func (r *Resource) MarshalYAML() ([]byte, error)
func (r *Resource) ToUnstructured() (*unstructured.Unstructured, error)
func (r *Resource) ToMap() (map[string]any, error)
```

The conversion boundary sits in `internal/cmdutil/render.go` — commands that need K8s types convert there, passing `[]*unstructured.Unstructured` to `internal/kubernetes/` and `internal/inventory/`.

### Decision 2: Matching moves entirely to CUE via `#MatchPlan`

**Alternatives considered:**

1. Keep Go-side matching (current `provider.Match()`) — duplicates CUE logic, can drift
2. CUE-native matching via `#MatchPlan` — single source of truth, richer diagnostics
3. Hybrid — match in Go, validate in CUE — worst of both worlds

**Decision:** Option 2. Go fills `#provider` and `#components` into `#MatchPlan` CUE definition, CUE computes the full cartesian match matrix, Go decodes the result.

**Rationale:** The factory proved this works. CUE's `#MatchPlan` produces structured diagnostics (missing labels, missing resources, missing traits per transformer) that the Go-side matching couldn't provide without significant additional code. Single source of truth eliminates drift between CUE schema and Go matching logic.

```
Go side:                                CUE side (#MatchPlan):
                                        
buildMatchPlan(cueCtx, dir,             #provider ──┐
  providerVal, schemaComponents)        #components ─┤
    │                                                ↓
    │  FillPath #provider ──────────▶  for comp, for tf:
    │  FillPath #components ─────────▶   check requiredLabels
    │                                    check requiredResources
    │  Decode result ◀──────────────   check requiredTraits
    ↓                                    ↓
  MatchPlan {                          matches[comp][tf] = {
    Matches,                             matched, missingLabels,
    Unmatched,                           missingResources, missingTraits
    UnhandledTraits,                   }
  }
```

### Decision 3: No `Pipeline` interface — concrete engine structs

**Alternatives considered:**

1. Keep `Pipeline` interface for testability
2. Concrete `ModuleRenderer`/`BundleRenderer` structs with direct method calls

**Decision:** Option 2.

**Rationale:** The `Pipeline` interface added indirection without value — there was only ever one implementation. Tests can use the concrete types directly. The factory validated this approach over months of testing.

### Decision 4: Loading IS building — no separate builder phase

**Alternatives considered:**

1. Keep separate `loader` + `builder` phases
2. Unified loading: load release package → gate validation → finalize → extract metadata

**Decision:** Option 2. `pkg/loader/` handles the full pipeline from CUE file loading through to a fully-constructed `*ModuleRelease` or `*BundleRelease`.

**Rationale:** The separate builder existed because the old system loaded modules and releases differently — modules were loaded, then a builder injected values via FillPath. The factory's release-centric loading (`release.cue` + `values.cue` as one CUE instance) eliminates this split. The CUE evaluation naturally handles what the builder did imperatively: unification, defaults, UUID generation, label computation.

```
Old (3 phases):                     New (1 phase):
                                    
loader.LoadModule()                 loader.LoadReleasePackage()
    ↓                                   ↓
builder.Build()                     loader.DetectReleaseKind()
  - resolve values files                ↓
  - validate values                 loader.LoadModuleReleaseFromValue()
  - FillPath chain                    - Module Gate (validateConfig)
  - concrete check                    - Concrete check
  - extract metadata                  - Extract metadata
  - auto-secrets inject               - Finalize (Syntax+BuildExpr)
    ↓                                 - Extract DataComponents
pipeline.Render()                       ↓
  - provider.Match()                engine.Render()
  - matchPlan.Execute()               - buildMatchPlan() (CUE)
                                      - executeTransforms()
```

### Decision 5: Validation gates with structured error parsing

**Alternatives considered:**

1. Keep factory's raw `ConfigError` — minimal, loses field-level detail
2. Parse CUE errors into `FieldError`/`ConflictError` — user-facing diagnostics with file/line/path
3. New error type from scratch

**Decision:** Option 2. Gate validation calls `validateConfig()` (factory logic), then parses the resulting CUE error tree into `pkg/errors.FieldError` and `pkg/errors.ConflictError` types from the existing CLI error system.

**Rationale:** The current CLI already has the parsing logic in `internal/builder/validation.go`. The factory has the gate logic in `pkg/loader/validate.go`. Merging them produces gates with rich, actionable error output. The `ConfigError` type SHOULD be retained as an intermediate representation that carries the raw CUE error for programmatic access, with a method to parse into structured `FieldError` slices.

```go
// pkg/loader/validate.go
func validateConfig(schema, values cue.Value, context, name string) *ConfigError

// pkg/errors/domain.go (existing, moved from internal/errors)
type FieldError struct {
    File, Path, Message string
    Line, Column        int
}
type ConflictError struct {
    Path, Message string
    Locations     []ConflictLocation
}

// ConfigError gains a method:
func (e *ConfigError) FieldErrors() []FieldError  // parse RawError into structured fields
```

### Decision 6: Types in `pkg/` with typed accessors for ModuleRelease

**Decision:** `ModuleRelease` carries `Schema` and `DataComponents` behind typed accessors `MatchComponents()` and `ExecuteComponents()`, making incorrect usage a compile error instead of a runtime bug (DEBT.md fix).

```go
// pkg/modulerelease/release.go
type ModuleRelease struct {
    Metadata *ReleaseMetadata
    Module   module.Module

    schema         cue.Value  // unexported — access via methods
    dataComponents cue.Value  // unexported — access via methods
}

func (r *ModuleRelease) MatchComponents() cue.Value    // returns schema (has #resources, #traits)
func (r *ModuleRelease) ExecuteComponents() cue.Value   // returns finalized data (constraint-free)
```

### Decision 7: `Component` Go type eliminated

**Decision:** No `Component` struct in Go. CUE-native matching doesn't need Go to inspect `#resources`, `#traits`, or `#blueprints`. For display purposes (command output showing component names), the engine returns component summaries derived from the `MatchPlan` result — no need for a full Go type.

### Decision 8: Error package moves to `pkg/errors`

**Decision:** All existing error types (`DetailError`, `ExitError`, `TransformError`, `ValidationError`, `ValuesValidationError`, `FieldError`, `ConflictError`, sentinel errors) move from `internal/errors/` to `pkg/errors/`. Import alias `oerrors` SHOULD be used at call sites to avoid collision with stdlib `errors`.

### Decision 9: CUE boundary enforcement

**Decision:** After migration, CUE imports (`cuelang.org/go/cue`) MUST NOT appear in:

- `internal/kubernetes/`
- `internal/inventory/`
- `internal/output/` (uses `Resource` methods, not raw `cue.Value`)

The conversion boundary is `internal/cmdutil/render.go`, which calls `Resource.ToUnstructured()` and passes K8s-native types downstream.

### Decision 10: DEBT.md fixes during promotion

All 15 items from `experiments/factory/DEBT.md` MUST be fixed during promotion:

| # | Item | Fix |
|---|------|-----|
| 1 | Silent metadata decode errors | Propagate error; log at WARN in non-strict, return error in strict mode |
| 2 | `joinErrors` loses identity | Already fixed in factory (uses `errors.Join`) — carry forward |
| 3 | Non-deterministic Warnings() | Sort map keys and inner trait slices before iteration |
| 4 | Dead `pkgName` field | Remove field and stale comment from `pkg/module/` |
| 5 | Fail-fast/fail-slow asymmetry | Fail-slow at both levels (BundleRenderer collects all release errors) |
| 6 | Fragile `isSingleResource` | Define explicit CUE output contract; validate output shape |
| 7 | Loader has too many responsibilities | Split: `pkg/loader/` (loading), `pkg/loader/validate.go` (gates) — finalize stays in loader as it's intrinsic to loading |
| 8 | Dual Schema/DataComponents | Typed accessors: `MatchComponents()`, `ExecuteComponents()` |
| 9 | Dead `LoadRelease` function | Remove from `pkg/loader/` |
| 10 | Extension-based directory detection | Use `os.Stat()` + `IsDir()` in `resolveReleaseFile()` |
| 11 | `fmt.Printf` in business logic | Use `charmbracelet/log` for diagnostics; resource output via `io.Writer` |
| 12 | Value/pointer embedding inconsistency | Consistent pointer convention for metadata; value for small embedded structs |
| 13 | Duplicated extract pattern | Single `extractReleaseMetadata()` used from both module and bundle paths |
| 14 | `BundleRelease.Schema` never consumed | Remove field |
| 15 | `fmt.Printf` mixed into business logic | Already covered by #11 |

## Risks / Trade-offs

**[Big-bang scope]** 23+ files changing at once means the codebase won't compile during migration. → Mitigation: Work on a feature branch. The old packages and new `pkg/` packages coexist during development; deletion happens last. Run `task check` only after all adaptations are complete.

**[CUE evaluation performance]** CUE-native `#MatchPlan` loads the `./core/matcher` package and evaluates the full cartesian product. For modules with many components/transformers this could be slower than Go-side matching. → Mitigation: Factory benchmarks showed acceptable performance for current module sizes. Profile after migration if needed.

**[`pkg/errors` name collision]** The package name `errors` conflicts with stdlib. → Mitigation: Convention: `import oerrors "github.com/opmodel/cli/pkg/errors"` at all call sites. Document in AGENTS.md.

**[CUE boundary leakage]** `pkg/core.Resource` has `cue.Value` as a public field, which means any importer of `pkg/core` transitively depends on the CUE SDK. → Mitigation: Acceptable for now — `pkg/` types are explicitly CUE-aware. If needed later, `Resource` could expose only conversion methods and hide the `Value` field.

**[Test rewrite volume]** 5 test files construct core types as struct literals and will break completely. → Mitigation: These are mechanical rewrites — the test logic doesn't change, only the types used. Test helpers can reduce boilerplate.

**[Integration test dependency]** Integration tests require a running `kind` cluster. → Mitigation: Run `task cluster:create` before `task test:integration`. This is existing behavior, not new.
