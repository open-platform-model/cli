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
