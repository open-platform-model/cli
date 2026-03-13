## 1. Extract shared helper

- [x] 1.1 Create `renderPreparedModuleRelease` function in `internal/workflow/render/render.go` using the common tail of `Release`
- [x] 1.2 Modify `Release` to load the provider earlier and call `renderPreparedModuleRelease`
- [x] 1.3 Modify `ReleaseFile` to load the provider earlier and call `renderPreparedModuleRelease`

## 2. Validation

- [x] 2.1 Run `task fmt` to format Go code
- [x] 2.2 Run `task lint` to ensure no linting errors
- [x] 2.3 Run `task test` to verify all tests still pass with the refactored code
