## MODIFIED Requirements

### Requirement: mod apply uses ownership inventory for pruning

`opm mod apply` SHALL use the current ownership inventory to compute stale resources (previous owned set minus current rendered set). It SHALL NOT require inventory change-history fields to perform stale-set computation or pruning.

#### Scenario: Apply computes stale set from ownership inventory

- **WHEN** a previous ownership inventory tracks resources `A`, `B`, and `C`
- **AND** the current render contains `A` and `B`
- **THEN** `C` SHALL be considered stale and eligible for pruning

### Requirement: mod apply writes current ownership inventory after successful apply

After all resources are successfully applied, `opm mod apply` SHALL persist the current ownership inventory for the release. The persisted form SHALL store the current owned resource set directly instead of a history-bearing inventory shape.

#### Scenario: Successful apply persists current ownership set

- **WHEN** `opm mod apply` successfully applies all rendered resources
- **THEN** the persisted inventory SHALL record the current owned resource entries for that release
- **AND** the inventory SHALL be sufficient for later prune and resource enumeration

### Requirement: mod delete uses ownership inventory for resource enumeration

`opm mod delete` SHALL use the persisted ownership inventory to enumerate resources for deletion when it exists. If no persisted inventory exists, the command MAY fall back to legacy discovery behavior.

#### Scenario: Delete with ownership inventory

- **WHEN** running `opm mod delete` and a persisted ownership inventory exists
- **THEN** only resources listed in that inventory SHALL be deleted
