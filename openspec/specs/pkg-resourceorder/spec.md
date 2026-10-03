# Capability: pkg-resourceorder

## Purpose

`pkg/resourceorder` assigns every Kubernetes GVK an integer weight that fixes apply order (CRDs and namespaces before RBAC and config, workloads before ingress and policy) so a module's resources land in a dependency-safe sequence. It is deliberately public and dependency-light, importing only `k8s.io/apimachinery`, so the operator and other tools can order resources exactly the way the CLI does.

## Requirements

### Requirement: Public resource ordering API
The `pkg/resourceorder` package SHALL export a `GetWeight` function that returns an integer ordering weight for any Kubernetes GVK, enabling deterministic resource apply ordering. Lower weights SHALL be applied first.

#### Scenario: Known GVK returns specific weight
- **WHEN** `resourceorder.GetWeight(gvk)` is called with a known GVK (e.g., CustomResourceDefinition)
- **THEN** it SHALL return the pre-defined weight for that GVK (e.g., -100 for CRDs)

#### Scenario: Unknown GVK returns default weight
- **WHEN** `resourceorder.GetWeight(gvk)` is called with an unrecognized GVK
- **THEN** it SHALL return `WeightDefault` (1000)

#### Scenario: Kind-only fallback
- **WHEN** `resourceorder.GetWeight(gvk)` is called with a GVK whose full Group/Version/Kind is not in the table but whose Kind is recognized
- **THEN** it SHALL return the weight for that Kind

### Requirement: No CLI dependencies
The `pkg/resourceorder` package SHALL NOT import any CLI-specific or internal packages. Its only external dependency SHALL be `k8s.io/apimachinery`.

#### Scenario: Clean dependency tree
- **WHEN** `pkg/resourceorder/` is compiled
- **THEN** its dependency tree contains only standard library and `k8s.io/apimachinery`

### Requirement: One stable weight sort

The `pkg/resourceorder` package SHALL export a generic sort that orders a slice in place by `GetWeight` of each element's GVK, read through a caller-supplied accessor, ascending or descending, and SHALL keep elements of equal weight in their input order. Every CLI path that orders resources for apply, delete, prune or the operator install and uninstall plans SHALL use it rather than a sort of its own. Adding the sort SHALL NOT widen the package's dependency tree beyond the standard library and `k8s.io/apimachinery/pkg/runtime/schema`.

#### Scenario: Ascending order for apply

- **WHEN** the sort runs ascending over a Deployment, a CustomResourceDefinition and a ConfigMap
- **THEN** the result is CustomResourceDefinition, ConfigMap, Deployment

#### Scenario: Descending order for delete

- **WHEN** the sort runs descending over the same three objects
- **THEN** the result is Deployment, ConfigMap, CustomResourceDefinition

#### Scenario: Equal weights keep input order

- **WHEN** the sort runs over two ConfigMaps `b` then `a`
- **THEN** the result is `b` then `a`
