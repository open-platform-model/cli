## REMOVED Requirements

### Requirement: Internal release parsing returns a barebones BundleRelease without validation
**Reason**: The bundle path is unreachable dead code (enhancement 0002 D15, supersedes D7). No `internal/cmd` command targets it, and the only producer of `kind: "BundleRelease"` is the deprecated `catalog/core/v1alpha1/bundlerelease` module — no bundle kind exists in `core`/`catalog_opm`/`catalog_kubernetes`. The internal `GetInstanceFile` bundle parse arm and helpers (`bareBundleRelease`, `mustBundleReleaseMetadata`, `bestEffortBundleMetadata`) are deleted.
**Migration**: None. Bundle files were never rendered — they were parsed only to be rejected. A `kind: "BundleRelease"` file now fails earlier at kind detection with `unknown instance kind: "BundleRelease"`. If bundle support is built in future, this capability is reintroduced under the Instance vocabulary (`BundleInstance`).

### Requirement: BundleRelease exposes processing-stage fields
**Reason**: The backing type `pkg/bundle.Release` (with `Metadata`, `Bundle`, `Releases`, `Config`, `Values`) is deleted along with the entire `pkg/bundle` package.
**Migration**: None — no production code constructed or consumed this type.

### Requirement: ProcessBundleRelease validates bundle values and establishes the public API shape
**Reason**: `pkg/render.ProcessBundleRelease` was a validate-then-`not implemented yet` stub with zero production callers (referenced only from `pkg/render/process_test.go`). It is deleted with `pkg/render/process_bundlerelease.go`.
**Migration**: None. Module instances continue to be processed by `ProcessModuleInstance`; there is no bundle equivalent until bundle support is built.
