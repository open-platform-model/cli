## REMOVED Requirements

### Requirement: Operator-owned delete delegates to the operator's finalizer
**Reason**: It promised that an operator applied with kubectl from a release manifest is found, which nobody runs. Restated without that case as "Operator-owned delete hands cleanup to the operator's finalizer"; the scenario "Operator applied with kubectl is found" goes with it. The readiness check itself does not change.
**Migration**: None.

## ADDED Requirements

### Requirement: Operator-owned delete hands cleanup to the operator's finalizer

`opm instance delete` against an operator-owned instance SHALL delete the `ModuleInstance` CR and delegate workload cleanup to the operator's `opmodel.dev/cleanup` finalizer, waiting bounded and reporting completion. Before deleting, the CLI SHALL verify the operator is ready and refuse when it is not — deleting a finalizer-armed CR with no running controller wedges the CR in terminating state. The readiness check SHALL locate the operator by its fixed names, as the `operator-lifecycle` capability's running-operator check defines.

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
