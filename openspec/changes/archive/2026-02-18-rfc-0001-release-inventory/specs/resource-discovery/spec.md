## ADDED Requirements

### Requirement: Inventory Secret excluded from workload discovery

`DiscoverResources()` SHALL exclude Secrets that carry the label `opmodel.dev/component: inventory` from its results. This prevents the inventory Secret from appearing in workload resource queries used by diff, status, and the label-based delete fallback.

#### Scenario: Inventory Secret not returned by DiscoverResources

- **WHEN** `DiscoverResources()` scans a namespace containing an inventory Secret with label `opmodel.dev/component: inventory`
- **AND** the inventory Secret also has `app.kubernetes.io/managed-by: open-platform-model`
- **THEN** the inventory Secret SHALL NOT appear in the returned resource list

#### Scenario: Non-inventory Secrets still returned

- **WHEN** `DiscoverResources()` scans a namespace containing a regular Secret with OPM labels but without `opmodel.dev/component: inventory`
- **THEN** the Secret SHALL appear in the returned resource list

### Requirement: Inventory-based resource discovery

A new `DiscoverResourcesFromInventory` function SHALL accept an `InventorySecret` and perform targeted GET calls for each tracked resource. It SHALL return both the live resources found and a list of inventory entries for resources that no longer exist on the cluster (missing). This provides fast, precise discovery (N API calls for N resources) compared to the label-scan approach (hundreds of API calls).

#### Scenario: All tracked resources exist

- **WHEN** the inventory contains 5 entries and all 5 resources exist on the cluster
- **THEN** 5 live resources SHALL be returned and the missing list SHALL be empty

#### Scenario: Some tracked resources missing

- **WHEN** the inventory contains 5 entries and 2 resources have been manually deleted
- **THEN** 3 live resources SHALL be returned and 2 entries SHALL be in the missing list

#### Scenario: Empty inventory

- **WHEN** the inventory contains 0 entries
- **THEN** both the live list and missing list SHALL be empty
