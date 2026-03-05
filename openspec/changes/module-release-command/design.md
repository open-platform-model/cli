## Context

Today, `ModuleRelease` is an internal construct built ephemerally during the render pipeline's BUILD phase. The builder loads `#ModuleRelease` from `opmodel.dev/core@v1`, then injects the module, release name, namespace, and values via `FillPath`. Users interact only with modules (source files) and values (override files) — the release is never materialized as a file.

The CLI command surface splits `opm mod` into two archetypes:

- **Render-based** (build, vet, apply, diff): accept a module path, use `RenderFlags`, call the render pipeline
- **Cluster-query** (status, tree, events, delete, list): accept `ReleaseSelectorFlags` (--release-name/--release-id), use `K8sFlags`, query inventory Secrets

The cluster-query commands already operate on releases, not modules. They belong under a `rel` group.

### Prior Art: Module Import Experiment

The `experiments/module-import/` experiment (completed 2026-03-02) validated the core CUE mechanics this change depends on. Key findings:

1. **Flattened module embedding works with CUE imports** — modules using `core.#Module` embedded at package root (not nested under a named field) are fully importable via CUE's native module system. Hidden definitions (`#config`, `#components`) remain accessible across package boundaries.

2. **`#ModuleRelease` integration works** — the pattern `#module: importedModule` + `values: {...}` correctly unifies, resolves components with concrete values, and computes UUIDs/labels. This is exactly the mechanism release files will use.

3. **`values.cue` breaks importability** — when a module package includes `values.cue` at package root, the extra `values` field violates `#Module`'s closedness on import. **This means published modules must not include `values.cue`** — it stays as a local dev-only file. Module defaults belong in `#config` via `| *defaultValue` syntax.

4. **`debugValues` is unaffected** — the `debugValues` field is part of the `#Module` definition itself, so it survives import. This reinforces Decision 5 (`opm mod vet` using `debugValues`).

