## MODIFIED Requirements

### Requirement: Lint runs on every push
The CI workflow SHALL run `golangci-lint` (the version the file `.golangci-lint-version` names, with `govet` enabled as one of its linters in `.golangci.yml`, so no separate `go vet` step exists) on every push to any branch.

#### Scenario: Lint passes
- **WHEN** a push is made to any branch
- **THEN** the lint job runs and exits zero if no lint errors are found

#### Scenario: Lint fails
- **WHEN** a push introduces a lint violation
- **THEN** the lint job exits non-zero and the commit is marked failed

## ADDED Requirements

### Requirement: The linter configuration is verified without the network
Every `Lint` job that runs golangci-lint (in `.github/workflows/ci.yml` and `.github/workflows/pr.yml`) SHALL verify `.golangci.yml` against the golangci-lint JSON schema committed in the repository, and SHALL NOT download a schema at run time. The golangci-lint action's own configuration check, which downloads the schema, SHALL be off. The verification SHALL give the same result when no network is reachable, and `task lint` SHALL run the same verification before the linters.

#### Scenario: A valid configuration passes with the linter's website unreachable
- **WHEN** the configuration check runs for a valid `.golangci.yml` and `golangci-lint.run` cannot be reached
- **THEN** the check exits zero

#### Scenario: An invalid configuration fails the job
- **WHEN** `.golangci.yml` holds a key the schema does not allow
- **THEN** the configuration check exits non-zero and prints the schema violation

#### Scenario: The action's own check is refused
- **WHEN** a workflow step uses the golangci-lint action without turning its configuration check off
- **THEN** the configuration check exits non-zero and names the workflow

### Requirement: One file names the linter version
The file `.golangci-lint-version` SHALL be the only place that names the golangci-lint version CI installs. Every use of the golangci-lint action SHALL read it and SHALL NOT name a version of its own, every use SHALL pin the same action commit, and each workflow that uses the action SHALL run the configuration check. The committed schema SHALL be the one for that version's minor line, SHALL be the only schema file in its directory, and SHALL match its recorded SHA-256 checksum. The configuration check SHALL fail when any of these does not hold, and when the installed golangci-lint is of another minor line than the file names.

#### Scenario: A workflow names its own version
- **WHEN** a step that uses the golangci-lint action sets a `version` input, or does not read `.golangci-lint-version`
- **THEN** the configuration check exits non-zero and names the workflow

#### Scenario: The version moves to a minor line without a committed schema
- **WHEN** `.golangci-lint-version` names a version whose minor line has no committed schema file
- **THEN** the configuration check exits non-zero and names the missing file

#### Scenario: The committed schema was edited
- **WHEN** the bytes of the committed schema do not match its recorded checksum
- **THEN** the configuration check exits non-zero

#### Scenario: The two workflows pin different action commits
- **WHEN** `pr.yml` and `ci.yml` use the golangci-lint action at different commit SHAs
- **THEN** the configuration check exits non-zero and prints both
