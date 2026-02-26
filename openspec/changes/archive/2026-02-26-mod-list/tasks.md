## 1. Export Health Primitives (internal/kubernetes)

- [x] 1.1 Export `healthStatus` type as `HealthStatus` and all constants (`HealthReady`, `HealthNotReady`, `HealthComplete`, `HealthUnknown`, `HealthMissing`, `HealthBound`) in `health.go`
- [x] 1.2 Export `evaluateHealth` as `EvaluateHealth` in `health.go`
- [x] 1.3 Add `IsHealthy(HealthStatus) bool` helper function in `health.go`
- [x] 1.4 Add `QuickReleaseHealth(resources []*unstructured.Unstructured, missingCount int) (HealthStatus, int, int)` in `health.go`
- [x] 1.5 Update all internal references in `status.go` and `tree.go` to use exported names
- [x] 1.6 Run `task test:unit` to verify no regressions from rename

## 2. Inventory Listing (internal/inventory)

- [x] 2.1 Add `ListInventories(ctx context.Context, client *kubernetes.Client, namespace string) ([]*InventorySecret, error)` in new file `list.go`
- [x] 2.2 Use `core.LabelManagedBy`/`core.LabelComponent` constants for label selector
- [x] 2.3 Handle unmarshal failures gracefully (log warning, skip corrupt Secrets)
- [x] 2.4 Sort results by `ReleaseMetadata.ReleaseName`
- [x] 2.5 Support all-namespaces via empty string namespace (K8s `Secrets("").List(...)` convention)

## 3. List Command (internal/cmd/mod)

- [x] 3.1 Create `list.go` with `NewModListCmd(cfg *config.GlobalConfig)` following existing command patterns
- [x] 3.2 Add flags: `-n`/`--namespace`, `-A`/`--all-namespaces`, `-o`/`--output`, `--kubeconfig`, `--context` (reuse `K8sFlags`)
- [x] 3.3 Implement `runList` function: resolve config, create K8s client, call `ListInventories`
- [x] 3.4 Implement parallel health evaluation with bounded worker pool (resource discovery + `QuickReleaseHealth` per release)
- [x] 3.5 Extract display metadata from each inventory: release name, module name, version (from latest change), release ID, last applied time, age
- [x] 3.6 Implement table output: NAME, MODULE, VERSION, STATUS, AGE columns; prepend NAMESPACE with `-A`
- [x] 3.7 Implement wide output: add RELEASE-ID (full UUID) and LAST-APPLIED columns
- [x] 3.8 Implement JSON and YAML structured output with all fields
- [x] 3.9 Handle empty result: print message and exit 0
- [x] 3.10 Register command in `mod.go` via `c.AddCommand(NewModListCmd(cfg))`

## 4. Integration Tests

- [x] 4.1 Create `tests/integration/mod-list/main.go` following existing integration test patterns (`//go:build ignore`, `kind-opm-dev` cluster)
- [x] 4.2 Implement setup: create test namespaces, deploy multiple releases with inventory Secrets and backing resources (ConfigMaps, Services)
- [x] 4.3 Scenario: list in specific namespace returns correct count and release names
- [x] 4.4 Scenario: list all namespaces returns releases from all test namespaces
- [x] 4.5 Scenario: list in empty namespace returns zero results
- [x] 4.6 Scenario: health status accuracy — delete a resource, verify NotReady status and correct ready/total counts
- [x] 4.7 Scenario: metadata correctness — verify module name, version, release ID match what was written
- [x] 4.8 Implement cleanup: delete all test resources, inventory Secrets, and test namespaces
- [x] 4.9 Add test to Taskfile integration test target if needed

## 5. Validation

- [x] 5.1 Run `task fmt` and fix any formatting issues
- [x] 5.2 Run `task lint` and fix any linter warnings (9 issues fixed in our files; 51 pre-existing remain)
- [x] 5.3 Run `task test:unit` and verify all unit tests pass (20/20 packages pass)
- [x] 5.4 Run `task test:integration` (with `kind-opm-dev` cluster) and verify mod-list integration tests pass
