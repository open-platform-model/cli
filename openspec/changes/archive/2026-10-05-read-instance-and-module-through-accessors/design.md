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
- `initValues` (`internal/instinit/values.go:32`). The library has no accessor for it, and this change moves only the reads library PR 194 added accessors for.
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

One input differs. The old decode was best-effort: when `metadata` failed to decode, it kept the fields that did decode. The library decode is all or nothing and returns nil, so the result is now zero metadata. Nil maps to zero by choice: a nil result is carried as a module with no metadata, the same as an absent subtree.

That input is reachable. A module the kernel acquires on its own passes the shape gate, which requires concrete `metadata.name`, `modulePath` and `version`. An instance file does not: the instance shape gate requires only the instance's `metadata.name` and `metadata.namespace`, and checks only the `kind` of its embedded `#module`, not the concreteness of the module's identity. So an instance whose embedded module still has an open `metadata.version` (for example a local replacement of a module in authoring) loads. It renders as long as no matched transformer reads the module version (core's `#transformer` exposes it as `#moduleMetadata.version`). In that gap the old decode kept the name and module path, and the new path gives zero metadata, which `spec.module` and the instance log line then show. The case is narrow, it involves a module that is not publishable yet, and the change is pinned by a `TestModuleMetadataOf` case so it is visible. The old debug log line "could not decode module metadata" goes with the helper. It printed only under debug logging.

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

The returned module is not the one `gateKernelLoad` acquires through the kernel. That gate is skipped when an identity field is open, so the wrapper and its fallback cover a schema that admits an open `Version`. With core `v2.0.0-beta.4` that case is not reachable: `#IdentityPackage` refuses an open `Version`, and the kernel-load gate refuses an open `metadata.version`, so every run that reaches values validation has decodable metadata. The fallback is defensive, and keeps vet's behaviour the same if the schema loosens. Whether vet should admit an open `Version` at all is tracked in cli issue 326. The wrapped root is also built in the runtime the identity schema lives in, which the comment above `cueCtx` in `vet.go` keeps on purpose. Using the kernel's module would make it depend on which runtime the kernel acquires in.

The module is built right after `loadPackage` succeeds and is returned on every path that returns `root`, including the identity-package and `cue.mod` read refusals. Only the paths that returned `cue.Value{}` return a nil module: the input precondition and directory errors, the root load error, and the no-`cue.mod` refusal. Vet only reaches the module when the plan has no refusals and no error.

### Vet: the log name comes from the plan

Vet names the module in its log prefix with the authored `metadata.name`, else the directory's base name. If an open `Version` reached this point, the library's decode would fail as a whole, so `mod.Metadata` would be nil while `metadata.name` is concrete, and a name taken from `mod.Metadata` would show the directory name instead. Current core refuses an open `Version` earlier (see above), so `Plan.ModuleName` is defensive: it keeps the log prefix independent of whether the metadata decodes.

So `VetChecks` records `metadata.name` on the plan as `ModuleName` (empty when absent, non-concrete or not a string), read in `VetChecks` right after the root loads. The vet output e2e test pins the prefix (`m:demo:` for a module vetted from a directory named `module`). Vet uses `plan.ModuleName` and falls back to `filepath.Base(modulePath)`, as it does today. The publish package reads the authored tree before the kernel touches it. That is its job, and the `pkg-types` rule about acquired modules does not cover it. `internal/cmd/module/vet.go` then reads no module field off a CUE value. It still looks up `#IdentityPackage` in the identity schema it loads, which is not a module read.

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
- [`Plan.ModuleName` is a second place that reads `metadata.name`] → It is the only `metadata.name` read on the vet path; vet's own read goes away. Publish's authored-tree gates (`gatePackageName`, the members check) already read it the same way.
- [Signature changes collide with the next cli change that touches `init.go`, `render/module.go` and `render.go`] → That change is planned to land after this one and builds on it. Each section here is small.

## Migration Plan

None. Internal signatures only. The `render-parity` program is built and run in section 3.

## Open Questions

None.
