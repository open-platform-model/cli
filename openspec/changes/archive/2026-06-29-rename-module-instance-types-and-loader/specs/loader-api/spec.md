## MODIFIED Requirements

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
The `pkg/loader` package SHALL export a `DetectInstanceKind` function that reads the `kind` field from a loaded instance package.

#### Scenario: ModuleInstance kind detection
- **WHEN** `DetectInstanceKind(pkg)` is called and the `kind` field is "ModuleInstance"
- **THEN** it returns "ModuleInstance"

#### Scenario: BundleRelease kind detection
- **WHEN** `DetectInstanceKind(pkg)` is called and the `kind` field is "BundleRelease"
- **THEN** it returns "BundleRelease"

#### Scenario: Unknown kind
- **WHEN** `DetectInstanceKind(pkg)` is called with an unrecognized kind
- **THEN** it returns an error

### Requirement: LoadModuleInstanceFromValue builds a ModuleInstance
The `pkg/loader` package SHALL export a `LoadModuleInstanceFromValue` function that validates, finalizes, and extracts a `*ModuleInstance` from an already-loaded CUE package value.

#### Scenario: Full loading pipeline
- **WHEN** `LoadModuleInstanceFromValue(cueCtx, pkg, fallbackName)` is called with a valid package
- **THEN** it runs Module Gate → concreteness check → metadata extraction → finalization → DataComponents extraction, returning a fully populated `*ModuleInstance`

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
