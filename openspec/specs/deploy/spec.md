# CLI Deploy Commands

## Purpose

The `opm mod apply` and `opm instance delete` commands manage the deployment lifecycle of OPM modules on Kubernetes clusters. `mod apply` renders the module via the Pipeline interface and applies resources using server-side apply. `instance delete` discovers and removes module resources from the persisted instance inventory record without requiring the original source.

## Design Rationale

1. **Go API integration**: Commands call `build.NewPipeline().Render()` directly, not subprocess.
2. **Inventory-first discovery**: `instance delete` discovers resources via the persisted instance inventory record instead of re-rendering.
3. **Server-side apply**: Use SSA with force for idempotent operations.
4. **Weighted ordering**: Resources applied/deleted in weight order for dependency handling.

## Dependencies

- **render-pipeline-v1**: Consumes Pipeline interface, RenderResult, Resource types
- **build-v1**: Uses Pipeline implementation

---

## User Stories

### User Story 1 - Deploy Module to Kubernetes (Priority: P1)

A developer wants to deploy their rendered module to a Kubernetes cluster.

**Independent Test**: Given a valid module, `opm mod apply` deploys resources successfully.

**Acceptance Scenarios**:

1. **Given** a valid module, **When** running `opm mod apply`, **Then** resources are deployed.
2. **Given** a deployed module, **When** running `opm instance delete`, **Then** its tracked resources are removed, and any CRD or Namespace it rendered is left behind and listed.
3. **Given** a module with CRDs and CRs, **When** running `opm mod apply`, **Then** CRDs are created first.
4. **Given** pending changes, **When** running `opm mod apply`, **Then** changes are applied.
5. **Given** dry-run request, **When** running `opm mod apply --dry-run`, **Then** no changes are made.

### User Story 2 - Delete Module Without Source (Priority: P2)

A developer wants to delete a deployed module after deleting the source files.

**Independent Test**: Deploy module, delete source, `opm instance delete` still works.

**Acceptance Scenarios**:

1. **Given** a deployed module, **When** source is deleted and `opm instance delete <name> -n <ns>` runs, **Then** its tracked resources are removed, and any CRD or Namespace it rendered is left behind and listed.
2. **Given** delete request, **When** running `opm instance delete --dry-run`, **Then** resources to delete are listed but not removed.

---

## Functional Requirements

### mod apply

| ID | Requirement |
|----|-------------|
| FR-D-001 | `mod apply` MUST call Pipeline.Render() to get resources. |
| FR-D-002 | `mod apply` MUST use server-side apply with force for conflicts. |
| FR-D-003 | `mod apply` MUST apply resources in ascending weight order. |
| FR-D-004 | `mod apply` MUST add OPM labels to all resources. |
| FR-D-005 | `mod apply` MUST support `--dry-run` (server-side). |
| FR-D-006 | `mod apply` MUST support `--wait` with timeout. |
| FR-D-007 | `mod apply` MUST support `--values` / `-f` (repeatable). |
| FR-D-008 | `mod apply` MUST support `--namespace` / `-n`. |
| FR-D-009 | `mod apply` MUST log warning on field conflicts. |
| FR-D-010 | `mod apply` MUST fail fast if render has errors AND no resources were partially rendered. When the apply flow includes inventory write and pruning, render errors SHALL prevent the entire flow (apply + prune + inventory write) from executing. |

### mod delete

| ID | Requirement |
|----|-------------|
| FR-D-020 | `instance delete` MUST discover resources via the persisted instance inventory record (the `ModuleInstance` CR); when none exists it MUST exit 5 (not found) rather than fall back to labels. |
| FR-D-021 | `instance delete` MUST NOT require module source. |
| FR-D-022 | `instance delete` MUST delete in descending weight order. |
| FR-D-023 | `instance delete` MUST support `--force` to skip confirmation. |
| FR-D-024 | `instance delete` MUST support `--dry-run` to preview. |
| FR-D-025 | `instance delete` MUST take the instance as a positional `<file|name|uuid>` argument. The namespace comes from `--namespace` / `-n`, the instance file, or the configured default. |
| FR-D-026 | `instance delete` MUST prompt for confirmation (unless --force). |
| FR-D-027 | `instance delete` MUST accept an instance UUID as the positional argument, resolved by matching `status.instanceUUID` across the namespace's `ModuleInstance` CRs. |
| FR-D-028 | `instance delete` MUST use ownership-inventory-based enumeration from the persisted instance inventory record; there is no label-based enumeration path. |

