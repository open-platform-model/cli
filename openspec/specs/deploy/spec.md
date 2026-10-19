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
| FR-D-023 | `instance delete` MUST support `--yes` / `-y` to skip confirmation; `--force` stays as its deprecated alias (capability `flag-conventions`). |
| FR-D-024 | `instance delete` MUST support `--dry-run` to preview. |
| FR-D-025 | `instance delete` MUST take the instance as a positional `<file|name|uuid>` argument. The namespace comes from `--namespace` / `-n`, the instance file, or the configured default. |
| FR-D-026 | `instance delete` MUST prompt for confirmation (unless `--yes`). |
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

### Requirement: Instance delete re-checks live ownership before each delete

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again and SHALL take the decision to delete it from the delete verdict the CLI shares with the operator and with its own prune, asked with the instance identity stored in the record. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Resource no longer managed by OPM is left behind

- **WHEN** a tracked resource's live `app.kubernetes.io/managed-by` label is missing or not an OPM value
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that OPM does not manage it

#### Scenario: Resource owned by another instance is left behind

- **WHEN** a tracked resource's live `module-instance.opmodel.dev/uuid` label differs from the instance's recorded UUID
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that it belongs to another instance

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

#### Scenario: Resource adopted by another instance is left behind

- **WHEN** a tracked resource carries an OPM managed-by value and the instance's UUID
- **AND** its live `opmodel.dev/adopt` annotation names another instance
- **THEN** the resource SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind` with the reason that another instance is adopting it
- **AND** the closing line SHALL say that one resource was left behind

#### Scenario: Delete carries a UID precondition

- **WHEN** a tracked resource passes the ownership check
- **THEN** the delete request SHALL carry a precondition on the UID of the object that was read

#### Scenario: Object replaced between its re-read and its delete

- **WHEN** the API server refuses a delete because the UID precondition does not match
- **THEN** the resource SHALL be reported as failed and SHALL NOT be counted as deleted
- **AND** the `ModuleInstance` record SHALL NOT be deleted

### Requirement: Instance delete follows the shared deletion plan

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator, built from the resources of the instance's `ModuleInstance` record that the command found in the cluster or could not read, and from the instance identity stored in the record. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Resources are deleted in the plan's order

- **WHEN** running `opm instance delete` for a CLI-owned instance whose record lists a ConfigMap, a Service and a Deployment
- **THEN** the delete requests SHALL be sent in descending kind-weight order: the Deployment, then the Service, then the ConfigMap

#### Scenario: Delete carries what the plan names

- **WHEN** the plan names the delete of a tracked Deployment
- **THEN** the delete request SHALL carry Foreground propagation and a precondition on the UID of the object that was read

#### Scenario: Failed resource holds the record

- **WHEN** the delete of one tracked resource fails with Forbidden and every other resource is deleted
- **THEN** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the command SHALL exit 1

#### Scenario: Left-behind resources do not hold the record

- **WHEN** the plan leaves a tracked Namespace and a resource owned by another instance in place, and deletes every other resource
- **THEN** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

#### Scenario: Kept claim is outside the plan

- **WHEN** the record lists a PersistentVolumeClaim and `--delete-data` is not set
- **THEN** the claim SHALL NOT be read for a delete verdict and SHALL NOT be deleted
- **AND** it SHALL be listed as `kept`
- **AND** the `ModuleInstance` record SHALL be deleted when no other resource failed

#### Scenario: Resource that appears after the instance was read

- **WHEN** a recorded ConfigMap does not exist when the command reads the instance
- **AND** it exists again before the deletes are sent
- **THEN** the ConfigMap SHALL NOT be deleted
- **AND** the `ModuleInstance` record SHALL be deleted when no planned resource failed

#### Scenario: Dry run follows the plan without deleting

- **WHEN** running `opm instance delete --dry-run`
- **THEN** each tracked resource SHALL be read and judged as on a real run
- **AND** no delete request SHALL be sent
- **AND** the `ModuleInstance` record SHALL still exist

### Requirement: Instance delete never deletes a Namespace or a CustomResourceDefinition

`opm instance delete` of a CLI-owned instance SHALL NOT delete a `Namespace` in the core API group or a `CustomResourceDefinition` in the `apiextensions.k8s.io` group, even when the instance's inventory tracks it. There SHALL be no flag that overrides this.

#### Scenario: No flag overrides the protection

- **WHEN** `opm instance delete` of a CLI-owned instance meets a tracked `Namespace` in the core API group or `CustomResourceDefinition` in the `apiextensions.k8s.io` group
- **THEN** it SHALL NOT delete it, and there SHALL be no flag that overrides this

### Requirement: Instance delete lists a protected resource as left behind

`opm instance delete` of a CLI-owned instance SHALL NOT delete a `Namespace` in the core API group or a `CustomResourceDefinition` in the `apiextensions.k8s.io` group, even when the instance's inventory tracks it. Each such resource SHALL be listed as left behind, naming its kind, namespace and name with the status `left behind` and the reason, in the dry run and in the real run.

#### Scenario: CRD tracked by the instance is left behind

- **WHEN** running `opm instance delete` for a CLI-owned instance whose inventory tracks a `CustomResourceDefinition`
- **THEN** the CRD SHALL NOT be deleted
- **AND** the output SHALL list it as `left behind`

#### Scenario: Dry run reports left-behind resources

- **WHEN** running `opm instance delete --dry-run` for an instance whose inventory tracks a Namespace
- **THEN** the output SHALL list the Namespace as `left behind`
- **AND** the dry-run summary SHALL count it separately from the resources that would be deleted

### Requirement: A left-behind resource does not fail the delete

`opm instance delete` of a CLI-owned instance SHALL NOT delete a `Namespace` in the core API group or a `CustomResourceDefinition` in the `apiextensions.k8s.io` group, even when the instance's inventory tracks it. A left-behind resource SHALL NOT count as a failure: the remaining tracked resources SHALL be deleted, the `ModuleInstance` record SHALL be deleted last, and the command SHALL exit 0. The closing line SHALL state how many resources were left behind and how to remove them.

#### Scenario: Namespace tracked by the instance is left behind

- **WHEN** running `opm instance delete` for a CLI-owned instance whose inventory tracks a Deployment and the Namespace it runs in
- **THEN** the Deployment SHALL be deleted
- **AND** the Namespace SHALL NOT be deleted
- **AND** the output SHALL list the Namespace as `left behind`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

### Requirement: A tracked resource is deleted only when its live labels and annotation show the instance owns it

`opm instance delete` SHALL delete each tracked resource only when the live `app.kubernetes.io/managed-by` label carries an OPM value, its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID, and its `opmodel.dev/adopt` annotation does not name another instance. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: Ownership conditions for the delete

- **WHEN** `opm instance delete` re-reads a tracked resource
- **THEN** it SHALL delete the resource only when the live `app.kubernetes.io/managed-by` label carries an OPM value, its `module-instance.opmodel.dev/uuid` label matches the instance's recorded UUID, and its `opmodel.dev/adopt` annotation does not name another instance

### Requirement: An object with no UUID label is judged without the UUID comparison

Before deleting each tracked resource, `opm instance delete` SHALL delete the resource only as listed in requirement "A tracked resource is deleted only when its live labels and annotation show the instance owns it". An object with no UUID label SHALL be judged without the UUID comparison, and so SHALL every object when the instance has no recorded UUID. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: No UUID comparison without a UUID

- **WHEN** an object has no UUID label, or the instance has no recorded UUID
- **THEN** the object SHALL be judged without the UUID comparison

### Requirement: A resource that fails the ownership check is left behind and does not fail the delete

Before deleting each tracked resource, `opm instance delete` SHALL delete the resource only as listed in requirement "A tracked resource is deleted only when its live labels and annotation show the instance owns it". A resource that fails the check SHALL be left behind and listed with the reason, and SHALL NOT count as a failure. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A resource that fails the check is listed with the reason

- **WHEN** a tracked resource fails the check
- **THEN** it SHALL be left behind and listed with the reason, and SHALL NOT count as a failure

### Requirement: The closing line of a delete gives the number of resources left behind

Before deleting each tracked resource, `opm instance delete` SHALL delete the resource only as listed in requirement "A tracked resource is deleted only when its live labels and annotation show the instance owns it". The closing line SHALL give the number of resources left behind. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: The closing line gives the number of resources left behind

- **WHEN** a tracked resource fails the check
- **THEN** the closing line SHALL give the number of resources left behind

### Requirement: Each delete carries a precondition on the UID of the object that was read

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again. Each delete SHALL carry a precondition on the UID of the object that was read; a delete the API server refuses on that precondition SHALL count as a failure for that resource and SHALL NOT be reported as deleted. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A refused precondition is a failure

- **WHEN** the API server refuses a delete on the UID precondition
- **THEN** the delete SHALL count as a failure for that resource and SHALL NOT be reported as deleted

### Requirement: A resource that is already gone is no failure and any other read error is

Before deleting each tracked resource, `opm instance delete` SHALL read the live object again. A resource that is already gone, whether its re-read or its delete call returns NotFound, SHALL count neither as deleted nor as a failure. Any other read error SHALL count as a failure for that resource, so the `ModuleInstance` record is kept and a re-run retries. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Already gone counts neither as deleted nor as a failure

- **WHEN** a resource is already gone, whether its re-read or its delete call returns NotFound
- **THEN** it SHALL count neither as deleted nor as a failure

### Requirement: The apply goes in two stages in ascending kind-class weight

`opm instance apply` and `opm module apply` SHALL apply the rendered resources in two stages, each in ascending order of the library's kind-class weight (`opm/k8s/object.Weight`) with build order breaking ties (enhancement 0012, "Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: The second stage follows weight order

- **WHEN** the second stage holds a Deployment (weight 100), a Service (weight 50) and a ConfigMap (weight 15) in that build order
- **THEN** they are applied ConfigMap, Service, Deployment

### Requirement: The first stage holds every CustomResourceDefinition and every Namespace

`opm instance apply` and `opm module apply` SHALL apply the rendered resources in two stages. The first stage SHALL hold every `CustomResourceDefinition` (group `apiextensions.k8s.io`) and every `Namespace` (core group); the second stage SHALL hold everything else.

#### Scenario: A Namespace is applied before the objects in it

- **WHEN** a module renders a ConfigMap in namespace `demo` before the `Namespace` `demo`, and the namespace does not exist
- **THEN** the Namespace is applied before the ConfigMap and both succeed on the first run

### Requirement: After the first stage the apply waits until the CustomResourceDefinitions are established

Outside a dry run, after the first stage the command (`opm instance apply` and `opm module apply`) SHALL wait until every CustomResourceDefinition the first stage applied without error reports the condition `Established=True`, bounded by `--timeout` (default 5m) counted from the start of the apply, with a timeout reporting the time elapsed since then, and only then apply the second stage.

#### Scenario: A CRD and its custom resource apply in one run

- **WHEN** a module renders a custom resource before the CustomResourceDefinition that defines its kind, and `opm instance apply` runs against a cluster that has neither
- **THEN** the CustomResourceDefinition is applied first, the command waits until it is established, the custom resource is applied after it, and the command succeeds on its first run

### Requirement: A wait that does not complete fails the apply

Outside a dry run, after the first stage the command (`opm instance apply` and `opm module apply`) SHALL wait until every CustomResourceDefinition the first stage applied without error reports the condition `Established=True`. When the wait does not complete, the command SHALL fail naming the CustomResourceDefinitions still pending, SHALL NOT apply the second stage, and SHALL NOT prune or write the inventory.

#### Scenario: A CRD that never becomes established fails the apply

- **WHEN** a CustomResourceDefinition of the first stage does not report `Established=True` within `--timeout`
- **THEN** the command exits non-zero naming that CustomResourceDefinition, applies nothing of the second stage, and neither prunes nor writes the inventory

### Requirement: A resource that fails to apply does not stop the rest

`opm instance apply` and `opm module apply` SHALL apply the rendered resources in two stages. A resource that fails to apply SHALL be reported and SHALL NOT stop the remaining resources of its stage or the next stage.

#### Scenario: A failed resource is reported and the stages go on

- **WHEN** a resource fails to apply
- **THEN** it SHALL be reported and SHALL NOT stop the remaining resources of its stage or the next stage

### Requirement: A dry run skips a custom resource whose CRD the same apply creates and warns

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a custom resource whose group and kind are defined by a CustomResourceDefinition of the same apply that does not yet exist on the cluster. The command SHALL log a warning naming the skipped resource and its CustomResourceDefinition, SHALL count it as skipped in the dry-run summary, and SHALL NOT treat it as an error.

#### Scenario: A dry run with a new CRD skips its custom resource

- **WHEN** `opm instance apply --dry-run` runs for a module rendering CustomResourceDefinition `foos.example.com` and one `Foo`, and the cluster has no such CustomResourceDefinition
- **THEN** the `Foo` is not sent, a warning names `Foo` and `foos.example.com`, the summary reports one resource skipped, and the command exits 0

### Requirement: A dry run sends a custom resource whose CRD already exists

On `--dry-run`, `opm instance apply` and `opm module apply`, a custom resource whose CustomResourceDefinition already exists on the cluster SHALL be sent as usual.

#### Scenario: A dry run with an existing CRD sends its custom resource

- **WHEN** the same dry run runs against a cluster where `foos.example.com` already exists
- **THEN** the `Foo` is sent to the server-side dry run and nothing is skipped

### Requirement: A dry run does not wait for any CustomResourceDefinition

On `--dry-run`, `opm instance apply` and `opm module apply`, a dry run SHALL NOT wait for any CustomResourceDefinition.

#### Scenario: A dry run does not wait

- **WHEN** a dry run applies a CustomResourceDefinition
- **THEN** it SHALL NOT wait for any CustomResourceDefinition

### Requirement: A custom resource in a namespace the same apply creates is warned about once

On `--dry-run`, `opm instance apply` and `opm module apply`, a custom resource that is also in a namespace the same apply creates SHALL be warned about once, for its CustomResourceDefinition.

#### Scenario: One warning for a custom resource in a new namespace

- **WHEN** a dry run meets a custom resource that is also in a namespace the same apply creates
- **THEN** it SHALL be warned about once, for its CustomResourceDefinition

### Requirement: An instance deploys the operator when one of three signals holds

An instance deploys the operator when any one of these SHALL hold, read from its record without rendering: it is named `opm-operator` in the namespace `opm-operator-system`; its module path, with any `@<major>` suffix removed, is `opmodel.dev/modules/opm_operator`; or its recorded inventory holds a `CustomResourceDefinition` (group `apiextensions.k8s.io`) whose name, after its first `.`, is exactly `opmodel.dev`.

#### Scenario: The operator module under another name is guarded

- **WHEN** `opm instance delete ops -n platform --yes` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_operator@v0` while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: A record holding the operator's CRDs is guarded

