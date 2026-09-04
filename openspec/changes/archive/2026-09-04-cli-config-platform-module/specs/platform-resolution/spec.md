## ADDED Requirements

### Requirement: Local default platform is a CUE module

The local default platform SHALL be a CUE module directory (`~/.opm/platform/`, sibling of the resolved config file so `--config`/`OPM_CONFIG` overrides move both together): a `cue.mod/module.cue` under the reserved-unpublished module path `opmodel.dev/platforms/local@v0` pinning core and every subscribed catalog, and a `platform.cue` embedding `core.#Platform` with one `#registry` entry per catalog carrying the catalog by import (0019 D5). The build a catalog entry materializes SHALL be named exactly once, as the module's `cue.mod` dependency; `platform.cue` SHALL carry no version scalars. Maintenance is editing `cue.mod` (by hand or `cue mod get`) and verifying with `opm config vet`; the CLI SHALL NOT require any other tool to keep the platform current.

#### Scenario: The module is the resolution

- **WHEN** the platform module's `cue.mod` pins `opmodel.dev/catalogs/opm@v4` at `4.0.1` and a newer catalog is published
- **THEN** the platform still evaluates catalog `4.0.1` bytes until the pin is edited, with no lockfile and no re-resolution

#### Scenario: Pin bump loop

- **WHEN** a user edits the platform module's `cue.mod` to a newer published catalog build and runs `opm config vet`
- **THEN** vet builds the module against the new pin and reports success, or fails naming the dependency when the pinned build does not exist

#### Scenario: Key-to-import drift refuses

- **WHEN** a `#registry` entry is keyed at one catalog path but embeds an import of a different catalog
- **THEN** building the platform module fails with a conflict at a path naming that entry (the D5 binding), and vet surfaces it

## REMOVED Requirements

### Requirement: Local platform file is a data-only CR-spec projection

**Reason**: 0019 D5 removes the subscription shape the projection mirrored (`version!` scalars are gone from core's `#Platform`; a registry entry carries its catalog by import), and the library wave deletes `SynthesizePlatform`, the consumer the decode fed. A data-only file cannot carry imports, so the local default platform becomes a CUE module (see the added requirement).

**Migration**: run `opm config init --force`: it writes `~/.opm/platform/` (the module form, both first-party catalogs pinned in `cue.mod`) and removes the legacy `~/.opm/platform.cue`. Until re-init, `opm config vet` fails naming the legacy file. The `--platform` flag's move from a data file to a module directory, and the cluster CR spec's decode, are reworked in the sibling changes (`cli-render-switch`, `cli-platform-cr-generation`); their requirements in this capability are unchanged by this change.
