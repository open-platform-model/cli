# values-load-isolation

Experiment investigating and proving a fix for a production bug in the CUE module loader where multiple `values*.cue` files in the same module directory cause CUE unification conflicts.

## Problem

`loader.Load()` calls `load.Instances([]string{"."}, cfg)`, which loads **all** `.cue` files in the module directory into a single CUE package evaluation. CUE unifies all of them. When a module directory contains multiple `values*.cue` files (e.g. `values.cue`, `values_forge.cue`, `values_testing.cue`), their concrete fields conflict during unification:

```
opm mod vet . --release-name mc
→ conflicting values "FORGE" and "PAPER"
```

A second consequence: even when the user provides `--values`, `values.cue` is already baked into `mod.Raw` via the package load. When the builder tries to inject user values via `FillPath("values", selectedValues)`, the concrete values already in `mod.Raw` cause a second CUE conflict.

## Approaches Tested

### Approach A — Explicit filtered file list

Filter `values*.cue` from the file list passed to `load.Instances`. Load `values.cue` separately via `ctx.CompileBytes`.

**Result: Works.** Filtering is complete — no conflict, `#components` and metadata intact, `FillPath` succeeds.

Key findings:

- Explicit `.cue` filenames from the same directory still produce **exactly one instance** (grouped by package, not by argument count).
- After filtering, `values.serverType` in `baseVal` is abstract — the schema constraint remains but no default is baked in.
- `values.cue` loaded via `ctx.CompileBytes` provides concrete defaults cleanly, outside package unification.

### Approach B — CUE overlay (stub shadowing)

Replace every `values*.cue` file in the overlay with a package-declaration-only stub (`"package main\n"`), suppressing all concrete fields.

**Result: Works.** Achieves the same isolation as Approach A.

Extra complexity: requires extracting the package name from the file before constructing the stub so it stays in the same CUE package.

### Approach C — Rogue-file validator

Before loading, scan the module directory and error immediately if any file matches `values*.cue` but is not exactly `values.cue`.

**Result: Works as a pre-check gate.** Does not fix the load strategy alone, but enforces the contract that only `values.cue` is allowed inside the module directory.

Error message:

```
module directory contains 2 unexpected values file(s): values_forge.cue, values_testing.cue
Only values.cue is allowed inside the module directory.
Move environment-specific files outside the module and use --values to reference them.
```

## Chosen Solution

**Approach A + C combined.**

- Approach A is simpler than B (no package-name extraction, no overlay construction).
- Approach C layers on top as a validation gate: modules with rogue files are rejected before loading, giving a clear error message.

## New Loader Internals

```
loader.Load(cueCtx, modulePath, registry) — same signature

Step 1  ResolvePath (unchanged)

Step 2  Enumerate .cue files in module dir (non-recursive, excluding cue.mod/)
        Separate into:
          moduleFiles  — files that do NOT match values*.cue
          valuesFiles  — files that match values*.cue

Step 3  Validate rogue files (Approach C)
        if any valuesFiles where base != "values.cue" → error

Step 4  Validate non-empty package
        if len(moduleFiles) == 0 → error

Step 5  Load package from explicit file list (Approach A)
        load.Instances(moduleFiles, &load.Config{Dir: mod.ModulePath})

Step 6  BuildInstance (unchanged)

Step 7  Extract metadata, #config, #components (unchanged)

Step 8  Populate mod.Values (priority order):
        a. If "values.cue" present → load via ctx.CompileBytes, extract "values" field
        b. Else → fallback to mod.Raw.LookupPath("values") for inline concrete defaults
        c. Else → mod.Values stays zero

Step 9  Set mod.Raw (unchanged)
```

See [PLAN.md](PLAN.md) for the full implementation plan, pipeline change, and test coverage matrix.

## Files

```
├── PLAN.md                   Implementation plan and design decisions
├── baseline_conflict_test.go Reproduces the production bug
├── approach_a_test.go        Fix: explicit filtered file list
├── approach_b_test.go        Fix: CUE overlay (package stubs)
├── approach_c_test.go        Fix: rogue-file validator
├── helpers_test.go           Shared helpers (path resolution, file scanning, pattern matching)
└── testdata/
    ├── external_values.cue   Simulates a --values / -f file passed from outside
    └── module/
        ├── cue.mod/module.cue
        ├── module.cue        Schema (#config) and metadata
        ├── components.cue    #components definition (server + proxy)
        ├── values.cue        Default values: serverType="PAPER"
        ├── values_forge.cue  Rogue: serverType="FORGE"
        └── values_testing.cue Rogue: serverType="SPIGOT"
```

## Running

```bash
go test ./experiments/values-load-isolation/... -v
```

All 27 tests pass.
