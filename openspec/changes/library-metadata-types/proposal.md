## Why

`pkg/module` declares `ModuleMetadata` and `InstanceMetadata` with the same fields and JSON tags as the library's `opm/schema.ModuleMetadata` and `opm/schema.InstanceMetadata`, which every acquired artifact already carries decoded (`Module.Metadata`, `Instance.Metadata`). The render workflow copies the library's decoded instance metadata field by field into the CLI type and decodes the embedded module's metadata into the other. Two declarations of one shape drift (the CLI copy already lacks the instance `fqn` the library added). Slice 07 of the kernel diet.

## What Changes

- The render `Result` carries the library's metadata types: `Result.Instance` is `module.InstanceMetadata`, `Result.Module` is `module.ModuleMetadata` (the library's `opm/module` aliases of the `opm/schema` types). The instance metadata is assigned from the acquired instance whole, so `Annotations` and `FQN` arrive with it; the `--namespace`/env override still applies after.
- The embedded module's metadata is still decoded from the instance package (the library keeps that decoder private and exposes no accessor on the instance), now into the library type.
- `CanonicalModuleRef` becomes a function in `internal/workflow/render`, `CanonicalModuleRef(m module.ModuleMetadata) (path, version string)`, keeping the `v`-prefix rule and its doc: the pair is written verbatim to `ModuleInstance.spec.module`, which the operator reads without normalising. Its two callers in `internal/workflow/apply` are updated.
- **BREAKING (Go importers of `pkg/`):** `pkg/module` is deleted.

**Not in this change:** a library accessor for the embedded module's metadata (a one-line library follow-up that would delete the CLI-side decode); any JSON output of `Result` (there is none).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pkg-types`: "Core types exported in pkg/" no longer lists `pkg/module`; the module and instance metadata types are the library's. The package list is restated to the packages that exist.
- `core-module`: "Module type location" is removed.

## Impact

**SemVer:** PATCH for the command surface: no flag, syntax or output change; the CR the apply workflow writes carries the same `spec.module` pair. For Go importers of `pkg/module` this is a breaking removal.

**Packages:** `internal/workflow/render` (`Result` field types, `CanonicalModuleRef`), `internal/workflow/apply` (two call sites), `pkg/module` (deleted). Commands affected: none visibly.

**Complexity justification (Principle VII):** net negative, one package deleted; one function moves.
