## 1. Rename function in module.go

- [x] 1.1 Rename `func Load(` to `func LoadModule(` in `internal/loader/module.go:41`
- [x] 1.2 Update doc comment on line 16 to reference `LoadModule`

## 2. Update call sites in tests and pipeline

- [x] 2.1 Update 13 call sites in `internal/loader/module_test.go`: `loader.Load(` → `loader.LoadModule(`
- [x] 2.2 Update call site in `internal/pipeline/pipeline.go:137`: `loader.Load(` → `loader.LoadModule(`
- [x] 2.3 Update comment in `internal/pipeline/pipeline.go:52`: `loader.Load()` → `loader.LoadModule()`
- [x] 2.4 Update 4 call sites in `internal/builder/builder_test.go`: `loader.Load(` → `loader.LoadModule(`
- [x] 2.5 Update call site in `tests/integration/values-flow/main.go:51`
- [x] 2.6 Update comment references in `experiments/values-flow/helpers_test.go:164,170`

## 3. Validation

- [x] 3.1 Run `task build` — compiler confirms no missed call sites
- [x] 3.2 Run `task test:unit` — all loader and builder tests pass
- [x] 3.3 Run `task lint` — no lint violations introduced by this change (66 pre-existing issues unrelated to rename)
