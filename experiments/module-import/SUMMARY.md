# Module Import Experiment — Summary

**Date**: 2026-03-02  
**Status**: ✅ Complete

## What We Tested

Can OPM modules keep the **flattened authoring style** (embedding `core.#Module` at package root) and still be importable via CUE's native module system?

## Answer

**YES**, with one critical constraint.

## Key Findings

### ✅ What Works

1. **Flattened embedding works perfectly**
   ```cue
   package jellyfin  // ← Changed from "main"
   
   core.#Module      // ← Embedded at root (no extra nesting)
   
   metadata: { ... }
   #config: { ... }
   #components: { ... }
   ```

2. **Hidden definitions are accessible**
   - `#config`, `#components`, `#policies` are fully visible across package boundaries
   - CUE's `#` prefix means "closed/validated", NOT "private"

3. **Full `#ModuleRelease` integration works**
   ```cue
   release: core.#ModuleRelease & {
       #module: importedModule  // ✓
       values: { replicas: 5 }
   }
   ```

### ❌ What Breaks

**Including `values.cue` at package root breaks importability.**

When `values.cue` writes:
```cue
package jellyfin

values: { ... }  // ← At package root, outside #Module
```

The package gains an extra field `values` that's NOT in the `#Module` definition. When this package is assigned to a field typed as `#Module`, CUE's closedness check rejects it:

```
Error: #Module.values: field not allowed
```

## Why This Happens

1. `core.#Module` is a **closed definition** (starts with `#`)
2. When referenced (e.g., `#module: importedPackage`), closedness is enforced
3. The `values` field from `values.cue` is written at **package scope**, not inside the `#Module` embedding
4. CUE sees it as an extra field and rejects the unification

## The Solution

**Use CUE's `@if(dev)` build tag to conditionally include `values.cue`.**

```cue
@if(dev)

package jellyfin

values: { ... }
```

### Why This Works Perfectly

- OPM CLI loads with `-t dev` tag → `values.cue` included and validated
- Import as dependency (no tags) → `values.cue` automatically excluded
- Works with both Go SDK AND pure CUE tooling (`cue eval`, etc.)
- Zero structural changes to `#Module` definition
- One line of ceremony for module authors

### Changes Needed

1. **Module authoring**:
   - Change `package main` → `package <modulename>`
   - Add `@if(dev)` as the first line of `values.cue`
   - Ensure `#config` has defaults via `| *defaultValue` for fields not in values.cue

2. **CLI changes**:
   - Pass `-t dev` tag (or `load.Config.Tags: ["dev"]`) when loading local modules
   - When loading imported modules, use standard CUE load (no tags)
   - User-provided values unify with `#config` at release time

## Test Results

All tests pass. See `module_import_test.go` for details:

- ✅ `TestFlattenedModuleImport` — both with and without values.cue
- ✅ `TestModuleAssignedToModuleField` — unification test (confirms values.cue breaks it)
- ✅ `TestModuleReleaseIntegration` — full end-to-end flow

## What This Means for OPM

You can **keep the flattened module authoring style** you wanted, AND get CUE module distribution for free. The changes:

```diff
  jellyfin/
  ├── cue.mod/module.cue
- ├── module.cue       // package main
+ ├── module.cue       // package jellyfin
  ├── components.cue
- └── values.cue       // No build tag
+ └── values.cue       // @if(dev) at top
```

Clean, minimal, and works with CUE's native dependency system. The `@if(dev)` tag is CUE-idiomatic and gives you the best of both worlds: validation during dev, clean imports in production.