These findings directly inform the design: release files that import modules from a registry will work because `#module: importedModule` unification is proven. The `--module` flag (local dev) uses `FillPath` injection which bypasses the closedness constraint entirely (same as today's builder).

## Goals / Non-Goals

**Goals:**

- Introduce `opm release` command group (alias: `rel`) with file-based release rendering and cluster-query commands
- Design `opm release` as a polymorphic surface that handles both `#ModuleRelease` and `#BundleRelease`; implement `ModuleRelease` only in this change
- Enable predefined `<name>_release.cue` files that define `#ModuleRelease` with inline values
- Support hybrid module resolution: registry import or `--module` flag for local dev
- Improve UX with positional arguments: file path for render commands, name/UUID for cluster commands
- Make `opm mod vet` use `debugValues` by default (no values file needed for module validation)
- Alias `opm mod build/apply` to construct ephemeral releases using the same pipeline

**Non-Goals:**

- Multi-release files (one `#ModuleRelease` per file)
- Provider selection in release files (stays as flag/config concern)
- Bundle/multi-module orchestration (future work)
- CRD-based release management on cluster
- Release file discovery by convention (always explicit file path)

## Decisions

### Decision 1: Release file loading via CUE evaluation

**Choice**: Load `<name>_release.cue` as a CUE file using `load.Instances()`, evaluate it, and extract the `#ModuleRelease` value. When `#module` is already filled (via import), skip module loading. When `#module` is open, require `--module` flag and fill via `FillPath`.

**Alternatives considered**:

- *Compile release file with `ctx.CompileBytes()`*: Simpler but can't resolve CUE imports (registry modules). Since registry import is the primary use case, `load.Instances()` is required.
- *New release-specific CUE schema*: Unnecessary — `#ModuleRelease` from `opmodel.dev/core@v1` already defines the shape. Reuse it directly.

**Rationale**: This approach reuses existing CUE infrastructure and the catalog's `#ModuleRelease` definition. CUE handles import resolution, schema validation, and field computation (UUID, labels) natively. The `experiments/module-import/` experiment confirmed this pattern works end-to-end (see Context section).

#### Release file structure

**Important constraint from experiment**: The imported module package must NOT contain `values.cue` at package root — CUE's closedness check on `#Module` rejects the extra `values` field. Published modules rely on `#config` defaults and `debugValues` only. This is already the v1alpha1 convention.

A release file using a registry-imported module:

```cue
// jellyfin_release.cue
package releases

import (
    "opmodel.dev/core@v1"
    "opmodel.dev/modules/jellyfin@v1"
)

core.#ModuleRelease & {
    #module: jellyfin
    metadata: name:      "jellyfin"
    metadata: namespace: "media"
    values: {
        image: tag: "10.9.11"
        storage: config: size: "10Gi"
    }
}
```

A release file for local development (used with `--module`):

```cue
// jellyfin_release.cue
package releases

import "opmodel.dev/core@v1"

core.#ModuleRelease & {
    // #module: filled by --module flag
    metadata: name:      "jellyfin"
    metadata: namespace: "media"
    values: {
        image: tag: "10.9.11"
        storage: config: size: "10Gi"
    }
}
```

A deployment repo structure:

```
deployment-repo/
├── media/
│   ├── jellyfin_release.cue       # production jellyfin
│   └── plex_release.cue           # production plex
├── dev/
│   ├── jellyfin_release.cue       # dev jellyfin (different values)
│   └── blog_release.cue           # dev blog
└── cue.mod/
    └── module.cue                 # deps: opmodel.dev/modules/jellyfin@v1, etc.
```

#### Loader function signature

New function in `internal/loader/release.go`:

```go
// LoadRelease loads a #ModuleRelease from a CUE file.
//
// The file must define a top-level value that conforms to #ModuleRelease.
// CUE imports (including registry module references) are resolved via
// load.Instances() using the file's parent directory for cue.mod resolution.
//
// The returned cue.Value may have #module unfilled if the release file
// does not import a module. The caller is responsible for filling #module
// via FillPath when --module is provided.
//
// Returns the evaluated CUE value and the directory used for CUE resolution
// (needed later for loading opmodel.dev/core@v1 for auto-secrets).
func LoadRelease(ctx *cue.Context, filePath string, registry string) (cue.Value, string, error)
```

Loading mechanics:

1. Resolve `filePath` to absolute path, derive parent directory
2. Check that `cue.mod/` exists in the parent directory (or an ancestor)
3. Set `CUE_REGISTRY` env var if `registry` is non-empty (same pattern as `LoadModule`)
4. Call `load.Instances([]string{filePath}, &load.Config{Dir: parentDir})`
5. Build and evaluate the instance via `ctx.BuildInstance()`
6. Return the evaluated value — do NOT validate concreteness yet (caller handles that)

#### Detecting whether `#module` is filled

After loading, the caller checks if `#module` is concrete:

```go
moduleVal := releaseVal.LookupPath(cue.MakePath(cue.Def("module")))
moduleIsFilled := moduleVal.Exists() && moduleVal.Validate(cue.Concrete(true)) == nil
```

If `moduleIsFilled` is false and `--module` was not provided, return a clear error:
`"#module is not filled in the release file — either import a module or use --module <path>"`

### Decision 2: Positional argument semantics split by command type

**Choice**: Render commands (`vet`, `build`, `apply`, `diff`) interpret the positional arg as a `.cue` file path. Cluster-query commands (`status`, `tree`, `events`, `delete`) interpret it as a release identifier (name or UUID, auto-detected by format). `list` takes no positional arg.

**Alternatives considered**:

- *Unified positional arg with file detection*: Ambiguous — a name like `jellyfin` could be a file or a release name. Splitting by command type eliminates ambiguity.

**Rationale**: Each command type has a clear, unambiguous contract. Render commands always need a file. Cluster commands always need an identifier.

#### Command flag and arg summary

```
RENDER COMMANDS (positional arg = file path)
─────────────────────────────────────────────
opm release vet   <release.cue> [--module <path>] [--provider <name>] [-v]
opm release build <release.cue> [--module <path>] [--provider <name>] [-o dir]
opm release apply <release.cue> [--module <path>] [--provider <name>] [--kubeconfig] [--context] [--dry-run]
opm release diff  <release.cue> [--module <path>] [--provider <name>] [--kubeconfig] [--context]

CLUSTER COMMANDS (positional arg = name or UUID)
──────────────────────────────────────────────────
opm release status <name|uuid> [-n namespace] [--kubeconfig] [--context] [--wide]
opm release tree   <name|uuid> [-n namespace] [--kubeconfig] [--context]
opm release events <name|uuid> [-n namespace] [--kubeconfig] [--context]
opm release delete <name|uuid> [-n namespace] [--kubeconfig] [--context] [--dry-run]
opm release list              [-n namespace] [--kubeconfig] [--context]
```

### Decision 3: Release identifier auto-detection (name vs UUID)

**Choice**: For cluster-query commands, detect whether the positional arg is a UUID (matches hex-dash UUID pattern) or a release name. Use the appropriate lookup method (label scan for name, direct GET for UUID).

**Rationale**: Replaces the verbose `--release-name` / `--release-id` flags with a single positional arg. The formats are unambiguous — UUIDs have a distinct `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` pattern that won't collide with kebab-case release names.

#### Implementation

New helper in `internal/cmdutil/`:

```go
// ResolveReleaseIdentifier inspects the positional argument and returns
// either a release name or a release UUID. The detection is based on the
// UUID v4/v5 pattern: 8-4-4-4-12 hex digits separated by dashes.
func ResolveReleaseIdentifier(arg string) (name string, uuid string) {
    uuidPattern := regexp.MustCompile(
        `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
    )
    if uuidPattern.MatchString(arg) {
        return "", arg
    }
    return arg, ""
}
```

This integrates with the existing `cmdutil.ResolveInventory()` which already accepts either `ReleaseName` or `ReleaseID`:

```go
name, uuid := cmdutil.ResolveReleaseIdentifier(args[0])
rsf := cmdutil.ReleaseSelectorFlags{
    ReleaseName: name,
    ReleaseID:   uuid,
    Namespace:   namespace,
}
inv, liveResources, missing, err := cmdutil.ResolveInventory(ctx, client, &rsf, namespace, logger)
```

### Decision 4: `opm mod build/apply` as thin aliases

**Choice**: `opm mod build` and `opm mod apply` remain as commands but internally construct an ephemeral `#ModuleRelease` from flags (--values, --namespace, --release-name) and feed it through the same release pipeline. They are not literal aliases — they use the same underlying pipeline but with a different entry point (flags → ephemeral release vs. file → predefined release).

