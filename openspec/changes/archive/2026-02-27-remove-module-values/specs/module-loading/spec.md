## REMOVED Requirements

### Requirement: Default values are extracted into Module.Values
**Reason**: In v1alpha1, `#Module` no longer has a `values` field. Values resolution has moved to the builder. The loader no longer loads `values.cue` or extracts inline values — it only filters `values*.cue` from the package load.
**Migration**: Values are now resolved by the builder at build time. The builder discovers `values.cue` from `mod.ModulePath` or uses `--values` files.

## MODIFIED Requirements

### Requirement: CUE instance is loaded with registry configuration
The loader SHALL load the CUE instance from the resolved module path using `load.Instances` with an explicit file list that excludes all `values*.cue` files. When a registry string is provided, the loader SHALL set `CUE_REGISTRY` for the duration of the load. The loader SHALL enumerate all top-level `.cue` files, filter out any file whose base name starts with `values` and ends with `.cue`, and pass only the remaining files to `load.Instances`.

#### Scenario: Registry is set during load
- **WHEN** a non-empty registry string is provided
- **THEN** `CUE_REGISTRY` is set before `load.Instances` is called and unset after

#### Scenario: No instances found returns error
- **WHEN** the module path contains no CUE instances
- **THEN** the loader returns a descriptive error

#### Scenario: Instance load error is surfaced
- **WHEN** `load.Instances` returns an instance with a non-nil error
- **THEN** the loader returns that error wrapped with context

#### Scenario: values*.cue files are excluded from package load
- **WHEN** the module directory contains files matching `values*.cue` (e.g., `values.cue`, `values_prod.cue`)
- **THEN** those files SHALL be excluded from the file list passed to `load.Instances`
- **AND** the remaining module files SHALL load without error
