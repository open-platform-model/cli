## 1. Flag & Namespace Cleanup

- [x] 1.1 Update `ReleaseSelectorFlags.AddTo` in `internal/cmdutil/flags.go` — change `--namespace` description from `"Target namespace (required)"` to `"Target namespace (default: from config)"`

## 2. `mod status` — Inventory-Only Discovery

- [x] 2.1 In `internal/cmd/mod/status.go` `runStatus`: replace the `if rsf.ReleaseID != ""` inventory block with a switch that handles both `--release-id` (via `inventory.GetInventory`) and `--release-name` (via `inventory.FindInventoryByReleaseName`)
- [x] 2.2 After inventory lookup: if `inv == nil`, return `noResourcesFoundError` (renders as `"release '<name>' not found in namespace '<ns>'"`) — respect `--ignore-not-found`
- [x] 2.3 Always call `inventory.DiscoverResourcesFromInventory` and populate `statusOpts.InventoryLive` and `statusOpts.MissingResources` — remove all label-scan fallback comments/debug messages
- [x] 2.4 Update `Long` doc string — remove "Resources are discovered via OPM labels" and "The --namespace flag is always required"

## 3. `mod delete` — Inventory-Only Discovery

- [x] 3.1 In `internal/cmd/mod/delete.go` `runDelete`: after the inventory lookup switch, if `inv == nil` return `noResourcesFoundError` — respect `--ignore-not-found`
- [x] 3.2 Remove all label-scan fallback comments and debug messages referencing "label-scan"
- [x] 3.3 Update `Long` doc string — remove "Resources are discovered via OPM labels" and "The --namespace flag is always required"

## 4. `mod diff` — Inventory-Only Orphan Detection

- [x] 4.1 In `internal/cmd/mod/diff.go` `runDiff`: simplify the inventory block — attempt `inventory.GetInventory`; if `inv != nil` set `diffOpts.InventoryLive`; if `inv == nil` leave `diffOpts.InventoryLive` as nil (no fallback, no debug message about label-scan)
- [x] 4.2 Remove debug log messages referencing "label-scan" from `diff.go`

## 5. `kubernetes/` Layer — Remove Label-Scan Branches

- [x] 5.1 In `internal/kubernetes/status.go` `GetReleaseStatus`: remove the `else` branch that calls `DiscoverResources` — resources is now always `opts.InventoryLive` (may be nil/empty). Update doc comment.
- [x] 5.2 In `internal/kubernetes/delete.go` `Delete`: remove the `else` branch that calls `DiscoverResources` — resources is now always `opts.InventoryLive`. Update doc comment.
- [x] 5.3 In `internal/kubernetes/diff.go` `findOrphans`: remove the `else` branch that calls `DiscoverResources` — `liveResources` is now always `inventoryLive` (nil = empty set = no orphans). Update doc comment.
- [x] 5.4 Remove unused imports of `DiscoverResources` from the three files above (if no longer imported elsewhere in each file)

## 6. Integration Tests

- [x] 6.1 In `tests/integration/inventory-ops/main.go`: remove scenario 6.6 (lines ~59–85, "Diff without inventory — label-scan fallback"). Update the file header comment to remove the 6.6 entry. Renumber `step()` calls if needed.
- [x] 6.2 In `tests/integration/deploy/main.go`: replace the two `kubernetes.DiscoverResources` calls (resource verification after apply, and verification after delete) with inventory-first discovery — write inventory after apply, use `inventory.FindInventoryByReleaseName` + `inventory.DiscoverResourcesFromInventory` to verify resources, clean up inventory Secret after the test.

## 7. Validation

- [x] 7.1 Run `task build` — binary compiles with no errors
- [x] 7.2 Run `task test` — all unit tests pass
- [x] 7.3 Run `task check` — fmt + vet + test all pass (28 pre-existing lint issues, none new)