**Alternatives considered**:

- *Deprecate `mod build/apply`*: Too disruptive. Module authors need the quick `opm mod build .` workflow.
- *Literal alias (generate temp file)*: Unnecessary indirection. Share the pipeline, not the file format.

**Rationale**: Module authors keep their workflow. The pipeline is unified. No code duplication.

#### How the alias works internally

`opm mod build [path] -f values.cue -n production --release-name my-app` maps to:

```go
// In runModBuild (internal/cmd/mod/build.go):
// This is conceptually identical to today's flow — the pipeline
// still constructs the ephemeral #ModuleRelease internally.
// The "alias" means the underlying pipeline.Render() is the same
// code path that opm rel build eventually calls after loading a release file.
result, err := cmdutil.RenderRelease(ctx, cmdutil.RenderReleaseOpts{
    Args:        args,           // module path
    Values:      rf.Values,      // --values files
    ReleaseName: rf.ReleaseName, // --release-name
    K8sConfig:   k8sConfig,      // resolved namespace, provider
    Config:      cfg,
})
```

No behavior change from today. The `RenderRelease` helper continues to work exactly as before for module-based rendering. The new `RenderFromReleaseFile` helper (see below) provides the file-based path.

### Decision 5: `opm mod vet` uses `debugValues` by default

