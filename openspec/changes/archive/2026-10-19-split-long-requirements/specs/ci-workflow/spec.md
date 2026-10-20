## REMOVED Requirements

### Requirement: Fixture modules are published to GHCR by CI
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A workflow publishes the repository's fixture modules to GHCR through `opm module publish`", "The fixture publish workflow triggers on pushes to `main` that touch the fixtures and offers `workflow_dispatch`", "The fixture publish workflow requests `packages: write` at the job level only and skips cleanly on forks", "Idempotency of the fixture publish lives in the caller `hack/fixtures.sh publish`".

### Requirement: One file names the linter version
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "One file names the linter version CI installs", "Every use of the golangci-lint action reads that file and pins one action commit", "The committed golangci-lint schema matches the linter version's minor line and its checksum", "The configuration check fails when the linter version rules do not hold or the installed golangci-lint is of another minor line".

## ADDED Requirements

### Requirement: A workflow publishes the repository's fixture modules to GHCR through `opm module publish`

A workflow (`.github/workflows/publish-fixtures.yml`) SHALL publish the repository's fixture modules to GHCR through `opm module publish`, using the binary built from the commit under test so a fixture that violates any publish gate fails CI.

#### Scenario: Bumped fixture publishes

- **WHEN** a fixture's identity version is bumped and the workflow runs
- **THEN** that fixture SHALL be published at its new tag

### Requirement: The fixture publish workflow triggers on pushes to `main` that touch the fixtures and offers `workflow_dispatch`

A workflow (`.github/workflows/publish-fixtures.yml`) SHALL trigger on pushes to `main` that touch `tests/fixtures/modules/**`, `hack/fixtures.sh` or the workflow file itself, and SHALL additionally offer `workflow_dispatch` so a new coordinate can be published from a branch before the consumers that pin it merge.

#### Scenario: A push that touches the fixtures triggers the workflow

- **WHEN** a push to `main` touches `tests/fixtures/modules/**`, `hack/fixtures.sh` or the workflow file itself
- **THEN** A workflow (`.github/workflows/publish-fixtures.yml`) SHALL trigger

### Requirement: The fixture publish workflow requests `packages: write` at the job level only and skips cleanly on forks

The workflow (`.github/workflows/publish-fixtures.yml`) SHALL request `packages: write` at the job level only, and SHALL skip cleanly on forks, which cannot obtain that permission.

#### Scenario: Fork run

- **WHEN** the workflow triggers on a fork
- **THEN** the publish job SHALL be skipped rather than fail on a missing token

### Requirement: Idempotency of the fixture publish lives in the caller `hack/fixtures.sh publish`

Idempotency SHALL live in the caller: `hack/fixtures.sh publish` SHALL attempt every fixture and treat a publish whose only refusal is that GHCR already holds the tag as "nothing to do", because publish itself always refuses an already-published tag and never skips.

#### Scenario: Unchanged fixture republished

- **WHEN** the workflow runs and every fixture's declared version is already on GHCR
- **THEN** each SHALL be reported as already present by the caller-side filter and the workflow SHALL succeed

### Requirement: One file names the linter version CI installs

The file `.golangci-lint-version` SHALL be the only place that names the golangci-lint version CI installs; `go.mod`, which the action reads first, SHALL NOT name one.

#### Scenario: go.mod names a linter version
- **WHEN** `go.mod` holds a golangci-lint module line
- **THEN** the configuration check exits non-zero, because the action would take the version from there

### Requirement: Every use of the golangci-lint action reads that file and pins one action commit

Every use of the golangci-lint action, in any letter case and quoted or not, SHALL read it and SHALL NOT name a version of its own, every use SHALL pin the same action commit, and each workflow that uses the action SHALL run the configuration check.
(Requirement: One file names the linter version CI installs)

#### Scenario: A workflow names its own version
- **WHEN** a step that uses the golangci-lint action sets a `version` input, or does not read `.golangci-lint-version`
- **THEN** the configuration check exits non-zero and names the workflow

#### Scenario: The two workflows pin different action commits
- **WHEN** `pr.yml` and `ci.yml` use the golangci-lint action at different commit SHAs
- **THEN** the configuration check exits non-zero and prints both

### Requirement: The committed golangci-lint schema matches the linter version's minor line and its checksum

The committed schema SHALL be the one for that version's minor line, SHALL be the only schema file in its directory, and SHALL match its recorded SHA-256 checksum.
(Requirement: One file names the linter version CI installs)

#### Scenario: The version moves to a minor line without a committed schema
- **WHEN** `.golangci-lint-version` names a version whose minor line has no committed schema file
- **THEN** the configuration check exits non-zero and names the missing file

#### Scenario: The committed schema was edited
- **WHEN** the bytes of the committed schema do not match its recorded checksum
- **THEN** the configuration check exits non-zero

### Requirement: The configuration check fails when the linter version rules do not hold or the installed golangci-lint is of another minor line

The configuration check SHALL fail when any of these does not hold, and when the installed golangci-lint is of another minor line than the file names.
(Requirements: One file names the linter version CI installs; Every use of the golangci-lint action reads that file and pins one action commit; The committed golangci-lint schema matches the linter version's minor line and its checksum)

#### Scenario: The installed linter is of another minor line
- **WHEN** the golangci-lint on `PATH` reports a minor line other than the one `.golangci-lint-version` names
- **THEN** the configuration check exits non-zero and prints both versions
