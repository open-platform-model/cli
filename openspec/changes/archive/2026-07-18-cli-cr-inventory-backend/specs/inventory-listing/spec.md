# Delta: inventory-listing (cli-cr-inventory-backend)

Listing moves from a Secret label-selector list to a native `ModuleInstance` CR list (enhancement 0006 D29).

## ADDED Requirements

### Requirement: Instance listing is a native ModuleInstance CR list

`opm instance list` SHALL list `ModuleInstance` custom resources in the target namespace (resolved by the standard namespace precedence). Results SHALL be sorted alphabetically by instance name. A CR that cannot be interpreted (e.g. malformed status) SHALL be reported with a warning and skipped, without failing the listing.

#### Scenario: List in a specific namespace

- **WHEN** `opm instance list -n production` runs and 3 `ModuleInstance` CRs exist in `production`
- **THEN** the command SHALL display those 3 instances sorted by name

#### Scenario: Empty namespace lists nothing

- **WHEN** `opm instance list` runs against a namespace with no `ModuleInstance` CRs
- **THEN** the command SHALL succeed with an empty result

### Requirement: Cluster-wide listing via --all-namespaces

`opm instance list --all-namespaces` (shorthand `-A`) SHALL perform a cluster-wide `ModuleInstance` list. On insufficient RBAC (cross-namespace list denied), the command SHALL fail with a clear, actionable error naming the missing permission. It MUST NOT degrade to a partial or label-based listing.

#### Scenario: Cluster-wide list

- **WHEN** `opm instance list -A` runs with cluster-wide list permission
- **THEN** instances from all namespaces SHALL be displayed with their namespaces

#### Scenario: Insufficient RBAC fails clearly

- **WHEN** `opm instance list -A` runs without cluster-wide `list` permission on `moduleinstances`
- **THEN** the command SHALL exit non-zero with an error naming the required permission
- **AND** SHALL NOT display a partial result

## REMOVED Requirements

### Requirement: ListInventories discovers all inventory Secrets in a namespace

**Reason**: Inventory Secrets no longer exist (enhancement 0006 D1/D8); the `ModuleInstance` CRs are the inventory, so listing CRs *is* the listing.
**Migration**: `opm instance list` lists `ModuleInstance` CRs (see ADDED requirements). Secret-tracked instances become visible after their one-time migration on next apply.

### Requirement: ListInventories uses existing label constants

**Reason**: Label-selector discovery of inventory Secrets is deleted with the Secret backend.
**Migration**: None — CR listing needs no label selector.