**Choice**: When `opm mod vet` is run without `-f` flags, the loader extracts `debugValues` from the module and uses it as the values source for the render pipeline. If `-f` is provided, it overrides and `debugValues` is ignored. If neither `debugValues` nor `-f` exists, error.

**Alternatives considered**:

- *Always require values*: Defeats the purpose of `debugValues` — module authors would still need a separate values file just to validate.

**Rationale**: `debugValues` exists precisely for this use case. Module authors embed test values in their module definition. `opm mod vet` should use them automatically for a zero-friction validation experience.

#### How debugValues extraction works

The module's `debugValues` is a CUE field on `#Module`:

```cue
// In catalog: v1alpha1/core/module.cue
#Module: {
    // ...
    debugValues: _   // open type, filled by module author
}
```

Module authors provide concrete values:

```cue
// In module.cue
debugValues: {
    web: replicas: 1
    web: image: { repository: "nginx", tag: "1.20.0", digest: "" }
    db: image: { repository: "postgres", tag: "14.0", digest: "" }
    db: volumeSize: "5Gi"
    db: password: value: "dev-password"
    db: host: value: "localhost"
}
```

Extraction in the builder:

```go
// New option in builder.Options:
type Options struct {
    Name          string
    Namespace     string
    DebugValues   bool   // when true, extract debugValues from module
}

// In Build(), before step 3 (resolve values files):
if opts.DebugValues {
    debugVal := mod.Raw.LookupPath(cue.ParsePath("debugValues"))
    if !debugVal.Exists() {
        return nil, fmt.Errorf("module does not define debugValues")
    }
    // Check that debugValues is not just `_` (open/unconstrained)
    if err := debugVal.Validate(cue.Concrete(true)); err != nil {
        return nil, &opmerrors.ValidationError{
            Message: "debugValues is not concrete — module must provide complete test values",
            Cause:   err,
        }
    }
    selectedValues = debugVal
    // Skip steps 3-4 (resolveValuesFiles, ValidateValues)
    // Proceed directly to step 5 (FillPath)
}
```

The `opm mod vet` command wires this:

```go
// In runVet (internal/cmd/mod/vet.go):
useDebugValues := len(rf.Values) == 0  // no -f flag provided

result, err := cmdutil.RenderRelease(ctx, cmdutil.RenderReleaseOpts{
    Args:        args,
    Values:      rf.Values,
    ReleaseName: rf.ReleaseName,
    K8sConfig:   k8sConfig,
    Config:      cfg,
    DebugValues: useDebugValues,  // new field
})
```

This requires threading `DebugValues` through `RenderReleaseOpts` → `RenderOptions` → `builder.Options`.

### Decision 6: `--module` flag for local module injection

**Choice**: Render commands under `opm rel` accept `--module <path>` flag. When provided, the CLI loads the module from the local directory and fills `#module` via `FillPath`, exactly as today's builder does. When not provided, `#module` must be filled via CUE import in the release file — if it's still open, CUE's concreteness check will report the error.

**Rationale**: Enables local development without publishing modules to a registry. Same mechanism as today's builder, just exposed as a flag.

#### FillPath injection for --module

```go
// In the release render path (new cmdutil.RenderFromReleaseFile):
if moduleFlag != "" {
    // Load module from local directory (same as pipeline.prepare)
    mod, err := loader.LoadModule(cueCtx, moduleFlag, registry)
    if err != nil {
        return nil, err
    }
    if err := mod.Validate(); err != nil {
        return nil, err
    }
    // Inject #module into the release CUE value
    releaseVal = releaseVal.FillPath(cue.MakePath(cue.Def("module")), mod.Raw)
    if err := releaseVal.Err(); err != nil {
        return nil, fmt.Errorf("filling #module from --module: %w", err)
    }
}
```

