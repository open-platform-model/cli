## Context

`ModuleRelease` is currently an internal, ephemeral construct — built in-memory during the render pipeline and never exposed to users. The CLI command surface splits `opm mod` into two archetypes:

- **Render-based** (build, vet, apply, diff): accept a module path, use `RenderFlags`, call `cmdutil.RenderRelease()`
- **Cluster-query** (status, tree, events, delete, list): accept `ReleaseSelectorFlags` (--release-name/--release-id), use `K8sFlags`, query inventory Secrets

The cluster-query commands already operate on releases, not modules. They belong under a `release` group.

### Post-refactor Architecture (promote-factory-engine)

The `promote-factory-engine` change replaced `internal/builder/`, `internal/pipeline/`, `internal/loader/`, and `internal/core/{component,transformer,provider}` with a clean `pkg/` surface. This directly affects how this change is implemented:

**Packages that no longer exist** (and what replaced them):

```text
GONE                            REPLACEMENT
──────────────────────────────────────────────────────────────────
internal/builder/               pkg/loader/ (validation gates)
internal/pipeline/              pkg/engine/ + internal/cmdutil/render.go
internal/loader/                pkg/loader/
internal/core/component/        (eliminated — CUE-native matching)
internal/core/transformer/      pkg/engine/execute.go
internal/core/provider/         pkg/provider/ (thin wrapper, no Match())
```

**The render orchestration layer** is now `internal/cmdutil/render.go`, which calls:

```text
pkg/loader.LoadReleasePackage()       → load CUE instance
pkg/loader.DetectReleaseKind()        → "ModuleRelease" or "BundleRelease"
pkg/loader.LoadModuleReleaseFromValue() → decode + validate + finalize
pkg/loader.LoadProvider()             → wrap provider CUE value
pkg/engine.ModuleRenderer.Render()    → CUE-native match + execute
resource.ToUnstructured()             → convert at cmdutil boundary
```

**Key implications for this change:**

- No builder to modify for `debugValues` — extraction moves to `cmdutil` layer
- No pipeline phases to skip — write a parallel `RenderFromReleaseFile()` in `cmdutil`
- `BuildFromRelease()` is unnecessary — `LoadModuleReleaseFromValue()` already does this
- `loader.LoadModule()` is gone — need a new `pkg/loader.LoadModulePackage()` for `--module` flag
- `DetectReleaseKind()` already exists and works exactly as designed
- Auto-secrets are free — CUE `#AutoSecrets` handles them in the loader

### Prior Art: Module Import Experiment

The `experiments/module-import/` experiment (completed 2026-03-02) validated the core CUE mechanics this change depends on. Key findings:

1. **Flattened module embedding works with CUE imports** — modules using `core.#Module` embedded at package root are fully importable via CUE's native module system. Hidden definitions (`#config`, `#components`) remain accessible across package boundaries.

2. **`#ModuleRelease` integration works** — the pattern `#module: importedModule` + `values: {...}` correctly unifies, resolves components with concrete values, and computes UUIDs/labels.

3. **`values.cue` breaks importability** — when a module package includes `values.cue` at package root, the extra `values` field violates `#Module`'s closedness on import. Published modules must not include `values.cue`. Module defaults belong in `#config` via `| *defaultValue` syntax.

4. **`debugValues` is unaffected** — the `debugValues` field is part of the `#Module` definition itself, so it survives import.

## Goals / Non-Goals

**Goals:**

- Introduce `opm release` command group (alias: `rel`) with file-based release rendering and cluster-query commands
- Design `opm release` as a polymorphic surface that handles both `#ModuleRelease` and `#BundleRelease`; implement `ModuleRelease` only in this change
- Enable predefined `<name>_release.cue` files that define `#ModuleRelease` with inline values
- Support hybrid module resolution: registry import or `--module` flag for local dev
- Improve UX with positional arguments: file path for render commands, name/UUID for cluster commands
- Make `opm mod vet` use `debugValues` by default (no values file needed for module validation)

**Non-Goals:**

- Multi-release files (one `#ModuleRelease` per file)
- Provider selection in release files (stays as flag/config concern)
- Bundle/multi-module orchestration (future work)
- CRD-based release management on cluster
- Release file discovery by convention (always explicit file path)
- Modifying `pkg/loader/` or `pkg/engine/` internals beyond adding new functions

## Decisions

### Decision 1: Release file loading lives in `pkg/loader/`

**Choice**: The new `LoadReleaseFile()` function belongs in `pkg/loader/release_file.go`, not in `internal/loader/` (which no longer exists). It uses `load.Instances()` with the file's parent directory for `cue.mod` resolution, enabling registry module imports.

