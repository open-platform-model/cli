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

On a cluster-facing apply where no `Platform` CR exists and resolution fell back to the local default, the CLI SHALL create the singleton `cluster` Platform from the resolved local spec using a plain create (field manager `opm-cli`), treating `AlreadyExists` as success-noop (0006 D22). The seeded document SHALL be the exact resolved platform spec the render consumed, carried through the render result — the CLI MUST NOT re-read the platform file at apply time (no TOCTOU) and MUST NOT seed an empty or partial spec. The CLI MUST NOT use server-side apply or update for this write, and MUST NOT overwrite an existing Platform. Creation failure (e.g. RBAC) SHALL degrade to a warning — the apply itself proceeds against the local platform (0006 D17).

The same write contract SHALL govern every caller that seeds the singleton, including `opm operator install`. Callers differ only in where the spec came from and in the provenance they report: an apply reports that it seeded from the local default platform, and an install reports the catalog coordinate and version it resolved from the registry. No caller SHALL report a provenance it did not use.

#### Scenario: Absent Platform is seeded

- **WHEN** an apply succeeds against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` SHALL be created from the local platform spec

#### Scenario: Seeded document matches the render-consumed spec

- **WHEN** the render resolved the local default platform with `type` and at least one registry subscription, and the apply seeds the cluster Platform
- **THEN** the created Platform's spec SHALL carry that same non-empty `type` and the same registry subscriptions (paths, `enable`, `version`)
- **AND** the seeded document SHALL NOT be derived from a second read of the platform file

#### Scenario: Concurrent create tolerated

- **WHEN** the create returns `AlreadyExists`
- **THEN** the CLI SHALL treat it as success and SHALL NOT modify the existing Platform

#### Scenario: RBAC-denied create degrades

- **WHEN** the create is forbidden
- **THEN** the CLI SHALL warn and the apply SHALL still complete

#### Scenario: Install reports its own provenance

- **WHEN** `opm operator install` seeds the Platform from a registry-resolved catalog version
- **THEN** the reported provenance SHALL name the catalog module path and the resolved version
- **AND** it SHALL NOT claim the spec came from the local default platform file

### Requirement: Legacy Cluster CR Tolerance

Decoding a **cluster CR spec** SHALL tolerate legacy stored shapes permanently: a `filter` key SHALL be ignored without error, and a subscription lacking `version` SHALL decode with an empty version and fail only at synthesis via the kernel's missing-version refusal, which the CLI SHALL wrap with a hint that the cluster Platform may predate the scalar-version shape and needs its spec re-applied. This tolerance applies to CR reads only — never to files.

#### Scenario: Filter-shaped stored CR still decodes

- **WHEN** the cluster Platform CR carries a legacy `filter` block and no `version`
- **THEN** decode succeeds, and materialization fails with the missing-version error plus the legacy-CR hint

### Requirement: Registry-resolved catalog subscription

The CLI SHALL be able to resolve the version a Platform subscription pins by querying the registry for the published versions of a major-suffixed catalog module path, rather than reading a version written by hand. Resolution SHALL be scoped to the major carried by that path, so a build from a different major can never be selected.

Two selection modes exist, and the caller chooses between them explicitly:

- **Release (default)**: the highest published version whose SemVer prerelease part is empty.
- **Prerelease (opt-in)**: the highest published version whose prerelease part begins with a non-numeric identifier.

Development builds SHALL NEVER be selected in either mode. A development build is identified by a prerelease part whose first identifier is numeric, which is the form catalog branch-publish pipelines produce so SemVer ranks those builds below every named prerelease. Versions the registry returns that are not valid SemVer SHALL be ignored rather than treated as candidates.

When no published version satisfies the active mode, resolution SHALL refuse rather than substitute a version from the other mode. The refusal SHALL name the module path queried, the registry consulted, the highest version it did see, and the action that would select it. Resolution SHALL NOT infer maturity, widen the major, or fall back to a hand-pinned literal.

#### Scenario: Stable release selected by default

- **WHEN** the catalog major has published `2.0.0-alpha.3`, `2.0.0`, and `2.1.0`
- **THEN** the default mode SHALL resolve `2.1.0`

#### Scenario: Prerelease-only history refuses under the default

- **WHEN** the catalog major has published only `2.0.0-alpha.1`, `2.0.0-alpha.2`, and `2.0.0-alpha.3`
- **THEN** the default mode SHALL refuse
- **AND** the refusal SHALL name the highest prerelease it saw and the action that selects a prerelease

#### Scenario: Prerelease mode ignores development builds

- **WHEN** the catalog major has published `2.0.0-alpha.3` and a later-pushed `2.0.0-0.dev.1754899200.g9ea5927`
- **THEN** the prerelease mode SHALL resolve `2.0.0-alpha.3`

#### Scenario: Development builds are never the answer

- **WHEN** every published version of the catalog major is a development build
- **THEN** both modes SHALL refuse, and no development build SHALL be offered as a candidate

#### Scenario: Empty history refuses

- **WHEN** the catalog major has no published versions
- **THEN** resolution SHALL refuse naming the module path and the registry

#### Scenario: Registry failure is not a refusal

- **WHEN** the version listing fails in transport
- **THEN** the failure SHALL be reported as a connectivity failure naming the lookup and the registry, distinct from a refusal, because no version was ever judged
