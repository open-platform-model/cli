# Capability: health-export

## Purpose

`internal/kubernetes/health.go` exports the health vocabulary that the status, tree, and list commands share: the `HealthStatus` type and its constants, `EvaluateHealth` for a single resource, `IsHealthy` for the "what counts as healthy" rule, and `QuickInstanceHealth` for folding a pre-fetched resource set into one aggregate verdict. Exporting them keeps a single definition of health for every command and for the integration programs under `tests/integration/`.

## Requirements

### Requirement: Health status type is exported

The kubernetes package SHALL export the `healthStatus` type as `HealthStatus` and its constants as `HealthReady`, `HealthNotReady`, `HealthComplete`, `HealthUnknown`, `HealthMissing`, `HealthBound`. All existing internal references SHALL be updated to use the exported names.

#### Scenario: External package uses HealthStatus

- **WHEN** the `internal/cmd/module/` package imports `internal/kubernetes`
- **THEN** it SHALL be able to reference `kubernetes.HealthStatus`, `kubernetes.HealthReady`, etc.

#### Scenario: Existing behavior unchanged

- **WHEN** the health evaluation functions are called with the same inputs as before export
- **THEN** they SHALL return the same results (export is a rename, not a behavior change)

### Requirement: EvaluateHealth is exported

The `evaluateHealth` function SHALL be exported as `EvaluateHealth` with the signature `func EvaluateHealth(resource *unstructured.Unstructured) HealthStatus`. It SHALL retain all existing evaluation logic unchanged.

#### Scenario: EvaluateHealth on a ready Deployment

- **WHEN** `EvaluateHealth` is called with an unstructured Deployment that has `Available` condition True
- **THEN** it SHALL return `HealthReady`

#### Scenario: EvaluateHealth on a passive ConfigMap

- **WHEN** `EvaluateHealth` is called with an unstructured ConfigMap
- **THEN** it SHALL return `HealthReady`

### Requirement: QuickInstanceHealth aggregates health from pre-fetched resources

The kubernetes package SHALL provide a `QuickInstanceHealth` function that accepts a slice of live unstructured resources and a missing resource count, and returns the aggregate `HealthStatus`, a ready count, and a total count. A resource SHALL count as ready if its `EvaluateHealth` result is `HealthReady`, `HealthComplete`, or `HealthBound`. The aggregate SHALL be `HealthReady` when all resources are ready and missing count is zero, `HealthNotReady` when any resource is not ready or missing count is greater than zero, and `HealthUnknown` when the total is zero.

#### Scenario: All resources healthy

- **WHEN** `QuickInstanceHealth` is called with 5 live resources all evaluating to `HealthReady` and missing count 0
- **THEN** it SHALL return `(HealthReady, 5, 5)`

#### Scenario: Some resources unhealthy

- **WHEN** `QuickInstanceHealth` is called with 4 live resources (3 ready, 1 not ready) and missing count 1
- **THEN** it SHALL return `(HealthNotReady, 3, 5)`

#### Scenario: No resources

- **WHEN** `QuickInstanceHealth` is called with 0 live resources and missing count 0
- **THEN** it SHALL return `(HealthUnknown, 0, 0)`

#### Scenario: Missing resources count in total

- **WHEN** `QuickInstanceHealth` is called with 3 live healthy resources and missing count 2
- **THEN** it SHALL return `(HealthNotReady, 3, 5)`

### Requirement: IsHealthy helper function

The kubernetes package SHALL provide an `IsHealthy` function that accepts a `HealthStatus` and returns `true` if the status is `HealthReady`, `HealthComplete`, or `HealthBound`, and `false` otherwise. This centralizes the "what counts as healthy" logic.

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
