## ADDED Requirements

### Requirement: The running-operator check locates the operator by its fixed names

Every CLI command that checks for a running operator before it acts SHALL locate the operator without reading the CLI's embedded operator manifest. The CLI is moving to install the operator as an OPM module and will stop carrying that manifest, so the check must not depend on it.

The operator's objects SHALL be its fixed names, which every operator release since `v1.0.0-alpha.18` uses and every install of the operator module keeps: the `Deployment` `opm-operator-controller-manager` in the namespace `opm-operator-system`, and the `CustomResourceDefinition`s `moduleinstances.opmodel.dev`, `modulepackages.opmodel.dev`, `platforms.opmodel.dev` and `transformerregistrations.opmodel.dev`. The `Namespace` is checked only as the `Deployment`'s namespace; the check reads no `Namespace` object and no instance record. An operator from an earlier release, which lacks one of these CRDs or serves its CRDs in another group, SHALL be reported as not ready.

The operator SHALL count as running only when every one of these `CustomResourceDefinition`s reports `Established=True` and the `Deployment` has completed its rollout. Otherwise the check SHALL report the operator as not ready, name each object that failed, and point at `opm operator install`. A read of any of these objects that fails SHALL count that object as not ready, so the command proceeds only on a positive finding.

#### Scenario: Manifest-installed operator is found by its fixed names

- **WHEN** the cluster runs an operator applied from a release manifest, with kubectl or by `opm operator install`
- **AND** its four CRDs are `Established` and `opm-operator-controller-manager` has rolled out
- **THEN** the running-operator check SHALL report the operator as running

#### Scenario: Module-installed operator is found by the same names

- **WHEN** the cluster holds the `ModuleInstance` `opm-operator` in `opm-operator-system` whose render created the four CRDs and the `Deployment` `opm-operator-controller-manager` in `opm-operator-system`
- **AND** every CRD is `Established` and the `Deployment` has rolled out
- **THEN** the running-operator check SHALL report the operator as running

#### Scenario: Operator older than the fourth CRD is not ready

- **WHEN** the cluster serves `moduleinstances`, `modulepackages` and `platforms` in `opmodel.dev` but not `transformerregistrations.opmodel.dev`, and `opm-operator-controller-manager` has rolled out
- **THEN** the running-operator check SHALL report the operator as not ready and name `transformerregistrations.opmodel.dev`

#### Scenario: No operator at all

- **WHEN** the cluster holds none of the four CRDs and no `opm-operator-controller-manager` Deployment
- **THEN** the running-operator check SHALL report the operator as not ready, naming the missing objects, and point at `opm operator install`
