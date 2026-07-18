# Delta: loader-api (cli-kernel-adoption)

The CLI loader API shrinks to what survives kernel adoption: `LoadInstanceFile` (instance-arg metadata extraction), `LoadValuesFile`, and provenance detection. Everything else is kernel surface (0006 D9).

## REMOVED Requirements

### Requirement: LoadInstancePackage loads instance CUE files

**Reason**: The CLI's copy is deleted; the kernel's `LoadInstancePackage` loads instance packages.
**Migration**: Kernel `LoadInstancePackage`.

### Requirement: DetectInstanceKind identifies instance type

**Reason**: Kind gating happens in the kernel's shape gate; the helper had no production callers.
**Migration**: Kernel load/shape-gate errors.

### Requirement: LoadModuleInstanceFromValue builds a ModuleInstance

**Reason**: Replaced by kernel `ProcessModuleInstance`/`SynthesizeInstance`.
**Migration**: Kernel entry points.

### Requirement: LoadProvider loads a provider from CUE

**Reason**: Providers are retired (0006 D39).
**Migration**: `platform-resolution` capability.

### Requirement: finalizeValue strips CUE constraints

**Reason**: Finalization is kernel `Finalize`.
**Migration**: Kernel `Finalize`.

### Requirement: `SynthesizeModuleInstanceFromPackage` builds an instance `cue.Value` from a module directory

**Reason**: Replaced by kernel `SynthesizeInstance` over a staged local source (0002 carryover).
**Migration**: `kernel-render` / `module-synthetic-instance`.

### Requirement: Synth wrapper pins the catalog at the user module's pinned version

**Reason**: There is no synth wrapper; version binding is the kernel's.
**Migration**: Kernel contract (enhancement 0001).

### Requirement: No filesystem writes inside the user's module

**Reason**: Kept as behavior (the kernel stages overlays in memory; `stageLocalModuleSource` only reads) but no longer a CLI-loader requirement — the CLI synth path is deleted.
**Migration**: `module-synthetic-instance` "No synthetic wrapper module" scenario covers the no-writes behavior.
