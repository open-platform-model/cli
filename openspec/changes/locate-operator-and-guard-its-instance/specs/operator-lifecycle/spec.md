## ADDED Requirements

### Requirement: The running-operator check locates the operator through its instance record, else by its fixed names

Every CLI command that checks for a running operator before it acts SHALL locate the operator without reading the CLI's embedded operator manifest. Source: 0028:D3:R13, 0028:D3:R15.

The check SHALL first read the operator's instance record, the `ModuleInstance` named `opm-operator` in the namespace `opm-operator-system` (0028:D3:R18). When that record exists, the operator's objects SHALL be the `CustomResourceDefinition` entries (group `apiextensions.k8s.io`) and the `Deployment` entries (group `apps`) its recorded inventory lists. A record whose inventory lists no `CustomResourceDefinition` or no `Deployment` SHALL make the check report the operator as not ready, naming the record.

When the record does not exist, including when the `ModuleInstance` CRD itself is absent, the operator's objects SHALL be its fixed names, which every released operator manifest uses and every module install keeps (0028:D2:R10): the `Deployment` `opm-operator-controller-manager` in the `Namespace` `opm-operator-system`, and the `CustomResourceDefinition`s `moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev` and `transformerregistrations.opmodel.dev`.

The operator SHALL count as running only when every located `CustomResourceDefinition` reports `Established=True` and every located `Deployment` has completed its rollout. Otherwise the check SHALL report the operator as not ready, name each object that failed, and point at `opm operator install`. A read of the instance record that fails for any reason other than NotFound SHALL fail the check closed, naming the read that failed, so the command proceeds only on a positive finding.

While the CLI still embeds an operator manifest, that manifest's `CustomResourceDefinition` names and its controller `Deployment` name and namespace SHALL equal the fixed names above.

#### Scenario: Manifest-installed operator is found by its fixed names

- **WHEN** the cluster runs an operator applied from a release manifest, with kubectl or by `opm operator install`, and holds no `ModuleInstance` `opm-operator` in `opm-operator-system`
- **AND** its four CRDs are `Established` and `opm-operator-controller-manager` has rolled out
- **THEN** the running-operator check SHALL report the operator as running

#### Scenario: Module-installed operator is found through its record

- **WHEN** the cluster holds the `ModuleInstance` `opm-operator` in `opm-operator-system` whose inventory lists the operator's CRDs and its controller `Deployment`
- **AND** every listed CRD is `Established` and the listed `Deployment` has rolled out
- **THEN** the running-operator check SHALL report the operator as running

#### Scenario: Record lists a CRD the fixed names do not

- **WHEN** the operator's instance record lists a fifth `CustomResourceDefinition` that is not `Established`
- **THEN** the running-operator check SHALL report the operator as not ready and name that CRD

#### Scenario: Record without a controller Deployment fails closed

- **WHEN** the operator's instance record exists and its inventory lists no `Deployment`
- **THEN** the running-operator check SHALL report the operator as not ready, naming the record, and SHALL NOT fall back to the fixed names

#### Scenario: Unreadable record fails closed

- **WHEN** reading the `ModuleInstance` `opm-operator` in `opm-operator-system` fails with an error other than NotFound, such as a permission denial
- **THEN** the running-operator check SHALL fail, naming the record read, and the command that asked SHALL NOT proceed

#### Scenario: No operator at all

- **WHEN** the cluster holds no operator instance record, none of the four CRDs and no `opm-operator-controller-manager` Deployment
- **THEN** the running-operator check SHALL report the operator as not ready, naming the missing objects, and point at `opm operator install`

#### Scenario: Embedded manifest keeps the fixed names

- **WHEN** the CLI's source is tested while it still embeds an operator manifest
- **THEN** the manifest's CRD names and its controller Deployment's name and namespace SHALL equal the fixed names
