## Context

The rendering logic inside `internal/workflow/render/render.go` implements two main functions: `Release` (rendering from a module directory) and `ReleaseFile` (rendering from a `#ModuleRelease` file). While they take different sets of parameters and prepare their `pkgrender.ModuleRelease` and `values` differently, the end of both functions is virtually identical. They both handle namespace overrides, load the provider, execute the core render engine, and convert the resulting resources into unstructured formats.

This duplication makes the code harder to follow and maintain, violating the goal of a clear, staged pipeline (Prepare -> Process -> Emit).

## Goals / Non-Goals

**Goals:**

- Extract the shared tail logic into a single internal helper function: `renderPreparedModuleRelease`.
- Shrink the size of `Release` and `ReleaseFile` to make them clear "preparation adapters".
- Prepare for future refactorings by expecting the provider to be loaded *before* the processing phase.

**Non-Goals:**

- Do not change how CUE values are evaluated or merged.
- Do not refactor `ModuleRelease` parsing or initialization (this will be done in separate iterations).
- Do not change any Kubernetes apply/emit behavior.

## Decisions

1. **Helper Signature**: The new helper will have the following signature:

   ```go
   func renderPreparedModuleRelease(
       ctx context.Context,
       rel *pkgrender.ModuleRelease,
       valuesVals []cue.Value,
       p *provider.Provider,
       namespaceOverride string,
   ) (*Result, error)
   ```

2. **Provider Loading**: Since the helper expects a `*provider.Provider`, the responsibility of loading the provider (`loader.LoadProvider`) shifts up into `Release` and `ReleaseFile` just before calling the helper. This aligns perfectly with the future goal of fully resolving all inputs during the "Prepare" phase before entering the "Process" phase.
3. **Error Handling**: The shared tail will maintain the exact same error types and formatting, returning `*opmexit.ExitError` where appropriate, ensuring CLI behavior and exit codes remain unchanged.

## Risks / Trade-offs

- **Risk**: Modifying the render tail might subtly change error wrapping or printed messages if not copied exactly.
  - **Mitigation**: The code for the shared tail will be copy-pasted directly from the existing tail of `Release`, ensuring parity in error handling (e.g., using `printValidationError`).
