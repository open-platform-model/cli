## Context

The inventory Secret currently stores release identity in a single `data.metadata` JSON blob where `"name"` holds the module name and `"releaseName"` holds the release name — inverted relative to the `kind: ModuleRelease` declaration in the same blob. The Secret labels similarly carry two module-scoped labels (`module.opmodel.dev/name`, `module.opmodel.dev/namespace`) on what is fundamentally a release object.

The fix is to split the single metadata blob into two typed, independently evolvable records — one for the release (`releaseMetadata`), one for the module (`moduleMetadata`) — and clean up the label set to be fully release-scoped.

No public CLI API (flags, commands, output) changes. This is a pure internal data model restructure.

## Goals / Non-Goals

**Goals:**

- `data.releaseMetadata["name"]` = release name
- `data.moduleMetadata["name"]` = module name
- Secret labels are fully scoped to the release (no `module.opmodel.dev/*` labels)
- `moduleMetadata` is a first-class typed record (enables future Module CRD migration)
- `releaseId` → `uuid` field rename for consistency with `moduleMetadata.uuid`
- Metadata is write-once: set at create time, preserved verbatim on all updates

**Non-Goals:**

- Migration of existing on-cluster Secrets (clean break, no backward compat)
- Exposing moduleMetadata to command output (future work)
- Changes to change entry format (`change-sha1-*` keys)

## Decisions

### Split `metadata` into `releaseMetadata` + `moduleMetadata`

**Decision**: Two separate Secret data keys, each with their own typed Go struct.

**Rationale**: The two records have different ownership and lifecycles. `releaseMetadata` is always present and required; `moduleMetadata.uuid` may be absent (local modules, older module schemas). Splitting avoids a single struct that is half-populated in some cases.

**Alternative considered**: Keep a single key but add a `moduleName` field. Rejected — it's the same confusion in a different direction, and doesn't position for future Module CRD migration cleanly.

---

### `moduleName` threaded only at create time via `WriteInventory`

**Decision**: `WriteInventory(ctx, client, inv, moduleName, moduleUUID string)` — takes module identity at the call site but only uses it when `inv` is newly created (no previous Secret). On update, `inv.ModuleMetadata` is already populated from `UnmarshalFromSecret`.

**Rationale**: Metadata is write-once. The apply path already distinguishes create (`prevInventory == nil`) from update. Threading `moduleName` only to `WriteInventory` avoids leaking it through `MarshalToSecret` internals and keeps the struct self-contained after construction.

**Alternative considered**: Store `moduleName` in `InventorySecret` as a non-serialized field (like `resourceVersion`). Rejected — it's confusing to have a field that is sometimes populated and sometimes not depending on how the struct was constructed.

---

### Remove `module.opmodel.dev/name` and `module.opmodel.dev/namespace` labels

**Decision**: Drop both labels from the inventory Secret label set. Replace `module.opmodel.dev/namespace` with `module-release.opmodel.dev/namespace`.

**Rationale**: `module.opmodel.dev/name` is never used as a selector in production code — it was informational only. The module name is now in `moduleMetadata` for anyone who needs it programmatically. `module.opmodel.dev/namespace` was a namespace label on a release object, which is semantically wrong.

**Impact on `LabelModuleName` constant in `kubernetes/discovery.go`**: Remove the constant. `LabelModuleNamespace` already points to `module-release.opmodel.dev/namespace` — verify and update if needed.

---

### `moduleMetadata.uuid` is omitempty

**Decision**: `json:"uuid,omitempty"` — the field is silently absent when the module has no identity UUID (local modules, test fixtures without upstream schema).

**Rationale**: Empty UUIDs are not actionable. Absence is cleaner than a zero-value string.

## Risks / Trade-offs

- **Existing on-cluster Secrets become unreadable** — `UnmarshalFromSecret` will fail on the old `"metadata"` key since it now looks for `"releaseMetadata"`. Any cluster with live inventory Secrets will need to re-apply all modules after this change. This is acceptable given the project is under heavy development.
  → Mitigation: Document as a known breaking change. Operators re-run `opm mod apply` for all releases.

- **`moduleMetadata` absent on first read** — if a Secret was written without `moduleMetadata` (e.g. in tests), `UnmarshalFromSecret` must treat the missing key as non-fatal (not an error).
  → Mitigation: Missing `moduleMetadata` key → zero-value `ModuleMetadata{}` struct, no error.

## Migration Plan

No cluster migration. Clean break:

1. Merge this change
2. Operators re-apply all releases with `opm mod apply` to rewrite inventory Secrets in the new format
3. Old Secrets with `data.metadata` key will fail to unmarshal — operator must delete and re-apply

Rollback: revert the commit; old Secrets are still on-cluster and readable by the old code.

## Open Questions

- None. All decisions made during exploration.
