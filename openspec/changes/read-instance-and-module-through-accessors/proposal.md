## Why

The cli still reads three kernel artifact subtrees off the raw CUE value with hand-built paths, because the library had no accessor for them: an instance's merged values (`schema.Values`), the metadata of the module an instance embeds (`schema.Module`, decoded by a local `decodeModuleMetadata` whose comment says "the instance exposes no accessor for this subtree"), and a module's `debugValues` (`schema.DebugValues`). Library `v1.0.0-beta.6` (library PR 194), which the cli already pins, adds `Instance.Values()`, `Instance.ModuleMetadata()` and `Module.DebugValues()` for exactly these reads. Reading through them keeps the cli from knowing where the kernel keeps these fields, so a later schema move is a library change only.

The accessors were added on purpose for these cli sites, as additive library API ahead of the beta. The cli's scaffold reads already moved to the module's decoded metadata in the archived change `read-scaffold-metadata-from-module`. This change moves the reads library PR 194 added accessors for.

## What Changes

- `internal/workflow/render/render.go`: the render result's values come from `inst.Values()` (the JSON decode in `decodeUnifiedValues` stays), and its module metadata comes from `inst.ModuleMetadata()`. A nil result means a module with no metadata and maps to zero metadata, which is what the local decode returned for an absent subtree. The local `decodeModuleMetadata` is deleted.
- `publish.VetChecks` returns a `*module.Module` built over the root it loaded instead of a bare `cue.Value`, and records the authored module name on the plan. `opm module vet` reads `#config` through `Module.ConfigSchema()` and takes its log name from the plan, so `internal/cmd/module/vet.go` reads no module field off a CUE value.
- `render.ResolveModuleValues`, `render.DebugValuesSource` and `instinit.PickValues` take a `*module.Module` and read `debugValues` through `Module.DebugValues()`. Their callers move with them: `internal/workflow/render/module.go`, `internal/cmd/module/vet.go`, `internal/cmd/instance/init.go` and the `tests/integration/render-parity` program. `initValues` is still read from `mod.Package`, because the library has no accessor for it, and this change moves only the reads library PR 194 added accessors for.
- The `pkg-types` requirement that already rules where the cli reads module identity gains the three accessor reads, with scenarios.

No user-visible change: every message, log line, output field and exit code stays the same. Tests pin the render result's module metadata and values, the vet log name for a module whose version is still open, the nil-metadata mapping, and the one input where the result differs (embedded metadata that does not decode, see design).

Not in this change:
- The duplicate-identity check. It already uses `object.Duplicates` (cli PR 325).
- An `initValues` accessor. The library does not have one, and this change moves only the reads library PR 194 added accessors for.
- Any library change, and any change to the library pin.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pkg-types`: the requirement "Core types exported in pkg/" also says the cli reads an instance's module metadata and values, and a module's `debugValues` and `#config`, through the library's accessors.

## Impact

- Code: `internal/workflow/render/render.go`, `values.go`, `module.go`; `internal/publish/vet.go` (and the `Plan` type); `internal/cmd/module/vet.go`; `internal/instinit/values.go`; `internal/cmd/instance/init.go`; `tests/integration/render-parity/main.go`; the tests next to each.
- API: `ResolveModuleValues`, `DebugValuesSource`, `PickValues` and `VetChecks` change signature. All are in `internal/`, except that the `tests/integration/render-parity` program imports `DebugValuesSource` and moves in the same section. Nothing in `pkg/` changes.
- Dependencies: none. Library pin stays `v1.0.0-beta.6`.
- Release class: refactor.
