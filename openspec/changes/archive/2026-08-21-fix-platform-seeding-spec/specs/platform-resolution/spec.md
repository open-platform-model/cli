## MODIFIED Requirements

### Requirement: Solo-cluster Platform write-if-absent

On a cluster-facing apply where no `Platform` CR exists and resolution fell back to the local default, the CLI SHALL create the singleton `cluster` Platform from the resolved local spec using a plain create (field manager `opm-cli`), treating `AlreadyExists` as success-noop (0006 D22). The seeded document SHALL be the exact resolved platform spec the render consumed, carried through the render result — the CLI MUST NOT re-read the platform file at apply time (no TOCTOU) and MUST NOT seed an empty or partial spec. The CLI MUST NOT use server-side apply or update for this write, and MUST NOT overwrite an existing Platform. Creation failure (e.g. RBAC) SHALL degrade to a warning — the apply itself proceeds against the local platform (0006 D17).

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
