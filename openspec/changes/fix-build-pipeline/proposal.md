# Proposal: Fix Build Pipeline

## Why

The `opm mod build` command fails with "no components found" because OPM modules define `#components` (a CUE definition/schema) while the module loader looks for `components` (a concrete field). This is a fundamental mismatch between how modules are authored (using CUE definitions) and how the CLI loads them.

## What Changes

- Add synthetic `#ModuleRelease` creation during module loading to force CUE definitions to become concrete values
- The loader will wrap the loaded `#Module` in a `#ModuleRelease` structure, which:
  - Provides the required `metadata.namespace` from the CLI `--namespace` flag
  - Extracts `components` from `#module.#components` (forcing concreteness)
  - Validates that all required fields are satisfied (failing early with clear errors)
- Developers using `opm mod build` won't need to create a separate release file during development

## Capabilities

### New Capabilities

- `module-release-synthesis`: Automatic creation of `#ModuleRelease` from `#Module` during build, extracting concrete components without requiring a separate release file.

### Modified Capabilities

_None. This is an internal implementation change to the module loader that doesn't change the render pipeline interface requirements._

## Impact

- **Packages**: `internal/build/module.go` - ModuleLoader will need to synthesize a release
- **Dependencies**: Requires access to `opmodel.dev/core@v0` schema definitions for `#ModuleRelease`
- **User Experience**: `opm mod build` will work with standard OPM modules without requiring a release file
- **Error Handling**: Must surface CUE validation errors clearly when required fields are missing (e.g., `container.name`, `workload-type` label)
