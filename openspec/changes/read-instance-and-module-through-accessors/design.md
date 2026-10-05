## Context

Library `v1.0.0-beta.6` (library PR 194) adds three nil-safe accessors:

- `Instance.Values() cue.Value`: `Package.LookupPath(schema.Values)`; the zero value when absent.
- `Instance.ModuleMetadata() *ModuleMetadata`: decodes `metadata` of the embedded `#module` (`schema.Module`) with the library's own decoder. It returns nil for a nil receiver, an instance with no `#module`, or metadata that does not decode. The decode is all or nothing.
- `Module.DebugValues() cue.Value`: `Package.LookupPath(schema.DebugValues)`; the zero value when the module declares none.

`Module.ConfigSchema()` already existed. The cli pins `v1.0.0-beta.6` at `origin/main` (cli PR 322).

The raw reads at `origin/main` (`e5b8c502`):

| Site | Today | After |
| --- | --- | --- |
| `internal/workflow/render/render.go:283` | `decodeUnifiedValues(inst.Package.LookupPath(schema.Values))` | `decodeUnifiedValues(inst.Values())` |
| `internal/workflow/render/render.go:306` | `decodeModuleMetadata(inst.Package.LookupPath(schema.Module))`, helper at `:377-389` | `inst.ModuleMetadata()`, nil mapped to zero; helper deleted |
| `internal/workflow/render/values.go:43` | `pkg.LookupPath(schema.DebugValues)` in `DebugValuesSource` | `mod.DebugValues()` |
| `internal/instinit/values.go:39` | `pkg.LookupPath(schema.DebugValues)` in `PickValues` | `mod.DebugValues()` |
| `internal/cmd/module/vet.go:131` | `modVal.LookupPath(cue.ParsePath("metadata.name"))` | `plan.ModuleName` |
| `internal/cmd/module/vet.go:217` | `modVal.LookupPath(schema.Config)` | `mod.ConfigSchema()` |

Callers that move with the signatures: `internal/workflow/render/module.go:118`, `internal/cmd/module/vet.go:147`, `internal/cmd/instance/init.go:289`, `tests/integration/render-parity/main.go:146`.

## Goals / Non-Goals

**Goals:**
- Every site in the table reads through a library accessor, or (vet's log name) through the plan the publish checks already build from the authored tree.
- Output, log lines and exit codes stay the same for every input.

**Non-Goals:**
- `initValues` (`internal/instinit/values.go:32`). The library has no accessor, and the decision covers only `debugValues`.
- The duplicate-identity check (already `object.Duplicates`, cli PR 325).
- Changing which CUE runtime vet validates in.

## Decisions

### Render: nil module metadata is zero metadata

```go
result.Module = moduleMetadataOf(inst)

// moduleMetadataOf is the instance's embedded module metadata, or zero
// metadata when the instance carries none.
func moduleMetadataOf(inst *module.Instance) module.ModuleMetadata {
	if m := inst.ModuleMetadata(); m != nil {
		return *m
	}
	return module.ModuleMetadata{}
}
```

The old decode returned zero metadata for an absent `#module` or `metadata`. The library returns nil there, so the helper maps nil to zero. That keeps `result.Module` a value, as `CanonicalModuleRef`, the instance log line and `spec.module` expect.

One input differs. The old decode was best-effort: when `metadata` failed to decode, it kept the fields that did decode. The library returns nil, so the result is now zero metadata. This cannot happen on a render path. The instance has already rendered, and the kernel refuses to load a module whose identity fields are not concrete (`ErrMissingRequiredField` in the library's shape gate). So the embedded module's metadata always decodes. The old debug log line "could not decode module metadata" goes with the helper. It printed only under debug logging.

`decodeUnifiedValues` stays. It is the cli's own conversion to the JSON map for `spec.values`, and it already handles the zero value through `Exists()`.

### Vet: `VetChecks` returns a module over the root it loaded

`VetChecks` changes from `(*Plan, cue.Value, error)` to `(*Plan, *module.Module, error)`. The module wraps the same root value vet reads today:

```go
mod, err := module.NewModuleFromValue(root)
if err != nil {
	// An open identity field (an authoring state vet allows) leaves the
	// metadata undecodable; the package is still what vet validates.
	mod = &module.Module{Package: root}
}
```

The returned module is not the one `gateKernelLoad` acquires through the kernel. That module is not always there: the gate is skipped when an identity field is open, and vet allows an open `Version`. Vet validates `#config` and `debugValues` in that case too. The wrapped root is also built in the runtime the identity schema lives in, which the comment above `cueCtx` in `vet.go` keeps on purpose. Using the kernel's module would make it depend on which runtime the kernel acquires in.

Every path that returned `cue.Value{}` now returns a nil module. Vet only reaches the module when the plan has no refusals and no error. The paths that return a zero root (no `cue.mod`, identity package refusals) always carry a refusal.

### Vet: the log name comes from the plan

Vet names the module in its log prefix with the authored `metadata.name`, else the directory's base name. With an open `Version`, the library's decode fails as a whole, so `mod.Metadata` is nil while `metadata.name` is concrete. If vet took its name from `mod.Metadata`, an open-version module's log lines would show the directory name instead. That would be a visible change.

So `VetChecks` records `metadata.name` on the plan as `ModuleName` (empty when absent, non-concrete or not a string), read from the root next to the other authored-tree reads in `gateDerivation`. Vet uses `plan.ModuleName` and falls back to `filepath.Base(modulePath)`, as it does today. The publish package reads the authored tree before the kernel touches it. That is its job, and the `pkg-types` rule about acquired modules does not cover it. `internal/cmd/module/vet.go` no longer reads from any CUE value.

### `debugValues` through the module

```go
func ResolveModuleValues(k *kernel.Kernel, mod *module.Module, moduleDir string, valuesFiles []string) ([]kernel.Source, error)
func DebugValuesSource(k *kernel.Kernel, mod *module.Module, origin string) (kernel.Source, error)
func PickValues(mod *module.Module) ([]byte, ValuesSource, error)
```

`mod.DebugValues()` returns the same value the raw lookup did, including the zero value when absent. The `!Exists()` checks, error texts and the `FromEmpty` fallback stay as they are. A nil module behaves like a module without `debugValues`. `PickValues` keeps reading `initValues` from `mod.Package`, guarded for a nil module.

The unit tests that compile a bare package (`packageWithConfig`, the `PickValues` table) wrap it as `&module.Module{Package: pkg}`. That is the same construction vet's fallback uses, and it needs no kernel.

## Risks / Trade-offs

- [The vet module is built by the cli, not acquired by the kernel] → It is the same value vet validated before; only the type around it changes. The design and the `VetChecks` godoc say so, so no reader takes it for a kernel-acquired module.
- [`Plan.ModuleName` is a second place that reads `metadata.name`] → It is the only reader. Vet's own read goes away, so the count stays at one.
- [Signature changes collide with the next cli change that touches `init.go`, `render/module.go` and `render.go`] → That change is planned to land after this one and builds on it. Each section here is small.

## Migration Plan

None. Internal signatures only. The `render-parity` program is built and run in section 3.

## Open Questions

None.
