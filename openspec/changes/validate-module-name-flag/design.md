## Context

OPM module names originate from CUE `metadata.name` and flow through the Go pipeline into Kubernetes labels. The companion catalog change (`k8s-name-constraints`) redefines `#NameType` to enforce RFC 1123 DNS label format (lowercase alphanumeric with hyphens, max 63 characters) at the CUE schema level. Names loaded from CUE are therefore guaranteed valid.

However, the `--name` CLI flag bypasses CUE validation entirely. Without independent validation, `opm mod apply ./blog --name MyBlog` would inject an invalid name into labels.

Additionally, `mod delete` currently returns exit 0 with "0 resources deleted" when no matching module exists — a silent no-op that masks real errors.

Current flow: `CUE name → Go ModuleMetadata.Name → injectLabels() → label value` — no validation of the `--name` flag at any step.

## Goals / Non-Goals

**Goals:**

- `--name` flag values are validated against RFC 1123 DNS label format; invalid values are rejected with an actionable error.
- `mod delete` exits non-zero with an actionable error when the module is not found on the cluster.

**Non-Goals:**

- Normalizing or transforming the `--name` flag value (e.g., converting `MyBlog` to `my-blog`). Invalid input is rejected, not fixed.
- Normalizing delete `--name` input. The lookup remains strict and case-sensitive.
- Smart module discovery or "did you mean?" suggestions for delete.
- Changing how module names appear in log output or display.
- CUE schema changes — handled by catalog change `k8s-name-constraints`.

## Decisions

### D1: CUE is the source of truth for name format

**Choice:** Do not normalize names in Go. The catalog's `#NameType` constraint enforces RFC 1123 DNS labels at definition time. Names arriving in Go from CUE are already valid.

**Alternatives considered:**

- *Normalize in `injectLabels()` via `strings.ToLower()`*: Would silently convert names, but cannot insert hyphens for PascalCase→kebab-case conversion. Even a full PascalCase→kebab conversion would duplicate CUE's logic in Go, creating two sources of truth that can diverge. Rejected.
- *Normalize in the loader*: Would lowercase `ModuleMetadata.Name` everywhere — logs, display output, status. Changes the module's identity as the author defined it. Rejected.

**Rationale:** Single source of truth. CUE defines the constraint; Go trusts it. No transformation logic duplicated across repositories.

### D2: Validate `--name` flag against RFC 1123 format

**Choice:** Validate `opts.Name` in `applyOverrides()` (`module.go:237`) using the same regex as CUE's `#NameType`: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` with max 63 characters. Return a validation error with an actionable message if invalid.

**Error message example:**

```
Error: validation failed
  Field: --name

  invalid module name "MyBlog": must be a DNS label (lowercase alphanumeric with hyphens, 1-63 characters)

Hint: use kebab-case, e.g., "my-blog"
```

**Alternatives considered:**

- *Silently convert `--name` to kebab-case*: Would duplicate the PascalCase→kebab conversion from the catalog. Also violates the principle that explicit is better than implicit — the user should know what name ends up on the cluster. Rejected.
- *No validation at all — let it fail at the cluster level*: The label value would be invalid per Kubernetes conventions but might still be applied. Failing early with a clear message is better. Rejected.

**Rationale:** The `--name` flag is the one input path that bypasses CUE validation. A regex check at this boundary is minimal, explicit, and keeps Go in sync with CUE's constraint without duplicating transformation logic.

### D3: Return a sentinel `ErrNotFound` from `Delete()` when 0 resources discovered

**Choice:** When `DiscoverResources()` returns an empty slice, `Delete()` returns a `not found` error wrapping `ErrNotFound`, instead of `nil`.

**Alternatives considered:**

- *Keep returning nil, handle in `runDelete()`*: The command layer would check `result.Deleted == 0` and decide the exit code. This works but pushes domain knowledge (is 0-resources an error?) into the command layer. The business logic should express that deleting nothing is not success.
- *New sentinel error type*: Considered `ErrModuleNotFound` but the existing `ErrNotFound` + `NewNotFoundError()` already provide structured error output with a `Hint` field. No new types needed.

**Rationale:** `Delete()` is the authority on what happened. Returning `nil` for "nothing to delete" is misleading. Using `ErrNotFound` maps to exit code 5 (`ExitNotFound`) via the existing `ExitCodeFromError()` function — no new exit codes or error routing needed.

### D4: Use `NewNotFoundError()` with a hint

**Choice:** The error message includes the module name and namespace, and the hint reminds users that module names are case-sensitive and must be lowercase kebab-case.

**Rationale:** The most common cause of "not found" will be casing mismatch during the transition period. A good hint saves the user a debugging cycle.

## Risks / Trade-offs

**[Stricter `--name` validation]** → Scripts or users passing PascalCase names via `--name` will get errors instead of silent acceptance. Mitigated by: the catalog change enforces the same constraint in CUE, so the entire system is moving to kebab-case. Clear error messages guide users to the correct format.

**[Delete is now stricter]** → Scripts or CI pipelines that run `opm mod delete` and expect exit 0 even when the module doesn't exist will break. Mitigated by: this is the correct behavior — silent no-ops are a bug, not a feature. Any script relying on this was masking real errors.

**[Regex duplication between CUE and Go]** → The RFC 1123 regex exists in both CUE (`#NameType`) and Go (validation function). Mitigated by: this is a well-known, stable standard. The regex is unlikely to change. If it does, the CUE schema is authoritative and the Go regex is a defensive guard, not a source of truth.