- **WHEN** `opm instance delete crds -n platform --yes` runs for a CLI-owned record of another name and module path whose inventory holds the `CustomResourceDefinition` `moduleinstances.opmodel.dev`, while `default/hello` carries `opmodel.dev/cleanup`
- **THEN** the command SHALL exit 2 without deleting any object

#### Scenario: Look-alike names are not the operator

- **WHEN** `opm instance delete dash -n default --yes` runs for a CLI-owned record whose module path is `opmodel.dev/modules/opm_operator_dashboard@v0` and whose inventory holds only the `CustomResourceDefinition` `widgets.example.opmodel.dev.io`, while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

### Requirement: Instance delete does not delete an instance that deploys the operator

`opm instance delete` SHALL NOT delete an instance that deploys the operator while that instance is operator-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer. These, as listed in requirement "An instance deploys the operator when one of three signals holds", are the same signals by which the operator recognises the instance that deploys it, so the CLI and the operator agree on which instance that is.

#### Scenario: The guard names the operator's instance

- **WHEN** `opm instance delete` targets an instance that deploys the operator
- **THEN** it SHALL NOT delete it while that instance is operator-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer

### Requirement: The guard runs before any object is deleted, on both branches and on a dry run

`opm instance delete` SHALL NOT delete an instance that deploys the operator while that instance is operator-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer. The guard SHALL run after the target instance is resolved and before any object is deleted, on the CLI-owned and the operator-owned branch alike, and on a dry run with the same outcome as the real run.

