# platform-resolution: Delta

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Solo-cluster Platform write-if-absent

On a cluster-facing apply where no `Platform` CR exists and resolution fell back to the local default, the CLI SHALL create the singleton `cluster` Platform from the resolved local spec using a plain create (field manager `opm-cli`), treating `AlreadyExists` as success-noop (0006 D22). The CLI MUST NOT use server-side apply or update for this write, and MUST NOT overwrite an existing Platform. Creation failure (e.g. RBAC) SHALL degrade to a warning, and the apply itself proceeds against the local platform (0006 D17).

The same write contract SHALL govern every caller that seeds the singleton, including `opm operator install`. Callers differ only in where the spec came from and in the provenance they report: an apply reports that it seeded from the local default platform, and an install reports the catalog coordinate and version it resolved from the registry. No caller SHALL report a provenance it did not use.

#### Scenario: Absent Platform is seeded

- **WHEN** an apply succeeds against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` SHALL be created from the local platform spec

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
