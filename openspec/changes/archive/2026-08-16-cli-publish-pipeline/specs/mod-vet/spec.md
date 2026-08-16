# mod-vet — Delta

## ADDED Requirements

### Requirement: Identity and coordinate checks before values validation

`opm module vet` SHALL, after loading the module and before values resolution, verify: the identity package conforms to core's `#IdentityPackage` (by unification, surfacing CUE's diagnostic); `metadata.modulePath` and `metadata.version` derive from the identity package's values; and `cue.mod`'s `module:` line agrees with the declared module path. Failures SHALL report through the standard validation-error rendering with exit code 2, in the same aligned two-value form the publish refusals use, and SHALL be reported even for a module that declares no `debugValues`.

#### Scenario: Coordinate drift caught at vet

- **WHEN** `cue.mod` and the identity package state different module paths
- **THEN** vet fails with both values and both files named, before any values validation

#### Scenario: Replaced derivation caught at vet

- **WHEN** `metadata.version` states a literal that disagrees with the identity package's `Version`
- **THEN** vet fails naming both values and the derivation fix

## MODIFIED Requirements

### Requirement: Registry-aware loading

Vet's module load SHALL resolve dependencies using the resolved registry (flag > env > config precedence), not the ambient process environment alone. Failing to reach the registry for the core-schema fetch SHALL exit 3 (connectivity), distinct from check failures (2).

#### Scenario: Registry flag respected

- **WHEN** vet runs with `--registry` pointing at a reachable mapping
- **THEN** the module's dependencies resolve through that mapping
