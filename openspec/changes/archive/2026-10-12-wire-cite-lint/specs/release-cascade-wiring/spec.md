## ADDED Requirements

### Requirement: Comments follow the shared cite-lint rule locally and in CI

The cli SHALL run the shared comment rule through `task cite-lint` and a plain call of `.github`'s `cite-lint.yml`, pinned like every `.github` reference. Both SHALL exclude only `tests/fixtures/`.

#### Scenario: A decision number in a comment fails the check

- **WHEN** a Go or CUE comment outside `tests/fixtures/` names a decision number
- **THEN** `task cite-lint` exits non-zero and the CI call fails the pull request

#### Scenario: The call is plain

- **WHEN** `task cascade:wiring:check` runs
- **THEN** it accepts the call only if its job has the keys `name`, `permissions`, `uses` and `with`, with `permissions` exactly `contents: read`
