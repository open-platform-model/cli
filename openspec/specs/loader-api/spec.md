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

---

## Removed Requirements

### Requirement: Module loader function name
**Reason**: `LoadModule()` is replaced by a release-centric loading approach. The new loader operates on release packages (`release.cue + values.cue`) instead of module directories.

**Migration**: Replace `loader.LoadModule(cueCtx, modulePath, registry)` with `loader.LoadReleasePackage(cueCtx, releaseFile, valuesFile)` followed by `loader.LoadModuleReleaseFromValue()`.

### Requirement: Shared CUE string-map extraction helper
**Reason**: `extractCUEStringMap` is no longer needed — the new loader uses `cue.Value.Decode()` into typed structs instead of manual field-by-field extraction.

**Migration**: No migration needed — internal helper eliminated.
