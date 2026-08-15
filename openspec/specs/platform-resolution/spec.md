# Capability: platform-resolution

## Purpose

Platform-source resolution by precedence with visible provenance (enhancement 0006 D11/D12/D17/D21/D22/D39). All sources converge on `synth.PlatformInput` → `SynthesizePlatform` → `Materialize` — the operator's own ingestion path.

## Requirements

### Requirement: Platform source precedence

The CLI SHALL resolve the platform for every render by precedence: `--platform <file>` (highest, explicit local override) > cluster `Platform` CR spec (cluster-facing commands only) > local default `~/.opm/platform.cue`. Every command that renders SHALL report which platform source it resolved. The `--provider` flag SHALL NOT exist (superseded by `--platform`, 0006 D21).

#### Scenario: Flag wins

- **WHEN** `opm instance apply --platform ./my-platform.cue` runs against a cluster that has a `Platform` CR
- **THEN** the render SHALL use `./my-platform.cue`
- **AND** the output SHALL report the platform source as the flag-provided file

#### Scenario: Cluster CR used when no flag

- **WHEN** `opm instance apply` runs with no `--platform` against a cluster with a readable `Platform` CR
- **THEN** the render SHALL use the cluster CR's spec
- **AND** the output SHALL report the platform source as the cluster CR

#### Scenario: Fallback to local default warns

- **WHEN** `opm instance apply` runs with no `--platform` and the cluster `Platform` CR is absent or unreadable (RBAC denied)
- **THEN** the render SHALL use `~/.opm/platform.cue`
- **AND** a warning SHALL state that the cluster Platform was not used and why

#### Scenario: Offline commands never read the cluster

- **WHEN** `opm instance build` or `opm module build` runs
- **THEN** the CLI SHALL NOT attempt any cluster read for platform resolution
- **AND** the platform SHALL come from `--platform` or the local default only

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

### Requirement: Materialization mirrors the operator

The resolved platform spec SHALL be materialized via kernel `SynthesizePlatform` → `Materialize` — the same calls the operator's `PlatformReconciler` makes. The CLI MUST NOT persist the materialized result.

#### Scenario: Same kernel calls

- **WHEN** any platform source is resolved
- **THEN** the CLI SHALL call `SynthesizePlatform` then `Materialize` on the invocation's kernel
- **AND** no materialized platform SHALL be written to disk or cluster

### Requirement: Solo-cluster Platform write-if-absent

On a cluster-facing apply where no `Platform` CR exists and resolution fell back to the local default, the CLI SHALL create the singleton `cluster` Platform from the resolved local spec using a plain create (field manager `opm-cli`), treating `AlreadyExists` as success-noop (0006 D22). The CLI MUST NOT use server-side apply or update for this write, and MUST NOT overwrite an existing Platform. Creation failure (e.g. RBAC) SHALL degrade to a warning — the apply itself proceeds against the local platform (0006 D17).

#### Scenario: Absent Platform is seeded

- **WHEN** an apply succeeds against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` SHALL be created from the local platform spec

#### Scenario: Concurrent create tolerated

- **WHEN** the create returns `AlreadyExists`
- **THEN** the CLI SHALL treat it as success and SHALL NOT modify the existing Platform

#### Scenario: RBAC-denied create degrades

- **WHEN** the create is forbidden
- **THEN** the CLI SHALL warn and the apply SHALL still complete

### Requirement: Legacy Cluster CR Tolerance

Decoding a **cluster CR spec** SHALL tolerate legacy stored shapes permanently: a `filter` key SHALL be ignored without error, and a subscription lacking `version` SHALL decode with an empty version and fail only at synthesis via the kernel's missing-version refusal, which the CLI SHALL wrap with a hint that the cluster Platform may predate the scalar-version shape and needs its spec re-applied. This tolerance applies to CR reads only — never to files.

#### Scenario: Filter-shaped stored CR still decodes

- **WHEN** the cluster Platform CR carries a legacy `filter` block and no `version`
- **THEN** decode succeeds, and materialization fails with the missing-version error plus the legacy-CR hint