### Kubernetes Integration

| ID | Requirement |
|----|-------------|
| FR-D-050 | MUST use client-go with default rate limiting. |
| FR-D-051 | MUST support kubeconfig from flag, env, or default path. |
| FR-D-052 | MUST support context selection via flag. |
| FR-D-053 | MUST fail fast with clear error on connectivity issues. |

### Resource Labeling

| ID | Requirement |
|----|-------------|
| FR-D-060 | All resources MUST have `app.kubernetes.io/managed-by: open-platform-model`. |
| FR-D-061 | All resources MUST have `module.opmodel.dev/name: <name>`. |
| FR-D-062 | ~~All resources MUST have `module.opmodel.dev/namespace: <ns>`.~~ Removed — namespace scoping is handled by K8s API calls, not labels. |
| FR-D-063 | All resources MUST have `module.opmodel.dev/version: <version>`. |
| FR-D-064 | All resources MUST have `component.opmodel.dev/name: <component>`. |
| FR-D-065 | All resources MUST have `module-instance.opmodel.dev/uuid: <instance-uuid>` when the instance identity is available. |
| FR-D-066 | All resources MUST have `module.opmodel.dev/uuid: <module-uuid>` when the module identity is available. |

---

## Requirements

### Requirement: mod apply uses ownership inventory for pruning

`opm mod apply` SHALL use the current ownership inventory to compute stale resources (previous owned set minus current rendered set). It SHALL NOT require inventory change-history fields to perform stale-set computation or pruning.

#### Scenario: Apply computes stale set from ownership inventory

- **WHEN** a previous ownership inventory tracks resources `A`, `B`, and `C`
- **AND** the current render contains `A` and `B`
- **THEN** `C` SHALL be considered stale and eligible for pruning

### Requirement: mod apply writes current instance inventory record after successful apply

After all resources are successfully applied, `opm mod apply` SHALL persist the current instance inventory record for the instance. The persisted form SHALL store top-level `createdBy`, `instanceMetadata`, `moduleMetadata`, and the current owned resource set directly instead of a history-bearing inventory shape.

#### Scenario: Successful apply persists current instance inventory record

- **WHEN** `opm mod apply` successfully applies all rendered resources
- **THEN** the persisted instance inventory record SHALL record the current owned resource entries for that instance
- **AND** the record SHALL preserve top-level `createdBy`, `instanceMetadata`, and `moduleMetadata`
- **AND** the ownership inventory in that record SHALL be sufficient for later prune and resource enumeration

#### Scenario: Failed apply does not write inventory

- **WHEN** `opm mod apply` fails to apply one or more resources
- **THEN** the persisted instance inventory record SHALL NOT be written or updated

### Requirement: mod apply stores deployed module version in module metadata

When `opm mod apply` persists the instance inventory record, it SHALL store the deployed module version in `moduleMetadata.version` rather than relying on inventory change history.

#### Scenario: Apply persists deployed module version in module metadata

- **WHEN** `opm mod apply` persists a instance inventory record for a versioned module
- **THEN** the persisted record SHALL contain that deployed module version under `moduleMetadata.version`

### Requirement: mod apply supports --no-prune flag

The `--no-prune` flag SHALL skip the stale resource pruning step. Default is `false` (pruning enabled).

#### Scenario: --no-prune skips pruning

- **WHEN** running `opm mod apply --no-prune`
- **THEN** stale resources SHALL NOT be deleted
- **AND** the persisted instance inventory record SHALL still be written

### Requirement: mod apply supports --force for empty render

The `--force` flag SHALL allow `opm mod apply` to proceed when the render produces zero resources and a previous ownership inventory exists. Without `--force`, this situation SHALL fail with an error.

#### Scenario: Empty render blocked without --force

- **WHEN** running `opm mod apply` and the render produces 0 resources with a non-empty previous ownership inventory
- **THEN** the command SHALL fail with an error indicating all resources would be pruned

#### Scenario: Empty render allowed with --force

- **WHEN** running `opm mod apply --force` and the render produces 0 resources
- **THEN** all previously tracked resources SHALL be pruned

### Requirement: mod delete uses ownership inventory for resource enumeration

`opm instance delete` SHALL use the ownership inventory stored in the persisted instance inventory record to enumerate resources for deletion when it exists. If an inventory exists, only resources tracked in that ownership inventory SHALL be deleted (no label-scan), and the `ModuleInstance` record itself SHALL be deleted last. If no record exists, the command SHALL exit 5 (not found).

