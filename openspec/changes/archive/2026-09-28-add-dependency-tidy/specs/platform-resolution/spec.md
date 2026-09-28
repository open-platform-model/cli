## MODIFIED Requirements

### Requirement: Local default platform is a CUE module

The local default platform SHALL be a CUE module directory (`~/.opm/platform/`, sibling of the resolved config file so `--config`/`OPM_CONFIG` overrides move both together): a `cue.mod/module.cue` under the reserved-unpublished module path `opmodel.dev/platforms/local@v0` pinning core and every subscribed catalog, and a `platform.cue` embedding `core.#Platform` with one `#registry` entry per catalog carrying the catalog by import (0019:D5). The build a catalog entry materializes SHALL be named exactly once, as the module's `cue.mod` dependency; `platform.cue` SHALL carry no version scalars. Maintenance is editing `cue.mod` (by hand or `cue mod get`), pinning whatever the new build needs with `opm module tidy` on the platform directory, and verifying with `opm config vet`; the CLI SHALL NOT require any other tool to keep the platform current.

#### Scenario: The module is the resolution

- **WHEN** the platform module's `cue.mod` pins `opmodel.dev/catalogs/opm@v4` at `4.0.1` and a newer catalog is published
- **THEN** the platform still evaluates catalog `4.0.1` bytes until the pin is edited, with no lockfile and no re-resolution

#### Scenario: Pin bump loop

- **WHEN** a user edits the platform module's `cue.mod` to a newer published catalog build and runs `opm config vet`
- **THEN** vet builds the module against the new pin and reports success, or fails naming the dependency when the pinned build does not exist

#### Scenario: Key-to-import drift refuses

- **WHEN** a `#registry` entry is keyed at one catalog path but embeds an import of a different catalog
- **THEN** building the platform module fails with a conflict at a path naming that entry (the 0019:D5 binding), and vet surfaces it
