## Why

`opm mod vet` validation errors are hard to act on: error paths show `#config.X` instead of `values.X` (the user's perspective), "field not allowed" errors lack file/line positions entirely, and when multiple `-f` files are passed there's no way to tell which file introduced a bad field. Module authors waste time hunting down the source of errors that the tool should pinpoint.

## What Changes

- Replace the per-field isolation validation (`validateValuesAgainstConfig`) with a recursive field walker that:
  - Checks closedness manually using `cue.Value.Allows()` instead of relying on CUE's internal closedness checker (which produces sparse position info)
  - Validates type constraints by unifying each allowed field with its resolved schema counterpart
  - Handles pattern constraints (`[Name=string]: { ... }`) via `cue.Str(key).Optional()` resolution
- Rewrite error paths from `#config.X` to `values.X` so errors reflect the user's input, not CUE internals
- Attribute source positions to every error using `cue.Value.Pos()` and `cue.Value.Expr()` decomposition for unified values
- Keep error output flat (not grouped by file) — each error line includes its source file:line:col

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `build`: The values-against-config validation logic in `ReleaseBuilder` (Step 4b) changes. Requirements FR-B-063 and FR-B-064 are unaffected. New requirements cover recursive closedness checking, path rewriting, and source position attribution in validation errors.
- `mod-vet`: The CUE error details format in validation output changes. Error paths and positions become more specific. The scenario "Module with CUE validation errors fails with details" gets stronger guarantees about what "details" includes.

## Impact

- **Code**: `internal/build/errors.go` (replace `validateValuesAgainstConfig`, add recursive walker, add path rewriting wrapper type), `internal/build/release_builder.go` (update Step 4b call site), `internal/build/errors_test.go` (update/expand tests)
- **APIs**: No public API changes. `ReleaseValidationError` struct unchanged. `formatCUEDetails` unchanged.
- **User-facing**: Error output for `opm mod vet` and `opm mod build` improves. No flag changes, no new commands.
- **SemVer**: PATCH — error message improvements, no behavioral changes to success paths
- **Dependencies**: No new dependencies. Uses existing CUE SDK APIs (`Value.Allows`, `Value.Expr`, `Selector.Optional`)
