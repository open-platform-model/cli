## 1. Ownership-only public inventory contract

- [x] 1.1 Replace the public `pkg/inventory` contract with ownership-only inventory types (`Inventory`, `InventoryEntry`) plus pure identity/stale-set helpers.
- [x] 1.2 Add optional top-level ownership summary fields (`revision`, `digest`, `count`) to the new inventory shape where justified.
- [x] 1.3 Remove public change-history helpers and types from `pkg/inventory` (`ChangeEntry`, `ChangeSource`, `Index`, `Changes`, `ComputeChangeID`, `UpdateIndex`, `PruneHistory`, `PrepareChange`).

## 2. Persisted release inventory record redesign

- [x] 2.1 Introduce a persisted release inventory record envelope with top-level `createdBy`, `releaseMetadata`, `moduleMetadata`, and ownership-only `inventory`.
- [x] 2.2 Add deployed module version to `moduleMetadata` and remove the need to read it from inventory history.
- [x] 2.3 Redesign persistence to store the release inventory record as a single JSON document instead of the old history-bearing Secret format.
- [x] 2.4 Keep Secret naming/labels aligned with the new release inventory record only where still needed by the CLI.
- [x] 2.5 Add tests covering release inventory record persistence round trips.

## 3. CLI workflow alignment

- [x] 3.1 Update apply/prune logic to use ownership inventory only for stale-set computation and resource ownership checks.
- [x] 3.2 Update delete/status/list flows to rely on ownership inventory for resource enumeration, while reading release/module metadata from the persisted release inventory record instead of inventory change history.
- [x] 3.3 Remove inventory change-history assumptions from query/output code paths touched by the redesign.

## 4. Validation

- [x] 4.1 Add unit coverage for ownership equality, K8s identity equality, inventory digest/count behavior, and stale-set computation.
- [x] 4.2 Add focused workflow coverage for apply/delete/status/list behavior using ownership-only inventory plus persisted release/module metadata.
- [x] 4.3 Run `task fmt`, `task lint`, and targeted tests for inventory and affected workflows.
