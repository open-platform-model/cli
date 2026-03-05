# Module Import Experiment

## Purpose

Test whether the **flattened module authoring style** (embedding `core.#Module` at package root) is compatible with CUE's native module/import system.

## Context

Currently, OPM modules use this flattened pattern:

```cue
package main  // ← Currently "main", needs to change

core.#Module  // ← Embedded at package root

metadata: { ... }
#config: { ... }
#components: { ... }
```

This eliminates one level of nesting compared to:

```cue
myModule: core.#Module & {
    metadata: { ... }
    #config: { ... }
}
```

The question: **Can we keep the flattened style AND make modules importable via CUE's module system?**

## Key Questions

| # | Question | Test Coverage |
|---|----------|---------------|
| **Q1** | Can a flattened module (core.#Module embedded at root) be loaded? | `TestFlattenedModuleImport/simple_module_without_values` |
| **Q2** | Are hidden definitions (`#config`, `#components`) accessible from importing packages? | `TestFlattenedModuleImport` (both cases) |
| **Q3** | If `values.cue` adds an extra `values` field (not in `#Module`), does assignment to a `#module` field fail due to closedness? | `TestFlattenedModuleImport/module_with_values_field` + `TestModuleAssignedToModuleField` |
| **Q4** | Does excluding `values.cue` avoid any issues? | `TestFlattenedModuleImport/simple_module_without_values` |
| **Q5** | Does the full `#ModuleRelease` flow work: `#module: importedPkg, values: {...}` → resolved components? | `TestModuleReleaseIntegration` |

## Test Structure

```
experiments/module-import/
├── README.md
├── module_import_test.go
└── testdata/
    ├── simple_module/          # WITHOUT values.cue
    │   ├── cue.mod/module.cue  # module: "test.dev/modules/simple@v0"
    │   ├── module.cue          # package simple; core.#Module embedded
    │   └── components.cue      # package simple; #components
    │
    └── module_with_values/     # WITH values.cue
        ├── cue.mod/module.cue
        ├── module.cue
        ├── components.cue
        └── values.cue          # Adds "values: {...}" at package root
```

## Expected Behavior

### CUE Import Semantics

When you `import "test.dev/modules/simple@v0"`, CUE gives you the **package scope as a struct**. If `core.#Module` is embedded at the root, the imported value should contain:

- All fields from `core.#Module` (apiVersion, kind, metadata, #config, #components, etc.)
- Any additional fields defined in the package files

### The `values.cue` Question

The `#Module` definition does NOT have a `values` field. If a module package includes `values.cue`, which writes:

```cue
values: { ... }
```

...at the package root (outside `#Module`), this creates an extra field.

**Potential issue**: When assigning the imported package to a field typed as `#Module`, the closedness of `#Module` might reject the extra `values` field.

**From CUE spec**:
> "An embedded value of type struct is unified with the struct in which it is embedded, but disregarding the restrictions imposed by closed structs."

This suggests the `values` field should be allowed in the **authoring** context (where `#Module` is embedded), but when the package is **imported and assigned to a `#module: #Module` field**, the closedness might apply.

## Possible Outcomes

### Scenario A: Everything Works

- ✅ Flattened modules load
- ✅ Hidden fields (`#config`, `#components`) are accessible
- ✅ Extra `values` field is allowed (embedding exception applies even on import)
- ✅ `#ModuleRelease` flow works

**Implication**: Authors can use the flattened style, change to `package <name>`, and ship modules as-is (including `values.cue` for debug/defaults).

### Scenario B: `values.cue` Breaks It

- ✅ Flattened modules load
- ✅ Hidden fields are accessible
- ❌ Extra `values` field causes unification error when assigned to `#module: #Module`
- ⚠️ `#ModuleRelease` fails if module includes `values`

**Implication**: Authors must exclude `values.cue` from published modules. Options:
1. Keep `values.cue` local (dev-only, not in published package)
2. Use a separate sub-package for defaults
3. Add `values` field to `#Module` definition itself

### Scenario C: Flattened Style Doesn't Work

- ❌ Flattened modules have issues (closedness, visibility, etc.)
- Need to switch to explicit named exports: `#Module: core.#Module & { ... }`

**Implication**: Accept one more level of nesting for importability.

## Running the Tests

```bash
cd experiments/module-import
go test -v
```

## Findings

**All tests pass.** The experiment confirms **Scenario B**: Flattened modules work, but `values.cue` breaks importability.

### Q1: Can flattened modules be loaded?
- [x] **Yes** ✓

Modules with `core.#Module` embedded at package root load successfully. The package scope contains all `#Module` fields plus any additional fields defined in the package.

### Q2: Are `#config` and `#components` accessible?
- [x] **Yes** ✓

Hidden definitions (`#config`, `#components`, `#policies`) are fully accessible from the loaded package. CUE's `#` prefix means "closed/validated", NOT "private". Cross-package visibility works as expected.

### Q3: Does `values.cue` cause conflicts?
- [x] **Conflict when assigned to `#module` field**

When a package includes `values.cue` (which adds a `values` field at package root), the loaded value has this structure:

```
Module fields:  apiVersion, kind, metadata, #config, #components, debugValues
Extra field:    values  ← NOT in #Module definition
```

When unified with `#Module` (e.g., `#module: importedPackage`), CUE rejects the extra field:

```
Error: #Module.values: field not allowed
```

**Root cause**: `#Module` is a closed definition. When referenced, it doesn't allow fields outside its schema. The `values` field from `values.cue` is written at package root, outside the `#Module` embedding.

### Q4: Does excluding `values.cue` fix it?
- [x] **Yes** ✓

Modules WITHOUT `values.cue` (only `module.cue` + `components.cue`) unify cleanly with `#Module`. The flattened style works perfectly when the package contains only the fields defined in `#Module`.

### Q5: Does `#ModuleRelease` integration work?
- [x] **Yes (simple_module)** ✓
- [ ] Yes (module_with_values) — would fail due to Q3
- [x] **Partial (only without values)** ✓

The full `#ModuleRelease` flow works when the imported module doesn't include extra fields:

```cue
release: core.#ModuleRelease & {
    #module: importedModule  // ✓ Works without values.cue
    values: { replicas: 5 }  // User-provided config
}
```

Components resolve correctly, with user values flowing through to component specs.

## Conclusion

**The flattened authoring style IS compatible with CUE module imports**, with one constraint: **module packages must not include extra fields beyond what `#Module` defines**.

This means:
- ✅ Keep `core.#Module` embedded at package root
- ✅ Change `package main` → `package <modulename>`
- ✅ `#config`, `#components`, `#policies` work perfectly
- ❌ `values.cue` at package root breaks importability

## Recommendations

### Option A: Exclude `values.cue` from Published Modules (Recommended)

Module authors should:
1. Keep the flattened style in `module.cue` and `components.cue`
2. Use `values.cue` for local development/testing only
3. Exclude `values.cue` from published packages (via `.cue` ignore or build tags)
4. Rely on `#config` defaults (via `| *defaultValue`) for module defaults

**Rationale**: `#config` already supports defaults via CUE's default value syntax. The `values` field is redundant for published modules — it was primarily for the CLI's current loading strategy.

### Option B: Add `values` Field to `#Module` Definition

Modify `core.#Module` to include:

```cue
#Module: {
    // ... existing fields ...
    values?: _  // Optional: module author defaults
}
```

**Pros**: Modules can ship with concrete defaults in `values.cue`  
**Cons**: Adds a field to `#Module` that's only used for defaults, not semantically meaningful for the module definition

### Option C: Use a Sub-Package for Defaults

Module authors could create a separate package for defaults:

```
jellyfin/
├── module.cue       # package jellyfin; core.#Module
├── components.cue   # package jellyfin
└── defaults/
    └── values.cue   # package defaults; values: {...}
```

**Cons**: More complex, doesn't integrate cleanly with current `#ModuleRelease` flow

### Option D: `@if(dev)` Build Tag (Recommended)

Use CUE's native `@if()` build attribute to conditionally include `values.cue`:

```cue
// values.cue
@if(dev)

package jellyfin

values: { ... }
```

**How it works**:
- The OPM CLI loads modules with `-t dev` (or `load.Config.Tags: ["dev"]` in Go)
- `values.cue` is included during local development — validated against `#config`
- When imported as a CUE dependency (no tags active), `values.cue` is automatically excluded
- The imported package contains only `#Module` fields — clean unification

**Why this is the best option**:

| Criterion                                  | @if(dev) | Exclude from publish | Add to #Module | Separate pkg |
|--------------------------------------------|----------|----------------------|----------------|--------------|
| values.cue validated against #config       | [x]      | [ ]                  | [x]            | [ ]          |
| Works with pure CUE tooling (not just CLI) | [x]      | [ ]                  | [x]            | [x]          |
| No changes to #Module definition           | [x]      | [x]                  | [ ]            | [x]          |
| Author ceremony                            | 1 line   | build/publish config | 0              | new directory |
| CUE-idiomatic                              | [x]      | [ ]                  | [x]            | [ ]          |

CUE also supports `@ignore()` which unconditionally excludes a file from ALL builds,
but that loses the ability to validate values locally. `@if(dev)` is strictly better
because it gives you both worlds.

## Alternative Approaches to `values.cue` Exclusion

Four approaches were considered for handling the `values.cue` conflict:

### 1. `@if(dev)` build tag (Recommended)

Add `@if(dev)` as the first line of `values.cue`. The OPM CLI sets the `dev` tag
during local loading (`load.Config.Tags: ["dev"]`). When the module is imported
as a dependency, no tags are active and the file is skipped.

- ✅ values.cue validated against #config during local dev
- ✅ Excluded automatically on import
- ✅ Works with Go SDK AND pure CUE tooling
- ✅ One line of ceremony for authors

### 2. `@ignore()` unconditional exclusion

Add `@ignore()` as the first line of `values.cue`. The file is excluded from ALL
CUE builds unconditionally. The CLI must load and compile it separately.

- ❌ values.cue can never participate in CUE evaluation (no #config validation)
- ✅ Excluded on import
- ⚠️ CLI must handle values.cue as raw file, outside CUE's evaluation

### 3. Programmatic Go-side filtering (current Approach A)

The existing `module-release-cue-eval` experiment already filters `values*.cue`
from `load.Instances` file lists in Go code. This works for the OPM CLI but does
not affect CUE's native import resolution.

- ❌ Only works via Go SDK — pure CUE imports still include values.cue
- ✅ Already implemented in existing experiments
- ⚠️ Defeats the goal of making modules importable via CUE's native system

### 4. Separate package for values

Put values in a different package (`package jellyfin_values`) so they're not
loaded when importing the main package.

- ❌ values.cue can't reference #config (different package scope)
- ❌ More complex directory structure
- ✅ Works with CUE's native import system

## Future Test: `@if(dev)` Verification

A test case should be added to `module_import_test.go` to prove the `@if(dev)`
approach works end-to-end:

1. Create `testdata/module_with_tagged_values/` — same as `module_with_values`
   but with `@if(dev)` as the first line of `values.cue`
2. **Test A**: Load WITH tag (`load.Config{Tags: ["dev"]}`) → `values` field
   exists, module works locally
3. **Test B**: Load WITHOUT tag (`load.Config{}`) → `values` field excluded,
   module unifies cleanly with `#Module`
4. **Test C**: Full `#ModuleRelease` integration with the tagged module (no tag)
   → works identically to `simple_module`

This would confirm that `@if(dev)` is the production-ready solution and can be
documented as the standard convention for OPM module authoring.

## Next Steps

Based on findings:
- [x] Flattened style works for imports ✓
- [x] `values.cue` causes conflicts ✓
- [x] `@if(dev)` is the recommended solution ✓
- [ ] Implement `@if(dev)` verification test
- [ ] Update module authoring guidelines
- [ ] Convert example modules to `package <name>` with `@if(dev)` on values.cue
- [ ] Update CLI loader to pass `-t dev` tag when loading local modules
