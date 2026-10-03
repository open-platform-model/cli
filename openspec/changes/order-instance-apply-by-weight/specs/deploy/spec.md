## ADDED Requirements

### Requirement: Apply stages CustomResourceDefinitions and Namespaces before the rest

`opm instance apply` and `opm module apply` SHALL apply the rendered resources in two stages, each in ascending `pkg/resourceorder` weight order with build order breaking ties. The first stage SHALL hold every `CustomResourceDefinition` (group `apiextensions.k8s.io`) and every `Namespace` (core group); the second stage SHALL hold everything else. Outside a dry run, after the first stage the command SHALL wait until every CustomResourceDefinition the first stage applied without error reports the condition `Established=True`, bounded by `--timeout` (default 5m) counted from the start of the apply, with a timeout reporting the time elapsed since then, and only then apply the second stage. When the wait does not complete, the command SHALL fail naming the CustomResourceDefinitions still pending, SHALL NOT apply the second stage, and SHALL NOT prune or write the inventory. A resource that fails to apply SHALL be reported and SHALL NOT stop the remaining resources of its stage or the next stage.

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

### Requirement: A dry run skips a custom resource whose CRD the same apply creates

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a custom resource whose group and kind are defined by a CustomResourceDefinition of the same apply that does not yet exist on the cluster. The command SHALL log a warning naming the skipped resource and its CustomResourceDefinition, SHALL count it as skipped in the dry-run summary, and SHALL NOT treat it as an error. A dry run SHALL NOT wait for any CustomResourceDefinition. A custom resource whose CustomResourceDefinition already exists on the cluster SHALL be sent as usual. A namespaced object in a Namespace that the same apply creates is not skipped: its dry run is sent and fails namespace admission as before.

#### Scenario: A dry run with a new CRD skips its custom resource

- **WHEN** `opm instance apply --dry-run` runs for a module rendering CustomResourceDefinition `foos.example.com` and one `Foo`, and the cluster has no such CustomResourceDefinition
- **THEN** the `Foo` is not sent, a warning names `Foo` and `foos.example.com`, the summary reports one resource skipped, and the command exits 0

#### Scenario: A dry run with an existing CRD sends its custom resource

- **WHEN** the same dry run runs against a cluster where `foos.example.com` already exists
- **THEN** the `Foo` is sent to the server-side dry run and nothing is skipped
