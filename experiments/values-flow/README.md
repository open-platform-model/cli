# values-flow experiment

Proves the complete end-to-end data flow for values processing that
`loader.Load()` and `builder.Build()` will implement.

## What is being proven

```
Approach A load (values*.cue excluded from load.Instances)
  → moduleVal   (mod.Raw — no concrete values baked in)
  → defaultVals (values.cue loaded separately via ctx.CompileBytes)
                 OR inline values directly from moduleVal

Approach C validation (runs before any load)
  → error if any values*.cue other than values.cue is present

selectValues(moduleVal, defaultVals, userFile)
  → Layer 2 (highest): user --values file  — completely replaces defaults
  → Layer 1 (default): separate values.cue — used when no user file
  → Fallback:          inline values in module.cue

validateAgainstConfig(#config, selectedValues)
  → validates only the values actually in use
  → module defaults ignored by user file are never validated

buildRelease(schema, moduleVal, name, ns, selectedValues)
  → FillPath chain: #module → metadata.name → metadata.namespace → values
  → concrete #ModuleRelease
  → release.values is the final, validated, concrete values
```

## Design decisions confirmed

**No `values: #config` in module.cue.** Module authors write either:
- `values.cue` with concrete defaults (pattern A)
- `values: { ... }` inline in `module.cue` (pattern B)

The catalog's `#ModuleRelease` enforces `values: close(#module.#config)` at
the release level. This is a framework concern, not an author concern.

**Only `values.cue` is allowed inside the module directory.** Any other
`values*.cue` file (e.g. `values_forge.cue`) is rejected before load.
Environment-specific overrides belong outside the module directory and are
referenced via `--values`.

**User values completely replace defaults — no partial merge.** When `--values`
is provided, the module author's defaults are entirely ignored. Validation runs
only on the values that are actually used.

## Test files

| File | What it tests |
|---|---|
| `helpers_test.go` | Shared helpers: `loadCatalog`, `loadModuleApproachA`, `selectValues`, `validateAgainstConfig`, `buildRelease`, etc. |
| `approach_c_test.go` | Rogue file validation — detects `values_forge.cue` before any CUE load |
| `approach_a_test.go` | File filtering mechanics — explicit file list, single instance, no concrete values in moduleVal |
| `select_values_test.go` | Layer 1 / Layer 2 selection — defaults used when no user file; user file replaces defaults entirely |
| `schema_validation_test.go` | `#config` validation — valid passes, invalid fails, only selected values validated |
| `release_build_test.go` | `buildRelease` — release is concrete, `release.values` correct, FillPath is immutable |
| `inline_module_test.go` | Pattern B — inline values, multi-file package, fallback selection, user override |

## Fixtures

```
testdata/
  values_module/          Pattern A: separate values.cue
    cue.mod/
    module.cue            metadata + #config + #components — NO values field
    values.cue            { values: { image: "nginx:latest", replicas: 1 } }

  inline_module/          Pattern B: inline values, multi-file
    cue.mod/
    module.cue            metadata + #config + values: { image: "nginx:stable", replicas: 2 }
    components.cue        #components.web + #components.sidecar

  rogue_module/           Pattern C: rogue file → error
    cue.mod/
    module.cue
    values.cue            legitimate default
    values_forge.cue      rogue — triggers Approach C error

  user_values.cue         Layer 2 override: { values: { image: "custom:2.0", replicas: 5 } }
  invalid_values.cue      Violates #config: { values: { image: "nginx:1.0", replicas: 0 } }
```

## Note on release build tests

`testdata/values_module` and `testdata/inline_module` use a free-form `spec`
in their `#components` that does not satisfy the catalog's strict closed
`#Component` schema (which requires `#resources` and `#traits`). The fixture
modules are sufficient for all Approach A / C / selection / validation tests.

For `buildRelease` tests — which inject the module into the real `#ModuleRelease`
schema — `_testModule` from the catalog is used instead. It is a known-valid
`#Module` with proper `#resources` and `#traits`:

```
_testModule #config: { replicaCount: int & >=1, image: string }
_testModule values:  { replicaCount: 2, image: "nginx:12" }
```

## Running

```bash
go test ./experiments/values-flow/... -v
```

No registry required. The catalog is loaded from local source (`catalog/v0/core`).

## Relationship to other experiments

| Experiment | Relationship |
|---|---|
| `values-load-isolation` | Proves Approach A and C in isolation. `values-flow` builds on those results and adds the selection + validation + build phases. |
| `module-release-cue-eval` | Proves the `buildRelease` FillPath sequence. `values-flow` reuses that pattern and adds the values selection layer above it. |
