## ADDED Requirements

### Requirement: cluster:operator builds the CLI it installs with

The `cluster:operator` task SHALL ensure the CLI binary it invokes exists and is current before
invoking it, by depending on the build task rather than assuming a previous build. The dependency
SHALL reuse the build task's existing up-to-date check, so a tree whose binary is already current
performs no rebuild and the task stays idempotent.

The task SHALL therefore succeed from a clean checkout with no manual preparation beyond a running
cluster, and SHALL be callable unattended — a caller that cannot answer a prompt or run a suggested
command by hand, such as an automated test cleanup, SHALL NOT need one.

#### Scenario: Clean tree with no binary

- **WHEN** a developer runs `task cluster:operator` in a checkout that has never been built, against
  a running kind cluster
- **THEN** the CLI binary is built first and the operator install proceeds
- **AND** the task SHALL NOT fail reporting that the binary does not exist

#### Scenario: Binary already current

- **WHEN** `task cluster:operator` runs again with no source changes since the last build
- **THEN** no rebuild is performed
- **AND** the task completes as it did before, without duplicating any operator configuration

#### Scenario: Called unattended by a test cleanup

- **WHEN** a destructive e2e test invokes `task cluster:operator` to rebuild the dev operator it tore
  down, in a worktree where the binary was never built
- **THEN** the task builds what it needs and restores the operator without developer intervention
