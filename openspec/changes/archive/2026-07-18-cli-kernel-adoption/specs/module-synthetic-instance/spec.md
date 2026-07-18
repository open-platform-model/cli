# Delta: module-synthetic-instance (cli-kernel-adoption)

Synthetic builds go through kernel `SynthesizeInstance` and emit `kind: "ModuleInstance"` (0002 carryover: retires the last `#ModuleRelease` application). Values selection, metadata defaults, output banner, and bundle rejection are unchanged and stay in the main spec.

## MODIFIED Requirements

### Requirement: Synthesize a `#ModuleInstance` from a module-package directory

The CLI SHALL synthesize a concrete instance from a module CUE package directory without requiring an `instance.cue` file, via kernel `SynthesizeInstance`. The synthesis SHALL load the module as a whole CUE package (matching `cue eval`/`cue vet` semantics) and pass it, with resolved values and synthetic metadata, to the kernel; the kernel unifies against the resolved `#ModuleInstance` schema so uuid, components, auto-secrets, and standard labels derive in CUE. The produced instance SHALL have `kind: "ModuleInstance"` — the synthesis SHALL NOT apply `#ModuleRelease` and SHALL NOT import `opmodel.dev/core/v1alpha1/modulerelease@v1`.

#### Scenario: Module directory loads as a whole CUE package

- **WHEN** synthesis is called with a module-package directory containing multiple `.cue` files in the same package
- **THEN** all files in the package SHALL be loaded as a single CUE instance via the kernel's module-package loading
- **AND** no individual file path SHALL be accepted as the synthesis input

#### Scenario: Emitted kind is ModuleInstance

- **WHEN** `opm module build` or `opm instance build <dir>` synthesizes and renders
- **THEN** the built instance SHALL carry `kind: "ModuleInstance"`
- **AND** no production code path SHALL reference `#ModuleRelease`

#### Scenario: No synthetic wrapper module

- **WHEN** synthesis runs
- **THEN** no temporary CUE module (synthetic `cue.mod/module.cue`) SHALL be created
- **AND** no files SHALL be created, modified, or left behind inside the module directory

#### Scenario: One CUE context

- **WHEN** the module value and synthesized instance are composed
- **THEN** both SHALL be produced from the kernel's single `*cue.Context`

## REMOVED Requirements

### Requirement: User module must declare a catalog dep

**Reason**: The catalog-dep pin existed to version-match the deleted synthetic wrapper; kernel synthesis binds the schema through the kernel's own resolution (0006 D9, 0002 carryover).
**Migration**: Modules keep whatever deps they genuinely import; no wrapper-driven pin requirement remains. Version-binding errors surface from the kernel with the module's declared language/schema versions.
