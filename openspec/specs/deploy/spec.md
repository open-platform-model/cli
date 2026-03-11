# CLI Deploy Commands

## Purpose

The `opm mod apply` and `opm mod delete` commands manage the deployment lifecycle of OPM modules on Kubernetes clusters. `mod apply` renders the module via the Pipeline interface and applies resources using server-side apply. `mod delete` discovers and removes module resources from the persisted release inventory record when available, falling back to OPM labels without requiring the original source.

## Design Rationale

1. **Go API integration**: Commands call `build.NewPipeline().Render()` directly, not subprocess.
2. **Inventory-first discovery**: `mod delete` discovers resources via the persisted release inventory record when available, falling back to labels instead of re-rendering.
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
2. **Given** a deployed module, **When** running `opm mod delete`, **Then** all resources are removed.
3. **Given** a module with CRDs and CRs, **When** running `opm mod apply`, **Then** CRDs are created first.
4. **Given** pending changes, **When** running `opm mod apply`, **Then** changes are applied.
5. **Given** dry-run request, **When** running `opm mod apply --dry-run`, **Then** no changes are made.

### User Story 2 - Delete Module Without Source (Priority: P2)

A developer wants to delete a deployed module after deleting the source files.

**Independent Test**: Deploy module, delete source, `opm mod delete` still works.

**Acceptance Scenarios**:

1. **Given** a deployed module, **When** source is deleted and `opm mod delete -n <ns> --name <name>` runs, **Then** resources are removed.
2. **Given** delete request, **When** running `opm mod delete --dry-run`, **Then** resources to delete are listed but not removed.

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
| FR-D-020 | `mod delete` MUST discover resources via the persisted release inventory record when available, falling back to OPM labels when no inventory exists. |
| FR-D-021 | `mod delete` MUST NOT require module source. |
| FR-D-022 | `mod delete` MUST delete in descending weight order. |
| FR-D-023 | `mod delete` MUST support `--force` to skip confirmation. |
| FR-D-024 | `mod delete` MUST support `--dry-run` to preview. |
| FR-D-025 | `mod delete` MUST require at least one of `--name` or `--release-id` for identification. The `--namespace` / `-n` flag remains required in all cases. |
| FR-D-026 | `mod delete` MUST prompt for confirmation (unless --force). |
| FR-D-027 | `mod delete` MUST support `--release-id` flag for discovery by release identity UUID. |
| FR-D-028 | `mod delete` MUST use ownership-inventory-based enumeration when a persisted release inventory record exists. When no inventory exists, it MUST fall back to label-based discovery. Dual-strategy discovery (both release-id and name+namespace selectors) applies only in the label-based fallback path. |

### Inventory Integration

#### Requirement: mod apply uses ownership inventory for pruning

`opm mod apply` SHALL use the current ownership inventory to compute stale resources (previous owned set minus current rendered set). It SHALL NOT require inventory change-history fields to perform stale-set computation or pruning.

#### Scenario: Apply computes stale set from ownership inventory

- **WHEN** a previous ownership inventory tracks resources `A`, `B`, and `C`
- **AND** the current render contains `A` and `B`
- **THEN** `C` SHALL be considered stale and eligible for pruning

#### Requirement: mod apply writes current release inventory record after successful apply

After all resources are successfully applied, `opm mod apply` SHALL persist the current release inventory record for the release. The persisted form SHALL store top-level `createdBy`, `releaseMetadata`, `moduleMetadata`, and the current owned resource set directly instead of a history-bearing inventory shape.

#### Scenario: Successful apply persists current release inventory record

- **WHEN** `opm mod apply` successfully applies all rendered resources
- **THEN** the persisted release inventory record SHALL record the current owned resource entries for that release
- **AND** the record SHALL preserve top-level `createdBy`, `releaseMetadata`, and `moduleMetadata`
- **AND** the ownership inventory in that record SHALL be sufficient for later prune and resource enumeration

#### Scenario: Failed apply does not write inventory

- **WHEN** `opm mod apply` fails to apply one or more resources
- **THEN** the persisted release inventory record SHALL NOT be written or updated

#### Requirement: mod apply stores deployed module version in module metadata

When `opm mod apply` persists the release inventory record, it SHALL store the deployed module version in `moduleMetadata.version` rather than relying on inventory change history.

#### Scenario: Apply persists deployed module version in module metadata

- **WHEN** `opm mod apply` persists a release inventory record for a versioned module
- **THEN** the persisted record SHALL contain that deployed module version under `moduleMetadata.version`

#### Requirement: mod apply supports --no-prune flag

The `--no-prune` flag SHALL skip the stale resource pruning step. Default is `false` (pruning enabled).

#### Scenario: --no-prune skips pruning

- **WHEN** running `opm mod apply --no-prune`
- **THEN** stale resources SHALL NOT be deleted
- **AND** the persisted release inventory record SHALL still be written

#### Requirement: mod apply supports --force for empty render

