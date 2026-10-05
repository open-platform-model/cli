## MODIFIED Requirements

### Requirement: Apply stages CustomResourceDefinitions and Namespaces before the rest

`opm instance apply` and `opm module apply` SHALL apply the rendered resources in two stages, each in ascending order of the library's kind-class weight (`opm/k8s/object.Weight`) with build order breaking ties (0012:D5:R1). The first stage SHALL hold every `CustomResourceDefinition` (group `apiextensions.k8s.io`) and every `Namespace` (core group); the second stage SHALL hold everything else. Outside a dry run, after the first stage the command SHALL wait until every CustomResourceDefinition the first stage applied without error reports the condition `Established=True`, bounded by `--timeout` (default 5m) counted from the start of the apply, with a timeout reporting the time elapsed since then, and only then apply the second stage. When the wait does not complete, the command SHALL fail naming the CustomResourceDefinitions still pending, SHALL NOT apply the second stage, and SHALL NOT prune or write the inventory. A resource that fails to apply SHALL be reported and SHALL NOT stop the remaining resources of its stage or the next stage.

#### Scenario: A CRD and its custom resource apply in one run

- **WHEN** a module renders a custom resource before the CustomResourceDefinition that defines its kind, and `opm instance apply` runs against a cluster that has neither
- **THEN** the CustomResourceDefinition is applied first, the command waits until it is established, the custom resource is applied after it, and the command succeeds on its first run

#### Scenario: A Namespace is applied before the objects in it

- **WHEN** a module renders a ConfigMap in namespace `demo` before the `Namespace` `demo`, and the namespace does not exist
- **THEN** the Namespace is applied before the ConfigMap and both succeed on the first run

#### Scenario: The second stage follows weight order

- **WHEN** the second stage holds a Deployment (weight 100), a Service (weight 50) and a ConfigMap (weight 15) in that build order
- **THEN** they are applied ConfigMap, Service, Deployment

#### Scenario: A CRD that never becomes established fails the apply

- **WHEN** a CustomResourceDefinition of the first stage does not report `Established=True` within `--timeout`
- **THEN** the command exits non-zero naming that CustomResourceDefinition, applies nothing of the second stage, and neither prunes nor writes the inventory
