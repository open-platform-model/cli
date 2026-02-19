## Context

The OPM codebase models two distinct concepts:

- **Module** — a CUE definition directory on disk. Has a path, a name from `metadata.name`, schema definitions. Immutable from the deployer's perspective.
- **ModuleRelease** — a concrete deployed instance of a Module, produced by combining a Module with user-supplied values. Has a release name (from `--release-name`), a release UUID, namespace, and version. Lives on a cluster.

A thorough audit identified 27 naming violations where Go identifiers use "Module" when they mean "ModuleRelease". These violations span function names, type names, field names, variable names, log messages, and error strings across 15 files. The violations actively mislead contributors reading or extending the code — particularly around the `inventory` package where `ModuleRef.Name` stores a release name while being called `Name` on a type called `ModuleRef`.

This is a pure refactor. No behavior changes, no flag changes, no user-visible message changes (with the deliberate exception of internal debug/log messages).

## Goals / Non-Goals

**Goals:**
- All Go identifiers that represent release concepts are named with "Release" where appropriate
- `InventoryMetadata` fields unambiguously identify whether they hold a module name or a release name/namespace
- The `ModuleRef` type is renamed and restructured to reflect its actual role
- Dead code (`build.ModuleValidationError`) is removed
- Internal log and debug messages accurately describe what they're logging
- The `m:` terminal prefix and all user-facing messages remain unchanged

**Non-Goals:**
- JSON wire format changes — existing inventory Secrets on clusters must remain readable
- User-facing message changes (command descriptions, success messages, `m:` prefix)
- Structural/architectural changes (no file moves, no package restructuring)
- Changes that overlap with `reorganize-cmd-pkg` structurally (that change moves files; this one renames identifiers within them)

## Decisions

### Preserve all JSON tags

**Decision**: All JSON struct tags remain unchanged even when the Go field name changes.

**Rationale**: Inventory Secrets already exist on clusters serialized with the current JSON keys (`"name"`, `"namespace"`, `"module"`). Changing JSON tags would silently break deserialization of existing inventory data. The Go identifier is what matters for code clarity; the JSON tag is the wire format.

**Alternative considered**: Add a migration path (read old tags, write new tags). Rejected — the complexity is unjustified for a pure naming refactor. The internal confusion exists in Go source, not in serialized data.

### Rename `ModuleRef` → `ChangeSource`, rename field `Name` → `ReleaseName`

**Decision**: The inventory type `ModuleRef` becomes `ChangeSource`, the field `Name` becomes `ReleaseName`. The `ChangeEntry.Module` field becomes `ChangeEntry.Source`. JSON tags (`"module"`, `"name"`) are unchanged.

**Rationale**: `ModuleRef` suggests a reference to a Module. It actually records the source context of a change entry — the module path, the module version, and crucially the *release name* (not module name) under which this change was applied. `ChangeSource` reflects this accurately. The `Name` → `ReleaseName` rename eliminates the comment that currently apologizes: *"Name is the module release name, not the canonical module definition name."* With `ReleaseName`, no comment is needed.

**Alternative considered**: Keep `ModuleRef`, only rename the field. Rejected — the type name itself is the primary source of confusion. Someone reading `module := inventory.ModuleRef{...}` in mod_apply.go would reasonably assume the variable holds module data.

### Rename `output.ModuleLogger` → `output.ReleaseLogger`, keep `m:` prefix

**Decision**: The Go function is renamed `ReleaseLogger`. The rendered `m:<name>:` prefix in terminal output is unchanged. Local variables `modLog` become `releaseLog`.

**Rationale**: The function is always called with a release name, never a module name. Renaming the Go function fixes the internal confusion without changing user-visible output. The `m:` prefix is intentional UX — users think in terms of "my module deployment" and changing it to `r:` would be unexplained noise in terminal output.

### Remove `build.ModuleValidationError`

**Decision**: Delete the `ModuleValidationError` type from `internal/build/errors.go` and its tests.

**Rationale**: The type is defined but never instantiated anywhere in the codebase. It is not type-asserted against anywhere. Its intended role is covered by `build.ReleaseValidationError` (an alias for `release.ValidationError`). Keeping it creates false expectations for future contributors who might try to use it.

### Fix `"module validation failed"` error message

**Decision**: Change the string `"module validation failed"` in `release/builder.go` to `"release validation failed"`.

**Rationale**: This message is set on a `release.ValidationError` that fires during release building — after values injection, when checking that the concrete release tree has no CUE errors. The *module* definition may be perfectly valid; this error means the specific release (module + values) is invalid. The message `"release validation failed"` is accurate.

**Exception**: The message `"module missing 'values' field..."` is intentionally preserved — in that case, it is correct that the *module* definition is missing the field. This is a module-level problem, not a release-level problem.

## Risks / Trade-offs

**[Risk] Two changes touching the same files** → The `reorganize-cmd-pkg` change also modifies `internal/cmd/mod_*.go`. If that change lands before this one, this change's callsite updates target `internal/cmd/mod/*.go` instead of `internal/cmd/mod_*.go`. If this lands first, `reorganize-cmd-pkg` moves the already-renamed code. Both orderings work — the renames are within function bodies, not structural. Mitigation: note the dependency in tasks.

**[Risk] JSON tag / Go field name mismatch is confusing** → After this change, `ChangeSource.ReleaseName` serializes to `"name"` in JSON. This is a deliberate tradeoff (see Decision 1). The mismatch is documented with a comment on the field. Mitigation: add explicit comments to each field whose JSON tag differs from the Go name.

**[Risk] Missing a callsite** → With 15 files and 27 violations, a grep-and-replace approach could miss usages. Mitigation: use `task build` and `task test` as validation gates. The compiler will catch any missed rename.

## Open Questions

None. All decisions are made.
