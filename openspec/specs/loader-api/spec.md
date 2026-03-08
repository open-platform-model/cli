# Loader API

## Purpose

Defines the public API of `pkg/loader` — the release-centric loading functions that replace the old `loader.LoadModule()` directory-based approach.

## Requirements

### Requirement: LoadReleasePackage loads release CUE files
The `pkg/loader` package SHALL export a `LoadReleasePackage` function that loads a release CUE package (release.cue + values.cue) and returns the raw evaluated `cue.Value` and the release directory path.

#### Scenario: Load with explicit values file
- **WHEN** `LoadReleasePackage(cueCtx, releaseFile, valuesFile)` is called with a non-empty valuesFile
- **THEN** it loads exactly the two specified files as one CUE instance

#### Scenario: Load with default values file
- **WHEN** `LoadReleasePackage(cueCtx, releaseFile, "")` is called with empty valuesFile
- **THEN** it loads `release.cue` and `values.cue` from the release directory

#### Scenario: Directory path resolves to release.cue
- **WHEN** releaseFile is a directory path (not ending in .cue)
- **THEN** the loader resolves it to `<directory>/release.cue` using `os.Stat()` and `IsDir()`

### Requirement: DetectReleaseKind identifies release type
The `pkg/loader` package SHALL export a `DetectReleaseKind` function that reads the `kind` field from a loaded release package.

#### Scenario: ModuleRelease kind detection
- **WHEN** `DetectReleaseKind(pkg)` is called and the `kind` field is "ModuleRelease"
- **THEN** it returns "ModuleRelease"

#### Scenario: BundleRelease kind detection
- **WHEN** `DetectReleaseKind(pkg)` is called and the `kind` field is "BundleRelease"
- **THEN** it returns "BundleRelease"

#### Scenario: Unknown kind
- **WHEN** `DetectReleaseKind(pkg)` is called with an unrecognized kind
- **THEN** it returns an error

### Requirement: LoadModuleReleaseFromValue builds a ModuleRelease
The `pkg/loader` package SHALL export a `LoadModuleReleaseFromValue` function that validates, finalizes, and extracts a `*ModuleRelease` from an already-loaded CUE package value.

#### Scenario: Full loading pipeline
- **WHEN** `LoadModuleReleaseFromValue(cueCtx, pkg, fallbackName)` is called with a valid package
- **THEN** it runs Module Gate → concreteness check → metadata extraction → finalization → DataComponents extraction, returning a fully populated `*ModuleRelease`

### Requirement: LoadBundleReleaseFromValue builds a BundleRelease
The `pkg/loader` package SHALL export a `LoadBundleReleaseFromValue` function that validates and extracts a `*BundleRelease` with its contained module releases.

#### Scenario: Full bundle loading pipeline
- **WHEN** `LoadBundleReleaseFromValue(cueCtx, pkg, fallbackName)` is called with a valid bundle package
- **THEN** it runs Bundle Gate → per-release Module Gate + finalization, returning a `*BundleRelease` with populated `Releases` map

### Requirement: LoadProvider loads a provider from CUE
The `pkg/loader` package SHALL export a `LoadProvider` function that loads a named provider from the CUE `#Registry`.

#### Scenario: Named provider loading
- **WHEN** `LoadProvider(cueCtx, name, cueModuleDir)` is called with a specific provider name
- **THEN** it loads the providers CUE package and extracts the named provider

#### Scenario: Auto-selection with single provider
- **WHEN** `LoadProvider(cueCtx, "", cueModuleDir)` is called with empty name and only one provider exists
- **THEN** it auto-selects that provider

### Requirement: finalizeValue strips CUE constraints
The loader SHALL provide a `finalizeValue` function (internal or exported) that uses `Syntax(cue.Final())` + `BuildExpr` to strip schema constraints from a CUE value, producing a plain data value suitable for `FillPath` injection.

#### Scenario: Finalization strips matchN validators
- **WHEN** `finalizeValue(cueCtx, v)` is called on a concrete CUE value with schema constraints
- **THEN** it returns a new CUE value with `matchN` validators, `close()` enforcement, and definition fields removed

#### Scenario: Non-expr syntax produces clear error
- **WHEN** `finalizeValue()` produces `*ast.File` instead of `ast.Expr`
- **THEN** it returns an error indicating unresolved imports or definition fields that should have been resolved upstream

### Requirement: SynthesizeModuleRelease builds a ModuleRelease without a release.cue file

The `pkg/loader` package SHALL export a `SynthesizeModuleRelease` function that constructs a `*modulerelease.ModuleRelease` from a loaded module CUE value and a concrete values CUE value, without requiring a `release.cue` file.

The function signature SHALL be:
```
SynthesizeModuleRelease(cueCtx *cue.Context, modVal cue.Value, valuesVal cue.Value, releaseName string, namespace string) (*modulerelease.ModuleRelease, error)
```

The function SHALL:
1. Run the Module Gate: validate `valuesVal` against `modVal.LookupPath("#config")` using `validateConfig`
2. Fill `#config` with the provided values: `filledMod := modVal.FillPath(cue.ParsePath("#config"), valuesVal)`
3. Extract schema components from `filledMod.LookupPath("#components")`
4. Wrap components under a regular `components` field so `MatchComponents()` can find them
5. Finalize components via `finalizeValue` for constraint-free execution
6. Decode module metadata from `modVal.LookupPath("metadata")`
7. Construct `ReleaseMetadata` with `releaseName` and `namespace`; leave UUID empty
8. Return `NewModuleRelease(relMeta, module.Module{Metadata: modMeta, Raw: modVal}, syntheticSchema, dataComponents)`

#### Scenario: SynthesizeModuleRelease succeeds with valid module and debugValues
- **WHEN** `SynthesizeModuleRelease` is called with a loaded module value and its concrete `debugValues`
- **THEN** the returned `*ModuleRelease` SHALL have non-nil `Metadata`, `Module.Metadata`, and non-empty `dataComponents`
- **AND** `MatchComponents()` SHALL return a value with `components` that can be iterated by the match plan

#### Scenario: SynthesizeModuleRelease fails Module Gate on invalid values
- **WHEN** `SynthesizeModuleRelease` is called with values that violate `#config` constraints
- **THEN** the function SHALL return a non-nil error describing the constraint violation

#### Scenario: SynthesizeModuleRelease produces concrete components
- **WHEN** `SynthesizeModuleRelease` is called with concrete `debugValues` satisfying `#config`
- **THEN** `ExecuteComponents()` SHALL return a fully concrete, constraint-free CUE value
- **AND** `dataComponents.Validate(cue.Concrete(true))` SHALL return nil

#### Scenario: Synthesized ModuleRelease UUID is empty
- **WHEN** `SynthesizeModuleRelease` is called successfully
- **THEN** `ModuleRelease.Metadata.UUID` SHALL be an empty string
- **AND** the `apply` command SHALL skip inventory tracking when UUID is empty (existing guard at `apply.go:187`)

---

## Removed Requirements

### Requirement: Module loader function name
**Reason**: `LoadModule()` is replaced by a release-centric loading approach. The new loader operates on release packages (`release.cue + values.cue`) instead of module directories.

**Migration**: Replace `loader.LoadModule(cueCtx, modulePath, registry)` with `loader.LoadReleasePackage(cueCtx, releaseFile, valuesFile)` followed by `loader.LoadModuleReleaseFromValue()`.

### Requirement: Shared CUE string-map extraction helper
**Reason**: `extractCUEStringMap` is no longer needed — the new loader uses `cue.Value.Decode()` into typed structs instead of manual field-by-field extraction.

**Migration**: No migration needed — internal helper eliminated.
