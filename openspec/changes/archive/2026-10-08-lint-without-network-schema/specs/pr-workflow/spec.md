## MODIFIED Requirements

### Requirement: Lint and unit mirror the push workflow
The `lint` job SHALL run golangci-lint at the version the file `.golangci-lint-version` names and the `unit` job SHALL run `go test ./internal/...`, both on Go 1.26.0, exactly as the push-triggered CI workflow does.

#### Scenario: Lint violation fails the PR
- **WHEN** the pull request introduces a lint violation
- **THEN** the `lint` job exits non-zero

#### Scenario: Unit failure fails the PR
- **WHEN** the pull request introduces a failing unit test
- **THEN** the `unit` job exits non-zero
