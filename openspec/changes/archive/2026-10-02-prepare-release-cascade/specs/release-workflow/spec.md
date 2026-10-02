## ADDED Requirements

### Requirement: Commit types decide whether a release is cut
`release-please-config.json` SHALL list `feat`, `fix`, `perf`, `revert`, `deps` and `refactor` as visible changelog sections, so a commit of one of those types on `main` opens or updates a release PR. It SHALL list `docs`, `test`, `ci`, `build` and `chore` as hidden sections, so a commit carrying only those types cuts no release. Source: workspace RELEASING.md, section "Pin classes".

#### Scenario: Docs-only commit cuts no release
- **WHEN** only `docs(...)` commits land on `main` after the last release
- **THEN** release-please opens no release PR

#### Scenario: Refactor still releases
- **WHEN** a `refactor(...)` commit lands on `main`
- **THEN** release-please opens or updates the release PR and lists it under Code Refactoring

#### Scenario: Docs commit beside a fix
- **WHEN** a `docs(...)` commit and a `fix(...)` commit land on `main`
- **THEN** the release PR's changelog lists the fix and omits the docs commit
