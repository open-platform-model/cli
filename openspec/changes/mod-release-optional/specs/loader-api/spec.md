## ADDED Requirements

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
