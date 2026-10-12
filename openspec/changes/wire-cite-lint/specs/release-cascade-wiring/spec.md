## ADDED Requirements

### Requirement: Comments follow the shared cite-lint rule locally and in CI

The cli SHALL run the shared comment rule through `task cite-lint` and through `.github/workflows/cite-lint.yml`, which holds one plain call of `open-platform-model/.github/.github/workflows/cite-lint.yml` at the same pinned SHA as every other `.github` reference. The workflow SHALL trigger on `pull_request` to `main` and on `push` to `main` only, and `.tasks/cascade/wiring-check.yaml` SHALL list it under `extra-references` as kind `cite-lint`. The task SHALL run the tool at the commit that the reusable workflow uses by default. Both SHALL exclude only `tests/fixtures/`, and each SHALL name the other in a comment. Source: `.github` README, "cite-lint", at `2b66da51`.

#### Scenario: A decision number in a comment fails the check

- **WHEN** a Go or CUE comment outside `tests/fixtures/` names a decision number
- **THEN** `task cite-lint` exits non-zero and the CI call fails the pull request

#### Scenario: The call is plain

- **WHEN** `task cascade:wiring:check` runs
- **THEN** it accepts the call only if its job has the keys `name`, `permissions`, `uses` and `with`, with `permissions` exactly `contents: read`
