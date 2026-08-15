# config-commands — Delta

## MODIFIED Requirements

### Requirement: Seeded Platform Subscriptions

The command SHALL NOT create `~/.opm/cue.mod/` and SHALL NOT run `cue mod tidy` or any CUE module operation. The seeded `platform.cue` SHALL subscribe to `opmodel.dev/catalogs/opm@v2` only — the sole first-party catalog since the consolidation — with an explicit pinned scalar `version` naming one published catalog build. The pin is load-bearing: it is bumped by hand as catalog releases ship, kept aligned with `hack/platform.cue` and the operator's sample Platform.

#### Scenario: opm config init seeds the platform file

- **WHEN** the user runs `opm config init` with no existing `~/.opm/platform.cue`
- **THEN** the file SHALL contain exactly one `registry` entry, keyed `opmodel.dev/catalogs/opm@v2`
- **AND** that entry SHALL carry an explicit concrete `version`
- **AND** the file SHALL contain no `filter` vocabulary and no `opmodel.dev/catalogs/kubernetes` entry
