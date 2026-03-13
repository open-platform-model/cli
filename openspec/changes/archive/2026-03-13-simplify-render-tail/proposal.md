## Why

The render workflow (`internal/workflow/render/render.go`) currently has two large entrypoints (`Release` and `ReleaseFile`) that converge late and duplicate the same 30-line tail logic. This duplication adds orchestration complexity and makes the pipeline harder to read and extend. We want to reduce this complexity without changing user-visible behavior, render semantics, or Kubernetes functionality. This is the first improvement outlined in the `render-pipeline-simplification-plan.md`.

## What Changes

- Extract the common tail logic from `Release` and `ReleaseFile` into a single helper function `renderPreparedModuleRelease`.
- The common tail includes:
  - Applying namespace override
  - Loading the provider
  - Calling `pkgrender.ProcessModuleRelease`
  - Converting resources to `*unstructured.Unstructured`
  - Assembling the workflow `Result`
- The two public entrypoints (`Release` and `ReleaseFile`) will remain but become thin preparation adapters that call the common helper.

## Capabilities

### New Capabilities

*(None - this is an internal refactoring)*

### Modified Capabilities

*(None - this is an internal refactoring)*

## Impact

- `internal/workflow/render/render.go` will be shorter and more obviously staged.
- Resource conversion and `Result` assembly will exist in one place.
- Easier to test the shared tail separately from input preparation.
- Prepares the ground for further simplifications (like loading the provider earlier).
