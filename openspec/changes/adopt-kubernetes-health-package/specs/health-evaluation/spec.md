## ADDED Requirements

### Requirement: Readiness is judged by the library health package

Every cli path that judges whether a Kubernetes object or an instance is ready SHALL use the library's `opm/k8s/health`: `health.Evaluate` for one live object, `health.IsHealthy` for the healthy set (`Ready`, `Applied`, `Complete`, `Bound`), and `health.Aggregate` to fold an instance's or a component's statuses into one verdict with a ready count and a total. This covers `opm instance status`, `opm instance list`, `opm instance tree`, the `--wait` readiness wait of `opm instance apply` and `opm module apply`, and the operator readiness wait of `opm operator install`. Source: 0012:D3:R6.

#### Scenario: Status uses the library verdict

- **WHEN** `opm instance status` evaluates a Deployment whose `Available` condition is True but whose `updatedReplicas` is lower than `spec.replicas`
- **THEN** its row SHALL read `NotReady`, the value `health.Evaluate` returns for it
- **AND** the command SHALL exit 2

#### Scenario: A passive kind is applied, not ready

- **WHEN** `opm instance status` evaluates a ConfigMap
- **THEN** its row SHALL read `Applied`
- **AND** it SHALL count as healthy toward the aggregate

#### Scenario: Missing and unreadable objects count against the aggregate

- **WHEN** `opm instance list` folds an instance with three healthy live objects, one tracked object missing from the cluster and one that could not be read
- **THEN** the instance SHALL show `NotReady (3/5)`

#### Scenario: Nothing to fold is unknown

- **WHEN** `opm instance tree` builds a component at depth 0, where no resource node is evaluated
- **THEN** the component status SHALL be `Unknown`

### Requirement: The cli carries no readiness evaluator of its own

No non-test Go file in the cli SHALL declare a health status type, a readiness evaluator, a healthy-set rule or an aggregate fold of its own, nor an alias to the library's. A unit test SHALL fail when a top-level declaration named `HealthStatus`, `EvaluateHealth`, `QuickInstanceHealth` or `IsHealthy`, or a function named `evaluate…Health`, appears in any non-test Go file of the repository. Source: 0012:D3:R6.

#### Scenario: A reintroduced evaluator fails the unit tests

- **WHEN** a change adds `func EvaluateHealth(obj *unstructured.Unstructured) string` to any non-test Go file
- **THEN** `task test:unit` SHALL fail, naming the file, the line and `opm/k8s/health` as the package to use

### Requirement: Health status strings are stable output

The status strings the cli prints and serialises SHALL be exactly `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied` and `Bound`, plus a PersistentVolumeClaim's raw `status.phase` and, in `opm instance tree`, a pod's raw phase. The JSON and YAML output of `opm instance status`, `opm instance list` and `opm instance tree` SHALL carry them under the same keys (`status`, `aggregateStatus`) as before the cli adopted the library package, byte for byte.

#### Scenario: JSON status output is unchanged

- **WHEN** the user runs `opm instance status my-app -n prod -o json` on an instance whose Deployment is ready and whose ConfigMap exists
- **THEN** the output SHALL carry `"status": "Ready"` for the Deployment, `"status": "Applied"` for the ConfigMap and `"aggregateStatus": "Ready"`

#### Scenario: A missing object keeps its spelling

- **WHEN** the user runs `opm instance status my-app -n prod -o yaml` and a tracked object is missing from the cluster
- **THEN** its row SHALL carry `status: Missing`
