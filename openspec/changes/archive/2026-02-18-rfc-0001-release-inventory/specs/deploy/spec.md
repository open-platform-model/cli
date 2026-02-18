## ADDED Requirements

### Requirement: mod apply writes inventory Secret after successful apply

After all resources are successfully applied, `opm mod apply` SHALL write an inventory Secret to the same namespace as the release. The Secret SHALL contain the current change entry (module ref, values, manifest digest, timestamp, inventory entries) and maintain a history index. The inventory SHALL only be written when all resources apply without error.

#### Scenario: Successful apply writes inventory

- **WHEN** `opm mod apply` successfully applies all rendered resources
- **THEN** an inventory Secret SHALL be created or updated in the release namespace
- **AND** the Secret SHALL contain the current change entry with all rendered resource entries

#### Scenario: Failed apply does not write inventory

- **WHEN** `opm mod apply` fails to apply one or more resources
- **THEN** the inventory Secret SHALL NOT be written or updated

### Requirement: mod apply supports --no-prune flag

The `--no-prune` flag SHALL skip the stale resource pruning step. Default is `false` (pruning enabled).

#### Scenario: --no-prune skips pruning

- **WHEN** running `opm mod apply --no-prune`
- **THEN** stale resources SHALL NOT be deleted
- **AND** the inventory SHALL still be written

### Requirement: mod apply supports --max-history flag

The `--max-history` flag SHALL set the maximum number of change entries in the inventory. Default SHALL be 10.

#### Scenario: Default max-history

- **WHEN** running `opm mod apply` without `--max-history`
- **THEN** the inventory SHALL retain at most 10 change entries

### Requirement: mod apply supports --force for empty render

The `--force` flag SHALL allow `opm mod apply` to proceed when the render produces zero resources and a previous inventory exists. Without `--force`, this situation SHALL fail with an error.

#### Scenario: Empty render blocked without --force

- **WHEN** running `opm mod apply` and the render produces 0 resources with a non-empty previous inventory
- **THEN** the command SHALL fail with an error indicating all resources would be pruned

#### Scenario: Empty render allowed with --force

- **WHEN** running `opm mod apply --force` and the render produces 0 resources
- **THEN** all previously tracked resources SHALL be pruned

### Requirement: mod delete uses inventory for resource enumeration

`opm mod delete` SHALL first attempt to read the inventory Secret to enumerate resources for deletion. If an inventory exists, only resources tracked in the inventory SHALL be deleted (no label-scan). The inventory Secret itself SHALL be deleted last. If no inventory exists, the command SHALL fall back to label-based discovery.

#### Scenario: Delete with inventory

- **WHEN** running `opm mod delete` and an inventory Secret exists
- **THEN** only resources listed in the inventory SHALL be deleted
- **AND** the inventory Secret SHALL be deleted after all tracked resources

#### Scenario: Delete without inventory (fallback)

- **WHEN** running `opm mod delete` and no inventory Secret exists
- **THEN** the command SHALL fall back to label-based discovery via `DiscoverResources()`

#### Scenario: Delete does not remove derived resources

- **WHEN** running `opm mod delete` with an inventory
- **AND** derived resources (e.g., Endpoints) exist with OPM labels but are not in the inventory
- **THEN** the derived resources SHALL NOT be deleted

## MODIFIED Requirements

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
| FR-D-020 | `mod delete` MUST discover resources via inventory Secret when available, falling back to OPM labels when no inventory exists. |
| FR-D-021 | `mod delete` MUST NOT require module source. |
| FR-D-022 | `mod delete` MUST delete in descending weight order. |
| FR-D-023 | `mod delete` MUST support `--force` to skip confirmation. |
| FR-D-024 | `mod delete` MUST support `--dry-run` to preview. |
| FR-D-025 | `mod delete` MUST require at least one of `--name` or `--release-id` for identification. The `--namespace` / `-n` flag remains required in all cases. |
| FR-D-026 | `mod delete` MUST prompt for confirmation (unless --force). |
| FR-D-027 | `mod delete` MUST support `--release-id` flag for discovery by release identity UUID. |
| FR-D-028 | `mod delete` MUST use inventory-based enumeration when an inventory Secret exists. When no inventory exists, it MUST fall back to label-based discovery. Dual-strategy discovery (both release-id and name+namespace selectors) applies only in the label-based fallback path. |

### Command Syntax

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
      --max-history int     Maximum change history entries (default: 10)
      --force               Allow empty render to prune all resources
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```
