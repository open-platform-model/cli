## 1. Package scaffold

- [x] 1.1 Create `internal/builder/` directory
- [x] 1.2 Create `internal/builder/builder.go` with package declaration, `Options` struct (Name, Namespace string), and empty `Build(ctx *cue.Context, mod *core.Module, opts Options, valuesFiles []string) (*core.ModuleRelease, error)` signature
- [x] 1.3 Create `internal/builder/values.go` with package declaration and empty `selectValues` and `loadValuesFile` function stubs

## 2. `values.go` — value selection

- [x] 2.1 Implement `loadValuesFile(ctx *cue.Context, path string) (cue.Value, error)`: resolve abs path, read file, `ctx.CompileBytes` with filename, return error if compile fails
- [x] 2.2 Implement `selectValues(ctx *cue.Context, mod *core.Module, valuesFiles []string) (cue.Value, error)`: if files provided, load and unify them, extract top-level `values` field; else fall back to `mod.Values`
- [x] 2.3 Return `*core.ValidationError` when no `values` field found in provided files, and when `mod.Values` is absent with no files given
- [x] 2.4 Confirm unification of multiple files applies later files with precedence (via `Unify` chain) and propagates any `unified.Err()`

## 3. `builder.go` — Approach C injection sequence

- [x] 3.1 Load `opmodel.dev/core@v0` via `load.Instances([]string{"opmodel.dev/core@v0"}, &load.Config{Dir: mod.ModulePath})` using the provided `ctx`
- [x] 3.2 Call `ctx.BuildInstance` on the core instance and extract `#ModuleRelease` via `coreVal.LookupPath(cue.ParsePath("#ModuleRelease"))`; return error if it does not exist or has an error
- [x] 3.3 Call `selectValues` to get the selected values; validate them against `mod.Raw.LookupPath(cue.ParsePath("#config"))` via `Unify`; return `*core.ValidationError` on mismatch
- [x] 3.4 Perform the FillPath chain in order: `#module` → `metadata.name` → `metadata.namespace` → `values` (referencing design Decision 4)
- [x] 3.5 Call `result.Validate(cue.Concrete(true))`; wrap any error as `*core.ValidationError` identifying which fields remain non-concrete
- [x] 3.6 Read back `metadata.uuid`, `metadata.version`, and `metadata.labels` from the concrete result via `LookupPath` + typed decode
- [x] 3.7 Call `core.ExtractComponents(result.LookupPath(cue.ParsePath("components")))` to extract components map
- [x] 3.8 Construct and return `*core.ModuleRelease` with populated Metadata, Module embed, Components, and Values

## 4. Tests — `values.go`

- [x] 4.1 Table-driven test for `selectValues`: no files + module has values → returns `mod.Values`
- [x] 4.2 Table-driven test for `selectValues`: no files + module has no values → returns `ValidationError`
- [x] 4.3 Table-driven test for `selectValues`: one file provided → loads file, extracts `values` field
- [x] 4.4 Table-driven test for `selectValues`: two files → unified result reflects both files, later file takes precedence on conflict
- [x] 4.5 Test for `selectValues`: file with no `values` field → returns `ValidationError`

## 5. Tests — `builder.go`

- [x] 5.1 Test `Build` with a real module fixture and valid values (requires `OPM_REGISTRY`; skip if not set) — assert non-nil release, UUID matches UUID regex, labels populated, components non-empty
- [x] 5.2 Test `Build` with values that violate `#config` schema — assert `*core.ValidationError` returned before injection
- [x] 5.3 Test `Build` with values that leave `#ModuleRelease` non-concrete — assert `*core.ValidationError` from concreteness check
- [x] 5.4 Test UUID determinism: call `Build` twice with identical inputs, assert `Metadata.UUID` is equal across both calls

## 6. Validation gates

- [x] 6.1 Run `task fmt` — all files pass gofmt
- [x] 6.2 Run `task test` — all tests pass (registry-dependent tests skipped if `OPM_REGISTRY` unset)
