## Context

Module authors need a fast validation feedback loop. Today they must run `opm mod build` (which outputs YAML manifests) or `opm mod apply` (which requires a cluster) just to verify their module is valid. There's no dedicated "does this compile and validate?" command.

The CLI already has a robust render pipeline (`build.Pipeline.Render()`) that loads CUE, builds `#ModuleRelease`, matches components to transformers, and executes transformers. Both `mod build` and `mod apply` use it. The new `mod vet` command needs the same pipeline but only cares about the pass/fail result — not the manifest output.

Additionally, two output improvements are pending:
- `mod build --verbose` lists generated resources in a plain text format that doesn't match `mod apply`'s styled `FormatResourceLine` output (issue #15).
- `config vet` prints a single unstyled line on success with no per-check visibility.

These three changes share a common need: styled per-resource and per-check validation output rendered by the `internal/output` package.

## Goals / Non-Goals

**Goals:**

- Add `opm mod vet` as a standalone validation command that reuses the existing render pipeline
- Add `StatusValid` and `FormatVetCheck` to the output package's vocabulary
- Update `mod build --verbose` to use `FormatResourceLine` with `StatusValid` for generated resources
- Update `config vet` to print per-check `FormatVetCheck` lines instead of a single summary line
- Keep all validation output formatting in `internal/output` so it's reusable across commands

**Non-Goals:**

- Adding `cue.Final()` to any validation call site (incompatible with CUE `#definition` types used throughout OPM modules)
- Embedding `cue mod tidy` (blocked by Go internal package boundary in CUE SDK)
- Adding a `--quiet` flag to `mod vet` (YAGNI — can be added later if requested)
- Changing error formatting in `formatCUEDetails` (already provides colorized output, works well)
- Changing the `mod build` manifest output format or the `mod apply` resource output format

## Decisions

### Decision 1: `mod vet` is a thin command that delegates to the render pipeline

`mod vet` creates a `build.Pipeline`, calls `Render()`, and consumes the `RenderResult` for validation output only. It does not render manifests to stdout. This means:
- Zero new CUE loading or validation logic
- Same validation semantics as `mod build` (any module that passes `mod vet` will also pass `mod build`)
- Same flag surface (`--values`, `--namespace`, `--name`, `--provider`, `--strict`) minus output flags (`-o`, `--split`, `--out-dir`, `--verbose-json`)

The command adds `--verbose` for matching decision details (reuses `writeVerboseOutput`) and prints per-resource validation lines on success.

**Alternative considered:** A separate lightweight validator that only does CUE `value.Validate()` without transformer matching. Rejected because it would miss render-phase errors (unmatched components, transform failures) which are the most common real-world validation failures.

### Decision 2: Per-resource output uses existing `FormatResourceLine` with new `StatusValid`

Rather than a new formatting function, `mod vet` and `mod build --verbose` both use the existing `FormatResourceLine(kind, ns, name, "valid")`. This requires only adding `StatusValid = "valid"` and a case in `statusStyle()` that returns green (`82`), the same color as `"created"`.

This keeps the resource line format consistent across commands:
```
r:StatefulSet/default/jellyfin          valid     ← mod vet / mod build --verbose
r:StatefulSet/default/jellyfin          created   ← mod apply
r:StatefulSet/default/jellyfin          deleted   ← mod delete
```

The `"valid"` status shares green with `"created"` because both are positive outcomes. Users scanning terminal output see green = good, yellow = changed, red = removed/failed.

**Alternative considered:** A distinct color for `"valid"` (e.g., cyan or white). Rejected because valid/created are both "success" states and splitting them visually adds cognitive load without information gain.

### Decision 3: `FormatVetCheck` is a new helper separate from `FormatResourceLine`

`FormatResourceLine` has the semantic pattern `r:<Kind/ns/name>  <status>` — it formats Kubernetes resource identifiers. Validation checks have a different semantic: a check label and an optional detail string (usually a file path).

`FormatVetCheck(label, detail string)` renders:
```
✔ Config file found              ~/.opm/config.cue
✔ Module metadata found          ~/.opm/cue.mod/module.cue
✔ CUE evaluation passed
```

Key design choices:
- Always renders a green checkmark — failed checks don't use this function (they return errors through the normal `ExitError` path)
- Detail text is dim/faint, right-aligned at column 34 from label start
- No `passed bool` parameter (avoids a red-X styling path that would be inconsistent with error reporting)

This function is used by `config vet` and can be used by `mod vet` for module-level checks. Per-resource validation lines in `mod vet` use `FormatResourceLine` instead.

### Decision 4: `mod build --verbose` resource section uses `FormatResourceLine`

The current `writeVerboseHuman` in `verbose.go` prints resources as plain text:
```
Generated Resources:
  StatefulSet/jellyfin [default] from app
```

This changes to use `FormatResourceLine` with `StatusValid`:
```
Generated Resources:
  r:StatefulSet/default/jellyfin          valid
```

The implementation modifies `writeVerboseHuman` in `internal/output/verbose.go` to call `FormatResourceLine` instead of `fmt.Sprintf`. The JSON verbose output (`--verbose-json`) is unchanged — it already includes structured resource data.

**Alternative considered:** Adding a separate `WriteResourceValidationLines` function. Rejected because `writeVerboseHuman` already handles the resource section and just needs its formatting updated.

### Decision 5: `config vet` prints checks as they pass, not collected

Each check in `config vet` prints its `FormatVetCheck` line immediately after passing. This means:
1. User sees real-time progress (CUE evaluation can be slow due to registry resolution)
2. On failure, preceding checkmarks are visible, giving context about what succeeded
3. The error is returned through the normal `ExitError` path

The existing check structure is preserved — only the output statements change.

### Decision 6: `mod vet` reuses `printValidationError` and `printRenderErrors` from `mod_build.go`

These functions are already defined in the `cmd` package and handle the two error categories from the render pipeline:
- `printValidationError` — CUE validation errors with file paths and line:col positions
- `printRenderErrors` — unmatched components, transform failures, unhandled traits

`mod vet` calls the same functions for identical error presentation. No code duplication.

### Decision 7: Exit code 2 for validation failures

`mod vet` uses exit code 2 for all validation failures (CUE errors, unmatched components, render failures), consistent with `mod build`'s use of `ExitValidationError = 2`. Exit code 1 is reserved for usage errors (cobra default). Exit code 0 means validation passed.

This matches the convention used by `cue vet` and other validation tools.

## Risks / Trade-offs

**[Risk] `mod vet` requires a configured provider** → The render pipeline needs a provider to match components to transformers. A module can't be fully validated without provider context. Mitigation: this is the correct behavior — `mod vet` validates the full render, not just CUE syntax. Users who want CUE-only validation can use `cue vet ./...` directly.

**[Risk] Output tests can't capture styled output** → `output.Println` writes to `os.Stdout`, not `cmd.OutOrStdout()`. Cobra's test `cmd.SetOut()` doesn't intercept it. Mitigation: test that the command exits with the correct code; unit test `FormatVetCheck` and `FormatResourceLine` formatting separately. This is a known limitation (documented in existing `config_init_test.go` comments).

**[Trade-off] `mod vet` flag surface is large for a validation command** → It mirrors `mod build` flags (`--values`, `--namespace`, `--name`, `--provider`, `--strict`). This is intentional: validation should match the build environment. A simpler `mod vet` that doesn't accept values wouldn't catch the most common errors (incomplete/incorrect values).

**[Trade-off] `FormatVetCheck` alignment column (34) is hardcoded** → If check labels grow longer than ~30 characters, the detail text won't align. Mitigation: current labels are 14-21 characters; 34 provides generous padding. If needed later, the column can be computed dynamically from the longest label in a batch.