The `--force` flag SHALL allow `opm mod apply` to proceed when the render produces zero resources and a previous ownership inventory exists. Without `--force`, this situation SHALL fail with an error.

#### Scenario: Empty render blocked without --force

- **WHEN** running `opm mod apply` and the render produces 0 resources with a non-empty previous ownership inventory
- **THEN** the command SHALL fail with an error indicating all resources would be pruned

#### Scenario: Empty render allowed with --force

- **WHEN** running `opm mod apply --force` and the render produces 0 resources
- **THEN** all previously tracked resources SHALL be pruned

#### Requirement: mod delete uses ownership inventory for resource enumeration

`opm mod delete` SHALL use the ownership inventory stored in the persisted release inventory record to enumerate resources for deletion when it exists. If an inventory exists, only resources tracked in that ownership inventory SHALL be deleted (no label-scan), and the inventory Secret itself SHALL be deleted last. If no inventory exists, the command SHALL fall back to label-based discovery.

#### Scenario: Delete with ownership inventory

- **WHEN** running `opm mod delete` and a persisted release inventory record exists
- **THEN** only resources listed in that record's ownership inventory SHALL be deleted
- **AND** the inventory Secret SHALL be deleted after all tracked resources

#### Scenario: Delete without inventory (fallback)

- **WHEN** running `opm mod delete` and no inventory Secret exists
- **THEN** the command SHALL fall back to label-based discovery via `DiscoverResources()`

#### Scenario: Delete does not remove derived resources

- **WHEN** running `opm mod delete` with ownership inventory
- **AND** derived resources (e.g., Endpoints) exist with OPM labels but are not in the inventory
- **THEN** the derived resources SHALL NOT be deleted

### Kubernetes Integration

| ID | Requirement |
|----|-------------|
| FR-D-050 | MUST use client-go with default rate limiting. |
| FR-D-051 | MUST support kubeconfig from flag, env, or default path. |
| FR-D-052 | MUST support context selection via flag. |
| FR-D-053 | MUST fail fast with clear error on connectivity issues. |

### Requirement: CLI mutating workflows refuse controller-managed releases

Before mutating an existing release, CLI workflows that apply or delete a release SHALL inspect inventory ownership. If the inventory indicates `createdBy: "controller"`, the CLI MUST refuse the mutation.

#### Scenario: Apply blocked for controller-managed release

- **WHEN** the user runs `opm mod apply` for a release whose inventory records `createdBy: "controller"`
- **THEN** the command SHALL fail before mutating cluster resources
- **AND** the error SHALL state that the release is controller-managed and cannot be changed by the CLI

#### Scenario: Delete blocked for controller-managed release

- **WHEN** the user runs `opm mod delete` for a release whose inventory records `createdBy: "controller"`
- **THEN** the command SHALL fail before deleting tracked resources
- **AND** the error SHALL state that the release is controller-managed and cannot be deleted by the CLI

### Requirement: Legacy CLI-managed releases remain mutable by the CLI

If an existing inventory has `createdBy: "cli"` or does not contain `createdBy`, CLI mutating workflows SHALL continue to operate normally.

#### Scenario: Apply allowed for legacy inventory

- **WHEN** the user runs `opm mod apply` for a release whose inventory has no `createdBy`
- **THEN** the CLI SHALL treat the release as CLI-managed and proceed normally

### Resource Labeling

| ID | Requirement |
|----|-------------|
| FR-D-060 | All resources MUST have `app.kubernetes.io/managed-by: open-platform-model`. |
| FR-D-061 | All resources MUST have `module.opmodel.dev/name: <name>`. |
| FR-D-062 | ~~All resources MUST have `module.opmodel.dev/namespace: <ns>`.~~ Removed — namespace scoping is handled by K8s API calls, not labels. |
| FR-D-063 | All resources MUST have `module.opmodel.dev/version: <version>`. |
| FR-D-064 | All resources MUST have `component.opmodel.dev/name: <component>`. |
| FR-D-065 | All resources MUST have `module-release.opmodel.dev/uuid: <release-uuid>` when the release identity is available. |
| FR-D-066 | All resources MUST have `module.opmodel.dev/uuid: <module-uuid>` when the module identity is available. |

---

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
| Module source deleted | `mod delete` works via labels |
| Empty RenderResult | Fail with error if previous inventory is non-empty (use --force to override) |
| Delete with neither --name nor --release-id | Return validation error: "either --name or --release-id is required" |
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
      --name string         Release name (default: module name)
      --provider string     Provider to use
      --dry-run             Server-side dry run
      --wait                Wait for resources to be ready
      --timeout duration    Wait timeout (default: 5m)
      --no-prune            Skip stale resource pruning
      --force               Allow empty render to prune all resources
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

### mod delete

```text
opm mod delete [flags]

Flags:
  -n, --namespace string    Target namespace (required)
      --name string         Module name (required if --release-id not provided)
      --release-id string   Release identity UUID (required if --name not provided)
      --force               Skip confirmation prompt
      --dry-run             Preview without deleting
      --wait                Wait for resources to be deleted
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
