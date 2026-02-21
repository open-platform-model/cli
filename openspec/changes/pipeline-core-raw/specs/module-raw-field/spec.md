## ADDED Requirements

### Requirement: Module exposes evaluated CUE value as public field
`core.Module` SHALL expose the fully-evaluated CUE value as a public field `Raw cue.Value` instead of through `CUEValue()` / `SetCUEValue()` accessor methods.

#### Scenario: Caller reads evaluated CUE value
- **WHEN** a caller holds a `*core.Module` after loading
- **THEN** the caller SHALL access the evaluated CUE value via `mod.Raw` directly

#### Scenario: Caller sets evaluated CUE value
- **WHEN** a loader assigns the evaluated CUE value to a module
- **THEN** the loader SHALL assign via `mod.Raw = v` directly (no setter method)

#### Scenario: Accessor methods are removed
- **WHEN** any code calls `mod.CUEValue()` or `mod.SetCUEValue(v)`
- **THEN** the build SHALL fail with a compile error (methods no longer exist)
