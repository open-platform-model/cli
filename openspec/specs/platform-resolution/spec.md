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

`~/.opm/platform.cue` (and any `--platform <file>`) SHALL be a data-only CUE file with no imports, shaped as the Platform CR spec projection: `name`, `type`, and `registry` (map of catalog module path to subscription with optional `enable` and `filter.range`/`allow`/`deny`). The CLI SHALL validate it against an embedded projection schema and decode it into `synth.PlatformInput`. One decode function SHALL serve all three sources (flag file, cluster CR spec, local default).

#### Scenario: Valid platform file decodes

- **WHEN** `~/.opm/platform.cue` declares `name`, `type`, and a `registry` entry for `opmodel.dev/catalogs/opm` with a `filter.range`
- **THEN** resolution SHALL produce a `PlatformInput` carrying that subscription and materialize it

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
