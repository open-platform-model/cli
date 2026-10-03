## ADDED Requirements

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