**Alternatives considered**:

- *`internal/loader/release.go`*: The package is gone. All loader functions live in `pkg/loader/`.
- *Compile release file with `ctx.CompileBytes()`*: Can't resolve CUE imports. Registry import is the primary use case.

**Rationale**: Consistent with the new package structure. `pkg/loader/` already has `LoadReleasePackage()` (module directory path) — `LoadReleaseFile()` is the parallel function for standalone `.cue` files.

#### How the two loader paths differ

```
LoadReleasePackage(ctx, dir, valuesFile)       LoadReleaseFile(ctx, filePath, registry)
────────────────────────────────────────       ───────────────────────────────────────
Input: module directory                        Input: standalone .cue file path
Loads: release.cue + values.cue               Loads: single .cue file with imports
Use:   opm mod build/vet/apply/diff            Use:   opm release build/vet/apply/diff
cue.mod: inside module directory               cue.mod: deployment repo (parent chain)
Registry: from config                          Registry: from config (env var)
```

Both return a `cue.Value` that can be passed directly to `DetectReleaseKind()` and `LoadModuleReleaseFromValue()` — those functions are agnostic to how the value was loaded.

#### Loader function signature

New function in `pkg/loader/release_file.go`:

```go
// LoadReleaseFile loads a #ModuleRelease or #BundleRelease from a standalone
// .cue file. CUE imports (including registry module references) are resolved
// via load.Instances() using the file's parent directory for cue.mod resolution.
//
// The returned cue.Value may have #module unfilled if the release file does not
// import a module. The caller is responsible for filling #module via FillPath
// when --module is provided.
//
// Returns the evaluated CUE value and the directory used for CUE resolution.
func LoadReleaseFile(ctx *cue.Context, filePath string, registry string) (cue.Value, string, error)
```

Loading mechanics:

1. Resolve `filePath` to absolute path, derive parent directory
2. Set `CUE_REGISTRY` env var if `registry` is non-empty
3. Call `load.Instances([]string{filePath}, &load.Config{Dir: parentDir})`
4. Build and evaluate the instance via `ctx.BuildInstance()`
5. Return the evaluated value — do NOT validate concreteness yet

#### Release file structure

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

**Rationale**: Replaces the verbose `--release-name` / `--release-id` flags with a single positional arg. The formats are unambiguous.

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

### Decision 4: `opm mod build/apply` require no changes

**Context**: The original design said `opm mod build/apply` would be refactored to "delegate to the release pipeline". After `promote-factory-engine`, this is already true — both commands call `cmdutil.RenderRelease()` which orchestrates `pkg/loader` and `pkg/engine` directly. The pipeline is unified.

**Decision**: No changes to `internal/cmd/mod/build.go` or `internal/cmd/mod/apply.go` for this purpose. The "alias" is already implemented.

### Decision 5: `debugValues` extraction at the `cmdutil` layer

**Context**: The original design placed `debugValues` extraction in `builder.Options`. The builder no longer exists.

**Choice**: Add `DebugValues bool` to `RenderReleaseOpts` in `internal/cmdutil/render.go`. When set, `RenderRelease()` extracts `debugValues` from the loaded module CUE package before calling `LoadModuleReleaseFromValue()`.

**Alternatives considered**:

- *`pkg/loader/` function*: Would require exposing a module-extraction function that's only needed for one CLI option. Better kept at the cmdutil layer as a CLI concern.
- *New `LoadReleasePackageWithDebugValues()`*: Leaks CLI-level concept into the loader. Rejected.

**Rationale**: `RenderRelease()` already orchestrates the full pipeline. The `DebugValues` flag is a CLI-level concern (replacing `--values` flags with module-embedded values). It belongs at the orchestration layer, not in the loader.

#### How debugValues extraction works

The module's `debugValues` is a CUE field on `#Module`. Module authors provide concrete values:

```cue
// In module.cue
debugValues: {
    web: replicas: 1
    web: image: { repository: "nginx", tag: "1.20.0", digest: "" }
    db: image: { repository: "postgres", tag: "14.0", digest: "" }
    db: volumeSize: "5Gi"
}
```

Extraction in `cmdutil.RenderRelease()`:

```go
// RenderReleaseOpts gains:
type RenderReleaseOpts struct {
    // ... existing fields ...
    DebugValues bool   // when true, extract debugValues from module instead of loading values files
}

// In RenderRelease(), before calling LoadReleasePackage():
if opts.DebugValues {
    // Load the module CUE package to extract debugValues
    modPkg, err := loader.LoadModulePackage(cueCtx, modulePath)
    if err != nil {
        return nil, &oerrors.ExitError{...}
    }
    debugVal := modPkg.LookupPath(cue.ParsePath("debugValues"))
    if !debugVal.Exists() {
        return nil, &oerrors.ExitError{..., Err: fmt.Errorf("module does not define debugValues")}
    }
    if err := debugVal.Validate(cue.Concrete(true)); err != nil {
        return nil, &oerrors.ExitError{..., Err: &oerrors.ValidationError{
            Message: "debugValues is not concrete — module must provide complete test values",
            Cause:   err,
        }}
    }
    // Write debugValues to a temp file and use it as the values source
    // OR pass the cue.Value directly through a new LoadReleasePackageWithValues() variant
    // Implementation detail: TBD during task 2
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
    DebugValues: useDebugValues,
})
```

### Decision 6: `--module` flag uses a new `LoadModulePackage()` loader function

**Context**: The original design called `loader.LoadModule()` for `--module` flag injection. `internal/loader/` and its `LoadModule()` function no longer exist.

**Choice**: Add `LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error)` to `pkg/loader/release_file.go`. This loads a module CUE package from a directory and returns the raw `cue.Value` for `FillPath` injection. It is the successor to `loader.LoadModule()` for this specific use case.

**Rationale**: The `pkg/loader/` package already handles all CUE loading. This is a small, focused addition that keeps the loading concern in one place.

#### New function signature

```go
// LoadModulePackage loads a module CUE package from a directory and returns
// the raw cue.Value. Used by the --module flag to inject a local module into
// a release file that does not import one from a registry.
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error)
```

#### FillPath injection for `--module` in `RenderFromReleaseFile()`

```go
// In RenderFromReleaseFile (internal/cmdutil/render.go):
if opts.ModulePath != "" {
    modVal, err := loader.LoadModulePackage(cueCtx, opts.ModulePath)
    if err != nil {
        return nil, &oerrors.ExitError{..., Err: fmt.Errorf("loading module from --module: %w", err)}
    }
    releaseVal = releaseVal.FillPath(cue.MakePath(cue.Def("module")), modVal)
    if err := releaseVal.Err(); err != nil {
        return nil, &oerrors.ExitError{..., Err: fmt.Errorf("filling #module from --module: %w", err)}
    }
}
```

### Decision 7: Command migration strategy

**Choice**: Migrate cluster-query commands (status, tree, events, delete, list) to `opm release`. Keep `opm mod` versions as deprecated aliases that delegate to `opm release` during a deprecation period.

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
            return runRelStatus(args, cfg, &rsf, &kf)
        },
    }
    rsf.AddTo(c)
    kf.AddTo(c)
    return c
}
```

Cobra's built-in `Deprecated` field prints `Command "status" is deprecated, use 'opm release status <name>' instead` and still executes the command.

### Decision 8: `RenderFromReleaseFile()` is a parallel orchestration function — no pipeline extension needed

**Context**: The original design proposed extending `Pipeline.Render()` with a `ReleaseValue *cue.Value` field to skip certain phases. The `Pipeline` interface no longer exists. The orchestration layer is `internal/cmdutil/render.go`.

**Choice**: Add `RenderFromReleaseFile(ctx, opts) (*RenderResult, error)` to `internal/cmdutil/render.go` as a parallel function to `RenderRelease()`. It calls the same `pkg/` functions but with a different loading entry point.

**Rationale**: The phase-skipping complexity was the entire motivation for the Option A/Option B debate in the original design. Without a pipeline, it evaporates. The two orchestration functions differ only in step 1 (how the CUE value is loaded) and step 3 (optional `--module` injection). Steps 2, 4, 5, 6, and 7 are identical and can share helpers.

#### New package structure

```
pkg/loader/
    release_file.go     # LoadReleaseFile() + LoadModulePackage() — NEW

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

internal/cmdutil/
    flags.go            # ReleaseFileFlags (--module) — UPDATED
    render.go           # RenderFromReleaseFile() + DebugValues in RenderReleaseOpts — UPDATED
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

#### `RenderFromReleaseFile()` — the new orchestration function

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

