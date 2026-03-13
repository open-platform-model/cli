## Context

`internal/workflow/ownership` contains a single function `EnsureCLIMutable` that checks whether a release was created by a controller and blocks CLI mutation if so. It imports `internal/inventory.ReleaseInventoryRecord` — a type from the internal inventory CRUD layer.

The challenge: moving `EnsureCLIMutable` to `pkg/ownership/` means it can't import `internal/inventory`. The function needs refactoring to accept a public interface or simpler parameters.

## Goals / Non-Goals

**Goals:**
- Move ownership policy to `pkg/ownership/`
- Refactor to not depend on `internal/inventory` types
- Keep the same policy semantics

**Non-Goals:**
- Move `internal/inventory` to `pkg/`
- Change the ownership policy itself

## Decisions

### Refactor to accept a simple string parameter
Instead of accepting `*inventory.ReleaseInventoryRecord`, refactor to accept the `createdBy` string value directly (or a small struct with release name, namespace, and createdBy). The caller extracts these from whatever inventory representation they use.

**Alternative considered**: Define an interface `type OwnershipRecord interface { NormalizedCreatedBy() string; ReleaseName() string; ReleaseNamespace() string }`. This is over-engineered for a single function.

**Decision**: Accept `createdBy`, `releaseName`, `releaseNamespace` as plain parameters. Simple, no dependency on any inventory type.

```go
func EnsureCLIMutable(createdBy, releaseName, releaseNamespace string) error
```

The caller in the CLI workflow extracts these from `inventory.ReleaseInventoryRecord` before calling.

### Define createdBy constants in pkg/ownership
Move or duplicate the `CreatedByController` and `CreatedByCLI` constants here so the package is self-contained.

## Risks / Trade-offs

- **[Low risk]**: Signature change requires updating callers. Only 1-2 callers exist.
- **[Trade-off]**: Slightly more verbose at the call site (extract fields before calling). Acceptable for the gain of zero internal dependencies.
