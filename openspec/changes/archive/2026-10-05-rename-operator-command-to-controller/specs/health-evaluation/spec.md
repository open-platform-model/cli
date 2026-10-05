## MODIFIED Requirements

### Requirement: Readiness is judged by the library health package

The cli SHALL take every decision the library's `opm/k8s/health` makes from that package: the status vocabulary (`health.Status` and its constants), the per-kind evaluation of one inventory object (`health.Evaluate`), the healthy set (`health.IsHealthy`: `Ready`, `Applied`, `Complete`, `Bound`), and the fold of an instance's or a component's statuses into one verdict with a ready count and a total (`health.Aggregate`). This covers `opm instance status`, `opm instance list`, `opm instance tree`, and the `--wait` readiness wait of `opm instance apply` and `opm module apply`. Three readiness checks are not decisions of the package and stay in the cli: the CustomResourceDefinition `Established` wait, the controller install wait's dispatch by kind (which judges a Deployment through `health.Evaluate` and treats other non-CRD kinds as ready once they exist), the display-only ReplicaSet and pod child rows of `opm instance tree`, which never feed an aggregate, and the display-only colouring of `FormatHealthStatus`, which groups the status strings for terminal styles. Source: 0012:D3:R6.

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
