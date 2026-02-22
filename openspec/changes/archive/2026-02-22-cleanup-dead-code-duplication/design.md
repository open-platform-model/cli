## Context

A full static analysis of the CLI codebase identified four categories of debt:

1. **Correctness bug** — `inventory/stale.go` has its own `kindToResource` that only
   appends `"s"`, producing wrong GVRs for types like `NetworkPolicy` and `StorageClass`.
   `kubernetes/resource.go` already has a correct, comprehensive implementation.

2. **Dead code** — Eight functions/methods unreachable from the main binary entry point
   (`ComputeReleaseUUID`, `ValidFormats`, `ListTemplateFiles`, `SetLogWriter`,
   `rewriteErrorPath`, `Info.String`, `DeleteInventory`, the builder no-op block),
   three struct fields never populated (`ModuleMetadata.Annotations`,
   `ReleaseMetadata.Annotations`, `ModuleMetadata.Labels`), and two CLI flags that
   are registered, help-text-documented, and silently ignored (`--wait`, `--timeout`).

3. **Copy-paste duplication** — The inventory-resolution switch block (28 lines)
   is byte-for-byte identical in `mod delete` and `mod status`. The 5-key resource
   sort comparator is independently implemented in both `pipeline` and `inventory`.
   The CUE `map[string]string` extraction loop is triplicated across `loader` and `builder`.

4. **Structural redundancy** — `TransformerComponentMetadata` is a copy of
   `ComponentMetadata` with only `omitempty` JSON tags differing.
   `TransformerMetadata.APIVersion` duplicates `Transformer.APIVersion`.

The change is a cross-cutting sweep touching 15+ packages. There are no user-visible
behavioral changes other than removing the non-functional `--wait`/`--timeout` flags.

## Goals / Non-Goals

**Goals:**
- Fix the `kindToResource` correctness bug in `inventory/stale.go`
- Remove all identified dead code; reduce binary surface
- Eliminate the two highest-impact copy-paste patterns (inventory resolution, resource sort)
- Extract reusable CUE string-map helper to avoid further triplication
- Remove non-functional CLI flags to avoid misleading users
- All existing tests continue to pass; `task test` green throughout

**Non-Goals:**
- Refactoring the dual `ValidationError` / `DetailError` types (separate analysis needed)
- Rationalizing the `Resource` dual accessor API (Get* vs plain) — add `GetLabels()` only
- Any behavioral changes to how commands work
- Performance optimization

## Decisions

### D1: Export `KindToResource` from `kubernetes` rather than moving it to `core`

`inventory` already imports `kubernetes`. Exporting from `kubernetes/resource.go`
requires zero new imports. Moving to `core` would require changing the `kubernetes`
package to also import from `core` and could create circular import chains.

_Alternative considered_: Put it in `internal/core` as a pure utility. Rejected
because `core` currently has no external package dependencies and adding string
pluralization logic there is out of scope for that package.

### D2: Circular import check before consolidating resource sort

Before `pipeline` imports `inventory.SortResources`, verify import graph direction.
If `inventory` imports `pipeline` (for types), consolidation moves `SortResources`
to `internal/core` instead. This must be checked during implementation.

_If circular_: Move `SortResources` to `core/resource.go` (already has `Resource`
type and `GetWeight`). Both `inventory` and `pipeline` already import `core`.

### D3: `cmdutil.ResolveInventory` takes a logger, not a context+log pair

The duplicated block in `delete.go` and `status.go` uses a `releaseLog` scoped to
the release name. The helper signature should be:

```go
func ResolveInventory(
    ctx context.Context,
    client *kubernetes.Client,
    rsf *ReleaseSelectorFlags,
    namespace string,
    ignoreNotFound bool,
    log *log.Logger,
) (*inventory.InventorySecret, []*core.Resource, error)
```

Returns discovered resources alongside the inventory Secret so callers don't need a
second call to `DiscoverResourcesFromInventory`. This collapses items 3.2 and 3.3 from
the plan into a single helper.

### D4: `SetLogWriter` moved to `output/export_test.go`, not deleted

`SetLogWriter` needs to touch the unexported `logger` package variable. The standard
Go pattern is `export_test.go` — a file that is compiled only during testing and can
expose internals without polluting the production binary. This keeps test isolation
without copying the function into every test file.

### D5: Dead `ModuleMetadata.Labels` — remove the field and its setter

`ModuleMetadata.Labels` is set in the loader (`extractModuleMetadata`) but never
consumed downstream (a TODO notes it should flow to CUE via `ToMap()`, but that
wiring was never implemented). Since the field is set-but-unused, removing it means
also removing the assignment in the loader. If module-level labels are needed in the
future, they will be re-introduced with full wiring at that time.

### D6: `TransformerComponentMetadata` → embed `*component.ComponentMetadata`

The two structs are functionally identical. The `omitempty` difference is in JSON
serialization only. The `TransformerContext.Components` map holds
`TransformerComponentMetadata` values. Replacing with `component.ComponentMetadata`
requires no interface changes — the type is only used internally within the
transformer context.

### D7: `DeleteInventory` — remove, document why

`mod delete` does not call `DeleteInventory` because the inventory Secret is tracked
as a regular resource in the inventory itself and is therefore deleted automatically
via the standard resource deletion path. The function is dead because the deletion
mechanism is correct. It will be removed with a code comment at the call site explaining
the pattern, so future contributors don't re-add it.

### D8: Orphaned `--wait` / `--timeout` flags — remove cleanly, no deprecation cycle

The flags were registered but the handler body was written with blank identifiers
(`_ /* wait */`). They never had any effect. No deprecation cycle needed.

## Risks / Trade-offs

- **Sort alignment risk** (`sortResourceInfos` in `output/manifest.go`)  
  This is a 3-key variant of the 5-key sort (omits group + kind). After consolidation
  it will explicitly document whether the divergence is intentional. If the output
  ordering differs from apply ordering in some edge case, that's a display-only issue
  with no operational impact.  
  → Mitigation: add a comment explaining the intentional divergence, or align to 5 keys.

- **Transformer context contract**  
  Replacing `TransformerComponentMetadata` with `*component.ComponentMetadata` changes
  the type stored in `TransformerContext.Components`. If any in-progress change
  (`transformer-matching-v2`, `mod-tree`) references this type directly, it will need
  to update its type assertions.  
  → Mitigation: check all in-progress changes for references before implementing D6.

- **`apply.go` behavior change** (`ResolveModulePath` default `""` → `"."`)  
  The inline extraction uses `""` as the default module path; `ResolveModulePath`
  returns `"."`. This affects `inventory.ComputeChangeID`. If the change ID
  computation is path-sensitive, stored inventory IDs for modules loaded with an
  explicit path could differ.  
  → Mitigation: read `ComputeChangeID` and `inventory.changeid.go` before changing;
  confirm `"."` and `""` resolve identically in that context.

## Open Questions

- None — all decisions made above. The circular import check (D2) is a
  verification step during implementation, not a design decision.
