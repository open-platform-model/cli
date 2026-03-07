# Validation Gates

## Overview

Two validation gates protect the render pipeline from invalid configuration before any finalization or rendering work begins. Both gates perform the same fundamental operation — unify consumer-supplied values with a `#config` schema and verify concreteness — but they operate at different scopes and run at different points in the loading flow.

```text
┌─────────────────────────────────────────────────────────────────────────┐
│  BUNDLE FLOW                                                             │
│                                                                          │
│  consumer values                                                         │
│       │                                                                  │
│       ▼                                                                  │
│  ┌─────────────┐   fail → clear error attributed to bundle release      │
│  │ Bundle Gate │                                                         │
│  └──────┬──────┘                                                         │
│         │ pass                                                           │
│         ▼                                                                │
│  comprehension output: releases map                                      │
│       │                                                                  │
│       ▼  (per release)                                                   │
│  ┌─────────────┐   fail → clear error attributed to instance name       │
│  │ Module Gate │                                                         │
│  └──────┬──────┘                                                         │
│         │ pass                                                           │
│         ▼                                                                │
│  concreteness check → finalize components → render                      │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  STANDALONE MODULE FLOW                                                  │
│                                                                          │
│  consumer values                                                         │
│       │                                                                  │
│       ▼                                                                  │
│  ┌─────────────┐   fail → clear error attributed to release name        │
│  │ Module Gate │                                                         │
│  └──────┬──────┘                                                         │
│         │ pass                                                           │
│         ▼                                                                │
│  concreteness check → finalize components → render                      │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Why Gates Exist

CUE already performs unification and validation internally. Any invalid values will eventually produce a CUE error during evaluation. The gates do not replace that — they intercept the same errors earlier, at the explicit values/schema boundary, before those errors propagate through the comprehension or finalization machinery and surface as deeply nested, hard-to-attribute CUE messages.

Without gates, a wrong value type in `values.cue` can produce an error like:

```
releases.server.let unifiedBundle.#instances.server.module.#config.rcon.password:
  incomplete value ...
```

With gates, the same mistake produces:

```
bundle "my-game-stack": values do not satisfy #config:
  - #bundle.#config.maxPlayers: conflicting values "fifty" and int & >=0
      bundle.cue:35:37
      values.cue:8:14
```

The gate output is attributed to the release or bundle name, points to the field that is wrong, and cites the file and line number in both the schema definition and the user's values file.

Gates also serve as the foundation for a future field-level diagnostic system. The `ConfigError` type preserves the raw CUE error tree, which can be walked to extract per-field paths, constraint descriptions, and source positions for structured output (e.g. JSON diagnostics, inline editor annotations).

---

## Shared Implementation

Both gates are implemented via a single shared function in `pkg/loader/validate.go`.

### `validateConfig`

```go
func validateConfig(schema cue.Value, values cue.Value, context string, name string) *ConfigError
```

**Parameters:**

- `schema` — the `#config` CUE value (from `#bundle.#config` or `#module.#config`)
- `values` — the consumer-supplied or instance-wired values
- `context` — `"bundle"` or `"module"` (used in error output)
- `name` — release or bundle name (used in error output)

**Two checks, in order:**

1. **Unify** — `schema.Unify(values)` then `.Err()`. Catches type mismatches and constraint violations: a string where an int is expected, a value outside an enum, a `matchN` violation, etc. This is the same unification that CUE performs internally in `#BundleRelease` and `#ModuleRelease` — done here explicitly so we control where the error surfaces.

2. **Concrete** — `.Validate(cue.Concrete(true))` on the unified value. Catches missing required fields: fields in `#config` that have no default and were not provided in `values`. A field with a default (`string | *"changeme"`) passes even if not supplied; a field declared as bare `string` does not.

**Returns** `*ConfigError` on failure, `nil` on success.

**If `schema` or `values` does not exist** (field missing from the CUE value), the function returns `nil` and lets downstream checks handle the structural problem. This is intentional — a missing `#config` or `values` field is a bug in the release definition, not a user config error.

### `ConfigError`

```go
type ConfigError struct {
    Context  string // "bundle" or "module"
    Name     string // bundle/release/instance name
    RawError error  // original CUE error tree
}
```

`Error()` formats a human-readable summary: one line per CUE error position, with file and line number when available.

`RawError` is preserved for future use. The `cue/errors` package provides `errors.Errors()` (unwrap to individual errors) and `errors.Details()` (position + message), which a future diagnostic layer can use to extract structured per-field information without re-parsing the string output.

---

## Bundle Gate

**What it validates:** consumer-supplied `values` against `#bundle.#config`.

**When it runs:** in `LoadBundleReleaseFromValue`, after metadata extraction and before the `releases` comprehension output is read.

**CUE paths accessed:**

- Schema: `pkg.LookupPath("#bundle.#config")`
- Values: `pkg.LookupPath("values")`

**What it catches:**

- Consumer provides a value of the wrong type (`maxPlayers: "fifty"` instead of `uint`)
- Consumer omits a required field (a field in `#bundle.#config` with no default)
- Consumer provides a value outside a constraint (`maxPlayers: 99999` where `<=10000`)

