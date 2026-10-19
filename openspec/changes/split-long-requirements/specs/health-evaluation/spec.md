## REMOVED Requirements

### Requirement: Readiness is judged by the library health package
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cli takes every readiness decision from the library health package", "The library health decisions cover instance status, list and tree and the apply wait", "The Established wait and the operator install dispatch by kind stay in the cli", "The tree child rows and the status colouring stay in the cli".

### Requirement: The cli carries no readiness evaluator of its own
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "No non-test Go file in the cli declares a copy of the library health package's decisions", "The readiness exemptions are not copies", "A unit test fails when a health decision is declared again or bound to a name of `opm/k8s/health`".

## ADDED Requirements

### Requirement: The cli takes every readiness decision from the library health package

The cli SHALL take every decision the library's `opm/k8s/health` makes from that package: the status vocabulary (`health.Status` and its constants), the per-kind evaluation of one inventory object (`health.Evaluate`), the healthy set (`health.IsHealthy`: `Ready`, `Applied`, `Complete`, `Bound`), and the fold of an instance's or a component's statuses into one verdict with a ready count and a total (`health.Aggregate`).

**Source**: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

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

### Requirement: The library health decisions cover instance status, list and tree and the apply wait

This SHALL cover `opm instance status`, `opm instance list`, `opm instance tree`, and the `--wait` readiness wait of `opm instance apply` and `opm module apply`.
(Requirement: The cli takes every readiness decision from the library health package)

**Source**: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: Instance status takes its row from the library verdict

- **WHEN** `opm instance status` evaluates a Deployment whose `Available` condition is True but whose `updatedReplicas` is lower than `spec.replicas`
- **THEN** its row SHALL read `NotReady`, the value `health.Evaluate` returns for it
- **AND** the command SHALL exit 2

### Requirement: The Established wait and the operator install dispatch by kind stay in the cli

Three readiness checks are not decisions of the package and SHALL stay in the cli: the CustomResourceDefinition `Established` wait, the operator install wait's dispatch by kind (which judges a Deployment through `health.Evaluate` and treats other non-CRD kinds as ready once they exist).
(Requirement: The cli takes every readiness decision from the library health package)

**Source**: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: The Established wait stays in the cli

- **WHEN** a readiness check is the CustomResourceDefinition `Established` wait
- **THEN** it is not a decision of the package and SHALL stay in the cli

### Requirement: The tree child rows and the status colouring stay in the cli

Three readiness checks are not decisions of the package and SHALL stay in the cli: the display-only ReplicaSet and pod child rows of `opm instance tree`, which never feed an aggregate, and the display-only colouring of `FormatHealthStatus`, which groups the status strings for terminal styles.
(Requirement: The cli takes every readiness decision from the library health package)

**Source**: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: The tree child rows never feed an aggregate

- **WHEN** `opm instance tree` shows the display-only ReplicaSet and pod child rows of a component
- **THEN** those rows SHALL stay in the cli and never feed an aggregate

### Requirement: No non-test Go file in the cli declares a copy of the library health package's decisions

No non-test Go file in the cli SHALL declare a copy of the package's decisions: a health status type, a per-kind evaluator of inventory objects, a healthy-set rule or an aggregate fold of its own, nor an alias to the library's.
(Requirement: The cli takes every readiness decision from the library health package)

**Source**: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: An alias of the library health type is refused

- **WHEN** a change adds `type Status = health.Status` or `var Healthy = health.IsHealthy` to a non-test Go file of `internal/kubernetes`
- **THEN** `task test:unit` SHALL fail, naming the file and the line

### Requirement: The readiness exemptions are not copies

The exemptions named in the requirements "The Established wait and the operator install dispatch by kind stay in the cli" and "The tree child rows and the status colouring stay in the cli" SHALL NOT be copies.
(Requirements: The Established wait and the operator install dispatch by kind stay in the cli; The tree child rows and the status colouring stay in the cli)

**Source**: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: The colouring of FormatHealthStatus is not a copy

- **WHEN** a non-test Go file holds the display-only colouring of `FormatHealthStatus`
- **THEN** it is not a copy of the package's decisions

### Requirement: A unit test fails when a health decision is declared again or bound to a name of `opm/k8s/health`

A unit test SHALL fail when a top-level declaration named `HealthStatus`, `EvaluateHealth`, `QuickInstanceHealth` or `IsHealthy`, or a function named `evaluate…Health`, appears in any non-test Go file of the repository, or when a non-test Go file of `internal/kubernetes` declares a top-level type alias or value bound directly to a name of `opm/k8s/health`.

**Source**: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: A reintroduced evaluator fails the unit tests

- **WHEN** a change adds `func EvaluateHealth(obj *unstructured.Unstructured) string` to any non-test Go file
- **THEN** `task test:unit` SHALL fail, naming the file, the line and `opm/k8s/health` as the package to use

#### Scenario: A re-export of the library package fails the unit tests

- **WHEN** a change adds `type Status = health.Status` or `var Healthy = health.IsHealthy` to a non-test Go file of `internal/kubernetes`
- **THEN** `task test:unit` SHALL fail, naming the file and the line
