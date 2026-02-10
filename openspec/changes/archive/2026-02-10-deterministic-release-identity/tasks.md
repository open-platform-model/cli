## 1. Shared Constants and Types

- [x] 1.1 Define OPM namespace UUID constant in a shared location (`internal/identity/identity.go`). Document its purpose and that it must match the catalog CUE constant.
- [x] 1.2 Add `LabelReleaseID = "module-release.opmodel.dev/uuid"` and `LabelModuleID = "module.opmodel.dev/uuid"` constants to `internal/kubernetes/discovery.go`.
- [x] 1.3 Add `Identity string` and `ReleaseIdentity string` fields to `build.ModuleMetadata` in `internal/build/types.go`.

## 2. Identity Extraction (Build Pipeline)

- [x] 2.1 In `release_builder.go` `extractMetadata`, extract `metadata.identity` from the CUE value via `LookupPath("metadata.identity")` into `ModuleMetadata.Identity`. Follow the existing pattern for version/FQN extraction. Empty string if field absent.
- [x] 2.2 Add a `computeReleaseIdentity` function that computes UUID v5 in Go using `github.com/google/uuid` (or equivalent). Input format: `"{fqn}:{name}:{namespace}"` with the OPM namespace UUID. Set result on `ModuleMetadata.ReleaseIdentity`.
- [x] 2.3 Add unit tests for `computeReleaseIdentity`: verify determinism, verify different inputs produce different UUIDs, verify empty FQN produces empty string (skip computation).

## 3. Identity Label Injection (Apply)

- [x] 3.1 In `internal/kubernetes/apply.go` `injectLabels`, add conditional injection of `LabelReleaseID` from `meta.ReleaseIdentity` and `LabelModuleID` from `meta.Identity`. Follow existing `if meta.Version != ""` pattern.
- [x] 3.2 Add unit tests for `injectLabels`: verify identity labels are set when non-empty, verify they are omitted when empty, verify existing user labels are preserved.

## 4. Dual-Strategy Discovery

- [x] 4.1 Define `DiscoveryOptions` struct in `internal/kubernetes/discovery.go` with fields: `ModuleName`, `Namespace`, `ReleaseID`.
- [x] 4.2 Add `BuildReleaseIDSelector(releaseID string) labels.Selector` function that builds a selector on `managed-by` + `release-id`.
- [x] 4.3 Refactor `DiscoverResources` to accept `DiscoveryOptions` instead of `(moduleName, namespace)`. Internally, run both selectors when both are available. Deduplicate by `metadata.uid`.
- [x] 4.4 Update all call sites of `DiscoverResources` to use the new signature: `delete.go`, `status.go`, `diff.go`.
- [x] 4.5 Add unit tests for `BuildReleaseIDSelector`: verify selector string output.
- [x] 4.6 Add unit tests for deduplication logic: verify union of overlapping sets produces unique results by UID.

## 5. Delete Command Changes

- [x] 5.1 Add `--release-id` flag to `mod_delete.go` (`deleteReleaseIDFlag string`).
- [x] 5.2 Remove `MarkFlagRequired("name")`. Add manual validation in `runDelete`: require `--namespace` always, require at least one of `--name` or `--release-id`. Return clear error message if neither provided.
- [x] 5.3 Pass `ReleaseID` from the flag into `DeleteOptions` and through to `DiscoveryOptions`.
- [x] 5.4 Add `ReleaseID string` field to `DeleteOptions` struct.

## 6. Status Command Changes

- [x] 6.1 Display "Module ID" and "Release ID" in `mod status` output when identity labels are present on discovered resources. Read from the first discovered resource's labels.
- [x] 6.2 Omit identity fields from output when not present (backwards compatibility).

## 7. Cross-Language Validation

- [x] 7.1 Create a test fixture CUE file that computes a known identity using `uuid.SHA1` with the OPM namespace UUID and a fixed input string.
- [x] 7.2 Add a Go test that computes the same identity in Go and asserts it matches the CUE-computed value from the fixture. This validates the OPM namespace UUID and input format are identical across languages.

## 8. Validation Gates

- [x] 8.1 Run `task fmt` — all Go files formatted.
- [x] 8.2 Run `task test` — all existing and new tests pass.
- [x] 8.3 Run `task check` — fmt + vet + test all pass. (Note: pre-existing lint issues remain in the codebase)
- [x] 8.4 Manual smoke test: apply a module with updated catalog, verify identity labels appear on cluster resources. Run `mod delete --release-id` and confirm resources are found and deleted. (Requires cluster access)
