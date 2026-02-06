## Context

OPM module names originate from CUE `metadata.name` (e.g., `"Blog"`) and flow through the Go pipeline without normalization into Kubernetes labels. The companion catalog change (`k8s-name-constraints`) enforces lowercase-with-hyphens at the CUE schema level. This CLI change adds defensive normalization in Go so that:

1. Labels are always lowercase regardless of input source (CUE value, `--name` flag override).
2. `mod delete` fails explicitly when no matching resources exist, instead of exiting 0.

Current flow: `CUE name → Go ModuleMetadata.Name → injectLabels() → label value` — no transformation at any step.

## Goals / Non-Goals

**Goals:**

- Labels for `module.opmodel.dev/name` are always lowercase on the cluster.
- `--name` flag overrides (which bypass CUE validation) are normalized before label injection.
- `mod delete` exits non-zero with an actionable error when the module is not found on the cluster.

**Non-Goals:**

- Normalizing delete `--name` input. The lookup remains strict and case-sensitive against lowercase labels.
- Smart module discovery or "did you mean?" suggestions for delete. That's a separate future change.
- Changing how module names appear in log output or display — only the label value is lowercased.
- CUE schema changes — handled by catalog change `k8s-name-constraints`.

## Decisions

### D1: Normalize at the label injection boundary

**Choice:** Lowercase `meta.Name` in `injectLabels()` (`apply.go:99`), not earlier in the pipeline.

**Alternatives considered:**

- *Normalize in the loader* (`module.go`, `extractMetadataFields`): Would lowercase `ModuleMetadata.Name` everywhere — logs, display output, status. This changes the module's identity as the author defined it. Rejected because the display name should reflect what the CUE author wrote.
- *Normalize in `BuildModuleSelector()`*: Would make delete case-insensitive. Rejected because the user explicitly wants strict matching — if labels are lowercase, users must type lowercase.

**Rationale:** The label is an infrastructure concern. The module author's name choice (`"Blog"` or `"blog"`) is preserved for display, but the label value is always normalized. This is the smallest surface area change with the clearest semantics.

### D2: Also normalize the `--name` flag override

**Choice:** Lowercase `opts.Name` in `applyOverrides()` (`module.go:240`) before assigning to `module.Name`.

**Rationale:** The `--name` flag bypasses CUE validation entirely. Without this, `opm mod apply ./blog --name MyBlog` would inject `"MyBlog"` into labels, defeating the normalization. This is the one place where an override enters the pipeline outside of CUE.

Note: This does mean `module.Name` (and therefore log output like `"loaded module name=myblog"`) will show the lowercased override, not the original flag value. This is acceptable because the override is explicitly replacing the CUE-authored name — the user chose to override, so the normalized form is the correct identity.

### D3: Return a sentinel `ErrNotFound` from `Delete()` when 0 resources discovered

**Choice:** When `DiscoverResources()` returns an empty slice, `Delete()` returns a `not found` error wrapping `ErrNotFound`, instead of `nil`.

**Alternatives considered:**

- *Keep returning nil, handle in `runDelete()`*: The command layer would check `result.Deleted == 0` and decide the exit code. This works but pushes domain knowledge (is 0-resources an error?) into the command layer. The business logic should express that deleting nothing is not success.
- *New sentinel error type*: Considered a `ErrModuleNotFound` but the existing `ErrNotFound` + `NewNotFoundError()` already provide structured error output with a `Hint` field. No new types needed.

**Rationale:** `Delete()` is the authority on what happened. Returning `nil` for "nothing to delete" is misleading. Using `ErrNotFound` maps to exit code 5 (`ExitNotFound`) via the existing `ExitCodeFromError()` function — no new exit codes or error routing needed.

### D4: Use `NewNotFoundError()` with a hint

**Choice:** The error message includes the module name and namespace, and the hint reminds users that module names are case-sensitive and lowercase.

**Rationale:** The most common cause of "not found" will be casing mismatch during the transition period. A good hint saves the user a debugging cycle.

## Risks / Trade-offs

**[Breaking label values]** → Existing cluster resources with PascalCase labels (e.g., `module.opmodel.dev/name=Blog`) will not be found by delete until re-applied. Mitigated by: the catalog change enforces lowercase names in new modules, and re-applying a module updates its labels. This is a one-time migration for existing deployments.

**[Delete is now stricter]** → Scripts or CI pipelines that run `opm mod delete` and expect exit 0 even when the module doesn't exist will break. Mitigated by: this is the correct behavior — silent no-ops are a bug, not a feature. Any script relying on this was masking real errors.

**[Lowercase override changes log output]** → If a user runs `--name MyBlog`, logs will show `myblog` not `MyBlog`. This is intentional — the override value should match what ends up on the cluster. But it could surprise users who expect the flag value echoed verbatim.
