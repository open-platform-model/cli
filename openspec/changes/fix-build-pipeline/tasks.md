# Tasks: Fix Build Pipeline

## 1. Namespace Resolution

- [x] 1.1 Add `resolveNamespace` function to extract namespace from flag or module's `defaultNamespace`
- [x] 1.2 Update `RenderOptions.Validate()` to defer namespace validation until after module load
- [x] 1.3 Add error type for missing namespace with actionable message

## 2. Module Type Detection

- [x] 2.1 Add `isModuleRelease` function to check if loaded value has concrete `components` field
- [x] 2.2 Update `ModuleLoader.Load()` to branch based on module type detection

## 3. Release Synthesis

- [x] 3.1 Add `synthesizeRelease` function that builds CUE expression wrapping module in `#ModuleRelease`
- [x] 3.2 Implement CUE unification of release expression with loaded module value
- [x] 3.3 Handle namespace injection from resolved value (flag or defaultNamespace)

## 4. Component Extraction

- [x] 4.1 Update `extractComponents` to read from `_release.components` path after synthesis
- [x] 4.2 Ensure metadata extraction works with both direct release and synthesized release paths

## 5. Error Handling

- [x] 5.1 Add error wrapper for CUE validation errors with component/field context
- [x] 5.2 Parse CUE error messages to extract field paths and constraint info
- [x] 5.3 Format user-friendly error messages with guidance on fixing

## 6. Testing

- [x] 6.1 Add test case: module with `#components` synthesizes correctly
- [x] 6.2 Add test case: explicit `#ModuleRelease` used directly
- [x] 6.3 Add test case: namespace from `--namespace` flag
- [x] 6.4 Add test case: namespace from `defaultNamespace` when flag omitted
- [x] 6.5 Add test case: error when neither namespace source available
- [x] 6.6 Add test case: validation error surfaces with component/field context

## 7. Integration

- [x] 7.1 Test `opm mod build` with `testing/blog` module
- [x] 7.2 Verify error messages are actionable for missing required fields
- [x] 7.3 Update blog module to provide missing required fields if needed
