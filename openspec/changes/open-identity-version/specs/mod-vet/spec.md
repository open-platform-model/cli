## ADDED Requirements

### Requirement: An open identity `Version` passes vet's identity checks and skips the render

`opm module vet` SHALL NOT refuse an open identity `Version` at the identity check, because
`--version` fills it at publish. Vet SHALL run every check that does not need a concrete version,
then skip the render, print why and how to fill the version, and exit 0 without the
`Module valid (<n> resources)` summary. An absent or wrong `Version` SHALL still fail with exit 2.

#### Scenario: Open Version skips the render with a reason

- **WHEN** `opm module vet .` runs on a module whose identity declares `Version: string` and whose
  `metadata.version` derives from it
- **THEN** the identity, coordinate and `#config` checks pass
- **AND** vet prints that the render was skipped because the identity `Version` is open, naming
  `opm module version set` and `opm module publish --version`
- **AND** vet does not print `Module valid (<n> resources)`
- **AND** the exit code is 0

#### Scenario: Absent Version still fails the identity check

- **WHEN** `opm module vet .` runs on a module whose identity package declares no `Version`
- **THEN** vet fails the identity check, naming the missing field, with exit code 2

#### Scenario: A malformed or skewed Version still fails the identity check

- **WHEN** the identity declares `Version: "abc"`, or `Version: "1.0.0"` under a `@v0` module path
- **THEN** vet fails the identity check with CUE's diagnostic and exit code 2

## MODIFIED Requirements

### Requirement: `opm module vet` verifies the identity package, the derived coordinates and the `cue.mod` module line before values resolution
`opm module vet` SHALL, after loading the module and before values resolution, verify: the identity package conforms to core's `#IdentityPackage` (by unification, surfacing CUE's diagnostic for a conflict, and refusing an absent `ModulePath` or `Version` by name); `metadata.modulePath` and `metadata.version` derive from the identity package's values; and `cue.mod`'s `module:` line agrees with the declared module path.

#### Scenario: Replaced derivation caught at vet

- **WHEN** `metadata.version` states a literal that disagrees with the identity package's `Version`
- **THEN** vet fails naming both values and the derivation fix
