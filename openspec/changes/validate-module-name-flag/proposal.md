## Why

`opm mod delete --name blog` silently succeeds with "0 resources deleted" when no matching module exists on the cluster. The label selector does an exact string match, so a miss is a no-op with exit code 0. Users have no indication their delete found nothing.

Separately, the OPM catalog is moving module names from PascalCase (`StatelessWorkload`) to k8s-style kebab-case (`stateless-workload`) via the `k8s-name-constraints` change, which redefines `#NameType` to enforce RFC 1123 DNS labels at the CUE schema level. The `--name` CLI flag bypasses CUE validation, so the CLI must independently validate that flag values conform to the same constraint.

## What Changes

- `--name` override flag (on apply/build) is validated against RFC 1123 DNS label format. Invalid names (e.g., `MyBlog`, `MY-BLOG`) are rejected with an actionable error.
- `mod delete` returns a non-zero exit code when no resources are found, instead of silently succeeding with exit 0.
- Delete label matching remains strict and case-sensitive — no normalization of delete input.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `build`: Module loading override (`--name` flag, FR-B-034) now validates the value against RFC 1123 DNS label format before use.

## Impact

- **`internal/build/module.go`**: `applyOverrides()` — validate `--name` override against RFC 1123 regex; return error if invalid.
- **`internal/kubernetes/delete.go`**: `Delete()` — return error (not nil) when 0 resources discovered.
- **`internal/cmd/mod_delete.go`**: `runDelete()` — handle the new error from `Delete()`, exit non-zero.
- **SemVer**: MINOR (stricter input validation on `--name` flag; delete exit code change is a behavioral fix).
- **Companion change**: Catalog repo enforces kebab-case names via `#NameType` (RFC 1123 DNS labels) and derives PascalCase for FQN via a custom `#KebabToPascal` function. That change is independent and not scoped here.
