## Purpose

Defines the contract for loading and building a concrete `*modulerelease.ModuleRelease` via the `pkg/loader` package. There is no separate builder phase — loading IS building, consistent with the `promote-factory-engine` architecture. The loader is responsible for value selection, schema validation (Module Gate), CUE-native evaluation of metadata and labels, and concreteness verification.

## Requirements

### Requirement: Loader validates consumer values and produces a concrete ModuleRelease

The `pkg/loader` package SHALL provide the full pipeline from CUE file loading through to a validated, concrete `*modulerelease.ModuleRelease`. There is no separate builder phase — loading IS building, consistent with the `promote-factory-engine` architecture.

The loader SHALL support two loading entry points:

1. **Module-directory path** (`LoadReleasePackage` + `LoadModuleReleaseFromValue`): used by `opm mod` commands. Accepts a directory containing `release.cue` + `values.cue`.
2. **Standalone release file** (`LoadReleaseFile` + `LoadModuleReleaseFromValue`): used by `opm release` commands. Accepts a single `.cue` file with CUE import resolution.

Both paths feed into the same `LoadModuleReleaseFromValue()` function which runs the Module Gate (validate values against `#module.#config`), concreteness check, metadata extraction, and value finalization.

#### Scenario: Successful load from module directory (existing behavior)

- **WHEN** `LoadReleasePackage()` is called with a module directory containing `release.cue` and `values.cue`
- **THEN** it returns a concrete `cue.Value` ready for `LoadModuleReleaseFromValue()`
- **AND** `LoadModuleReleaseFromValue()` returns a `*ModuleRelease` with all fields populated

#### Scenario: Successful load from release file

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file where `#module` is already filled via CUE import
- **THEN** it returns a concrete `cue.Value`
- **AND** `LoadModuleReleaseFromValue()` returns a `*ModuleRelease` with all fields populated (including auto-secrets handled by CUE `#AutoSecrets`)

#### Scenario: Release file with `--module` override

- **WHEN** `LoadReleaseFile()` returns a value where `#module` is not concrete
- **AND** the caller fills `#module` via `FillPath` using `LoadModulePackage()`
- **THEN** `LoadModuleReleaseFromValue()` successfully returns a `*ModuleRelease`

#### Scenario: Module Gate catches type mismatch

- **WHEN** consumer values contain a field with the wrong type
- **THEN** `LoadModuleReleaseFromValue()` returns a `*ConfigError` with structured `FieldError` details

#### Scenario: Auto-secrets are handled by CUE (no Go injection)

- **WHEN** a module's `#config` contains `#Secret` fields and concrete secret values are provided
- **THEN** the CUE `#AutoSecrets` mechanism in the loader automatically discovers and groups secrets
- **AND** the resulting `*ModuleRelease` contains the `opm-secrets` component
- **AND** no Go-side auto-secrets injection code is required

### Requirement: Values are validated against the module config schema before injection
The builder SHALL validate the selected values against the module's `#config` schema and return a descriptive error if they do not conform.

#### Scenario: Values match schema
- **WHEN** the selected values satisfy all constraints in `#config`
- **THEN** injection proceeds without error

#### Scenario: Values violate schema
- **WHEN** the selected values contain a field that violates a `#config` constraint (wrong type, out-of-range, missing required field)
- **THEN** the builder SHALL return an error identifying the offending field and the constraint that was violated

