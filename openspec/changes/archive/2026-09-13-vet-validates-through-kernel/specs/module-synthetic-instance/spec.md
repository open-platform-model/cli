## MODIFIED Requirements

### Requirement: Values selection mirrors `opm module vet`

The synthesis SHALL select values to fill the module's `#config` from the same sources `opm module vet` uses, through the same resolver (`render.ResolveModuleValues`): `-f`/`--values` files as file-backed kernel values sources in declaration order, falling back to the module's `debugValues` rendered as one source when no `-f` flags are given.

#### Scenario: Values from `-f` flags

- **WHEN** the user invokes synthesis with one or more `-f` files
- **THEN** each file SHALL become a kernel values source attributed to that file, in declaration order
- **AND** the sources SHALL be validated against the module's `#config` through the kernel before synthesis, so a violation or a conflict is attributed to the file it came from
- **AND** the sources SHALL be passed to the kernel's `SynthesizeInstance` as the values to fill into `#config`

#### Scenario: `debugValues` fallback

- **WHEN** the user invokes synthesis with no `-f` flag and the module defines a `debugValues` field
- **THEN** `debugValues` SHALL be rendered as the single values source, attributed to the module's `debugValues`

#### Scenario: Neither values flag nor `debugValues`

- **WHEN** the user invokes synthesis with no `-f` flag and the module does not define `debugValues`
- **THEN** the CLI SHALL return an actionable error stating that the module must define `debugValues` or values must be supplied with `-f`
