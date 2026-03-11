## 1. Public inventory package extraction

- [x] 1.1 Create `pkg/inventory` and move the reusable inventory model, Secret codec, naming helpers, change-history helpers, and identity helpers out of `internal/inventory`.
- [x] 1.2 Update CLI packages and tests to import `pkg/inventory` for the moved contract types/helpers without changing current behavior.
- [x] 1.3 Keep Kubernetes client operations in internal packages and add/adjust package-level docs so the public/internal boundary is explicit.

## 2. Provenance metadata and compatibility

- [x] 2.1 Extend release inventory metadata with `createdBy` support and preserve the existing Secret name, labels, JSON keys, and backward-compatible unmarshaling behavior.
- [x] 2.2 Ensure new CLI-created inventories write `createdBy=cli` and existing inventories without the field are interpreted as legacy CLI-managed releases.
- [x] 2.3 Add unit coverage for round-trip serialization, legacy inventories without `createdBy`, and metadata write-once preservation on updates.

## 3. Ownership enforcement in CLI workflows

- [x] 3.1 Add shared ownership resolution/check helpers for mutating workflows so controller-managed releases are rejected before apply/delete side effects.
- [x] 3.2 Update `opm mod apply` and `opm mod delete` to block controller-managed releases with clear user-facing errors.
- [x] 3.3 Add workflow/integration coverage for CLI-owned, legacy, and controller-owned release mutation paths.

## 4. Ownership visibility in read-only commands

- [x] 4.1 Update `opm mod list` summaries and outputs to show release ownership in table and structured formats.
- [x] 4.2 Update `opm mod status` headers/output to show ownership and warn when a release is controller-managed.
- [x] 4.3 Add tests covering ownership display for CLI-managed, legacy, and controller-managed inventories.

## 5. Validation

- [x] 5.1 Run focused unit and integration tests covering inventory, apply/delete workflows, and list/status output.
- [x] 5.2 Run `task fmt`, `task lint`, and `task test` and fix any failures.
