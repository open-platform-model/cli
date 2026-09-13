## MODIFIED Requirements

### Requirement: `opm mod vet` uses `debugValues` by default

The `opm mod vet` command SHALL use the module's `debugValues` field as the values source when no `-f` flag is provided. This validation SHALL happen in the module vet command itself rather than through `cmdutil.RenderRelease()`.

#### Scenario: `debugValues` used when no `-f` flag

- **WHEN** `opm mod vet` is run without `-f` flags
- **THEN** the module's `debugValues` field is extracted and used as the values source
- **AND** the vet output shows "debugValues" as the values source

#### Scenario: `-f` flag overrides `debugValues`

- **WHEN** `opm mod vet` is run with one or more `-f` flags
- **THEN** the explicit values files are used
- **AND** `debugValues` is ignored

#### Scenario: `debugValues` is `_` (unconstrained)

- **WHEN** `opm mod vet` is run without `-f` flags
- **AND** the module's `debugValues` field is `_` (open/unconstrained, not filled by the author)
- **AND** `#config` has a field without a default
- **THEN** `opm mod vet` SHALL refuse with the standard "values do not satisfy #config" validation block, naming the incomplete `#config` field at its schema position (an unconstrained `debugValues` carries no position of its own)
- **AND** the exit code SHALL be 2

#### Scenario: `debugValues` is `_` and every `#config` field has a default

- **WHEN** `opm mod vet` is run without `-f` flags
- **AND** the module's `debugValues` field is `_`
- **AND** every `#config` field carries a default
- **THEN** the kernel merges the open value with `#config` to a concrete result and `opm mod vet` SHALL report "Values satisfy #config" with `debugValues` as the source, the verdict `opm mod build` reaches for the same module
- **AND** the exit code SHALL be 0