#### Scenario: Dry run reports the same refusal

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --dry-run` runs for a CLI-owned record while an instance carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 with the same refusal as the real run, and nothing SHALL be deleted

### Requirement: An operator-owned target is refused with exit code 2

For an instance that deploys the operator, when the target is operator-owned, the command (`opm instance delete`) SHALL refuse with exit code 2 and delete and patch nothing, naming the signal that matched and the remedy of setting `spec.owner` to `cli`. The operator never reconciles, finalizes or prunes the instance that deploys it, so the operator-owned delete would wait on, and report, a cleanup that does not happen.

#### Scenario: Operator-owned instance of the operator is refused

- **WHEN** the record `opm-operator` in `opm-operator-system` is operator-owned, with `spec.prune: true`, and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **AND** `opm instance delete opm-operator -n opm-operator-system --yes` runs
- **THEN** the command SHALL exit 2 and the `ModuleInstance` `opm-operator` SHALL still exist
- **AND** the error SHALL name the matched signal and say to set `spec.owner` to `cli`, and SHALL NOT report that any resource was pruned

### Requirement: The guard lists ModuleInstance resources cluster-wide when the target is not operator-owned

For an instance that deploys the operator that is not operator-owned, the command (`opm instance delete`) SHALL list `ModuleInstance` resources cluster-wide. Deleting the operator while it still owes cleanup leaves every armed instance unable to finish deletion until an operator runs again.

#### Scenario: The guard lists every ModuleInstance

- **WHEN** `opm instance delete` targets an instance that deploys the operator that is not operator-owned
- **THEN** the command SHALL list `ModuleInstance` resources cluster-wide

### Requirement: An armed instance blocks the delete with exit code 2

For an instance that deploys the operator that is not operator-owned, when any `ModuleInstance` in the cluster carries `opmodel.dev/cleanup`, the command SHALL refuse with exit code 2, delete and patch nothing, and name each such instance as `namespace/name`, marking the target itself when it is one of them, and name `opm operator uninstall` with its `--remove-finalizers` choice and that choice's consequence (the named instances' workloads are orphaned).

#### Scenario: Refused while an instance waits on the operator's cleanup

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --yes` runs for a CLI-owned record while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object, and the record and every object its inventory lists SHALL still exist
- **AND** the error SHALL name `default/hello` and `opm operator uninstall --remove-finalizers` and state that this choice orphans its workloads

#### Scenario: The target itself carries the finalizer

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --yes` runs for a CLI-owned record while the `ModuleInstance` `opm-operator` itself carries `opmodel.dev/cleanup` and no other instance does
- **THEN** the command SHALL exit 2, naming `opm-operator-system/opm-operator` as the instance being deleted

#### Scenario: No armed instance, no refusal

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --yes` runs for a CLI-owned record and no `ModuleInstance` carries `opmodel.dev/cleanup`
- **THEN** the guard SHALL pass and the delete SHALL proceed as for any CLI-owned instance

### Requirement: A failed cluster-wide list fails closed

For an instance that deploys the operator that is not operator-owned, the command (`opm instance delete`) SHALL list `ModuleInstance` resources cluster-wide. When the cluster-wide list fails, the command SHALL fail closed with the exit code `opm` maps that Kubernetes error to (4 for a permission or authentication denial) and delete nothing.

#### Scenario: List failure fails closed

- **WHEN** the user may not list `moduleinstances` cluster-wide and runs `opm instance delete opm-operator -n opm-operator-system --force` for a CLI-owned record
- **THEN** the command SHALL exit 4 without deleting anything

### Requirement: Instance delete offers no flag that removes finalizers

`opm instance delete` SHALL NOT delete an instance that deploys the operator while that instance is operator-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer. `opm instance delete` SHALL offer no flag that removes finalizers and SHALL NOT write `spec.owner`.

#### Scenario: No finalizer flag and no owner write

- **WHEN** `opm instance delete` targets an instance that deploys the operator
- **THEN** it SHALL offer no flag that removes finalizers and SHALL NOT write `spec.owner`

### Requirement: The yes flag and its deprecated alias do not bypass either refusal

`opm instance delete` SHALL NOT delete an instance that deploys the operator while that instance is operator-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer. `--yes` SHALL keep its one meaning, skipping the confirmation prompt, and SHALL NOT bypass either refusal; neither SHALL `--force`, its deprecated alias on this command (capability `flag-conventions`).

#### Scenario: The deprecated alias does not bypass the guard

- **WHEN** `opm instance delete opm-operator -n opm-operator-system --force` runs for a CLI-owned record while `default/hello` carries the `opmodel.dev/cleanup` finalizer
- **THEN** the command SHALL exit 2 without deleting any object, exactly as with `--yes`

### Requirement: Every other instance's delete is unchanged by the guard

`opm instance delete` SHALL NOT delete an instance that deploys the operator while that instance is operator-owned, nor while any `ModuleInstance` in the cluster carries the `opmodel.dev/cleanup` finalizer. Every other instance's delete SHALL be unchanged by this requirement.

#### Scenario: Other instances are not guarded

- **WHEN** `opm instance delete hello -n default --yes` runs for a CLI-owned instance of another module while another instance carries `opmodel.dev/cleanup`
- **THEN** the delete SHALL proceed without the guard

### Requirement: An unreadable tracked resource counts as a failure and keeps the ModuleInstance

When `opm instance delete` deletes a CLI-owned instance and a tracked resource could not be read during discovery (the read failed with an error other than NotFound), that resource SHALL count as a failure for that resource, with the same outcome as a failed re-read: it SHALL be listed with its read error, the other tracked resources SHALL still be processed, the `ModuleInstance` record SHALL NOT be deleted, and the command SHALL exit 1.

#### Scenario: Discovery read error keeps the ModuleInstance

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a Deployment and a ConfigMap
- **AND** reading the ConfigMap during discovery fails with Forbidden
- **THEN** the Deployment SHALL be deleted
- **AND** the ConfigMap SHALL be listed with the Forbidden error
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the output SHALL say that the ModuleInstance was kept and that re-running is safe
- **AND** the command SHALL exit 1

### Requirement: The output says the ModuleInstance was kept and a dry run counts the resource as not checked

When `opm instance delete` deletes a CLI-owned instance and a tracked resource could not be read during discovery (the read failed with an error other than NotFound), on a real run the output SHALL say that the `ModuleInstance` was kept and that re-running is safe. A dry run SHALL count the resource among those it could not check.

