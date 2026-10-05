## MODIFIED Requirements

### Requirement: IsHealthy helper function

The kubernetes package SHALL provide an `IsHealthy` function that accepts a `HealthStatus` and returns `true` if the status is `HealthReady`, `HealthApplied`, `HealthComplete`, or `HealthBound`, and `false` otherwise. This centralizes the "what counts as healthy" logic: every command that decides whether a resource or instance is fine (status, tree, list, the exit code, the controller wait) SHALL go through it rather than comparing against individual statuses.

#### Scenario: Ready is healthy

- **WHEN** `IsHealthy(HealthReady)` is called
- **THEN** it SHALL return `true`

#### Scenario: NotReady is not healthy

- **WHEN** `IsHealthy(HealthNotReady)` is called
- **THEN** it SHALL return `false`

#### Scenario: Complete is healthy

- **WHEN** `IsHealthy(HealthComplete)` is called
- **THEN** it SHALL return `true`

#### Scenario: Bound is healthy

- **WHEN** `IsHealthy(HealthBound)` is called
- **THEN** it SHALL return `true`

#### Scenario: Applied is healthy

- **WHEN** `IsHealthy(HealthApplied)` is called
- **THEN** it SHALL return `true`
