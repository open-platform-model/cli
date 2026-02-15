## Why

Module authors have no dedicated command to validate their modules without generating manifests or applying to a cluster. The existing `opm mod build` produces manifests but doesn't surface per-resource validation results, and its `--verbose` output is not aligned with `opm mod apply` (see issue #15). Additionally, `opm config vet` prints a single unstyled line on success, giving no visibility into which checks passed. These gaps slow down the author feedback loop and make output harder to scan.

This change supersedes `improve-config-vet-output`, absorbing its config vet output improvements and `FormatVetCheck` helper into the broader validation output effort.

## What Changes

- **New `opm mod vet` command**: Standalone module validation that builds the module via the existing render pipeline and reports per-resource validation results with colorized output. No manifest output — purely a pass/fail validation tool.
- **Enhanced `opm mod build --verbose` output**: Align verbose output with `opm mod apply` per issue #15. Generated resources section uses `FormatResourceLine` with a new `"valid"` status, matching the `r:<Kind/ns/name>  <status>` format used by apply.
- **Improved `opm config vet` output** (absorbed from `improve-config-vet-output`): Replace single success line with per-check styled output. Each validation step (file exists, module.cue exists, CUE evaluation passes) prints a checkmark as it completes.
- **New `StatusValid` output constant**: Add `"valid"` to the status vocabulary in the styling system, rendered in green. Reusable by both `mod vet` and `mod build --verbose`.
- **New `FormatVetCheck` output helper** (absorbed from `improve-config-vet-output`): Renders validation check results with a green checkmark, label, and optional right-aligned detail text. Reusable by `config vet`, `mod vet`, and future vet commands.

**SemVer**: MINOR — adds a new command. No breaking changes.

## Capabilities

### New Capabilities

- `mod-vet`: The `opm mod vet` command — standalone module validation with per-resource output, reusable validation pipeline, and clear pass/fail summary.

### Modified Capabilities

- `build`: Enhanced `--verbose` output with per-resource validation lines using `FormatResourceLine` and `StatusValid`.
- `config-commands`: `config vet` success output changes from a single plain line to per-check styled checkmark output using `FormatVetCheck`.
- `log-output-style`: New `StatusValid` constant and style mapping. New `FormatVetCheck` helper for validation check output with right-aligned detail text.

## Impact

- **New files**: `internal/cmd/mod_vet.go`, `internal/cmd/mod_vet_test.go`
- **Modified commands**: `internal/cmd/config_vet.go` (output), `internal/cmd/mod_build.go` (verbose output)
- **Modified output**: `internal/output/styles.go` (StatusValid, FormatVetCheck), `internal/output/verbose.go` (resource validation lines)
- **Supersedes**: `improve-config-vet-output` change (to be archived as superseded)
- **Justification (Principle VII)**: `mod vet` fills a clear gap — module authors currently must run `mod build` and visually inspect YAML to validate. `FormatVetCheck` and `StatusValid` extend the established style vocabulary rather than introducing a new system. The validation output logic lives in the output package, making it reusable by any command.
