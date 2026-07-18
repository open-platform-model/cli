## Purpose

Defines how `opm instance list` discovers deployed instances by listing `ModuleInstance` custom resources, including namespace scoping and cluster-wide listing.

## Requirements

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
