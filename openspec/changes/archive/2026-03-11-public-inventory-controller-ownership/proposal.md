## Why

OPM's inventory Secret is currently trapped in `internal/inventory`, which prevents a future Kubernetes controller from reusing the exact inventory contract that the CLI already depends on for discovery, status, diff, and deletion. We need to make the inventory model public before controller work starts, and we need explicit ownership metadata so the CLI and controller can interoperate without taking over each other's releases.

## What Changes

- Extract the reusable inventory data model, serialization, naming, and change-history helpers into a public `pkg/inventory` package.
- Preserve the existing inventory Secret wire format and discovery labels so current CLI behavior and existing clusters remain compatible.
- Add write-once inventory provenance metadata (`createdBy`) so new releases record whether they were created by the CLI or a controller.
- Define backward-compatibility rules: inventory Secrets without `createdBy` are treated as legacy CLI-managed releases.
- Update CLI mutating workflows to block takeover of controller-managed releases while continuing to allow read-only inspection.
- Define the reciprocal ownership rule for future controller integrations: controllers must not take over CLI-managed releases.

## Capabilities

### New Capabilities
- `inventory-ownership`: Defines inventory provenance, legacy compatibility, and the no-takeover policy between CLI-managed and controller-managed releases.
- `public-inventory-package`: Defines the public reusable inventory package boundary so controller code can consume the same inventory contract as the CLI.

### Modified Capabilities
- `release-inventory`: Extend the inventory metadata contract with provenance while preserving Secret naming, labels, serialization, and write-once metadata semantics.
- `deploy`: Change mutating CLI behavior so apply/delete operations refuse to take over controller-managed releases.
- `mod-list`: Surface release ownership in list output so users can distinguish CLI-managed and controller-managed releases.
- `mod-status`: Surface release ownership in status output and warn when a release is not CLI-managed.

## Impact

- **Code**: `internal/inventory/`, `pkg/`, CLI apply/delete/list/status workflows, and future controller integration points.
- **Behavior**: Existing releases remain CLI-managed by default; new controller-managed releases become visible to the CLI but protected from CLI mutation.
- **Compatibility**: Inventory Secret labels and core data layout stay backward compatible; only an optional metadata field is added.
- **SemVer**: MINOR — adds new ownership-aware behavior and public package surface without changing existing flags or breaking legacy inventories.