### Decision 7: Command migration strategy

**Choice**: Migrate cluster-query commands (status, tree, events, delete, list) to `opm release`. Keep `opm mod` versions as aliases that delegate to `opm release` during a deprecation period, printing a notice suggesting the `release` equivalent.

**Alternatives considered**:

- *Immediate removal from `mod`*: Breaking change, violates SemVer MINOR commitment.
- *Keep in both permanently*: Confusing — two paths to the same thing with no signal about the canonical one.

**Rationale**: Gradual migration. Users discover `opm release` naturally. Aliases prevent breakage.

#### Deprecation notice pattern

```go
// In internal/cmd/mod/status.go (deprecated version):
func NewModStatusCmd(cfg *config.GlobalConfig) *cobra.Command {
    var rsf cmdutil.ReleaseSelectorFlags
    var kf cmdutil.K8sFlags

    c := &cobra.Command{
        Use:        "status",
        Short:      "Show status of a deployed release (use 'opm release status' instead)",
        Deprecated: "use 'opm release status <name>' instead",
        RunE: func(c *cobra.Command, args []string) error {
            // Delegate to the same logic as opm release status
            return runRelStatus(args, cfg, &rsf, &kf)
        },
    }
    rsf.AddTo(c)
    kf.AddTo(c)
    return c
}
```

Cobra's built-in `Deprecated` field prints `Command "status" is deprecated, use 'opm release status <name>' instead` and still executes the command.

### Decision 8: New package structure

```
internal/cmd/release/
    release.go          # NewReleaseCmd() — group container (Use: "release", Aliases: ["rel"])
    vet.go              # opm release vet <release.cue>
    build.go            # opm release build <release.cue>
    apply.go            # opm release apply <release.cue>
    diff.go             # opm release diff <release.cue>
    status.go           # opm release status <name|uuid>
    tree.go             # opm release tree <name|uuid>
    events.go           # opm release events <name|uuid>
    delete.go           # opm release delete <name|uuid>
    list.go             # opm release list

internal/loader/
    release.go          # LoadRelease() — load #ModuleRelease from .cue file

internal/cmdutil/
    flags.go            # ReleaseFileFlags (--module), updated ReleaseSelectorFlags
    release.go          # RenderFromReleaseFile() — shared release render preamble
```

#### New flag group

```go
// ReleaseFileFlags holds flags specific to release-file-based rendering.
type ReleaseFileFlags struct {
    Module   string   // --module <path> for local module injection
    Provider string   // --provider <name>
}

func (f *ReleaseFileFlags) AddTo(cmd *cobra.Command) {
    cmd.Flags().StringVar(&f.Module, "module", "",
        "Path to local module directory (fills #module in the release file)")
    cmd.Flags().StringVar(&f.Provider, "provider", "",
        "Provider to use (default: from config)")
}
```

#### Shared release render preamble

New function in `internal/cmdutil/release.go`:

```go
// RenderFromReleaseFileOpts holds inputs for RenderFromReleaseFile.
type RenderFromReleaseFileOpts struct {
    // ReleaseFilePath is the path to the .cue release file (required).
    ReleaseFilePath string
    // ModulePath is the path to a local module directory (optional, from --module).
    ModulePath string
    // K8sConfig is the pre-resolved Kubernetes configuration.
    K8sConfig *config.ResolvedKubernetesConfig
    // Config is the fully loaded global configuration.
    Config *config.GlobalConfig
}

// RenderFromReleaseFile loads a release file, optionally injects a local
// module, and executes the render pipeline. This is the release-file
// equivalent of RenderRelease (which does module-based rendering).
func RenderFromReleaseFile(ctx context.Context, opts RenderFromReleaseFileOpts) (*pipeline.RenderResult, error)
```

#### Pipeline integration

The new `RenderFromReleaseFile` function needs to interface with the existing pipeline. Two approaches:

