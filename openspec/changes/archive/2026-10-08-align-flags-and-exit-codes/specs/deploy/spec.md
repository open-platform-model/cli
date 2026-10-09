## MODIFIED Requirements

### Requirement: Instance delete of an instance that deploys the operator is guarded

`opm instance delete` SHALL NOT delete an instance that deploys the operator while that instance is operator-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer.

An instance deploys the operator when any one of these holds, read from its record without rendering: it is named `opm-operator` in the namespace `opm-operator-system`; its module path, with any `@<major>` suffix removed, is `opmodel.dev/modules/opm_operator`; or its recorded inventory holds a `CustomResourceDefinition` (group `apiextensions.k8s.io`) whose name, after its first `.`, is exactly `opmodel.dev`. These are the same signals by which the operator recognises the instance that deploys it, so the CLI and the operator agree on which instance that is.

The guard SHALL run after the target instance is resolved and before any object is deleted, on the CLI-owned and the operator-owned branch alike, and on a dry run with the same outcome as the real run.

- When the target is operator-owned, the command SHALL refuse with exit code 2 and delete and patch nothing, naming the signal that matched and the remedy of setting `spec.owner` to `cli`. The operator never reconciles, finalizes or prunes the instance that deploys it, so the operator-owned delete would wait on, and report, a cleanup that does not happen.
- Otherwise the command SHALL list `ModuleInstance` resources cluster-wide. When any carries `opmodel.dev/cleanup`, the command SHALL refuse with exit code 2, delete and patch nothing, and name each such instance as `namespace/name`, marking the target itself when it is one of them, and name `opm operator uninstall` with its `--remove-finalizers` choice and that choice's consequence (the named instances' workloads are orphaned). Deleting the operator while it still owes cleanup leaves every armed instance unable to finish deletion until an operator runs again. When the cluster-wide list fails, the command SHALL fail closed with the exit code `opm` maps that Kubernetes error to (4 for a permission or authentication denial) and delete nothing.

`opm instance delete` SHALL offer no flag that removes finalizers and SHALL NOT write `spec.owner`. `--yes` SHALL keep its one meaning, skipping the confirmation prompt, and SHALL NOT bypass either refusal; neither SHALL `--force`, its deprecated alias on this command (capability `flag-conventions`). Every other instance's delete SHALL be unchanged by this requirement.

#### Scenario: Refused while an instance waits on the operator's cleanup

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --yes` runs for a CLI-owned record while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object, and the record and every object its inventory lists SHALL still exist
- **AND** the error SHALL name `default/hello` and `opm operator uninstall --remove-finalizers` and state that this choice orphans its workloads

#### Scenario: Operator-owned instance of the operator is refused

- **WHEN** the record `opm-operator` in `opm-operator-system` is operator-owned, with `spec.prune: true`, and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **AND** `opm instance delete opm-operator -n opm-operator-system --yes` runs
- **THEN** the command SHALL exit 2 and the `ModuleInstance` `opm-operator` SHALL still exist
- **AND** the error SHALL name the matched signal and say to set `spec.owner` to `cli`, and SHALL NOT report that any resource was pruned

#### Scenario: Dry run reports the same refusal

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --dry-run` runs for a CLI-owned record while an instance carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 with the same refusal as the real run, and nothing SHALL be deleted

#### Scenario: The target itself carries the finalizer

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --yes` runs for a CLI-owned record while the `ModuleInstance` `opm-operator` itself carries `opmodel.dev/cleanup` and no other instance does
- **THEN** the command SHALL exit 2, naming `opm-operator-system/opm-operator` as the instance being deleted

#### Scenario: The operator module under another name is guarded

- **WHEN** `opm instance delete ops -n platform --yes` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_operator@v0` while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: A record holding the operator's CRDs is guarded

- **WHEN** `opm instance delete crds -n platform --yes` runs for a CLI-owned record of another name and module path whose inventory holds the `CustomResourceDefinition` `moduleinstances.opmodel.dev`, while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: Look-alike names are not the operator

- **WHEN** `opm instance delete dash -n default --yes` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_operator_dashboard@v0` and whose inventory holds only the `CustomResourceDefinition` `widgets.example.opmodel.dev.io`, while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

#### Scenario: No armed instance, no refusal

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --yes` runs for a CLI-owned record and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **THEN** the guard SHALL pass and the delete SHALL proceed as for any CLI-owned instance

#### Scenario: List failure fails closed

- **WHEN** the user may not list `moduleinstances` cluster-wide and runs `opm instance delete opm-operator -n opm-operator-system --force` for a CLI-owned record
- **THEN** the command SHALL exit 4 without deleting anything

#### Scenario: Other instances are not guarded

- **WHEN** `opm instance delete hello -n default --yes` runs for a CLI-owned instance of another module while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

#### Scenario: The deprecated alias does not bypass the guard

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --force` runs for a CLI-owned record while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object, exactly as with `--yes`
