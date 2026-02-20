## 1. Delete AST inspection files

- [x] 1.1 Delete `internal/build/module/inspector.go`
- [x] 1.2 Delete `internal/build/module/types.go`

## 2. Rewrite loader.go

- [x] 2.1 Inline `load.Instances()` and `inst.PkgName` extraction directly into `Load`, removing `inspectModule`
- [x] 2.2 Extend `extractModuleMetadata` to include `metadata.name` and `metadata.defaultNamespace` via `LookupPath` + `.String()`
- [x] 2.3 Remove the two-step `Inspection`-then-metadata flow from `Load`; populate `mod.Metadata` after `BuildInstance()` using `extractModuleMetadata`
- [x] 2.4 Remove the `"cuelang.org/go/cue/build"` import if no longer referenced

## 3. Update core.Module validation message

- [x] 3.1 Update `Validate()` error message for empty `Name` to remove "string literal" wording (e.g., `"metadata.name is empty or not concrete"`)

## 4. Update tests

- [x] 4.1 Delete `TestExtractMetadataFromAST_NilFiles` from `internal/build/module/loader_test.go`
- [x] 4.2 Update `TestLoad_ComputedMetadataName` in `loader_test.go`: assert `Name == "computed-module"` (not empty)
- [x] 4.3 Delete `TestExtractMetadataFromAST` table test from `internal/build/release/ast_test.go`
- [x] 4.4 Update `TestLoad_MissingMetadata` in `ast_test.go`: assert `Name == "computed-module"` (not empty), remove "AST walk" comment

## 5. Component extraction fixes (internal/core/component.go)

- [x] 5.1 Add `Blueprints: make(map[string]cue.Value)` to the struct literal in `extractComponent()`
- [x] 5.2 Add `#blueprints` extraction block in `extractComponent()` after the `#traits` block (same pattern: `LookupPath` → `Fields()` → iterate → map by selector key)
- [x] 5.3 Add `spec` extraction block in `extractComponent()` after `#traits`: `if specValue := value.LookupPath(cue.ParsePath("spec")); specValue.Exists() { comp.Spec = specValue }`
- [x] 5.4 Add test assertions in `TestExtractComponents` (internal/core/component_test.go): assert `comp.Spec.Exists()` for the `web` component; assert `comp.Blueprints` is non-nil for all components

## 6. Build concreteness gate (internal/build/release/builder.go)

- [x] 6.1 After `core.ExtractComponents()` in `Build()`, add a loop: for each component, if `!comp.IsConcrete()`, return `fmt.Errorf("component %q is not concrete after value injection", name)`
- [x] 6.2 Add a test case in the builder tests (or release integration tests) asserting `Build()` returns an error when a component is not concrete

## 7. Test fixture and UUID assertion

- [x] 7.1 Add `identity: "test-uuid-1234"` to `internal/build/testdata/test-module/module.cue` under `metadata`
- [x] 7.2 Add `assert.NotEmpty(t, mod.Metadata.UUID, "UUID extracted from CUE eval")` to `TestLoad_ValidModule` in `loader_test.go`

## 8. Validation

- [x] 8.1 Run `task test` — all tests pass
- [x] 8.2 Run `task check` — fmt, vet, and tests all pass (pre-existing lint issues unrelated to this change)
