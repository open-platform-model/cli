## REMOVED Requirements

### Requirement: Instance delete of an instance that deploys the operator is guarded
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-controller-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

### Requirement: Instance delete keeps the ModuleInstance when a tracked resource could not be read
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-controller-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## MODIFIED Requirements

### Requirement: Instance delete re-checks live ownership before each delete

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again and SHALL delete it only when its `app.kubernetes.io/managed-by` label carries an OPM value and its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID. As in the controller's prune, an object with no UUID label SHALL be judged on the managed-by label alone, and so SHALL every object when the instance has no recorded UUID. A resource that fails the check SHALL be left behind and listed with the reason, and SHALL NOT count as a failure. A resource that is already gone, whether its re-read or its delete call returns NotFound, SHALL count neither as deleted nor as a failure. Any other read error SHALL count as a failure for that resource, so the `ModuleInstance` record is kept and a re-run retries.

#### Scenario: Resource no longer managed by OPM is left behind

- **WHEN** a tracked resource's live `app.kubernetes.io/managed-by` label is missing or not an OPM value
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that OPM no longer manages it

#### Scenario: Resource owned by another instance is left behind

- **WHEN** a tracked resource's live `module-instance.opmodel.dev/uuid` label differs from the instance's recorded UUID
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that another instance owns it

#### Scenario: Resource without a UUID label is deleted on managed-by alone

- **WHEN** a tracked resource carries an OPM managed-by value and no `module-instance.opmodel.dev/uuid` label
- **THEN** the resource SHALL be deleted

#### Scenario: Resource gone before its delete

- **WHEN** a tracked resource no longer exists when it is read again
- **THEN** the command SHALL NOT report an error for it
- **AND** it SHALL NOT be counted as deleted

#### Scenario: Resource gone between its re-read and its delete

- **WHEN** a tracked resource passes the ownership check and its delete call returns NotFound
- **THEN** the command SHALL NOT report an error for it
- **AND** it SHALL NOT be counted as deleted

#### Scenario: Instance without a recorded UUID deletes on managed-by alone

- **WHEN** the instance has no recorded UUID
- **AND** a tracked resource carries an OPM managed-by value and any `module-instance.opmodel.dev/uuid` label
- **THEN** the resource SHALL be deleted

#### Scenario: Read error keeps the ModuleInstance

- **WHEN** re-reading a tracked resource fails with an error other than NotFound, such as Forbidden
- **THEN** the resource SHALL be reported as failed
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the command SHALL exit non-zero

### Requirement: A dry run skips a namespaced object whose namespace the same apply creates

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a namespaced object whose namespace the apply would create: a `Namespace` (core group) the same apply renders and that does not exist on the cluster, or the instance namespace that `--create-namespace` would create. A dry run persists neither, so the server's namespace admission would refuse every object in them although the real apply would not. The command SHALL log a warning naming the skipped object and its namespace and saying that a dry run cannot validate it, SHALL count it as skipped in the dry-run summary, SHALL NOT treat it as an error, and SHALL exit 0 when nothing else fails. A namespaced object in a namespace that already exists, and every cluster-scoped object, SHALL be sent as usual. Without `--create-namespace`, a missing instance namespace that no rendered `Namespace` creates SHALL NOT cause a skip: the dry run reports the error the real apply would hit. Outside a dry run nothing is skipped.

#### Scenario: A dry run with a new Namespace skips the objects in it

- **WHEN** `opm instance apply --dry-run` runs for a module rendering `Namespace` `demo` and ConfigMap `cfg` in `demo`, and the cluster has no namespace `demo`
- **THEN** the Namespace is sent to the server-side dry run, `cfg` is not sent, a warning names `ConfigMap/cfg` and `demo`, the summary reports one resource skipped, no error is reported, and the command exits 0

#### Scenario: A dry run with --create-namespace skips the objects in the new instance namespace

- **WHEN** `opm instance apply --dry-run --create-namespace` runs for an instance in namespace `media`, the cluster has no namespace `media`, and the module renders no `Namespace` object
- **THEN** the command reports that namespace `media` would be created, skips every namespaced object in `media` with a warning naming `media`, counts them as skipped, and exits 0

#### Scenario: A dry run in an existing namespace sends everything

- **WHEN** the same dry run runs against a cluster where the namespace already exists
- **THEN** every object is sent to the server-side dry run and nothing is skipped for its namespace

#### Scenario: A real apply skips nothing

