## Why

A code review surfaced four gaps in the module loading and component extraction pipeline: (1) the module loader uses an unnecessary AST inspection pass for metadata that CUE evaluation already provides; (2) `Component.Spec` is declared but never extracted; (3) `Component.Blueprints` is declared but never initialized or extracted; (4) `Build()` does not gate on `IsConcrete()` per component as specified. A fifth gap is a missing UUID test assertion that left the identity extraction path untested.

## What Changes

**Module loader simplification:**

- Remove `ExtractMetadataFromAST`, `extractFieldsFromMetadataStruct`, and the `Inspection` struct
- Delete `inspector.go` and `types.go` from `internal/build/module`
- Collapse `inspectModule` into `Load` — inline `load.Instances()` and `inst.PkgName` extraction directly
- Extend `extractModuleMetadata` to cover `name` and `defaultNamespace` via `LookupPath` + `.String()`
- Update tests that relied on AST inspection behavior; update `Validate()` error message

**Component extraction fixes (internal/core/component.go):**

- Extract `spec` field in `extractComponent()` → `comp.Spec`
- Initialize `Blueprints: make(map[string]cue.Value)` in `extractComponent()` and add `#blueprints` extraction block analogous to `#traits`

**Build concreteness gate (internal/build/release/builder.go):**

- After `core.ExtractComponents()` in `Build()`, loop over components and return an error for any component where `!comp.IsConcrete()`

**Test coverage:**

- Add `identity` field to `test-module` fixture; add `UUID` assertion to `TestLoad_ValidModule`

## Capabilities

### New Capabilities

- `module-metadata-extraction`: Unified metadata extraction from CUE evaluation — all `ModuleMetadata` scalar fields populated from a single evaluated `cue.Value` via `LookupPath`. No AST walk involved.
- `core-component-extraction`: Complete component field extraction — `Spec`, `Blueprints`, `Resources`, and `Traits` all populated from `extractComponent()`; `Blueprints` always initialized.

### Modified Capabilities

- `build`: Remove AST-inspection-based metadata extraction requirement; add explicit requirement that `Build()` returns an error if any component is not concrete after `FillPath`.

## Impact

- **internal/build/module**: `loader.go` simplified; `inspector.go` and `types.go` deleted
- **internal/core/component.go**: `extractComponent()` — add `Spec` extraction, `Blueprints` initialization and extraction
- **internal/build/release/builder.go**: `Build()` — add `IsConcrete()` gate after component extraction
- **internal/build/testdata/test-module/module.cue**: add `metadata.identity` field
- **internal/build/module/loader_test.go**: add UUID assertion; remove AST-only tests; update computed-name assertion
- **internal/build/release/ast_test.go**: remove `TestExtractMetadataFromAST`; update `TestLoad_MissingMetadata`
- **SemVer**: PATCH — no public API or behavioral change for well-formed modules
- **Breaking**: None
