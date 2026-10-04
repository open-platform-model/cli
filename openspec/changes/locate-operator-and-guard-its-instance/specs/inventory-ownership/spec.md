## MODIFIED Requirements

### Requirement: Operator-owned delete delegates to the operator's finalizer

`opm instance delete` against an operator-owned instance SHALL delete the `ModuleInstance` CR and delegate workload cleanup to the operator's `opmodel.dev/cleanup` finalizer, waiting bounded and reporting completion. Before deleting, the CLI SHALL verify the operator is ready and refuse when it is not — deleting a finalizer-armed CR with no running controller wedges the CR in terminating state. The readiness check SHALL locate the operator as the `operator-lifecycle` capability's running-operator check defines (through the operator's instance record, else by its fixed names), so an operator applied with kubectl from a release manifest is found as well as one the CLI installed.

Whether the operator removes the workloads is governed by `spec.prune`, which has no CRD default and which the CLI does not write: absent it, the operator removes the CR and deliberately orphans the workloads. The CR's disappearance therefore proves the finalizer completed, not that anything was pruned. The CLI SHALL report the outcome that actually occurred and MUST NOT claim a prune it has not established.

#### Scenario: Delete without spec.prune reports the orphaning

- **WHEN** `opm instance delete` removes an operator-owned instance whose `spec.prune` is unset
- **THEN** the CLI SHALL report that the resources were left running, and name the remedy
- **AND** SHALL NOT report that any resource was pruned

#### Scenario: Delete of an operator-owned instance

- **WHEN** `opm instance delete` targets an operator-owned instance and the operator is ready
- **THEN** the CLI SHALL delete the CR, wait for finalizer cleanup, and report the removal

#### Scenario: Delete refused while the operator is down

- **WHEN** the operator Deployment is not available
- **AND** `opm instance delete` targets an operator-owned instance
- **THEN** the CLI SHALL refuse, explaining the finalizer would wedge without a running operator

#### Scenario: Operator applied with kubectl is found

- **WHEN** the operator was applied with kubectl from a release manifest, so the cluster holds no operator instance record, and it is running
- **AND** `opm instance delete` targets an operator-owned instance
- **THEN** the readiness check SHALL pass and the CLI SHALL delete the CR as for any ready operator