#### Scenario: Dry run counts an unreadable resource as not checked

- **WHEN** running `opm instance delete --dry-run` and a tracked resource cannot be read during discovery
- **THEN** nothing SHALL be deleted
- **AND** the command SHALL report that 1 resource could not be checked
- **AND** SHALL exit 1

### Requirement: An unreadable Namespace or CustomResourceDefinition is left behind and is no failure

When `opm instance delete` deletes a CLI-owned instance and a tracked resource could not be read during discovery (the read failed with an error other than NotFound), a tracked `Namespace` in the core group or `CustomResourceDefinition` in `apiextensions.k8s.io` that could not be read SHALL instead be listed as `left behind`, as it would be if read, and SHALL NOT count as a failure.

#### Scenario: Unreadable Namespace is left behind, not a failure

- **WHEN** running `opm instance delete` and reading a tracked Namespace during discovery fails with Forbidden
- **AND** every other tracked resource is read and deleted
- **THEN** the Namespace SHALL be listed as `left behind`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

### Requirement: The operator-owned delete path is not affected by discovery read errors

When `opm instance delete` deletes a CLI-owned instance and a tracked resource could not be read during discovery (the read failed with an error other than NotFound), that resource SHALL count as a failure for that resource, with the same outcome as a failed re-read: it SHALL be listed with its read error, the other tracked resources SHALL still be processed, the `ModuleInstance` record SHALL NOT be deleted, and the command SHALL exit 1. The operator-owned delete path SHALL NOT be affected.

#### Scenario: Operator-owned delete ignores discovery read errors

- **WHEN** running `opm instance delete` for an operator-owned instance and a tracked resource cannot be read
- **THEN** the command SHALL delete the `ModuleInstance` and wait for the operator as before

### Requirement: A dry run skips a namespaced object whose namespace the apply would create

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a namespaced object whose namespace the apply would create: a `Namespace` (core group) the same apply renders and that does not exist on the cluster, or the instance namespace that `--create-namespace` would create. A dry run persists neither, so the server's namespace admission would refuse every object in them although the real apply would not.

#### Scenario: A dry run with a new Namespace skips the objects in it

- **WHEN** `opm instance apply --dry-run` runs for a module rendering `Namespace` `demo` and ConfigMap `cfg` in `demo`, and the cluster has no namespace `demo`
- **THEN** the Namespace is sent to the server-side dry run, `cfg` is not sent, a warning names `ConfigMap/cfg` and `demo`, the summary reports one resource skipped, no error is reported, and the command exits 0

#### Scenario: A dry run with --create-namespace skips the objects in the new instance namespace

- **WHEN** `opm instance apply --dry-run --create-namespace` runs for an instance in namespace `media`, the cluster has no namespace `media`, and the module renders no `Namespace` object
- **THEN** the command reports that namespace `media` would be created, skips every namespaced object in `media` with a warning naming `media`, counts them as skipped, and exits 0

### Requirement: A skipped object is warned about, counted and no error

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a namespaced object whose namespace the apply would create. The command SHALL log a warning naming the skipped object and its namespace and saying that a dry run cannot validate it, SHALL count it as skipped in the dry-run summary, SHALL NOT treat it as an error, and SHALL exit 0 when nothing else fails.

#### Scenario: A skipped object gets a warning and a count

- **WHEN** a dry run skips a namespaced object whose namespace the apply would create
- **THEN** the command SHALL log a warning naming the skipped object and its namespace and saying that a dry run cannot validate it, SHALL count it as skipped in the dry-run summary, SHALL NOT treat it as an error, and SHALL exit 0 when nothing else fails

### Requirement: A dry run sends every other object as usual

On `--dry-run`, `opm instance apply` and `opm module apply` SHALL NOT send a namespaced object whose namespace the apply would create. A namespaced object in a namespace that already exists, and every cluster-scoped object, SHALL be sent as usual. Without `--create-namespace`, a missing instance namespace that no rendered `Namespace` creates SHALL NOT cause a skip: the dry run reports the error the real apply would hit. Outside a dry run nothing is skipped.

#### Scenario: A dry run in an existing namespace sends everything

- **WHEN** the same dry run runs against a cluster where the namespace already exists
- **THEN** every object is sent to the server-side dry run and nothing is skipped for its namespace

#### Scenario: A real apply skips nothing

- **WHEN** `opm instance apply --create-namespace` runs without `--dry-run` for an instance whose namespace does not exist
- **THEN** the namespace is created and every object is applied

### Requirement: Create-namespace creates a missing namespace only after every check that can refuse the apply passed

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists, and SHALL create a missing namespace only after every check of the apply that can refuse has passed: the cluster gates, the status-permission check, the empty-render guard and the ownership guard (capability `apply-pruning`, "The ownership guard judges every apply"). The namespace SHALL be created before the first rendered resource is applied.

#### Scenario: An existing namespace is left alone

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace exists
- **THEN** no namespace SHALL be created and the apply SHALL proceed as without the flag

### Requirement: A check that refuses the apply leaves the cluster unchanged and creates no namespace

An apply that one of the checks listed in requirement "Create-namespace creates a missing namespace only after every check that can refuse the apply passed" refuses SHALL NOT create the namespace and SHALL leave the cluster unchanged.

#### Scenario: A refused first install creates no namespace

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **AND** the read of a rendered cluster-scoped resource fails with Forbidden
- **THEN** the command SHALL exit 4
- **AND** the namespace SHALL NOT exist afterwards
- **AND** no rendered resource SHALL be applied and no `ModuleInstance` SHALL be written

#### Scenario: An untracked cluster-scoped resource refuses before the namespace is created

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **AND** a rendered cluster-scoped resource already exists without OPM labels
- **THEN** the command SHALL fail naming that resource
- **AND** the namespace SHALL NOT exist afterwards

#### Scenario: A failed cluster gate creates no namespace

- **WHEN** `opm instance apply --create-namespace` runs against a cluster without the `ModuleInstance` CustomResourceDefinition
- **THEN** the command SHALL exit 2
- **AND** the namespace SHALL NOT exist afterwards

#### Scenario: A denied status permission creates no namespace

- **WHEN** `opm instance apply --create-namespace` runs and the caller may not patch `moduleinstances/status` in the instance namespace
- **THEN** the command SHALL exit 4
- **AND** the namespace SHALL NOT exist afterwards

### Requirement: While the namespace is missing the apply sends no read into it

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists. While the namespace is missing, the apply SHALL treat it as holding nothing and SHALL NOT send a read into it: it SHALL NOT read the `ModuleInstance` record, so the apply is a first install, and the ownership guard SHALL skip every rendered object in that namespace. Rendered cluster-scoped objects and objects in other namespaces SHALL be checked as usual.

#### Scenario: A missing namespace is a first install and is not read

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist and no check refuses
- **THEN** the command SHALL send no read of a `ModuleInstance` or of a rendered object into that namespace before it creates it
- **AND** the namespace SHALL be created before the first rendered resource is applied
- **AND** the `ModuleInstance` record SHALL be written at revision 1

#### Scenario: A caller without cluster-wide read is not refused by the missing namespace

- **WHEN** `opm instance apply --create-namespace` runs for an instance whose namespace does not exist
- **AND** the API server would answer Forbidden to the caller's read of a `ModuleInstance` or of a ConfigMap in a namespace that does not exist
- **THEN** the apply SHALL NOT fail on such a read

### Requirement: The status-permission check runs unchanged with a missing namespace

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists. The status-permission check SHALL run unchanged; it asks about the namespace by name and does not need it to exist.

#### Scenario: The status-permission check asks about the namespace by name

