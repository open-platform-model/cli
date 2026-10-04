## ADDED Requirements

### Requirement: Instance delete of the operator's own instance meets the finalizer guard

`opm instance delete` SHALL NOT delete the operator's own instance, the `ModuleInstance` named `opm-operator` in the namespace `opm-operator-system`, while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer, unless the user gives `--remove-finalizers`. Source: 0028:D9:R5.

The guard SHALL run after the target instance is resolved and before any object is deleted, on the CLI-owned and the operator-owned branch alike, and on a dry run. It SHALL list `ModuleInstance` resources cluster-wide. When any carries `opmodel.dev/cleanup`, the command SHALL refuse with exit code 2, delete nothing, and name each such instance as `namespace/name`, the `--remove-finalizers` flag and its consequence (the named instances' workloads are orphaned). When the cluster-wide list fails, including a permission denial, the command SHALL fail closed with the standard exit code for that Kubernetes error and delete nothing.

`--remove-finalizers` (boolean, default false) SHALL have the meaning it has on `opm operator uninstall`: the command SHALL try to remove exactly the `opmodel.dev/cleanup` finalizer from every armed instance, leaving every other finalizer intact, attempt every armed instance even after one fails, report each instance it orphaned, and proceed to the delete only when every armed instance was stripped. On a dry run, the flag SHALL make the command list the instances whose finalizer it would remove, and SHALL NOT patch any instance.

`--force` SHALL keep its meaning, skipping the confirmation prompt, and SHALL NOT bypass the guard. `--remove-finalizers` given for any instance other than the operator's own SHALL be refused as a usage error (exit code 1) before anything is deleted or patched. Every other instance's delete SHALL be unchanged by this requirement.

#### Scenario: Refused while an instance waits on the operator's cleanup

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --force` runs while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object
- **AND** the error SHALL name `default/hello` and `--remove-finalizers` and state that the override orphans its workloads

#### Scenario: Dry run reports the same refusal

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --dry-run` runs while an instance carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 with the same refusal as the real run

#### Scenario: Override strips the finalizer and proceeds

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --force --remove-finalizers` runs while `default/hello` carries `opmodel.dev/cleanup` and a foreign finalizer
- **THEN** `opmodel.dev/cleanup` SHALL be removed from `default/hello` and the foreign finalizer SHALL remain
- **AND** the command SHALL report that `default/hello`'s workloads are orphaned and then delete the operator's instance as it deletes any instance of its ownership

#### Scenario: Dry run with the override patches nothing

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --dry-run --remove-finalizers` runs while an instance carries `opmodel.dev/cleanup`
- **THEN** the command SHALL list that instance as one whose finalizer would be removed
- **AND** no instance's finalizers SHALL change

#### Scenario: No armed instance, no refusal

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --force` runs and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **THEN** the guard SHALL pass and the delete SHALL proceed as for any instance

#### Scenario: List failure fails closed

- **WHEN** the user may not list `moduleinstances` cluster-wide and runs `opm instance delete opm-operator -n opm-operator-system --force`
- **THEN** the command SHALL exit with the permission-denied exit code without deleting anything

#### Scenario: Override on another instance is a usage error

- **WHEN** `opm instance delete hello -n default --remove-finalizers` runs
- **THEN** the command SHALL exit 1, stating the flag applies only to the operator's instance `opm-operator-system/opm-operator`, and SHALL delete and patch nothing

#### Scenario: Other instances are not guarded

- **WHEN** `opm instance delete hello -n default --force` runs for a CLI-owned instance while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the finalizer guard