**What it does not catch:**

- Incorrect wiring inside the bundle definition itself (e.g. the bundle author wires the wrong field into an instance's `values`) — that is the Module Gate's responsibility
- Structural errors in the bundle definition (e.g. a malformed `#instances` entry) — caught by the `releasesVal.Err()` check that follows

**Error attribution:** `bundle "<bundleReleaseName>": values do not satisfy #config: ...`

**Call site:** `pkg/loader/bundle_release.go` → `LoadBundleReleaseFromValue`

---

## Module Gate

**What it validates:** instance values (from `#BundleInstance.values`) or consumer values (from `#ModuleRelease.values`) against `#module.#config`.

**When it runs:**

- **Bundle path:** in `extractBundleReleases`, per release, after the release entry error check and before the concreteness check
- **Standalone path:** in `LoadModuleReleaseFromValue`, after the top-level `pkg.Err()` check and before the concreteness check

**CUE paths accessed (bundle path, per release entry):**

- Schema: `schemaEntry.LookupPath("#module.#config")`
- Values: `schemaEntry.LookupPath("values")`

**CUE paths accessed (standalone path):**

- Schema: `releaseVal.LookupPath("#module.#config")`
- Values: `releaseVal.LookupPath("values")`

**What it catches:**

- Bundle author forgot to wire a required module field into the instance `values` (e.g. `rcon.password` not wired for minecraft)
- Bundle author wired a value of the wrong type into an instance field
- Standalone consumer omits a required module config field
- Standalone consumer provides a value that violates a module constraint

**What it does not catch (bundle path):** consumer-level bundle config errors — those are already caught by the Bundle Gate before the Module Gate runs.

**Error attribution:**

- Bundle path: `module "<instanceName>": values do not satisfy #config: ...` (wrapped with `release "<instanceName>":` by the caller)
- Standalone path: `module "<releaseName>": values do not satisfy #config: ...`

**Call sites:**

- `pkg/loader/bundle_release.go` → `extractBundleReleases` (per release)
- `pkg/loader/module_release.go` → `LoadModuleReleaseFromValue`

---

## Concreteness Check (Post-Gate)

After the relevant gate(s) pass, a `Validate(cue.Concrete(true))` check runs on the whole CUE value — the full release entry in the bundle path, or the full release value in the standalone path.

This check is intentionally kept separate from the gates. It covers fields that are not part of the consumer-facing `#config` — auto-generated fields like `metadata.uuid` (interpolated from name and namespace), computed labels, and other derived values. If this check fails after the gates have passed, it indicates a gap in the bundle wiring or a bug in the module definition — not a user config error. The error message reflects this:

```
release "server": not fully concrete after gate validation (bundle wiring may be incomplete): ...
```

**Scope:** always the whole release value — same in both the bundle path and the standalone path. The previous behaviour of validating only `components` in the bundle path was changed after confirming that `Validate(cue.Concrete(true))` on the full `schemaEntry` does not trip on `#module.#config` validators, because `#BundleInstance.values` passes concrete data (no open schema constraints) into the `#ModuleRelease`.

---

## Execution Order Summary

### Bundle path (`LoadBundleReleaseFromValue`)

```
1. pkg.Err()                          — top-level CUE evaluation error
2. extractBundleReleaseMetadata       — decode bundle release name/uuid
3. extractBundleInfo                  — decode bundle metadata
4. Bundle Gate                        — validateConfig(#bundle.#config, values)
5. releasesVal.Err()                  — comprehension-level CUE errors
6. per release:
   a. schemaEntry.Err()               — release entry CUE errors
   b. Module Gate                     — validateConfig(#module.#config, values)
   c. schemaEntry.Validate(Concrete)  — whole-release concreteness
   d. finalizeValue(components)       — strip schema constraints for rendering
   e. decodeModuleReleaseEntry        — decode Go struct
```

### Standalone path (`LoadModuleReleaseFromValue`)

```
1. releaseVal.Err()                   — top-level CUE evaluation error
2. Module Gate                        — validateConfig(#module.#config, values)
3. releaseVal.Validate(Concrete)      — whole-release concreteness
4. extractReleaseMetadata             — decode release name/namespace
5. extractModuleInfo                  — decode module metadata
6. finalizeValue(release)             — strip schema constraints for rendering
```

---

## Future: Field-Level Diagnostics

The current gate output is a formatted string summary. The next layer is structured per-field errors: instead of one combined message, a list of `FieldError` entries each carrying the field path, the constraint that was violated, and the source positions.

`ConfigError.RawError` is the entry point for this. The `cue/errors` package exposes:

```go
errors.Errors(rawErr)    // []cue.Error — individual errors in the combined list
ce.Position()            // token.Pos  — source position of the constraint
ce.InputPositions()      // []token.Pos — source positions of the inputs
errors.Details(ce, nil)  // string     — human-readable message for one error
```

A future `parseConfigError(ce *ConfigError) []FieldError` function would walk this list, extracting the field path from the error message (or from the CUE value path when available) and the positions for structured output. The gate infrastructure does not need to change — only the `ConfigError` consumer needs updating.