**Option A: Extend Pipeline.Render() with a new RenderOptions field**

```go
type RenderOptions struct {
    // ... existing fields ...

    // ReleaseValue is a pre-evaluated #ModuleRelease CUE value.
    // When set, the pipeline skips PREPARATION and BUILD phases
    // and uses this value directly.
    ReleaseValue *cue.Value
}
```

**Option B: Build the ModuleRelease outside the pipeline, then call Match+Generate**

```go
// In RenderFromReleaseFile:
// 1. Load release file
releaseVal, resolveDir, err := loader.LoadRelease(cueCtx, filePath, registry)
// 2. Optionally fill #module
if modulePath != "" { /* FillPath */ }
// 3. Build ModuleRelease from the pre-filled CUE value
rel, err := builder.BuildFromRelease(cueCtx, releaseVal, resolveDir)
// 4. Load provider
provider, err := loader.LoadProvider(cueCtx, providerName, providers)
// 5. Match + Generate (same as pipeline phases 4+5)
matchPlan := provider.Match(rel.Components)
resources, errs := matchPlan.Execute(ctx, rel)
```

**Choice: Option A.** Extending `RenderOptions` keeps the pipeline as the single orchestrator. Option B duplicates phase orchestration logic. With Option A, `pipeline.Render()` checks if `ReleaseValue` is set and skips phases 1+3, but still runs provider load (phase 2), matching (phase 4), and generate (phase 5) through the same code path.

The modified pipeline flow:

```
pipeline.Render(ctx, opts)
│
├── opts.ReleaseValue != nil ?
│   │
│   ├── YES (release-file path):
│   │   ├── Detect release type (ModuleRelease or BundleRelease)
│   │   ├── BundleRelease → error "bundle releases not yet supported"
│   │   ├── ModuleRelease:
│   │   │   ├── Skip Phase 1 (PREPARATION)
│   │   │   ├── Phase 2: PROVIDER LOAD
│   │   │   ├── Phase 3 alt: builder.BuildFromRelease(ctx, *opts.ReleaseValue, opts.ModulePath)
│   │   │   │     → validates concreteness
│   │   │   │     → extracts metadata, components, autoSecrets
│   │   │   │     → returns *modulerelease.ModuleRelease
│   │   │   ├── Phase 4: MATCHING
│   │   │   └── Phase 5: GENERATE
│   │
│   └── NO (module-directory path, existing behavior):
│       ├── Phase 1: PREPARATION (loader.LoadModule)
│       ├── Phase 2: PROVIDER LOAD
│       ├── Phase 3: BUILD (builder.Build with FillPath)
│       ├── Phase 4: MATCHING
│       └── Phase 5: GENERATE
```

#### Builder changes

New function in `internal/builder/builder.go`:

```go
// BuildFromRelease creates a *modulerelease.ModuleRelease from a pre-evaluated
// #ModuleRelease CUE value (loaded from a release file).
//
// Unlike Build(), this does NOT construct the #ModuleRelease via FillPath.
// The release value is expected to already have #module, metadata, and values
// filled (either by CUE import or by the caller via FillPath for --module).
//
// The function:
//  1. Validates full concreteness of the release value
//  2. Extracts release metadata (uuid, labels, annotations)
//  3. Extracts components via component.ExtractComponents()
//  4. Injects auto-secrets if autoSecrets is non-empty
//  5. Extracts module metadata for the ModuleRelease.Module field
//  6. Returns *modulerelease.ModuleRelease
//
// resolveDir is the directory containing the release file's cue.mod/,
// needed for loading opmodel.dev/resources/config@v1 during auto-secrets injection.
func BuildFromRelease(ctx *cue.Context, releaseVal cue.Value, resolveDir string) (*modulerelease.ModuleRelease, error)
```

This reuses steps 6-8 from the existing `Build()` function. The shared code can be extracted into internal helpers:

