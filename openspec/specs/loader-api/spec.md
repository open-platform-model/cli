# Loader API

## Purpose

Defines the public API of `pkg/loader` — the release-centric loading functions that replace the old `loader.LoadModule()` directory-based approach.

## Requirements

### Requirement: LoadInstancePackage loads instance CUE files
The `pkg/loader` package SHALL export a `LoadInstancePackage` function that loads an instance CUE package (instance.cue + values.cue) and returns the raw evaluated `cue.Value` and the instance directory path.

#### Scenario: Load with explicit values file
- **WHEN** `LoadInstancePackage(cueCtx, instanceFile, valuesFile)` is called with a non-empty valuesFile
- **THEN** it loads exactly the two specified files as one CUE instance

#### Scenario: Load with default values file
- **WHEN** `LoadInstancePackage(cueCtx, instanceFile, "")` is called with empty valuesFile
- **THEN** it loads `instance.cue` and `values.cue` from the instance directory

#### Scenario: Directory path resolves to instance.cue
- **WHEN** instanceFile is a directory path (not ending in .cue)
- **THEN** the loader resolves it to `<directory>/instance.cue` using `os.Stat()` and `IsDir()`

### Requirement: DetectInstanceKind identifies instance type
The `pkg/loader` package SHALL export a `DetectInstanceKind` function that reads the `kind` field from a loaded instance package. It SHALL recognize `"ModuleInstance"` and reject every other kind with an error; bundle kinds are no longer recognized (enhancement 0002 D15 — the bundle path is removed).

#### Scenario: ModuleInstance kind detection
- **WHEN** `DetectInstanceKind(pkg)` is called and the `kind` field is "ModuleInstance"
- **THEN** it returns "ModuleInstance"

#### Scenario: Unknown kind
- **WHEN** `DetectInstanceKind(pkg)` is called with an unrecognized kind (including the former `"BundleRelease"`)
- **THEN** it returns an `unknown instance kind` error

### Requirement: LoadModuleInstanceFromValue builds a ModuleInstance
The `pkg/loader` package SHALL export a `LoadModuleInstanceFromValue` function that validates, finalizes, and extracts a `*ModuleInstance` from an already-loaded CUE package value.

#### Scenario: Full loading pipeline
- **WHEN** `LoadModuleInstanceFromValue(cueCtx, pkg, fallbackName)` is called with a valid package
- **THEN** it runs Module Gate → concreteness check → metadata extraction → finalization → DataComponents extraction, returning a fully populated `*ModuleInstance`

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

### Requirement: `SynthesizeModuleInstanceFromPackage` builds an instance `cue.Value` from a module directory

The `pkg/loader` package SHALL export a function (working name: `SynthesizeModuleInstanceFromPackage`) that loads a module CUE package from a directory, composes a `#ModuleInstance` wrapper around it using a small synthetic CUE module pinned at the same catalog version the user's module already uses, fills synthetic metadata, and returns a `cue.Value` ready to feed into `LoadModuleInstanceFromValue`.

#### Scenario: Whole-package load of the user's module

- **WHEN** `SynthesizeModuleInstanceFromPackage(ctx, modulePath, opts)` is called with a directory containing a CUE module package
- **THEN** the loader SHALL evaluate every `.cue` file in that directory belonging to the same package via `load.Instances(["."], &load.Config{Dir: modulePath})`
- **AND** the loader SHALL return an error if the directory does not contain a single resolvable CUE package

#### Scenario: Synthetic wrapper resolves the catalog via the registry

- **WHEN** the synthesis composes the `#ModuleInstance` wrapper
- **THEN** the wrapper SHALL be a small CUE module declaring one dep on `opmodel.dev/core/v1alpha1@v1`
- **AND** the wrapper file SHALL import `mr "opmodel.dev/core/v1alpha1/modulerelease@v1"` and apply `mr.#ModuleInstance` at the top level (matching the shape of real instance files in `releases/<env>/<module>/instance.cue`)
- **AND** the wrapper's `cue.mod/module.cue` and `wrapper.cue` SHALL be served via `load.Config.Overlay` against a temp anchor created with `os.MkdirTemp` and removed before the function returns
- **AND** the catalog dep SHALL be resolved via the loader's default registry (`CUE_REGISTRY` env), reusing the local cache

#### Scenario: Two loads share one CUE context

- **WHEN** the synthesis loads both the user's module and the synthetic wrapper
- **THEN** both loads SHALL be performed against the same `*cue.Context`
- **AND** composition SHALL use `Value.Unify` and `Value.FillPath`, not string-based CUE source generation

#### Scenario: Synthetic metadata is filled by the loader

- **WHEN** the caller passes `opts.Name = "foo"` and `opts.Namespace = "bar"`
- **THEN** the returned `cue.Value` SHALL have `metadata.name = "foo"` and `metadata.namespace = "bar"` filled via `Value.FillPath`

#### Scenario: Caller-supplied values are preserved for downstream filling

- **WHEN** the function is called with values to use (either pre-loaded `-f` files or the module's `debugValues`)
- **THEN** the returned `cue.Value` SHALL have `#module` filled with the loaded module value but SHALL leave `values` to be filled by `LoadModuleInstanceFromValue`
- **AND** `LoadModuleInstanceFromValue` SHALL run the Module Gate (validate values vs `#config`), concreteness check, metadata extraction, and finalisation exactly as it does for instance-file inputs

### Requirement: Synth wrapper pins the catalog at the user module's pinned version

The synthesis SHALL parse the user's `cue.mod/module.cue` via `cuelang.org/go/mod/modfile.Parse`, look up the `opmodel.dev/core/v1alpha1@v1` dep, and reuse the same version string as the synth wrapper's pin for that dep. This guarantees that the user-module load and the synth-wrapper load resolve the catalog to the same registry artifact.

#### Scenario: Catalog version copied from user modfile

- **WHEN** the user's `cue.mod/module.cue` declares `"opmodel.dev/core/v1alpha1@v1": v: "v1.3.9"`
- **THEN** the synth wrapper's `cue.mod/module.cue` SHALL declare the same dep with `v: "v1.3.9"`

#### Scenario: User module declares no catalog dep

- **WHEN** the user's `cue.mod/module.cue` does not declare `opmodel.dev/core/v1alpha1@v1`
- **THEN** the loader SHALL return a `DetailError` whose hint instructs the user to add `opmodel.dev/core/v1alpha1@v1` as a dependency before building

#### Scenario: User modfile cannot be parsed

- **WHEN** the user's `cue.mod/module.cue` cannot be located or parsed
- **THEN** the loader SHALL return an error wrapping the parse failure with context `"reading module's cue.mod/module.cue"`

### Requirement: No filesystem writes inside the user's module

The synthesis path SHALL NOT create, modify, or remove any files inside the user's module directory or its `cue.mod/`. Anchor directories used for the synth wrapper SHALL live outside the user's module tree (created via `os.MkdirTemp`) and SHALL be removed before the function returns.

#### Scenario: User module tree unchanged

- **WHEN** synthesis runs to completion (success or error)
- **THEN** no files inside `<modulePath>` or `<modulePath>/cue.mod/` SHALL be created, modified, or deleted

#### Scenario: Anchor temp dir cleaned up

- **WHEN** synthesis returns (whether successful or with an error)
- **THEN** the temp anchor directory SHALL have been removed via `os.RemoveAll`