### Requirement: Release metadata and labels are derived by CUE evaluation
The builder SHALL load `#ModuleRelease` from `opmodel.dev/core@v1` (resolved from the module's own dependency cache) and inject the module, release name, namespace, and values via `FillPath`. UUID, labels, and derived metadata fields SHALL be computed by CUE evaluation, not by Go code.

#### Scenario: UUID is deterministic
- **WHEN** the same module, release name, and namespace are provided
- **THEN** the resulting `ModuleRelease.Metadata.UUID` SHALL be identical across builds

#### Scenario: Labels are populated from CUE evaluation
- **WHEN** the release is built successfully
- **THEN** `ModuleRelease.Metadata.Labels` SHALL contain all expected OPM labels as evaluated by `#ModuleRelease`

#### Scenario: Core v1 schema loaded
- **WHEN** the builder loads the core schema
- **THEN** it SHALL load `opmodel.dev/core@v1` (not `opmodel.dev/core@v0`)
- **THEN** error messages SHALL reference `opmodel.dev/core@v1`

### Requirement: The resulting release must be fully concrete
The builder SHALL validate that the `#ModuleRelease` value is fully concrete after injection, and return an error if any field remains abstract or unresolved.

#### Scenario: Incomplete values leave release non-concrete
- **WHEN** the provided values do not satisfy all required fields in `#config`
- **THEN** the builder SHALL return an error identifying which fields are not concrete

#### Scenario: Fully provided values produce a concrete release
- **WHEN** all required fields in `#config` are satisfied by the selected values
- **THEN** the builder SHALL return a `*core.ModuleRelease` where all components are concrete and ready for matching

### Requirement: Value selection falls back to module defaults when no files are given

When no `--values` files are provided, the builder SHALL discover values using the following priority:

1. When `release.cue` is present: auto-discover `values.cue` from the module directory (existing behavior)
2. When `release.cue` is absent and `debugValues` is defined in the module: use `debugValues` as the values source
3. When neither `release.cue` nor `values.cue` nor `debugValues` is available: return a descriptive error

The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` and `debugValues` SHALL both be ignored.

When using `LoadReleaseFile()` (release-file path), the `values` field is inline in the release CUE file itself. There is no `values.cue` fallback — the release file is self-contained.

#### Scenario: No values file, `values.cue` exists in module directory

- **WHEN** `LoadReleasePackage()` is called with no explicit values file
- **AND** `values.cue` exists in the module directory
- **THEN** `values.cue` is loaded alongside `release.cue` as part of the CUE instance

#### Scenario: Release file is self-contained

- **WHEN** `LoadReleaseFile()` is called
- **THEN** the `values` field is read from the release CUE file's inline definition
- **AND** no `values.cue` file is searched for or loaded

#### Scenario: No values files, no values.cue, debugValues defined

- **WHEN** no `--values` files are provided
- **AND** no `values.cue` file exists in the module directory
- **AND** the module defines a concrete `debugValues` field
- **THEN** the builder SHALL use `debugValues` as the values source

#### Scenario: No values files, no values.cue, no debugValues

- **WHEN** no `--values` files are provided
- **AND** no `values.cue` file exists in the module directory
- **AND** the module has no `debugValues` field
- **THEN** the builder SHALL return an error indicating the user must provide values via `values.cue`, `debugValues`, or `--values`

#### Scenario: Multiple `--values` files are unified

- **WHEN** more than one values file is provided via `--values`
- **THEN** the builder SHALL unify all files together before injection

### Requirement: `LoadReleaseFile()` loads a standalone `.cue` file with import resolution

The `pkg/loader` package SHALL export `LoadReleaseFile()` in `pkg/loader/release_file.go`. This function loads a standalone `.cue` release file using `load.Instances()` with the file's parent directory for `cue.mod` resolution, enabling CUE registry module imports.

```go
func LoadReleaseFile(ctx *cue.Context, filePath string, registry string) (cue.Value, string, error)
```

#### Scenario: Release file with registry import resolves successfully

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file that imports a module from `opmodel.dev/modules/jellyfin@v1`
- **AND** the file's parent directory contains a `cue.mod/module.cue` declaring the dependency
- **THEN** the import is resolved, the module is unified into `#module`, and the evaluated value is returned

#### Scenario: Release file without `cue.mod/` fails with clear error

- **WHEN** `LoadReleaseFile()` is called with a `.cue` file in a directory with no `cue.mod/` ancestor
- **THEN** the loader returns an error describing the missing module configuration

### Requirement: `LoadModulePackage()` loads a local module for `--module` flag injection

The `pkg/loader` package SHALL export `LoadModulePackage()` in `pkg/loader/release_file.go`. This function loads a module CUE package from a local directory and returns the raw `cue.Value` for `FillPath` injection into a release value. This replaces the deleted `internal/loader.LoadModule()` for this specific use case.

```go
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error)
```

#### Scenario: Local module loaded for `--module` injection

- **WHEN** `LoadModulePackage()` is called with a valid module directory
- **THEN** it returns the evaluated `cue.Value` of the module package
- **AND** the caller can inject it via `releaseVal.FillPath(cue.MakePath(cue.Def("module")), modVal)`

### Requirement: `opm mod vet` uses `debugValues` by default

The `opm mod vet` command SHALL use the module's `debugValues` field as the values source when no `-f` flag is provided. The extraction SHALL happen in `internal/cmdutil/render.go` via the `DebugValues bool` field on `RenderReleaseOpts`. The loader SHALL expose a `LoadReleasePackageWithValue()` variant that accepts a pre-loaded `cue.Value` instead of a values file path.

#### Scenario: `debugValues` used when no `-f` flag

- **WHEN** `opm mod vet` is run without `-f` flags
- **THEN** `RenderRelease()` is called with `DebugValues: true`
- **AND** the module's `debugValues` field is extracted and used as the values source
- **AND** the vet output shows "debugValues" as the values source

#### Scenario: `-f` flag overrides `debugValues`

- **WHEN** `opm mod vet` is run with one or more `-f` flags
- **THEN** `DebugValues` is `false` and the explicit values files are used
- **AND** `debugValues` is ignored

#### Scenario: `debugValues` is `_` (unconstrained)

- **WHEN** `opm mod vet` is run without `-f` flags
- **AND** the module's `debugValues` field is `_` (open/unconstrained, not filled by the author)
- **THEN** `RenderRelease()` returns an error: "debugValues is not concrete — module must provide complete test values"

### Requirement: SynthesizeModuleRelease builds a ModuleRelease without a release.cue file
The loader SHALL provide a `SynthesizeModuleRelease` function that constructs a `*modulerelease.ModuleRelease` from a loaded module CUE value and a concrete values CUE value, without requiring a `release.cue` file.

The function SHALL:
1. Run the Module Gate: validate `valuesVal` against `modVal.LookupPath("#config")` using the shared `validateConfig` function
2. Fill `#config` with the provided values: `filledMod := modVal.FillPath(cue.ParsePath("#config"), valuesVal)`
3. Extract schema components from `filledMod.LookupPath("#components")` (preserves `#resources`, `#traits` definition fields required by the CUE match plan evaluator)
4. Create a synthetic schema value by wrapping the components under a regular `components` field (so `ModuleRelease.MatchComponents()` can look up `"components"`, not `"#components"`)
5. Finalize components via `finalizeValue` for constraint-free execution
6. Decode module metadata from `modVal.LookupPath("metadata")`
7. Construct `ReleaseMetadata` with the provided `releaseName` and `namespace`; leave UUID empty
8. Return `NewModuleRelease(relMeta, module.Module{Metadata: modMeta, Raw: modVal}, syntheticSchema, dataComponents)`

#### Scenario: SynthesizeModuleRelease succeeds with valid module and debugValues
- **WHEN** `SynthesizeModuleRelease` is called with a loaded module value and its concrete `debugValues`
- **THEN** the returned `*ModuleRelease` SHALL have non-nil `Metadata`, `Module.Metadata`, and non-empty `dataComponents`
- **AND** `MatchComponents()` SHALL return a value with `components` that can be iterated by the match plan

#### Scenario: SynthesizeModuleRelease fails Module Gate on invalid values
- **WHEN** `SynthesizeModuleRelease` is called with values that violate `#config` constraints
- **THEN** the function SHALL return a non-nil error describing the constraint violation
- **AND** the error SHALL be formatted identically to the Module Gate error from the normal release path

#### Scenario: SynthesizeModuleRelease produces concrete components
- **WHEN** `SynthesizeModuleRelease` is called with concrete `debugValues` satisfying `#config`
- **THEN** `ExecuteComponents()` SHALL return a fully concrete, constraint-free CUE value
- **AND** `dataComponents.Validate(cue.Concrete(true))` SHALL return nil

#### Scenario: Synthesized ModuleRelease UUID is empty
- **WHEN** `SynthesizeModuleRelease` is called successfully
- **THEN** `ModuleRelease.Metadata.UUID` SHALL be an empty string
- **AND** the `apply` command SHALL skip inventory tracking when UUID is empty (existing behavior at `apply.go:187`)

### Requirement: RenderRelease supports synthesis mode when release.cue is absent
`cmdutil.RenderRelease` SHALL detect whether `release.cue` exists in the module path. When absent, it SHALL take a synthesis branch that:
1. Loads the module package
2. Extracts `debugValues` when `DebugValues: true` and no `-f` flag, or loads the `-f` values file
3. Resolves the release name from `opts.ReleaseName`, then `module.metadata.name`, then `filepath.Base(modulePath)`
4. Resolves the namespace from `module.metadata.defaultNamespace` (overridden post-synthesis by flag/env, identical to normal path)
5. Calls `SynthesizeModuleRelease`
6. Continues on the common tail: provider loading, engine rendering, resource conversion

When `release.cue` is present, existing behavior is unchanged.

#### Scenario: RenderRelease takes synthesis path when no release.cue
- **WHEN** `RenderRelease` is called with a module path that has no `release.cue`
- **AND** `DebugValues: true`
- **THEN** `RenderRelease` SHALL call `SynthesizeModuleRelease` instead of `LoadReleasePackage`
- **AND** the returned `*RenderResult` SHALL be populated identically to a release-backed render

#### Scenario: RenderRelease takes normal path when release.cue present
- **WHEN** `RenderRelease` is called with a module path that has a `release.cue`
- **THEN** `RenderRelease` SHALL use the existing `LoadReleasePackage` / `LoadReleasePackageWithValue` path
- **AND** behavior SHALL be unchanged from before this change

#### Scenario: Synthesis mode with no debugValues and no -f returns actionable error
- **WHEN** `RenderRelease` is called in synthesis mode
- **AND** `DebugValues: true` but the module has no `debugValues` field
- **THEN** `RenderRelease` SHALL return an error containing both remediation options: add `debugValues` to the module OR provide a values file with `-f`

---

## Removed Requirements

### Requirement: Separate release building phase
**Reason**: `builder.Build()` with its FillPath chain, values validation, auto-secrets injection, and component extraction is eliminated. All of this is handled by the new loader: CUE evaluation naturally handles value unification and defaults, gates handle validation, `#AutoSecrets` handles secrets, and `finalizeValue()` handles constraint stripping.

**Migration**: Replace `builder.Build(ctx, mod, opts, valuesFiles)` with `loader.LoadReleasePackage()` + `loader.LoadModuleReleaseFromValue()`.
