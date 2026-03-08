## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Value selection falls back to module defaults when no files are given

Supersedes the main spec scenario `"No values files, no values.cue → return error"` — that scenario is replaced by the 3-tier fallback below.

The builder SHALL discover values using the following priority when no `--values` files are provided:

1. When `release.cue` is present: auto-discover `values.cue` from the module directory (existing behavior)
2. When `release.cue` is absent and `debugValues` is defined in the module: use `debugValues` as the values source
3. When neither `release.cue` nor `values.cue` nor `debugValues` is available: return a descriptive error

The builder SHALL NOT read values from `Module.Values`. If `--values` files are provided, `values.cue` and `debugValues` SHALL both be ignored.

#### Scenario: No values files, values.cue exists in module directory
- **WHEN** no `--values` files are provided
- **AND** a `values.cue` file exists in the module directory
- **THEN** the builder SHALL load `values.cue`, extract the `values` field, and use it for injection

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

#### Scenario: --values files provided, values.cue exists
- **WHEN** `--values` files are provided
- **AND** a `values.cue` file also exists in the module directory
- **THEN** the builder SHALL use ONLY the `--values` files and completely ignore `values.cue`

#### Scenario: Multiple --values files are unified
- **WHEN** more than one values file is provided via `--values`
- **THEN** the builder SHALL unify all files together before injection
