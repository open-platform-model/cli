## 1. Ownership-only public inventory contract

- [ ] 1.1 Replace the public `pkg/inventory` contract with ownership-only inventory types (`Inventory`, `InventoryEntry`) plus pure identity/stale-set helpers.
- [ ] 1.2 Add optional top-level ownership summary fields (`revision`, `digest`, `count`) to the new inventory shape where justified.
- [ ] 1.3 Remove public change-history helpers and types from `pkg/inventory` (`ChangeEntry`, `ChangeSource`, `Index`, `Changes`, `ComputeChangeID`, `UpdateIndex`, `PruneHistory`, `PrepareChange`).

## 2. Persisted release inventory record redesign

- [ ] 2.1 Introduce a persisted release inventory record envelope with top-level `createdBy`, `releaseMetadata`, `moduleMetadata`, and ownership-only `inventory`.
- [ ] 2.2 Add deployed module version to `moduleMetadata` and remove the need to read it from inventory history.
- [ ] 2.3 Redesign persistence to store the release inventory record as a single JSON document instead of the old history-bearing Secret format.
- [ ] 2.4 Keep Secret naming/labels aligned with the new release inventory record only where still needed by the CLI.
- [ ] 2.5 Add tests covering release inventory record persistence round trips.

## 3. CLI workflow alignment

- [ ] 3.1 Update apply/prune logic to use ownership inventory only for stale-set computation and resource ownership checks.
- [ ] 3.2 Update delete/status/list flows to rely on ownership inventory for resource enumeration, while reading release/module metadata from the persisted release inventory record instead of inventory change history.
- [ ] 3.3 Remove inventory change-history assumptions from query/output code paths touched by the redesign.

## 4. Validation

- [ ] 4.1 Add unit coverage for ownership equality, K8s identity equality, inventory digest/count behavior, and stale-set computation.
- [ ] 4.2 Add focused workflow coverage for apply/delete/status/list behavior using ownership-only inventory plus persisted release/module metadata.
- [ ] 4.3 Run `task fmt`, `task lint`, and targeted tests for inventory and affected workflows.