- **WHEN** the instance namespace is missing
- **THEN** the status-permission check SHALL run unchanged; it asks about the namespace by name and does not need it to exist

### Requirement: A failed read or create of the namespace stops the apply with the exit code of its cause

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists. A failure of the first read (whether the namespace exists) SHALL stop the apply with the exit code of its cause, before any other step. A failure of the create SHALL stop the apply with the exit code of its cause, before any rendered resource is applied.

#### Scenario: The namespace read fails

- **WHEN** `opm instance apply --create-namespace` runs and the read of the namespace fails with Forbidden
- **THEN** the command SHALL exit 4 before any other call to the cluster

#### Scenario: The namespace create fails

- **WHEN** `opm instance apply --create-namespace` runs, every check passes and the create of the namespace fails with Forbidden
- **THEN** the command SHALL exit 4
- **AND** no rendered resource SHALL be applied and no `ModuleInstance` SHALL be written

### Requirement: A namespace that exists at the create stops the apply with exit code 1

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists. When the namespace turns out to exist at the create although the first read found it missing, the apply SHALL stop with exit code 1 before any change and SHALL tell the user to run the command again, because nothing inside that namespace was checked.

#### Scenario: The namespace appears during the checks

- **WHEN** `opm instance apply --create-namespace` found the namespace missing and another actor creates it before the apply does
- **THEN** the command SHALL exit 1, say that the namespace was created by someone else, and tell the user to run the command again
- **AND** no rendered resource SHALL be applied

### Requirement: A dry run creates no namespace and reports that it would

With `--create-namespace`, `opm instance apply` and `opm module apply` SHALL read first whether the instance namespace exists. A dry run SHALL create nothing and SHALL report first that the namespace would be created.

#### Scenario: A dry run reports first that the namespace would be created

- **WHEN** `opm instance apply --dry-run --create-namespace` runs for an instance whose namespace is missing
- **THEN** the dry run SHALL create nothing and SHALL report first that the namespace would be created

### Requirement: A failed record delete prints no success line and exits non-zero

When `opm instance delete` deletes a CLI-owned instance, every tracked resource was deleted, and the delete of the `ModuleInstance` record then fails with an error other than NotFound, the command SHALL NOT print a success line and SHALL exit non-zero. The output SHALL name the record and the error, SHALL say that the tracked resources were deleted and that the record remains, and SHALL say that re-running is safe.

#### Scenario: Forbidden record delete

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a ConfigMap
- **AND** the delete of the `ModuleInstance` record fails with Forbidden
- **THEN** the ConfigMap SHALL be deleted
- **AND** the output SHALL NOT contain `Instance deleted`
- **AND** the output SHALL say that the record remains and that re-running is safe
- **AND** the command SHALL exit 4

### Requirement: A failed record delete exits by the answer of the API server

When `opm instance delete` deletes a CLI-owned instance, every tracked resource was deleted, and the delete of the `ModuleInstance` record then fails with an error other than NotFound, the exit code SHALL be 4 when the API server denied the delete (Forbidden or Unauthorized), 3 when it answered with a server timeout or service unavailable, and 1 for any other failure. A NotFound answer SHALL count as deleted.

#### Scenario: Record delete fails with a server error

- **WHEN** the delete of the `ModuleInstance` record fails with an internal server error
- **THEN** the command SHALL exit 1 without a success line

### Requirement: A re-run after the cause is fixed deletes the record

When `opm instance delete` deletes a CLI-owned instance, every tracked resource was deleted, and the delete of the `ModuleInstance` record then fails with an error other than NotFound, re-running the command after the cause is fixed SHALL delete the record and exit 0.

#### Scenario: Re-run after the cause is fixed

- **WHEN** `opm instance delete` runs again after an earlier run deleted the resources but failed to delete the record
- **THEN** it SHALL delete the record and exit 0

### Requirement: A dry run is not affected by a failed record delete

When `opm instance delete` deletes a CLI-owned instance, every tracked resource was deleted, and the delete of the `ModuleInstance` record then fails with an error other than NotFound, the command SHALL NOT print a success line and SHALL exit non-zero. A dry run deletes no record and SHALL NOT be affected.

#### Scenario: A dry run deletes no record

- **WHEN** `opm instance delete --dry-run` runs
- **THEN** it deletes no record and SHALL NOT be affected

### Requirement: Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. After the delete the claim is in no inventory, so no later `opm` command deletes it.

#### Scenario: Claim is kept by default

- **WHEN** running `opm instance delete --yes` for a CLI-owned instance whose inventory tracks a Deployment and a PersistentVolumeClaim
- **THEN** the Deployment SHALL be deleted
- **AND** the PersistentVolumeClaim SHALL NOT be deleted
- **AND** the output SHALL list the claim with the status `kept`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0
- **AND** the closing output SHALL carry `kubectl delete pvc` with the name and namespace of the claim

### Requirement: Each kept claim is listed on its own line at the informational level

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. Each kept claim SHALL be listed on its own line, naming its kind, namespace and name with the status `kept`, in the dry run and in the real run, at the informational log level and never as a warning or an error.

#### Scenario: Dry run lists the claim as kept

- **WHEN** running `opm instance delete --dry-run` for an instance whose inventory tracks a PersistentVolumeClaim
- **THEN** the output SHALL list the claim with the status `kept`
- **AND** the dry-run output SHALL say how many claims would be kept and name `--delete-data`
- **AND** nothing SHALL be deleted

### Requirement: A kept claim does not count as a failure

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. A kept claim SHALL NOT count as a failure: the other tracked resources SHALL be deleted, the `ModuleInstance` record SHALL be deleted last, and the command SHALL exit 0.

#### Scenario: Unreadable claim is kept

- **WHEN** discovery could not read a tracked PersistentVolumeClaim
- **AND** `--delete-data` is not set
- **THEN** the claim SHALL be listed as `kept` and SHALL NOT count as a failure

### Requirement: The closing output counts the kept claims and names the command that deletes them

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. The closing output SHALL state how many claims were kept, SHALL print for each kept claim the `kubectl delete pvc` command that deletes it, and SHALL name `--delete-data`.

#### Scenario: The closing output names the claims kept

- **WHEN** `opm instance delete` of a CLI-owned instance keeps a claim
- **THEN** the closing output SHALL state how many claims were kept, SHALL print for each kept claim the `kubectl delete pvc` command that deletes it, and SHALL name `--delete-data`

### Requirement: With --delete-data a tracked claim is deleted under the live-ownership check

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. With `--delete-data`, a tracked claim SHALL be deleted under the same live-ownership check as every other tracked resource.

#### Scenario: Delete-data deletes the claim

- **WHEN** running `opm instance delete --yes --delete-data` for the same instance
- **THEN** the Deployment and the PersistentVolumeClaim SHALL be deleted
- **AND** the output SHALL list no resource with the status `kept`

### Requirement: The kept-claim rule covers the core group only and no StatefulSet template claim

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. The kept-claim rule SHALL apply to the core group only: a kind of the same name in another API group is deleted like any other resource. The command SHALL NOT handle claims that a StatefulSet created from `volumeClaimTemplates`: they are not in the inventory, with or without the flag.

#### Scenario: A claim of another group is deleted like any other resource

- **WHEN** a tracked resource has the kind of the same name in another API group
- **THEN** the kept-claim rule SHALL apply to the core group only, so that resource is deleted like any other resource

### Requirement: --delete-data on an operator-managed instance warns and changes nothing

On an operator-managed instance `--delete-data` SHALL NOT change what the operator does and SHALL NOT fail the command (`opm instance delete`): the command SHALL print a warning, before it asks for confirmation, that says so, names `spec.dataPolicy` of the `ModuleInstance` as the setting that an operator with that field obeys for PersistentVolumeClaims, and says that an older operator deletes them.

