# platform-resolution — Delta

## MODIFIED Requirements

### Requirement: Local platform file is a data-only CR-spec projection

`~/.opm/platform.cue` (and any `--platform <file>`) SHALL be a data-only CUE file with no imports, shaped as the Platform CR spec projection: `name`, `type`, and `registry` — a map from a **major-suffixed** catalog module path (`…@vN`) to a subscription with optional `enable` and required scalar `version` (bare SemVer naming exactly one catalog build). The filter vocabulary (`filter.range`/`allow`/`deny`) SHALL NOT be accepted. The CLI SHALL validate against the embedded projection schema and decode into `synth.PlatformInput`. One decode function SHALL serve all three sources (flag file, cluster CR spec, local default).

#### Scenario: Valid platform file decodes

- **WHEN** `~/.opm/platform.cue` declares `name`, `type`, and a `registry` entry for `opmodel.dev/catalogs/opm@v2` with `version: "2.0.0-alpha.3"`
- **THEN** decoding yields a `synth.PlatformInput` with that subscription's `Version` set

#### Scenario: Filter shape rejected in files

- **WHEN** a platform file carries a `filter:` block under a subscription
- **THEN** validation fails naming the unknown field

#### Scenario: Major-free key rejected

- **WHEN** a platform file's registry key carries no `@vN` suffix
- **THEN** validation fails before any synthesis

#### Scenario: Import-bearing platform file rejected

- **WHEN** the platform file contains a CUE `import` declaration
- **THEN** validation SHALL fail with an error stating the local platform file must be data-only

## ADDED Requirements

### Requirement: Legacy Cluster CR Tolerance

Decoding a **cluster CR spec** SHALL tolerate legacy stored shapes permanently: a `filter` key SHALL be ignored without error, and a subscription lacking `version` SHALL decode with an empty version and fail only at synthesis via the kernel's missing-version refusal, which the CLI SHALL wrap with a hint that the cluster Platform may predate the scalar-version shape and needs its spec re-applied. This tolerance applies to CR reads only — never to files.

#### Scenario: Filter-shaped stored CR still decodes

- **WHEN** the cluster Platform CR carries a legacy `filter` block and no `version`
- **THEN** decode succeeds, and materialization fails with the missing-version error plus the legacy-CR hint
