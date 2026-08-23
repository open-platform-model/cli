

## Requirements

### Requirement: Config init command creates configuration

The `opm config init` command SHALL create the default configuration files in `~/.opm/`.

The command creates:

- `~/.opm/config.cue` — scalar-only configuration file (registry, kubernetes, log) with no CUE imports
- `~/.opm/platform.cue` — data-only default platform file (name, type, registry subscriptions) with no CUE imports

The command SHALL NOT create `~/.opm/cue.mod/` and SHALL NOT run `cue mod tidy` or any CUE module operation. The seeded `platform.cue` SHALL subscribe to **both** first-party catalogs — `opmodel.dev/catalogs/opm@v2`, the abstraction catalog, and `opmodel.dev/catalogs/k8s@v1`, the raw Kubernetes passthrough catalog — each with an explicit pinned scalar `version` naming one published build of that catalog. Each pin is load-bearing: pins are bumped by hand as catalog releases ship, kept aligned with `hack/platform.cue` and the operator's sample Platform. A pin SHALL name a version that is actually published, because `opm config init` is normatively offline and cannot resolve one.

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
- **THEN** the file SHALL contain exactly two `registry` entries, keyed `opmodel.dev/catalogs/opm@v2` and `opmodel.dev/catalogs/k8s@v1`
- **AND** each entry SHALL carry an explicit concrete `version`
- **AND** the file SHALL contain no `filter` vocabulary
- **AND** the file SHALL contain no `opmodel.dev/catalogs/kubernetes` entry, which names a retired module path rather than the extracted catalog

#### Scenario: Seeded platform offers the raw escape hatch

- **WHEN** a module demands a contract from `opmodel.dev/catalogs/k8s@v1` and the platform is the seeded default
- **THEN** the subscription needed to match that contract SHALL already be present, without the user editing `platform.cue`

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

### Requirement: Config vet command validates configuration

The `opm config vet` command SHALL validate both `~/.opm` files using CUE evaluation against their embedded schemas.

Checks performed:

1. Config file exists at resolved path
2. Config file is syntactically valid CUE and satisfies the embedded config schema (no imports, no removed fields)
3. Platform file, when present, is syntactically valid CUE and satisfies the embedded platform projection schema

A missing `platform.cue` SHALL NOT fail vet (the file is optional until a render needs a local default); vet SHALL note its absence. Each check SHALL print a styled line to stdout using `FormatVetCheck` as it passes, giving the user real-time feedback. On failure, all previously-passing checks SHALL remain visible.

#### Scenario: Valid configuration passes validation

- **WHEN** `opm config vet` is run
- **WHEN** config.cue exists and is valid, and platform.cue exists and is valid
- **THEN** command succeeds
- **THEN** output SHALL contain a checkmark line for each passing check, including the platform file check

#### Scenario: Missing config file fails with actionable error

- **WHEN** `opm config vet` is run
- **WHEN** `~/.opm/config.cue` does not exist
- **THEN** command fails with not-found error
- **THEN** hint: "Run 'opm config init' to create default configuration"

#### Scenario: Missing platform file is noted, not fatal

- **WHEN** `opm config vet` is run
- **WHEN** config.cue is valid and `~/.opm/platform.cue` does not exist
- **THEN** command succeeds
- **THEN** output SHALL note that no local default platform is configured

#### Scenario: Invalid platform file fails

- **WHEN** `opm config vet` is run
- **WHEN** platform.cue contains an import declaration or violates the projection schema
- **THEN** the config checks SHALL print passing checkmark lines
- **THEN** command fails with a CUE error naming the platform file

#### Scenario: Stale providers block fails with migration hint

- **WHEN** `opm config vet` is run against a pre-D39 config.cue containing `providers:` or a `~/.opm/cue.mod/`
- **THEN** validation SHALL fail naming the removed field
- **AND** the hint SHALL say to re-run `opm config init` (or remove the field and `cue.mod/`)

### Requirement: Config vet respects path overrides

The `opm config vet` command SHALL respect `--config` flag and `OPM_CONFIG` environment variable for config path resolution.

#### Scenario: Validate custom config path via flag

- **WHEN** `opm config vet --config /custom/config.cue` is run
- **THEN** validation uses `/custom/config.cue` instead of default

#### Scenario: Validate custom config path via environment

- **WHEN** `OPM_CONFIG=/custom/config.cue` is set
- **WHEN** `opm config vet` is run
- **THEN** validation uses `/custom/config.cue` instead of default
