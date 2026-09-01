## MODIFIED Requirements

### Requirement: Config init command creates configuration

The `opm config init` command SHALL create the default configuration in `~/.opm/`.

The command creates:

- `~/.opm/config.cue` — scalar-only configuration file (registry, kubernetes, log) with no CUE imports (unchanged)
- `~/.opm/platform/` — the local default platform as a real CUE module: `cue.mod/module.cue` (module path `opmodel.dev/platforms/local@v0`, pinned dependencies on `opmodel.dev/core` and both first-party catalogs) and `platform.cue` (embeds `core.#Platform`, imports both catalogs, one `#registry` entry per catalog carrying `#catalog: <import>`)

The command SHALL NOT write a data-only `~/.opm/platform.cue`; when a legacy one exists it SHALL be removed after the module is written, with a printed note. The command SHALL remain normatively offline: it writes the pinned `cue.mod` without resolving anything, and it SHALL NOT run `cue mod tidy` or any registry operation. The seeded module SHALL subscribe to **both** first-party catalogs — `opmodel.dev/catalogs/opm@v4` (abstraction) and `opmodel.dev/catalogs/k8s@v1` (raw passthrough) — with each entry's build named exactly once, as the pinned dependency in `cue.mod/module.cue`; no `version` scalar appears in `platform.cue` (the entry's `version` is derived from the imported catalog, 0019 D5). Pins are bumped by hand as releases ship, mirrored with `hack/platform/` and `hack/kind-platform.yaml`, and SHALL name published builds.

#### Scenario: Initialize configuration for first time

- **WHEN** `opm config init` is run
- **WHEN** no configuration exists at `~/.opm/config.cue`
- **THEN** `~/.opm/` directory is created with 0700 permissions
- **THEN** `~/.opm/config.cue` is written with 0600 permissions
- **THEN** `~/.opm/platform/cue.mod/module.cue` and `~/.opm/platform/platform.cue` are written with 0600 permissions
- **THEN** no data-only `~/.opm/platform.cue` is written
- **THEN** success message lists created files
- **THEN** message suggests: "Validate with: opm config vet"

#### Scenario: Seeded platform subscriptions

- **WHEN** `opm config init` writes `~/.opm/platform/`
- **THEN** `cue.mod/module.cue` SHALL pin exactly one build for each of `opmodel.dev/core@v2`, `opmodel.dev/catalogs/opm@v4` and `opmodel.dev/catalogs/k8s@v1`
- **AND** `platform.cue` SHALL contain exactly two `#registry` entries, keyed `opmodel.dev/catalogs/opm@v4` and `opmodel.dev/catalogs/k8s@v1`, each embedding its catalog by import
- **AND** `platform.cue` SHALL contain no `version` scalar and no `filter` vocabulary

#### Scenario: Seeded platform offers the raw escape hatch

- **WHEN** a module demands a contract from `opmodel.dev/catalogs/k8s@v1` and the platform is the seeded default
- **THEN** the entry needed to match that contract SHALL already be present, without the user editing the platform module

#### Scenario: Legacy data-only platform file is migrated

- **WHEN** `opm config init --force` is run and a legacy data-only `~/.opm/platform.cue` exists
- **THEN** the platform module is written, the legacy file is removed, and the output notes the removal

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

The `opm config vet` command SHALL validate the `~/.opm` configuration.

Checks performed:

1. Config file exists at resolved path
2. Config file is syntactically valid CUE and satisfies the embedded config schema (no imports, no removed fields)
3. Platform module, when present at the sibling `platform/` directory, builds through the kernel's shape-gated platform loader: imports resolve, the value is a well-formed `#Platform`, and the schema's derived-entry tripwires (key-to-import binding, derived version) evaluate

The platform check evaluates a real CUE module: on a cold module cache it performs registry I/O; on a warm cache it is offline. A missing platform module SHALL NOT fail vet (it is optional until a render needs a local default); vet SHALL note its absence. A leftover legacy data-only `~/.opm/platform.cue` SHALL fail vet naming the file, with the hint to re-run `opm config init --force`. Each check SHALL print a styled line to stdout using `FormatVetCheck` as it passes; on failure, previously-passing checks SHALL remain visible.

#### Scenario: Valid configuration passes validation

- **WHEN** `opm config vet` is run
- **WHEN** config.cue is valid and `~/.opm/platform/` builds cleanly
- **THEN** command succeeds
- **THEN** output SHALL contain a checkmark line for each passing check, including the platform module check

#### Scenario: Missing config file fails with actionable error

- **WHEN** `opm config vet` is run
- **WHEN** `~/.opm/config.cue` does not exist
- **THEN** command fails with not-found error
- **THEN** hint: "Run 'opm config init' to create default configuration"

#### Scenario: Missing platform file is noted, not fatal

- **WHEN** `opm config vet` is run
- **WHEN** config.cue is valid and `~/.opm/platform/` does not exist
- **THEN** command succeeds
- **THEN** output SHALL note that no local default platform is configured

#### Scenario: Invalid platform file fails

- **WHEN** the platform module pins a catalog build that does not exist, or an entry's key disagrees with its imported catalog's `modulePath`
- **THEN** the config checks SHALL print passing checkmark lines
- **THEN** command fails with an error naming the platform module and the offending dependency or entry

#### Scenario: Legacy platform file fails with migration hint

- **WHEN** `opm config vet` is run and a data-only `~/.opm/platform.cue` exists
- **THEN** validation SHALL fail naming the legacy file
- **AND** the hint SHALL say to re-run `opm config init --force`

#### Scenario: Stale providers block fails with migration hint

- **WHEN** `opm config vet` is run against a pre-D39 config.cue containing `providers:` or a `~/.opm/cue.mod/`
- **THEN** validation SHALL fail naming the removed field
- **AND** the hint SHALL say to re-run `opm config init` (or remove the field and `cue.mod/`)