#### Scenario: Operator-managed instance

- **WHEN** running `opm instance delete --delete-data` for an operator-managed instance
- **THEN** the command SHALL warn that `--delete-data` does not change what the operator does and SHALL name `spec.dataPolicy`
- **AND** the warning SHALL say that an older operator deletes PersistentVolumeClaims
- **AND** the delete SHALL go on as without the flag

### Requirement: The kept-claim rule of the CLI is not claimed for an operator-managed instance

`opm instance delete` of a CLI-owned instance SHALL NOT delete a tracked `PersistentVolumeClaim` of the core API group unless `--delete-data` is set. The kept-claim rule of the CLI SHALL NOT be claimed for an operator-managed instance: the operator decides what it removes.

#### Scenario: The operator decides what it removes

- **WHEN** the instance is operator-managed
- **THEN** the kept-claim rule of the CLI SHALL NOT be claimed for it: the operator decides what it removes

### Requirement: An operator-managed delete that may have kept claims does not claim a full prune

When the delete of an operator-managed instance with `spec.prune` set completes, its inventory tracked at least one PersistentVolumeClaim, its `spec.dataPolicy` is not `Delete` and the installed `ModuleInstance` CRD has `spec.dataPolicy` or has no schema that can be read, the closing output SHALL NOT say that every tracked resource was pruned and SHALL NOT say that the claims are still in the cluster.

#### Scenario: No claim of a full prune and no claim of claims in the cluster

- **WHEN** the delete of an operator-managed instance with `spec.prune` set completes, its inventory tracked at least one PersistentVolumeClaim, its `spec.dataPolicy` is not `Delete` and the installed `ModuleInstance` CRD has `spec.dataPolicy` or has no schema that can be read
- **THEN** the closing output SHALL NOT say that every tracked resource was pruned and SHALL NOT say that the claims are still in the cluster

### Requirement: The closing output of an operator-managed delete that may have kept claims names them

When the delete of an operator-managed instance with `spec.prune` set completes, its inventory tracked at least one PersistentVolumeClaim, its `spec.dataPolicy` is not `Delete` and the installed `ModuleInstance` CRD has `spec.dataPolicy` or has no schema that can be read, the closing output SHALL name each such claim, SHALL say that an operator with `spec.dataPolicy` keeps them and that an operator older than its CRDs deleted them, and SHALL print the `kubectl delete pvc` command for each.

#### Scenario: Operator-managed delete that may have kept claims closes without claiming either outcome

- **WHEN** `opm instance delete --yes` completes for an operator-managed instance whose `spec.prune` is true, whose `spec.dataPolicy` is absent and whose inventory tracks the PersistentVolumeClaim `data` in namespace `apps`
- **AND** the installed `ModuleInstance` CRD has `spec.dataPolicy`
- **THEN** the closing output SHALL NOT say that the operator pruned every tracked resource
- **AND** it SHALL NOT say that the claim is still in the cluster
- **AND** it SHALL name the claim and carry `kubectl delete pvc data -n apps`
- **AND** it SHALL say that an operator older than its CRDs deleted the claim

### Requirement: Every other operator-managed delete closes as a prune of the tracked resources

In every other case than the one of requirement "An operator-managed delete that may have kept claims does not claim a full prune" the closing output SHALL report the prune of the tracked resources as before.

#### Scenario: Operator-managed delete with the Delete policy closes as a full prune

- **WHEN** the same delete completes for an instance whose `spec.dataPolicy` is `Delete`
- **THEN** the closing output SHALL say that the operator pruned the tracked resources
- **AND** it SHALL print no `kubectl delete pvc` command

#### Scenario: Operator-managed delete on an operator without the field closes as a full prune

- **WHEN** the same delete completes in a cluster whose `ModuleInstance` CRD has no `spec.dataPolicy`
- **THEN** the closing output SHALL say that the operator pruned the tracked resources
- **AND** no output of the command SHALL say that a PersistentVolumeClaim is kept

### Requirement: Instance delete reads the instance record before it asks for confirmation

`opm instance delete` SHALL read the instance record before it asks for confirmation. An instance that has no record SHALL be reported as not found before any prompt.

#### Scenario: Missing instance is reported before the prompt

- **WHEN** running `opm instance delete nosuch -n media` without `--yes`
- **THEN** the command SHALL exit 5 without printing a confirmation prompt

### Requirement: The prompt names the claims that delete-data deletes

When `--delete-data` is set, the instance is CLI-owned and its inventory tracks at least one live PersistentVolumeClaim, the confirmation prompt SHALL name every such claim by namespace and name and SHALL say that the data on them is deleted.

#### Scenario: Prompt lists the claims

- **WHEN** running `opm instance delete jellyfin -n media --delete-data` without `--yes`
- **AND** the inventory tracks the claims `config` and `media` in namespace `media`
- **THEN** the prompt SHALL name `media/config` and `media/media`
- **AND** the prompt SHALL say that their data is deleted

### Requirement: Without delete-data the prompt says PersistentVolumeClaims are kept

Without `--delete-data` the prompt for a CLI-owned instance SHALL say that PersistentVolumeClaims are kept.