```go
// Shared between Build() and BuildFromRelease():
func validateConcreteness(result cue.Value) error
func extractReleaseMetadata(result cue.Value, opts Options) (*modulerelease.ReleaseMetadata, error)  // already exists
func extractModuleMetadata(result cue.Value) (*module.ModuleMetadata, error)  // new: reads from #module in release
func injectAutoSecrets(ctx *cue.Context, result cue.Value, resolveDir string, components map[string]*component.Component) error  // already exists
```

### Decision 9: `opm mod vet` vs `opm rel vet` — distinct validation semantics

These serve different audiences with different questions:

```
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  opm mod vet [path]              opm release vet <release.cue>       │
│  ──────────────────              ───────────────────────────────     │
│  Audience: module author         Audience: platform operator         │
│  Question: "is my module         Question: "is this release          │
│    well-formed?"                   complete and deployable?"         │
│                                                                      │
│  Values source:                  Values source:                      │
│    debugValues (default)           inline in release file            │
│    or -f override                                                    │
│                                                                      │
│  Module source:                  Module source:                      │
│    local directory [path]          registry import or --module       │
│                                                                      │
│  Pipeline path:                  Pipeline path:                      │
│    RenderRelease() with            RenderFromReleaseFile()           │
│    DebugValues: true                                                 │
│                                                                      │
│  Output:                         Output:                             │
│    per-resource validation         per-resource validation           │
│    "Module valid (N resources)"    "Release valid (N resources)"     │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### Decision 10: Polymorphic release type detection

**Choice**: `opm release` is designed as a polymorphic command surface that handles both `#ModuleRelease` and `#BundleRelease` release files. This change implements `ModuleRelease` only. `BundleRelease` files are detected and return a clear "not yet supported" error, establishing the extensibility contract for when bundle CLI support is added.

**Alternatives considered**:

- *Separate `opm release` and `opm bundle-release` groups*: Unambiguous, but creates a two-top-level-commands problem — users would need to know which command to use before opening a file. The file itself describes its type; the command shouldn't require pre-knowledge.
- *Single monolithic `rel` command (original design)*: `rel` is ambiguous once bundles arrive — no indication whether the group handles one or both release types.

**Rationale**: The command group name `release` is the right abstraction level — it describes *what you're managing* (releases), not *which kind*. The CUE file is self-describing; the loader detects the type and routes accordingly. This aligns with Principle I (Type Safety First) — the CUE type system determines behavior, not flags or command names.

#### Cobra command definition

```go
// In internal/cmd/release/release.go:
c := &cobra.Command{
    Use:     "release",
    Aliases: []string{"rel"},
    Short:   "Release operations",
    Long:    `Release operations for OPM releases.`,
}
```

#### Type detection mechanism

After loading the CUE file via `load.Instances()`, the loader unifies the evaluated value against the catalog's `#ModuleRelease` and `#BundleRelease` definitions to determine the release type:

```go
// ReleaseType identifies the kind of release loaded from a .cue file.
type ReleaseType int

const (
    ModuleRelease ReleaseType = iota
    BundleRelease
)

// LoadRelease loads a release file and returns the evaluated CUE value
// along with the detected release type.
func LoadRelease(ctx *cue.Context, filePath string, registry string) (cue.Value, ReleaseType, string, error) {
    // ... load.Instances(), evaluate ...

    // Detect type by checking kind field (fastest path — no schema load needed)
    kindVal := val.LookupPath(cue.ParsePath("kind"))
    if kindVal.Exists() {
        kind, _ := kindVal.String()
        switch kind {
        case "ModuleRelease":
            return val, ModuleRelease, resolveDir, nil
        case "BundleRelease":
            return val, BundleRelease, resolveDir, nil
        }
    }

    return cue.Value{}, 0, "", fmt.Errorf("release file does not define a recognised release type (expected ModuleRelease or BundleRelease)")
}
```

The `kind` field is the most direct discriminator — both `#ModuleRelease` and `#BundleRelease` in the OPM catalog define `kind` as a concrete string literal, making this a reliable and cheap check.