// RenderFromReleaseFile loads a release file, optionally injects a local module,
// and executes the render pipeline. This is the release-file equivalent of
// RenderRelease() (which does module-directory rendering).
//
// Pipeline:
//   1. loader.LoadReleaseFile()              load .cue file with import resolution
//   2. loader.DetectReleaseKind()            error on BundleRelease (not yet supported)
//   3. [optional] loader.LoadModulePackage() + FillPath for --module flag
//   4. loader.LoadModuleReleaseFromValue()   validate + extract *ModuleRelease
//   5. loader.LoadProvider()                 wrap provider CUE value
//   6. engine.ModuleRenderer.Render()        CUE-native match + execute
//   7. Resource.ToUnstructured()             convert at cmdutil boundary
func RenderFromReleaseFile(ctx context.Context, opts RenderFromReleaseFileOpts) (*RenderResult, error)
```

#### How it compares to `RenderRelease()`

```
RenderRelease()                          RenderFromReleaseFile()
───────────────────────────────────      ─────────────────────────────────────────
1. resolve module path (args[0] or ".")  1. use ReleaseFilePath directly
2. LoadReleasePackage(dir, valuesFile)   2. LoadReleaseFile(filePath, registry)
   (dir contains release.cue+values.cue)    (standalone .cue with CUE imports)
   [debugValues: load module + extract]  3. [optional] LoadModulePackage()+FillPath
3. DetectReleaseKind()                   4. DetectReleaseKind()
4. LoadModuleReleaseFromValue()          5. LoadModuleReleaseFromValue()
5. LoadProvider()                        6. LoadProvider()
6. ModuleRenderer.Render()               7. ModuleRenderer.Render()
7. Resource.ToUnstructured()             8. Resource.ToUnstructured()
─── returns *RenderResult ────────────   ─── returns *RenderResult ──────────────
```

Steps 4–7 / 5–8 are identical. Only the loading entry point differs.

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
│  Orchestration:                  Orchestration:                      │
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

**Choice**: `opm release` handles both `#ModuleRelease` and `#BundleRelease` files. This change implements `ModuleRelease` only. `BundleRelease` files are detected and return a clear "not yet supported" error.

**Type detection**: `pkg/loader.DetectReleaseKind()` already exists and reads the `kind` field. No new code needed for detection — just enforce the guard at the command layer:

```go
releaseVal, resolveDir, err := loader.LoadReleaseFile(cueCtx, filePath, registry)
if err != nil {
    return err
}
kind, err := loader.DetectReleaseKind(releaseVal)
if err != nil {
    return err
}
if kind == "BundleRelease" {
    return fmt.Errorf("bundle releases are not yet supported — use a #ModuleRelease file")
}
// continue with ModuleRelease path...
```

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

#### Extensibility contract for future bundle support

When bundle CLI support is added, the `kind == "BundleRelease"` branch is the extension point. The `pkg/loader.DetectReleaseKind()` function already handles both kinds — only the command-layer guard needs to be replaced with actual bundle handling.

## Risks / Trade-offs

**[CUE registry resolution complexity]** → Loading release files with `load.Instances()` requires a `cue.mod/` directory with proper dependency declarations. Deployment repos need their own `cue.mod/module.cue` with module dependencies. Mitigation: document the deployment repo setup pattern clearly. Consider a future `opm rel init` command.

**[Module publishing must exclude values.cue]** → Per `experiments/module-import/` findings, published modules must not include `values.cue` at package root. Mitigation: this is already the v1alpha1 convention. The `--module` flag (local dev path) uses FillPath injection which bypasses this constraint entirely.

**[Two ways to do the same thing]** → `opm mod apply` and `opm rel apply <file>` both deploy releases. Mitigation: clear documentation that `mod` is the quick-start path, `rel` is the production/GitOps path. Deprecation notices guide migration for cluster-query commands.

**[debugValues may not cover all validation paths]** → A module author's `debugValues` might not exercise secrets, optional fields, or edge cases. Mitigation: `debugValues` is the default but `-f` still works for thorough validation.

**[Positional arg requires release name uniqueness per namespace]** → Release name lookup via label scan may return multiple matches if names aren't unique. Mitigation: the inventory system already enforces name+namespace uniqueness via the Secret naming convention.

**[Polymorphic surface may confuse users]** → Users might not know whether a given command behaves differently for bundle vs module release files. Mitigation: clear error messages when a `BundleRelease` file is detected.

**[debugValues implementation detail: values injection mechanism]** → `RenderRelease()` currently passes a `valuesFile` path to `LoadReleasePackage()`. To inject `debugValues`, we either write a temp file (fragile) or add a `LoadReleasePackageWithValue()` variant that accepts a `cue.Value` directly. This implementation detail is deferred to the task itself. The CUE-value-based variant is preferred — it avoids temp files and keeps everything in-memory.

## Open Questions

- **`opm rel init`?** Should there be a scaffold command for deployment repos? Deferred to a future change.
- **Release file validation without provider?** `opm rel vet` needs a provider for matching. Should there be a `--skip-matching` flag? Deferred — evaluate after initial implementation.
- **debugValues values injection mechanism**: Pass `cue.Value` directly into a new `LoadReleasePackageWithValue()` variant, or serialize to temp file? To be resolved during task 2 implementation.
