## Context

`internal/loader` now contains two loaders: the module loader (`module.go`) and the provider loader (`provider.go`, recently moved from `internal/provider`). Both live in the same Go package. The module loader's exported function was named `Load`, which is generic and ambiguous now that `LoadProvider` exists alongside it. This design covers the mechanical rename of `Load` → `LoadModule`.

## Goals / Non-Goals

**Goals:**

- Rename `func Load(...)` to `func LoadModule(...)` in `internal/loader/module.go`
- Update all call sites and references across the repository
- Make the `internal/loader` API self-documenting: `LoadModule` / `LoadProvider`

**Non-Goals:**

- Changing function signatures or behavior
- Moving any files
- Modifying the provider loader

## Decisions

**Rename only, no interface extraction**
A simple rename is sufficient. No interface or adapter layer is needed — all callers are internal and can be updated directly. Adding indirection here would violate Principle VII (YAGNI).

**Update comments and doc strings**
All references to `Load()` in comments (including `pipeline.go:52` and `experiments/`) will be updated to `LoadModule()` for accuracy.

## Risks / Trade-offs

No behavioral risk — pure rename with no logic changes. The only risk is a missed call site causing a compile error, which the Go compiler will catch immediately.

## Migration Plan

1. Rename `func Load` → `func LoadModule` in `module.go`
2. Update all `loader.Load(` → `loader.LoadModule(` call sites
3. Update comments referencing `loader.Load()`
4. Run `task build` to confirm no missed sites (compiler-verified)
5. Run `task test` to confirm tests pass
