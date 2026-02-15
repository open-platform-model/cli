## 1. Path-rewritten error wrapper type

- [x] 1.1 Implement `pathRewrittenError` struct in `internal/build/errors.go` that wraps `cueerrors.Error` and overrides `Path()` to return a custom path. Delegate all other methods (`Position()`, `InputPositions()`, `Msg()`, `Error()`) to the inner error.
- [x] 1.2 Implement `rewriteErrorPath(e cueerrors.Error, basePath []string) cueerrors.Error` helper that wraps a CUE error with prepended base path segments.
- [x] 1.3 Add unit tests for `pathRewrittenError`: verify `Path()` returns rewritten path, verify other methods delegate correctly, verify `formatCUEDetails` renders rewritten paths.

## 2. Source position finder

- [x] 2.1 Implement `findSourcePosition(v cue.Value) token.Pos` in `internal/build/errors.go`. Try `v.Pos()` first; if invalid, decompose via `v.Expr()` and return the first valid position from the conjunct parts. Return `token.NoPos` as final fallback.
- [x] 2.2 Add unit tests for `findSourcePosition`: single-source value returns `Pos()` directly, unified value returns a valid position from one of the parts, value with no position returns `token.NoPos`.

## 3. Recursive field validator

- [x] 3.1 Implement `validateFieldsRecursive(schema, data cue.Value, path []string, errs *cueerrors.Error)` in `internal/build/errors.go`. For each field in `data.Fields()`: check `schema.Allows(cue.Str(name))`, resolve schema field (literal then pattern via `.Optional()`), unify and check for type errors, recurse into struct children.
- [x] 3.2 Replace the body of `validateValuesAgainstConfig` to call `validateFieldsRecursive` with initial path `[]string{"values"}` and return the accumulated errors.
- [x] 3.3 Remove the old per-field isolation logic (single-field struct construction, `cueCtx` parameter if no longer needed).

## 4. Unit tests for recursive validator

- [x] 4.1 Update `TestValidateValuesAgainstConfig` — existing test cases should pass with updated error paths (`values.X` instead of `#config.X`). Update assertions accordingly.
- [x] 4.2 Add test: top-level disallowed field emits error with `values.` path and source position.
- [x] 4.3 Add test: nested disallowed field inside pattern-constrained struct emits error with full path (e.g., `values.media.tvshows.badField`) and source position.
- [x] 4.4 Add test: pattern constraint fields are accepted (no false "field not allowed" for `media.tvshows` when `[Name=string]` pattern exists).
- [x] 4.5 Add test: type mismatch at nested level (string where struct expected) emits error with correct path and does not recurse.
- [x] 4.6 Add test: optional fields in schema are accepted without error.
- [x] 4.7 Add test: deeply nested struct (3+ levels) validates correctly with full path in errors.
- [x] 4.8 Add test: multi-source unified value attributes error position to the correct source file via `Expr()` decomposition.

## 5. Integration with ReleaseBuilder

- [x] 5.1 Update `validateValuesAgainstConfig` function signature if needed (remove `cueCtx *cue.Context` parameter if no longer used by the new implementation).
- [x] 5.2 Update call site in `release_builder.go` Step 4b to match any signature changes.
- [x] 5.3 Verify the jellyfin example module: run `opm mod vet examples/jellyfin/ -f examples/jellyfin/values.cue -f examples/jellyfin/val.cue` and confirm errors show `values.` paths with file:line:col positions.

## 6. Validation gates

- [x] 6.1 Run `task fmt` — all Go files formatted.
- [x] 6.2 Run `task test` — all tests pass (including updated error assertions).
- [x] 6.3 Run `task check` — fmt + vet + test all pass.
