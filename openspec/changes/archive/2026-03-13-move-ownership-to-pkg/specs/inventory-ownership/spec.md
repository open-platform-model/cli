## MODIFIED Requirements

### Requirement: Inventory provenance defines release ownership
The inventory contract SHALL define release ownership through a `createdBy` field with allowed values matching `ownership.CreatedByCLI` and `ownership.CreatedByController` constants from `pkg/ownership`. The ownership enforcement function SHALL reside in `pkg/ownership` (previously `internal/workflow/ownership`).

#### Scenario: CLI-owned release
- **WHEN** an inventory records `createdBy` matching `ownership.CreatedByCLI`
- **THEN** the release SHALL be treated as CLI-managed

#### Scenario: Controller-owned release
- **WHEN** an inventory records `createdBy` matching `ownership.CreatedByController`
- **THEN** the release SHALL be treated as controller-managed
