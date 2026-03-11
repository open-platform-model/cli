## Why

The current public inventory contract still reflects an older CLI persistence model instead of the smaller ownership-focused interface a future controller will need. `pkg/inventory` currently mixes three concerns in one API surface: resource ownership, change history, and storage helpers. That makes the package harder to reuse from controller code and leaves the public contract misaligned with the controller shape we want (`ModuleRelease.status.inventory` / `BundleRelease.status.inventory`).

We want to make inventory a stable public interoperability contract that answers one question well: what resources does this release currently own? The CLI should keep using inventory for apply, prune, discovery, and status scoping, but it should stop treating inventory as the universal source for source metadata, raw values, and change history.

## What Changes

- Redefine the public inventory contract in `pkg/inventory` as ownership-only: current resource refs plus optional inventory digest/count/revision.
- Remove change-history types and helpers from the public inventory API (`ChangeEntry`, `ChangeSource`, `Index`, `Changes`, `ComputeChangeID`, `UpdateIndex`, `PruneHistory`, `PrepareChange`).
- Remove storage-oriented history concerns from the core inventory contract and redesign persistence around the new ownership-only shape.
- Update CLI workflows to use inventory only for ownership and prune logic, not as the source of version/history metadata.
- Prepare the CLI for a future controller where release/source/history data comes from release status while inventory remains the shared ownership contract.

## Capabilities

### Modified Capabilities

- `release-inventory`: Redefine inventory as current owned resources only, with optional digest/count/revision metadata.
- `public-inventory-package`: Restrict the public package to reusable ownership concerns and remove history/storage-specific API surface.
- `deploy`: Use ownership inventory for stale-set computation and pruning, without depending on inventory change-history fields.
- `mod-list`: Stop requiring inventory change history for release display metadata; inventory remains the release discovery and ownership scope.
- `mod-status`: Continue using inventory for tracked resource discovery, but stop depending on inventory change-history metadata for headers.

## Impact

- **Code**: `pkg/inventory`, inventory persistence, CLI apply/delete/list/status workflows, and future controller integration points.
- **Behavior**: Inventory remains sufficient for ownership-aware CLI operations, but history/source metadata moves out of the inventory contract.
- **Compatibility**: The CLI is still under heavy development, so the persisted inventory format can be redesigned directly to match the new ownership-only contract.
- **SemVer**: MINOR — public package surface changes in a way intended to prepare the next controller-facing interface while the CLI is still free to evolve.
