## Context

See proposal.md for motivation. What exists today:

- `internal/workflow/render/values.go` already turns `-f` files into kernel sources (`loadValuesSources`, via `Kernel.LoadSourceFromFile`) and `debugValues` into one source (`DebugValuesSource`: the field is formatted back to CUE and wrapped with `Kernel.LoadSourceFromBytes` under an origin the caller names). `internal/workflow/render/module.go` composes them in the unexported `resolveModuleValues(k, mod *module.Module, moduleDir, valuesFiles)`, which also validates `-f` sources before synthesis.
- `internal/cmd/module/vet.go` holds `modVal cue.Value` (the module package the publish gate loaded) rather than a `*module.Module`, resolves values into `[]cue.Value` through `pkg/loader.LoadValuesFile` or a `debugValues` lookup, checks each for concreteness, then calls `pkg/validate.Config`, which returns `*pkgerrors.ConfigError`.
- `cmdutil.PrintValidationError` has an `errors.As(*ConfigError)` branch and a fallback that groups any error carrying CUE positions (`GroupedErrorsFromError` + `hasPositions`).
- `Kernel.ValidateConfigDetailed(schema cue.Value, sources []kernel.Source) (cue.Value, error)` compiles the sources in the schema's own context, unifies them in order, walks disallowed fields, asserts concreteness, and returns the raw CUE error tree. `internal/cmd/module` already imports `internal/workflow/render` (`apply.go`, `build.go`), so vet can call an exported resolver without a new edge.

## Goals / Non-Goals

**Goals:**
- One values-resolution path and one validator for `vet` and `build`.
- Delete every CLI copy of kernel validation and values loading.

**Non-Goals:**
- Changing `FromModule`'s behaviour (it keeps validating `-f` sources before synthesis and leaving `debugValues` to the synthesized build).
- Restoring positions inside the module's own file for `debugValues` violations (see Risks).

## Research & Decisions

### Shared resolver: exported and value-based

**Context**: vet has a `cue.Value`, `FromModule` has a `*module.Module`; both need the same resolution.
**Explored**: `resolveModuleValues` and `DebugValuesSource` read only `mod.Package`.
**Options considered**:
1. vet wraps its value in a `module.Module{Package: modVal}` to call the existing function - a fake artifact, and `Module` is the library's type.
2. Both functions take the package value; `FromModule` passes `mod.Package`.
3. vet acquires the module a second time through `AcquireModuleFromDir` to get a `*module.Module` - a second load of the same package per invocation.
**Decision**: option 2. `render.ResolveModuleValues(k *kernel.Kernel, pkg cue.Value, moduleDir string, valuesFiles []string) ([]kernel.Source, error)`; `DebugValuesSource(k, pkg cue.Value, origin string)`.
**Rationale**: no fake artifact, no second load, one edit per caller.

### Validation happens at the callers, not inside the resolver

**Context**: today `resolveModuleValues` validates `-f` sources itself; vet must validate both `-f` and `debugValues` sources.
**Decision**: the resolver returns sources only. `FromModule` validates `-f` sources against `mod.ConfigSchema()` before synthesis, as it does today. vet validates whatever the resolver returned against `modVal.LookupPath(schema.Config)`.
**Rationale**: keeps `FromModule`'s "cheap failure before synthesis" property without validating `-f` files twice on the vet path.

### `ConfigError` is deleted, framing moves to the call site

**Context**: `ConfigError` carried a context, a name and the raw error, and `PrintValidationError` special-cased it.
**Explored**: after deleting `pkg/validate`, nothing constructs a `ConfigError`; the fallback branch in `PrintValidationError` groups any error whose CUE positions are valid, which the kernel's tree has (file-backed sources report the file's path, the `debugValues` source reports its origin with line numbers of the rendered text).
**Decision**: delete the type and the branch. vet wraps the kernel error as `fmt.Errorf("module %q: values do not satisfy #config: %w", modName, err)` for the exit error and prints through `cmdutil.PrintValidationError("values do not satisfy #config", err)`.
**Rationale**: the grouped output is produced by the branch that stays; the frame is one `fmt.Errorf`.

### The per-input concreteness pre-check goes

**Context**: vet checked each resolved value with `Validate(cue.Concrete(true))` before schema validation and printed "<source> not concrete".
**Decision**: rely on the kernel's concreteness assertion on the merged value.
**Rationale**: a pre-check per input refuses a stack whose merge is concrete (a base file with an open field completed by an override), which `build` accepts; the kernel checks the merge, so `vet` and `build` agree.

Data flow after the change:

```text
-f files ──► Kernel.LoadSourceFromFile ──┐
                                         ├──► []kernel.Source ──► Kernel.ValidateConfigDetailed(#config, sources)
debugValues ──► render.DebugValuesSource ─┘                              │
                                                              error tree ─┴─► cmdutil.PrintValidationError (grouped by position)
```

## Risks / Trade-offs

- [`debugValues` violations are attributed to `<module>/debugValues:line:col` of the rendered text, not to the module file's own line] → this is what `opm module build` already reports for the same input; the origin names the module directory so the author knows where to look. Carrying the field's original position is a library-side improvement, out of scope here.
- [The kernel's validator collects per-source schema errors and merge conflicts differently from the copy] → the copy was byte-identical to the kernel's validator at the time it was taken; the vet e2e cases behind the `validation-output` spec (grouped output, both error classes in one run) are the check, and task 2.1 runs them.
- [Go importers of `pkg/validate` or `pkg/errors.ConfigError` break] → both were CLI-internal in practice; the proposal marks the removal as breaking for `pkg/` importers.