- **WHEN** `opm instance apply --create-namespace` runs without `--dry-run` for an instance whose namespace does not exist
- **THEN** the namespace is created and every object is applied

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-D-001 | `mod apply` MUST be idempotent. |
| NFR-D-003 | No enforced limits on module complexity. |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-D-001 | New user can init, build, apply in under 3 minutes. |
| SC-D-003 | `mod apply` is fully idempotent. |

---

## Edge Cases

| Case | Handling |
|------|----------|
| Render errors | `mod apply` fails before touching cluster |
| Cluster unreachable | Fail fast with connectivity error |
| RBAC denied | Pass through Kubernetes API error |
| Field conflict | Log warning, take ownership |
| Module source deleted | `instance delete` works via the inventory record |
| Empty RenderResult | Fail with error if previous inventory is non-empty (use --force to override) |
| Delete without an instance argument | Return a usage error: the positional `<file|name|uuid>` argument is required |
| Catalog without identity support | Identity labels omitted; existing labeling unchanged |

---

## Command Syntax

### mod apply

```text
opm mod apply [path] [flags]

Arguments:
  path    Path to module directory (default: .)

Flags:
  -f, --values strings      Additional values files (can be repeated)
  -n, --namespace string    Target namespace
      --name string         Instance name (default: module name)
      --platform string     Platform module directory (overrides the cluster Platform and the module's own deps)
      --dry-run             Server-side dry run
      --create-namespace    Create target namespace if it does not exist
      --no-prune            Skip stale resource pruning
      --force               Allow empty render to prune all resources
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

### instance delete

```text
opm instance delete <file|name|uuid> [flags]

Arguments:
  file|name|uuid        Instance file, instance name, or instance UUID

Flags:
  -n, --namespace string    Target namespace
      --force               Skip confirmation prompt
      --dry-run             Preview without deleting
      --timeout duration    Wait for the controller to finish deleting a controller-owned instance
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage error |
| 2 | Render error |
| 3 | Kubernetes error |

## ADDED Requirements

### Requirement: Instance delete of an instance that deploys the controller is guarded

`opm instance delete` SHALL NOT delete an instance that deploys the controller while that instance is controller-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer.

An instance deploys the controller when any one of these holds, read from its record without rendering: it is named `opm-controller` in the namespace `opm-controller-system`; its module path, with any `@<major>` suffix removed, is `opmodel.dev/modules/opm_controller`; or its recorded inventory holds a `CustomResourceDefinition` (group `apiextensions.k8s.io`) whose name, after its first `.`, is exactly `opmodel.dev`. These are the same signals by which the controller recognises the instance that deploys it, so the CLI and the controller agree on which instance that is.

The guard SHALL run after the target instance is resolved and before any object is deleted, on the CLI-owned and the controller-owned branch alike, and on a dry run with the same outcome as the real run.

- When the target is controller-owned, the command SHALL refuse with exit code 2 and delete and patch nothing, naming the signal that matched and the remedy of setting `spec.owner` to `cli`. The controller never reconciles, finalizes or prunes the instance that deploys it, so the controller-owned delete would wait on, and report, a cleanup that does not happen.
- Otherwise the command SHALL list `ModuleInstance` resources cluster-wide. When any carries `opmodel.dev/cleanup`, the command SHALL refuse with exit code 2, delete and patch nothing, and name each such instance as `namespace/name`, marking the target itself when it is one of them, and name `opm controller uninstall` with its `--remove-finalizers` choice and that choice's consequence (the named instances' workloads are orphaned). Deleting the controller while it still owes cleanup leaves every armed instance unable to finish deletion until a controller runs again. When the cluster-wide list fails, the command SHALL fail closed with the exit code `opm` maps that Kubernetes error to (4 for a permission or authentication denial) and delete nothing.

`opm instance delete` SHALL offer no flag that removes finalizers and SHALL NOT write `spec.owner`. `--force` SHALL keep its meaning, skipping the confirmation prompt, and SHALL NOT bypass either refusal. Every other instance's delete SHALL be unchanged by this requirement.

#### Scenario: Refused while an instance waits on the controller's cleanup

- **WHEN** `opm instance delete opm-controller -n opm-controller-system --force` runs for a CLI-owned record while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object, and the record and every object its inventory lists SHALL still exist
- **AND** the error SHALL name `default/hello` and `opm controller uninstall --remove-finalizers` and state that this choice orphans its workloads

