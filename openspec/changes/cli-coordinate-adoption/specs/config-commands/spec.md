# config-commands — Delta

## MODIFIED Requirements

### Requirement: Config init command creates configuration

The `opm config init` command SHALL create the default configuration files in `~/.opm/`.

The command creates:

- `~/.opm/config.cue` — scalar-only configuration file (registry, kubernetes, log) with no CUE imports
- `~/.opm/platform.cue` — data-only default platform file (name, type, registry subscriptions) with no CUE imports

The command SHALL NOT create `~/.opm/cue.mod/` and SHALL NOT run `cue mod tidy` or any CUE module operation. The seeded `platform.cue` SHALL subscribe to `opmodel.dev/catalogs/opm@v2` only — the sole first-party catalog since the consolidation — with an explicit pinned scalar `version` naming one published catalog build. The pin is load-bearing: it is bumped by hand as catalog releases ship, kept aligned with `hack/platform.cue` and the operator's sample Platform.

#### Scenario: Initialize configuration for first time

- **WHEN** `opm config init` is run
- **WHEN** no configuration exists at `~/.opm/config.cue`
- **THEN** `~/.opm/` directory is created with 0700 permissions
- **THEN** `~/.opm/config.cue` is written with 0600 permissions
- **THEN** `~/.opm/platform.cue` is written with 0600 permissions
- **THEN** no `~/.opm/cue.mod/` directory is created
- **THEN** success message lists created files
- **THEN** message suggests: "Validate with: opm config vet"

#### Scenario: Seeded platform subscriptions

- **WHEN** `opm config init` writes `platform.cue`
- **THEN** the file SHALL contain exactly one `registry` entry, keyed `opmodel.dev/catalogs/opm@v2`
- **AND** that entry SHALL carry an explicit concrete `version`
- **AND** the file SHALL contain no `filter` vocabulary and no `opmodel.dev/catalogs/kubernetes` entry

#### Scenario: Refuse to overwrite existing configuration

- **WHEN** `opm config init` is run
- **WHEN** `~/.opm/config.cue` already exists
- **THEN** command fails with validation error
- **THEN** error message: "configuration already exists"
- **THEN** hint: "Use --force to overwrite existing configuration."

#### Scenario: Force overwrite existing configuration

- **WHEN** `opm config init --force` is run
- **WHEN** `~/.opm/config.cue` already exists
- **THEN** existing files are overwritten
- **THEN** success message lists created files