#### Command-layer enforcement (ModuleRelease only)

All render commands check the returned `ReleaseType` before proceeding:

```go
releaseVal, releaseType, resolveDir, err := loader.LoadRelease(cueCtx, filePath, registry)
if err != nil {
    return err
}
if releaseType == loader.BundleRelease {
    return fmt.Errorf("bundle releases are not yet supported — use a #ModuleRelease file")
}
// continue with ModuleRelease path...
```

The `--module` flag is also gated: if `releaseType == BundleRelease`, return an error early rather than silently ignoring the flag.

#### Extensibility contract for future bundle support

When bundle CLI support is added, the `BundleRelease` path in each command is the extension point:

- **Render commands**: The `BundleRelease` path will resolve the bundle into N `ModuleRelease` CUE values and run N pipeline executions. The `--module` flag does not apply (bundle modules are always registry-imported or defined inline).
- **Cluster-query commands**: Will add a bundle inventory lookup alongside the existing module release inventory lookup. Output format will differ (aggregate view for bundles, single-release view for modules). The positional arg auto-detection (`name` vs `UUID`) is already shared logic — the bundle path extends it with a second inventory search.

## Risks / Trade-offs

**[CUE registry resolution complexity]** → Loading release files with `load.Instances()` requires a `cue.mod/` directory with proper dependency declarations. Deployment repos need their own `cue.mod/module.cue` with module dependencies. Mitigation: document the deployment repo setup pattern clearly. Consider a future `opm rel init` command that scaffolds a deployment repo.

**[Module publishing must exclude values.cue]** → Per `experiments/module-import/` findings, modules published to a CUE registry must not include `values.cue` at package root — the extra `values` field violates `#Module` closedness on import. Mitigation: this is already the v1alpha1 convention (defaults live in `#config`, test values in `debugValues`). Document clearly in module authoring guidelines. The `--module` flag (local dev path) uses FillPath injection which bypasses this constraint entirely.

**[Two ways to do the same thing]** → `opm mod apply` and `opm rel apply <file>` both deploy releases. Mitigation: clear documentation that `mod` is the quick-start path, `rel` is the production/GitOps path. Deprecation notices guide migration for cluster-query commands.

**[debugValues may not cover all validation paths]** → A module author's `debugValues` might not exercise secrets, optional fields, or edge cases. Mitigation: `debugValues` is the default but `-f` still works for thorough validation. `debugValues` is "does my module compile" not "is my production config correct."

**[Positional arg requires release name uniqueness per namespace]** → Release name lookup via label scan may return multiple matches if names aren't unique. Mitigation: the inventory system already enforces name+namespace uniqueness via the Secret naming convention.

**[Pipeline interface expansion]** → Adding `ReleaseValue` to `RenderOptions` changes the pipeline interface. Mitigation: the field is optional — existing callers are unaffected. The `Pipeline` interface itself doesn't change (still `Render(ctx, RenderOptions)`), only the options struct gains a new optional field.

**[Polymorphic surface may confuse users]** → Users might not know whether a given command behaves differently for bundle vs module release files. Mitigation: clear error messages when a `BundleRelease` file is detected ("bundle releases are not yet supported — use a `#ModuleRelease` file"), preventing silent misbehaviour. Future docs will explain the behavioral differences between module and bundle releases.

**[BuildFromRelease code duplication]** → The new function shares logic with `Build()`. Mitigation: extract shared steps (concreteness validation, metadata extraction, auto-secrets injection) into internal helpers used by both functions.

## Open Questions

- **`opm rel init`?** Should there be a scaffold command for deployment repos (creates `cue.mod/`, example release file)? Deferred to a future change.
- **Release file validation without provider?** `opm rel vet` currently needs a provider for matching. Should there be a `--skip-matching` flag that only validates the release CUE structure and values? Deferred — evaluate after initial implementation.