#### Scenario: Delete with ownership inventory

- **WHEN** running `opm instance delete` and a persisted instance inventory record exists
- **THEN** only resources listed in that record's ownership inventory SHALL be deleted
- **AND** the `ModuleInstance` record SHALL be deleted after all tracked resources it deletes; resources left behind SHALL NOT block it

#### Scenario: Delete without inventory

- **WHEN** running `opm instance delete` and no `ModuleInstance` record exists for the instance
- **THEN** the command SHALL exit with code 5 and report the instance as not found

#### Scenario: Delete does not remove derived resources

- **WHEN** running `opm instance delete` with ownership inventory
- **AND** derived resources (e.g., Endpoints) exist with OPM labels but are not in the inventory
- **THEN** the derived resources SHALL NOT be deleted

### Requirement: CLI mutating workflows refuse controller-managed instances

Before mutating an existing instance, CLI workflows that apply or delete a instance SHALL inspect inventory ownership. If the inventory indicates `createdBy: "controller"`, the CLI MUST refuse the mutation.

#### Scenario: Apply blocked for controller-managed instance

- **WHEN** the user runs `opm mod apply` for a instance whose inventory records `createdBy: "controller"`
- **THEN** the command SHALL fail before mutating cluster resources
- **AND** the error SHALL state that the instance is controller-managed and cannot be changed by the CLI

#### Scenario: Delete blocked for controller-managed instance

- **WHEN** the user runs `opm instance delete` for a instance whose inventory records `createdBy: "controller"`
- **THEN** the command SHALL fail before deleting tracked resources
- **AND** the error SHALL state that the instance is controller-managed and cannot be deleted by the CLI

### Requirement: Legacy CLI-managed instances remain mutable by the CLI

If an existing inventory has `createdBy: "cli"` or does not contain `createdBy`, CLI mutating workflows SHALL continue to operate normally.

#### Scenario: Apply allowed for legacy inventory

- **WHEN** the user runs `opm mod apply` for a instance whose inventory has no `createdBy`
- **THEN** the CLI SHALL treat the instance as CLI-managed and proceed normally

---

### Requirement: Instance delete never deletes CRDs or Namespaces

`opm instance delete` of a CLI-owned instance SHALL NOT delete a `Namespace` in the core API group or a `CustomResourceDefinition` in the `apiextensions.k8s.io` group, even when the instance's inventory tracks it. There SHALL be no flag that overrides this. Each such resource SHALL be listed as left behind, naming its kind, namespace and name with the status `left behind` and the reason, in the dry run and in the real run. A left-behind resource SHALL NOT count as a failure: the remaining tracked resources SHALL be deleted, the `ModuleInstance` record SHALL be deleted last, and the command SHALL exit 0. The closing line SHALL state how many resources were left behind and how to remove them.

#### Scenario: Namespace tracked by the instance is left behind

- **WHEN** running `opm instance delete` for a CLI-owned instance whose inventory tracks a Deployment and the Namespace it runs in
- **THEN** the Deployment SHALL be deleted
- **AND** the Namespace SHALL NOT be deleted
- **AND** the output SHALL list the Namespace as `left behind`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: CRD tracked by the instance is left behind

- **WHEN** running `opm instance delete` for a CLI-owned instance whose inventory tracks a `CustomResourceDefinition`
- **THEN** the CRD SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind`

#### Scenario: Dry run reports left-behind resources

- **WHEN** running `opm instance delete --dry-run` for an instance whose inventory tracks a Namespace
- **THEN** the output SHALL list the Namespace as `left behind`
- **AND** the dry-run summary SHALL count it separately from the resources that would be deleted

### Requirement: Instance delete re-checks live ownership before each delete

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again and SHALL delete it only when its `app.kubernetes.io/managed-by` label carries an OPM value and its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID. As in the operator's prune, an object with no UUID label SHALL be judged on the managed-by label alone, and so SHALL every object when the instance has no recorded UUID. A resource that fails the check SHALL be left behind and listed with the reason, and SHALL NOT count as a failure. A resource that is already gone, whether its re-read or its delete call returns NotFound, SHALL count neither as deleted nor as a failure. Any other read error SHALL count as a failure for that resource, so the `ModuleInstance` record is kept and a re-run retries.

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
      --timeout duration    Wait for the operator to finish deleting an operator-owned instance
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