#### Scenario: Prompt without delete-data

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes`
- **THEN** the prompt SHALL say that PersistentVolumeClaims are kept

### Requirement: The prompt for an operator-managed instance says so and says what the operator does

For an operator-managed instance the prompt SHALL be the same with or without `--delete-data`, and SHALL say that the instance is operator-managed and what the operator does.

#### Scenario: Delete-data note comes before the question

- **WHEN** running `opm instance delete jellyfin -n media --delete-data` without `--yes` for an operator-managed instance
- **THEN** the warning that `--delete-data` does not change what the operator does SHALL be printed before the confirmation question

### Requirement: The prompt for an operator-managed instance without prune says the operator leaves the tracked resources running

For an operator-managed instance the prompt SHALL be the same with or without `--delete-data`, and SHALL say that the instance is operator-managed and what the operator does. When `spec.prune` is not set, it SHALL say that the operator leaves the tracked resources running.

#### Scenario: Prompt for an operator-managed instance without prune

- **WHEN** the `spec.prune` of an operator-managed instance is not set and its `spec.dataPolicy` is `Delete`
- **THEN** the prompt SHALL say that the operator leaves its tracked resources running
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are deleted

### Requirement: The prompt for an operator-managed instance with prune set and no claim says nothing about claims

For an operator-managed instance the prompt SHALL be the same with or without `--delete-data`, and SHALL say that the instance is operator-managed and what the operator does. When `spec.prune` is set and the inventory tracks no PersistentVolumeClaim, it SHALL say that the operator deletes the tracked resources and SHALL say nothing about PersistentVolumeClaims.

#### Scenario: Prompt for an instance that tracks no claim

- **WHEN** the `spec.prune` of an operator-managed instance is true and its inventory tracks no PersistentVolumeClaim
- **THEN** the prompt SHALL say that the operator deletes its tracked resources
- **AND** the prompt SHALL NOT mention PersistentVolumeClaims

### Requirement: With prune set and a tracked claim the prompt says one of four things decided by the CRD

When `spec.prune` is set and the inventory tracks at least one PersistentVolumeClaim, the prompt SHALL say one of four things, decided by whether the installed `ModuleInstance` CRD has `spec.dataPolicy`. The command SHALL take that from the CRD its operator-readiness check reads and SHALL NOT read the CRD a second time for it.

#### Scenario: The prompt depends on the CRD

- **WHEN** `spec.prune` is set and the inventory tracks at least one PersistentVolumeClaim
- **THEN** the prompt SHALL say one of four things, decided by whether the installed `ModuleInstance` CRD has `spec.dataPolicy`, and the command SHALL take that from the CRD its operator-readiness check reads

### Requirement: The dry run and the run with --yes state the same outcome for PersistentVolumeClaims

The dry run, and the run with `--yes`, SHALL state the same outcome for PersistentVolumeClaims in their output. (Requirement: Instance delete reads the instance record before it asks for confirmation)

#### Scenario: Same outcome in a dry run and with --yes

- **WHEN** the dry run or the run with `--yes` outputs the outcome for PersistentVolumeClaims
- **THEN** it SHALL state the same outcome for PersistentVolumeClaims in their output

### Requirement: The prompt says the operator deletes the claims and their data when the CRD has no policy field or the policy is Delete

When `spec.prune` is set and the inventory tracks at least one PersistentVolumeClaim, the prompt SHALL say one of four things. When the installed `ModuleInstance` CRD has no `spec.dataPolicy` field, the operator does not know it: the prompt SHALL say so and SHALL say that the operator deletes the tracked resources, PersistentVolumeClaims and their data included, whatever the instance carries. When `spec.dataPolicy` is `Delete`, it SHALL say the same and name the value.

#### Scenario: Prompt for an operator-managed instance with prune set

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for an operator-managed instance whose `spec.prune` is true and whose inventory tracks a PersistentVolumeClaim
- **AND** whose `spec.dataPolicy` is `Delete`
- **THEN** the prompt SHALL say that the instance is operator-managed and that the operator deletes its tracked resources, PersistentVolumeClaims and their data included
- **AND** the prompt SHALL name `spec.dataPolicy` and its value `Delete`
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are kept

#### Scenario: Prompt on an operator without the field

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for an operator-managed instance whose `spec.prune` is true and whose inventory tracks a PersistentVolumeClaim
- **AND** the installed `ModuleInstance` CRD has no `spec.dataPolicy`
- **THEN** the prompt SHALL say that the operator in the cluster has no `spec.dataPolicy` and deletes its tracked resources, PersistentVolumeClaims and their data included
- **AND** the prompt SHALL NOT say that PersistentVolumeClaims are kept

### Requirement: The prompt says the operator keeps the claims when the CRD has the field and the policy is Keep, absent or another value

When `spec.prune` is set and the inventory tracks at least one PersistentVolumeClaim, the prompt SHALL say one of four things. When the CRD has the field and `spec.dataPolicy` is `Keep`, absent, or any other value, the prompt SHALL say that the operator deletes the tracked resources and keeps PersistentVolumeClaims and their data, and SHALL add one sentence which says that an operator older than its CRDs deletes the claims whatever the field says.

#### Scenario: Prompt for an operator-managed instance with prune set and no data policy

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for an operator-managed instance whose `spec.prune` is true, whose inventory tracks a PersistentVolumeClaim and whose `spec.dataPolicy` is absent or `Keep`
- **AND** the installed `ModuleInstance` CRD has `spec.dataPolicy`
- **THEN** the prompt SHALL say that the operator deletes its tracked resources and keeps PersistentVolumeClaims and the data on them
- **AND** the prompt SHALL say that `spec.dataPolicy` is not set, or that it is `Keep`
- **AND** the prompt SHALL say that an operator older than its CRDs deletes the claims whatever the field says

### Requirement: The prompt does not say whether claims are kept when the CRD has no readable schema

When `spec.prune` is set and the inventory tracks at least one PersistentVolumeClaim, the prompt SHALL say one of four things. When the CRD has no schema that can be read and `spec.dataPolicy` is not `Delete`, the prompt SHALL NOT say that claims are kept or deleted: it SHALL say that an operator with the field keeps them, that an older operator deletes them, and that `opm` could not read the schema of the CRD to tell.

#### Scenario: Prompt when the CRD cannot be read

- **WHEN** the same prompt is built and the `ModuleInstance` CRD has no schema that can be read
- **THEN** the prompt SHALL NOT say that PersistentVolumeClaims are kept
- **AND** the prompt SHALL say that `opm` could not read the schema of the CRD to tell which operator runs

### Requirement: The prompt shows the value of spec.dataPolicy as it is

The prompt SHALL show the value of `spec.dataPolicy` as it is, or say that it is not set; a value other than `Keep` and `Delete` SHALL be shown quoted and named as read as `Keep`. (Requirement: Instance delete reads the instance record before it asks for confirmation)

#### Scenario: Prompt for an unknown data policy

- **WHEN** the `spec.dataPolicy` of an operator-managed instance with `spec.prune` set and a tracked PersistentVolumeClaim is `Retain`, and the CRD has the field
- **THEN** the prompt SHALL show `"Retain"` and say that it is read as `Keep`
- **AND** the prompt SHALL say that the operator keeps PersistentVolumeClaims

### Requirement: Every check that can refuse the delete runs before the prompt

Every check that can refuse the delete SHALL run before the prompt: the guard for an instance that deploys the operator, and, for an operator-managed instance, the operator-readiness check, whose failure includes a CRD that cannot be read. A refused delete SHALL NOT ask for confirmation.

#### Scenario: Refusal comes before the question

- **WHEN** running `opm instance delete jellyfin -n media` without `--yes` for an operator-managed instance
- **AND** the operator is not ready, or one of its CRDs or its controller Deployment cannot be read
- **THEN** the command SHALL exit 2 with the operator-readiness refusal
- **AND** it SHALL NOT print a confirmation prompt

### Requirement: A recorded resource that the command did not find is not in the deletion plan

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator. A recorded resource that the command did not find when it read the instance SHALL NOT be in the plan: it is not read again and not deleted. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A resource not found is not in the plan

- **WHEN** a recorded resource was not found when the command read the instance
- **THEN** it SHALL NOT be in the plan: it is not read again and not deleted

### Requirement: The command acts on a tracked resource only as the deletion plan names

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator. It SHALL read a tracked resource, delete it or leave it in place only when the plan names that action as the next one, and it SHALL send each delete with the propagation policy and the precondition the plan names. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Actions and delete options come from the plan

- **WHEN** `opm instance delete` acts on a tracked resource
- **THEN** it SHALL read a tracked resource, delete it or leave it in place only when the plan names that action as the next one, and it SHALL send each delete with the propagation policy and the precondition the plan names

### Requirement: The deletion plan does not hold a PersistentVolumeClaim that the command keeps

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator. The plan SHALL NOT hold a `PersistentVolumeClaim` that the command keeps ("Instance delete keeps a tracked PersistentVolumeClaim unless `--delete-data` is set"). Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A kept claim is outside the plan

- **WHEN** the command keeps a `PersistentVolumeClaim`
- **THEN** the plan SHALL NOT hold it

### Requirement: The ModuleInstance record is deleted only when the plan's release verdict allows it

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator. The command SHALL delete the `ModuleInstance` record only when the plan's release verdict allows it, which is when every planned resource was deleted or left in place and none failed. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: The release verdict holds the record

- **WHEN** `opm instance delete` of a CLI-owned instance has acted on the planned resources
- **THEN** the command SHALL delete the `ModuleInstance` record only when the plan's release verdict allows it, which is when every planned resource was deleted or left in place and none failed

### Requirement: A dry run follows the same plan and deletes nothing

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator. A dry run SHALL follow the same plan, SHALL send no delete, and SHALL NOT delete the record. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A dry run sends no delete

- **WHEN** `opm instance delete --dry-run` runs
- **THEN** it SHALL follow the same plan, SHALL send no delete, and SHALL NOT delete the record

### Requirement: The output lines and exit codes stay as the other requirements state them

`opm instance delete` of a CLI-owned instance SHALL take every action on a tracked resource from the deletion plan the CLI shares with the operator. The output lines, their wording and the exit codes of the command SHALL stay as the other requirements of this capability state them. The operator-owned delete path SHALL NOT be affected. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Output and exit codes are unchanged

- **WHEN** `opm instance delete` follows the plan
- **THEN** the output lines, their wording and the exit codes of the command SHALL stay as the other requirements of this capability state them, and the operator-owned delete path SHALL NOT be affected

### Requirement: Instance delete reports a resource as deleted when the API server accepts its delete

`opm instance delete` of a CLI-owned instance SHALL report a tracked resource as deleted when the API server accepts its delete request. Each delete uses Foreground propagation, so a deleted resource can still exist, with a `deletionTimestamp`, when the command exits: without `--wait` the command returns once every planned action is done. This requirement describes when a resource counts as deleted.

#### Scenario: Accepted delete of a resource with dependents

- **WHEN** running `opm instance delete` for a CLI-owned instance that tracks a Deployment with running Pods
- **AND** the API server accepts every delete request
- **THEN** the Deployment SHALL be listed as `deleted` and the command SHALL exit 0

### Requirement: Without --wait an accepted delete is no failure although the resource still exists

Without `--wait`, a resource whose delete was accepted SHALL NOT count as a failure because it still exists, also when a finalizer keeps it; with no other failure the `ModuleInstance` record is deleted and the exit code is 0.

#### Scenario: Resource held by a finalizer

- **WHEN** running `opm instance delete` without `--wait` for a CLI-owned instance that tracks a ConfigMap carrying a finalizer that no controller removes
- **THEN** the ConfigMap SHALL be listed as `deleted`
- **AND** the `ModuleInstance` record SHALL be deleted
- **AND** the command SHALL exit 0

### Requirement: With --wait the command also waits for the deleted resources to disappear

With `--wait` the command (`opm instance delete`) SHALL also wait, bounded, for the deleted resources to disappear ("Instance delete with `--wait` waits for the deleted resources to disappear").

#### Scenario: The wait is bounded

- **WHEN** `opm instance delete` runs with `--wait`
- **THEN** the command SHALL also wait, bounded, for the deleted resources to disappear

### Requirement: Instance delete accepts --wait and without it nothing changes

`opm instance delete` SHALL accept `--wait`. Without `--wait`, the output and the exit code of the command SHALL be the same as before this requirement.

#### Scenario: Without the flag nothing changes

- **WHEN** running `opm instance delete` without `--wait` for a CLI-owned instance whose resources are still terminating after their deletes are accepted
- **THEN** the command SHALL NOT read the resources again
- **AND** the `ModuleInstance` record SHALL be deleted and the command SHALL exit 0

### Requirement: Instance delete with `--wait` waits for the deleted resources to disappear

With `--wait`, on a real run of a CLI-owned instance in which no tracked resource failed, the command (`opm instance delete`) SHALL, after every planned action is done and before it deletes the `ModuleInstance` record, poll each resource whose delete the API server accepted until the resource is gone or `--timeout` (default `5m0s`) has passed since the wait started. When every such resource is gone, the command SHALL delete the `ModuleInstance` record and report as it does without `--wait`.

#### Scenario: Deleted resources go away within the timeout

- **WHEN** running `opm instance delete --wait` for a CLI-owned instance that tracks a Deployment
- **AND** the Deployment still exists with a `deletionTimestamp` after its delete is accepted, and is gone some seconds later
- **THEN** the command SHALL return only after a read of the Deployment returns NotFound
- **AND** the `ModuleInstance` record SHALL be deleted and the command SHALL exit 0

### Requirement: A resource is gone when its read returns NotFound or its UID differs

With `--wait`, on a real run of a CLI-owned instance in which no tracked resource failed, a resource is gone when its read returns NotFound, or when the live object under its name has a UID other than the one of the object that was deleted. A read that fails with another error SHALL NOT count as gone. (Requirement: Instance delete with `--wait` waits for the deleted resources to disappear)

#### Scenario: Name taken by a new object

- **WHEN** a deleted ConfigMap is gone and another object with the same name and a different UID exists
- **THEN** the wait SHALL count the ConfigMap as gone

### Requirement: While it waits the command prints how many deleted resources are still terminating

With `--wait`, on a real run of a CLI-owned instance in which no tracked resource failed, while it waits, the command (`opm instance delete`) SHALL print a line that says how many of the deleted resources are still terminating each time that number drops, and at least every 30 seconds otherwise.

#### Scenario: Progress while waiting

- **WHEN** the command waits for the deleted resources
- **THEN** it SHALL print a line that says how many of the deleted resources are still terminating each time that number drops, and at least every 30 seconds otherwise

### Requirement: The wait skips a resource the command did not delete

With `--wait`, on a real run of a CLI-owned instance in which no tracked resource failed, the command (`opm instance delete`) SHALL NOT wait for a resource it did not delete: a kept `PersistentVolumeClaim`, a resource left behind, or a resource that was already absent.

#### Scenario: Kept claim is not waited for

- **WHEN** running `opm instance delete --wait` without `--delete-data` for an instance that tracks a PersistentVolumeClaim and a ConfigMap
- **THEN** the command SHALL wait for the ConfigMap only
- **AND** it SHALL exit 0 while the claim still exists

#### Scenario: Resource left behind is not waited for

- **WHEN** running `opm instance delete --wait` for an instance whose record lists a ConfigMap that now belongs to another instance
- **THEN** that ConfigMap SHALL be listed as `left behind` and SHALL NOT be waited for

### Requirement: A timeout lists the resources still terminating and keeps the ModuleInstance record

With `--wait`, on a real run of a CLI-owned instance in which no tracked resource failed, when the timeout passes first, the command (`opm instance delete`) SHALL list each resource that is still terminating, with its finalizers when it has any, SHALL NOT delete the `ModuleInstance` record, SHALL NOT print a success line, and SHALL exit 1.

#### Scenario: Resource held by a finalizer times out

- **WHEN** running `opm instance delete --wait --timeout 30s` for a CLI-owned instance that tracks a ConfigMap carrying a finalizer that no controller removes
- **THEN** after 30 seconds the command SHALL list the ConfigMap as terminating and name its finalizer
- **AND** a resource that no read answered for before the timeout SHALL be listed with that as its error, not as gone
- **AND** the `ModuleInstance` record SHALL NOT be deleted
- **AND** the command SHALL exit 1

#### Scenario: Re-run after a timeout

- **WHEN** a run with `--wait` timed out, the cause was removed, and the same command runs again
- **THEN** the command SHALL find the instance, wait until the resources are gone, delete the `ModuleInstance` record and exit 0

### Requirement: --wait has no effect on a dry run, after a failed delete or on an operator-managed instance

With `--dry-run`, and in a run in which a tracked resource failed, `--wait` SHALL have no effect. For an operator-managed instance `--wait` SHALL have no effect: the command (`opm instance delete`) waits for the `ModuleInstance` to be gone, bounded by `--timeout`, with or without it.

#### Scenario: Dry run does not wait

- **WHEN** running `opm instance delete --wait --dry-run`
- **THEN** no delete request SHALL be sent and the command SHALL NOT wait

#### Scenario: Operator-managed instance

- **WHEN** running `opm instance delete --wait` for an operator-managed instance
- **THEN** the command SHALL behave as without `--wait`: it deletes the `ModuleInstance` and waits, bounded by `--timeout`, for it to be gone

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
  -y, --yes                 Skip the confirmation prompt
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
