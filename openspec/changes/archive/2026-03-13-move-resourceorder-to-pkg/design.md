## Context

`internal/resourceorder` is a pure data package — a GVK-to-weight lookup table and one exported function (`GetWeight`). It depends only on `k8s.io/apimachinery`. It has 4 callers, all in `internal/`. Independent of the `pkg/render/` merge.

## Goals / Non-Goals

**Goals:**
- Move to `pkg/resourceorder/` with no changes
- Update all 4 callers

**Non-Goals:**
- Add or change any weights
- Rename `GetWeight` or any constants

## Decisions

### Keep package name as-is
`resourceorder.GetWeight(gvk)` reads well as a public API. No rename needed.

### Straight move
No refactoring, no API changes. Copy files, change package declaration, update imports.

## Risks / Trade-offs

- **[None]**: Trivial mechanical change. 4 files to update, 1 exported function, 0 behavior changes.
