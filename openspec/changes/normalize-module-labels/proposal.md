## Why

`opm mod delete --name blog` silently succeeds with "0 resources deleted" when the module is labeled as `Blog` (PascalCase) on the cluster. The label selector does an exact string match, so the case mismatch causes a no-op with exit code 0. Users have no indication their delete failed due to casing.

This is part of a broader normalization effort: the OPM catalog is moving module names from PascalCase (`Blog`) to k8s-style lowercase-with-hyphens (`blog`). The CLI must normalize defensively to match, and fix the silent-success bug in delete.

## What Changes

- **BREAKING**: Module names in labels (`module.opmodel.dev/name`) are now stored as lowercase. Existing cluster resources with PascalCase labels will not be found by delete until re-applied.
- `injectLabels()` lowercases `meta.Name` before writing to the `module.opmodel.dev/name` label.
- The `--name` override flag (on apply/build) is lowercased before use in labels, ensuring overrides that bypass CUE validation are still normalized.
- `mod delete` returns a non-zero exit code when no resources are found, instead of silently succeeding with exit 0.
- Delete label matching remains strict and case-sensitive — no normalization of delete input.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `build`: Module loading override (`--name` flag, FR-B-034) now lowercases the value before label injection. Label injection in apply normalizes `meta.Name` to lowercase.

## Impact

- **`internal/kubernetes/apply.go`**: `injectLabels()` — add `strings.ToLower()` on `meta.Name`.
- **`internal/build/module.go`**: `applyOverrides()` — lowercase the `--name` override.
- **`internal/kubernetes/delete.go`**: `Delete()` — return error (not nil) when 0 resources discovered.
- **`internal/cmd/mod_delete.go`**: `runDelete()` — handle the new error from `Delete()`, exit non-zero.
- **SemVer**: MINOR (label value format change is internal; delete exit code change is a behavioral fix). The breaking label change is mitigated by the catalog-side CUE enforcement — new modules will already use lowercase names.
- **Companion change**: Catalog repo will enforce lowercase names in `#NameType` and derive PascalCase for FQN via CUE `strings.ToTitle`. That change is independent and not scoped here.
