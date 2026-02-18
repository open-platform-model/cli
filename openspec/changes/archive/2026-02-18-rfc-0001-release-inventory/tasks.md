## 1. Data Model & Types (sub-issue #34)

- [x] 1.1 Add `LabelComponent = "opmodel.dev/component"` and `labelComponentInventory = "inventory"` constants to `internal/kubernetes/discovery.go`
- [x] 1.2 Add `k8s.io/api` as a direct dependency (`go get k8s.io/api`)
- [x] 1.3 Create `internal/inventory/types.go` with `InventoryEntry`, `ChangeEntry`, `ModuleRef`, `InventoryList`, `InventoryMetadata`, and `InventorySecret` structs (all with JSON tags matching the RFC data model)
- [x] 1.4 Create `internal/inventory/entry.go` with `NewEntryFromResource(*build.Resource) InventoryEntry`, `IdentityEqual(a, b InventoryEntry) bool`, and `K8sIdentityEqual(a, b InventoryEntry) bool`
- [x] 1.5 Create `internal/inventory/secret.go` with `MarshalToSecret`, `UnmarshalFromSecret`, `SecretName`, and `InventoryLabels` — handle both `stringData` (marshal) and `data` (unmarshal from K8s GET)
- [x] 1.6 Create `internal/inventory/types_test.go` — roundtrip serialization (marshal then unmarshal), identity equality (version excluded), K8s identity equality (component excluded), Secret name computation, label correctness, empty inventory edge case, unmarshal from base64-encoded `data` field
- [x] 1.7 Run `task check` — verify fmt, vet, and all tests pass

## 2. Deterministic Manifest Digest (sub-issue #35)

- [x] 2.1 Create `internal/inventory/digest.go` with `ComputeManifestDigest(resources []*build.Resource) string` and a 5-key sort function (weight, group, kind, namespace, name)
- [x] 2.2 Update `internal/build/pipeline.go` sort to use the same 5-key total ordering (replace weight-only sort)
- [x] 2.3 Create `internal/inventory/digest_test.go` — same resources in different order produce same digest, content change produces different digest, added/removed resource changes digest, benchmark for typical module size (10-20 resources)
- [x] 2.4 Run `task check` — verify existing tests still pass with updated pipeline sort

## 3. Change ID & History Management (sub-issue #36)

- [x] 3.1 Create `internal/inventory/changeid.go` with `ComputeChangeID(modulePath, moduleVersion, values, manifestDigest string) string`, `UpdateIndex(index []string, changeID string) []string`, `PruneHistory(secret *InventorySecret, maxHistory int)`, and `PrepareChange(module ModuleRef, values string, manifestDigest string, entries []InventoryEntry) (string, *ChangeEntry)`
- [x] 3.2 Create `internal/inventory/changeid_test.go` — deterministic ID, version bump produces different ID, idempotent re-apply moves to front, pruning removes oldest, local module uses empty version
- [x] 3.3 Run `task check`

## 4. Inventory Secret CRUD Operations (sub-issue #37)

- [x] 4.1 Create `internal/inventory/crud.go` with `GetInventory(ctx, client, name, namespace, releaseID)`, `WriteInventory(ctx, client, inventorySecret)`, and `DeleteInventory(ctx, client, name, namespace, releaseID)`
- [x] 4.2 Implement GET with name-convention lookup, fallback to label-based list, and `nil, nil` for first-time apply
- [x] 4.3 Implement Write with full PUT semantics — create if new, update with `resourceVersion` for optimistic concurrency
- [x] 4.4 Implement Delete with idempotent 404 handling
- [x] 4.5 Update `DiscoverResources()` in `internal/kubernetes/discovery.go` to exclude Secrets with `opmodel.dev/component: inventory` label
- [x] 4.6 Create `internal/inventory/crud_test.go` — unit tests with fake clientset: GET by name, GET fallback to label, first-time nil return, create new, update existing, optimistic concurrency conflict, idempotent delete
- [x] 4.7 Run `task check`

## 5. Apply Flow with Pruning & Safety Checks (sub-issue #38)

- [x] 5.1 Create `internal/inventory/stale.go` with `ComputeStaleSet(previous, current []InventoryEntry) []InventoryEntry` and `ApplyComponentRenameSafetyCheck(stale, current []InventoryEntry) []InventoryEntry`
- [x] 5.2 Add `PreApplyExistenceCheck(ctx, client, entries []InventoryEntry) error` — GET each resource, fail on terminating or untracked
- [x] 5.3 Add `PruneStaleResources(ctx, client, stale []InventoryEntry) error` — reverse weight order, 404-as-success, exclude Namespaces by default
- [x] 5.4 Create `internal/inventory/stale_test.go` — resource removed, resource renamed, first-time empty stale, idempotent re-apply empty stale, component rename safety check filters correctly, genuine removal not filtered
- [x] 5.5 Add `--no-prune`, `--max-history`, and `--force` flags to `internal/cmd/mod_apply.go`
- [x] 5.6 Wire the 8-step apply flow in `mod_apply.go`: render, compute digest, compute change ID, read inventory, compute stale + safety checks, apply, prune (if success), write inventory (if success)
- [x] 5.7 Add empty render safety gate — fail when render produces 0 resources with non-empty previous inventory unless `--force`
- [x] 5.8 Integration test: rename scenario (old resources pruned, new applied)
- [x] 5.9 Integration test: partial failure (no prune, no inventory write)
- [x] 5.10 Integration test: first-time apply (pre-apply check, inventory created)
- [x] 5.11 Integration test: idempotent re-apply (same change ID, no stale)
- [x] 5.12 Run `task check`

## 6. Diff/Delete/Status with Inventory-First Discovery (sub-issue #39)

- [x] 6.1 Add `DiscoverResourcesFromInventory(ctx, client, inv *InventorySecret) (live, missing)` to `internal/kubernetes/discovery.go`
- [x] 6.2 Update `internal/kubernetes/diff.go` — inventory-first orphan detection in `findOrphans()` with label-scan fallback and debug log
- [x] 6.3 Update `internal/kubernetes/delete.go` — inventory-first enumeration, delete inventory Secret last, no derived resources (fixes #16)
- [x] 6.4 Update `internal/kubernetes/status.go` — inventory-first discovery, component grouping from inventory entries, show "Missing" for resources no longer on cluster
- [x] 6.5 Integration test: diff with inventory (orphan from set difference)
- [x] 6.6 Integration test: diff without inventory (label-scan fallback)
- [x] 6.7 Integration test: delete with inventory (no Endpoints deleted, inventory Secret deleted last)
- [x] 6.8 Integration test: status with inventory (component grouping, missing resource shown)
- [x] 6.9 Run `task check`

## 7. Final Validation

- [x] 7.1 Run full `task check` (fmt + vet + test)
- [x] 7.2 Run `task test:coverage` and verify new code has adequate coverage
