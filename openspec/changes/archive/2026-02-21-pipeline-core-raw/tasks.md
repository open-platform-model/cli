## 1. Update `core.Module`

- [x] 1.1 In `internal/core/module.go`, rename the private field `value cue.Value` to `Raw cue.Value`
- [x] 1.2 Remove the `CUEValue() cue.Value` method from `core.Module`
- [x] 1.3 Remove the `SetCUEValue(v cue.Value)` method from `core.Module`

## 2. Update callers in `internal/legacy/`

- [x] 2.1 In `internal/legacy/module/loader.go`, replace `mod.SetCUEValue(v)` with `mod.Raw = v`
- [x] 2.2 In `internal/legacy/release/builder.go`, replace `mod.CUEValue()` with `mod.Raw`

## 3. Fix any remaining callers

- [x] 3.1 Run `task build` and fix any compile errors from missed `CUEValue`/`SetCUEValue` call sites

## 4. Validation

- [x] 4.1 Run `task fmt` — all Go files formatted
- [x] 4.2 Run `task test` — all tests pass
