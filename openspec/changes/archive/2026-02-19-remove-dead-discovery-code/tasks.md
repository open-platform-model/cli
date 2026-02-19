## 1. Create New Files (Extract Survivors)

- [x] 1.1 Create `internal/kubernetes/labels.go` with label constants (LabelReleaseName, LabelComponent, labelReleaseID, labelModuleID, LabelReleaseUUID, fieldManagerName, LabelManagedBy, labelManagedByValue)
- [x] 1.2 Create `internal/kubernetes/errors.go` with error types (errNoResourcesFound, noResourcesFoundError, IsNoResourcesFound, ReleaseNotFoundError)
- [x] 1.3 Create `internal/kubernetes/errors_test.go` with TestNoResourcesFoundError moved from discovery_test.go

## 2. Modify Existing Files

- [x] 2.1 Add `sortByWeightDescending` function to `internal/kubernetes/delete.go` (move from discovery.go)
- [x] 2.2 Add `TestSortByWeightDescending` and `makeUnstructured` helper to `internal/kubernetes/delete_test.go` (move from discovery_test.go)

## 3. Delete Dead Code

- [x] 3.1 Delete `internal/kubernetes/discovery.go` (entire file)
- [x] 3.2 Delete `internal/kubernetes/discovery_test.go` (entire file)
- [x] 3.3 Delete `internal/kubernetes/integration_test.go` (entire file)

## 4. Verification

- [x] 4.1 Run `task build` to verify clean compilation
- [x] 4.2 Run `task test` to verify all tests pass
- [x] 4.3 Run `go vet ./...` to verify no issues
- [x] 4.4 Verify no references to deleted functions remain using grep