#### Scenario: Controller-owned instance of the controller is refused

- **WHEN** the record `opm-controller` in `opm-controller-system` is controller-owned, with `spec.prune: true`, and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **AND** `opm instance delete opm-controller -n opm-controller-system --force` runs
- **THEN** the command SHALL exit 2 and the `ModuleInstance` `opm-controller` SHALL still exist
- **AND** the error SHALL name the matched signal and say to set `spec.owner` to `cli`, and SHALL NOT report that any resource was pruned

#### Scenario: Dry run reports the same refusal

- **WHEN** `opm instance delete opm-controller -n opm-controller-system --dry-run` runs for a CLI-owned record while an instance carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 with the same refusal as the real run, and nothing SHALL be deleted

#### Scenario: The target itself carries the finalizer

- **WHEN** `opm instance delete opm-controller -n opm-controller-system --force` runs for a CLI-owned record while the `ModuleInstance` `opm-controller` itself carries `opmodel.dev/cleanup` and no other instance does
- **THEN** the command SHALL exit 2, naming `opm-controller-system/opm-controller` as the instance being deleted

#### Scenario: The controller module under another name is guarded

- **WHEN** `opm instance delete ops -n platform --force` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_controller@v0` while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: A record holding the controller's CRDs is guarded

- **WHEN** `opm instance delete crds -n platform --force` runs for a CLI-owned record of another name and module path whose inventory holds the `CustomResourceDefinition` `moduleinstances.opmodel.dev`, while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: Look-alike names are not the controller

- **WHEN** `opm instance delete dash -n default --force` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_controller_dashboard@v0` and whose inventory holds only the `CustomResourceDefinition` `widgets.example.opmodel.dev.io`, while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

#### Scenario: No armed instance, no refusal

- **WHEN** `opm instance delete opm-controller -n opm-controller-system --force` runs for a CLI-owned record and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **THEN** the guard SHALL pass and the delete SHALL proceed as for any CLI-owned instance

#### Scenario: List failure fails closed

- **WHEN** the user may not list `moduleinstances` cluster-wide and runs `opm instance delete opm-controller -n opm-controller-system --force` for a CLI-owned record
- **THEN** the command SHALL exit 4 without deleting anything

#### Scenario: Other instances are not guarded

- **WHEN** `opm instance delete hello -n default --force` runs for a CLI-owned instance of another module while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

### Requirement: Instance delete keeps the ModuleInstance record when a tracked resource is unreadable

When `opm instance delete` deletes a CLI-owned instance and a tracked resource could not be read during discovery (the read failed with an error other than NotFound), that resource SHALL count as a failure for that resource, with the same outcome as a failed re-read: it SHALL be listed with its read error, the other tracked resources SHALL still be processed, the `ModuleInstance` record SHALL NOT be deleted, and the command SHALL exit 1. On a real run the output SHALL say that the `ModuleInstance` was kept and that re-running is safe. A dry run SHALL count the resource among those it could not check. A tracked `Namespace` in the core group or `CustomResourceDefinition` in `apiextensions.k8s.io` that could not be read SHALL instead be listed as `left behind`, as it would be if read, and SHALL NOT count as a failure. The controller-owned delete path SHALL NOT be affected.

#### Scenario: Discovery read error keeps the ModuleInstance

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a Deployment and a ConfigMap
- **AND** reading the ConfigMap during discovery fails with Forbidden
- **THEN** the Deployment SHALL be deleted
- **AND** the ConfigMap SHALL be listed with the Forbidden error
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the output SHALL say that the ModuleInstance was kept and that re-running is safe
- **AND** the command SHALL exit 1

#### Scenario: Dry run counts an unreadable resource as not checked

- **WHEN** running `opm instance delete --dry-run` and a tracked resource cannot be read during discovery
- **THEN** nothing SHALL be deleted
- **AND** the command SHALL report that 1 resource could not be checked
- **AND** SHALL exit 1

#### Scenario: Unreadable Namespace is left behind, not a failure

- **WHEN** running `opm instance delete` and reading a tracked Namespace during discovery fails with Forbidden
- **AND** every other tracked resource is read and deleted
- **THEN** the Namespace SHALL be listed as `left behind`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: Controller-owned delete ignores discovery read errors

- **WHEN** running `opm instance delete` for a controller-owned instance and a tracked resource cannot be read
- **THEN** the command SHALL delete the `ModuleInstance` and wait for the controller as before
