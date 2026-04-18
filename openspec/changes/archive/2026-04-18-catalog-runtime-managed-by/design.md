## Context

The catalog's `#TransformerContext` is being refactored (sister change in `poc-controller/openspec/changes/catalog-runtime-managed-by`) so that the runtime identity behind `app.kubernetes.io/managed-by` is a mandatory field that the runtime MUST inject, rather than a hardcoded literal. The full motivation, alternatives, and trade-offs live in the sister change's `design.md`.

This change scope is narrow: pin the new catalog version, update `cli/pkg/render/execute.go` to fill the new `#runtimeName` field with `core.LabelManagedByValue` (`"opm-cli"`), and add tests that lock the contract.

## Goals / Non-Goals

**Goals:**

- CLI-rendered resources carry `app.kubernetes.io/managed-by: opm-cli` after this change merges. The previous behavior — every resource carrying the legacy `"open-platform-model"` value regardless of runtime — is corrected.
- CLI tests fail loudly if the catalog regresses (e.g., new catalog version reverts the `#runtimeName` field) or if the CLI render path forgets to fill it.
- No drift between the Go `core.LabelManagedByValue` constant and the CUE-side schema. Render-and-check test enforces.

**Non-Goals:**

- Catalog edits. Owned by the sister change in `poc-controller/`. This change consumes the published version.
- Coordinating with the controller. Each runtime updates its own render path independently after the catalog publishes.
- Backfilling existing applied resources. SSA apply naturally rewrites the label on next reconcile/apply of any given resource. `core.IsOPMManagedBy` continues to recognize the legacy value for resources the CLI hasn't reapplied yet.

## Decisions

### Decision 1: Wait for catalog publish, no in-repo catalog vendoring

**Choice**: do not vendor the catalog edits into this repo or fork the catalog locally. Wait for the sister change to publish the new catalog version, then pin via `cue.mod/module.cue`.

**Why**: the catalog is a shared schema published to OCI. Vendoring or local-forking would defeat the whole point of the catalog being a versioned external artifact. Pinning + waiting is the correct dependency model.

**Alternatives considered**:

- *Path-replace the catalog locally for development*: works for local iteration but must be undone before merge. Use `task update-deps` from workspace root after the catalog publishes.

### Decision 2: Mirror the controller's render-and-check test

**Choice**: write a test in `cli/pkg/render/` that constructs a minimal `#ModuleRelease`, renders it via the production CLI pipeline with `runtimeName = core.LabelManagedByValue`, and asserts the rendered resource's `managed-by` label exactly equals `core.LabelManagedByValue`. Add a negative test that omits `runtimeName` and asserts CUE evaluation fails.

**Why**: the catalog's mandatory field is the contract enforcer at the schema level; the test is the contract enforcer at the call-site level. Together they prevent both "Go forgot to set the field" and "catalog stopped requiring it" regressions.

**Alternatives considered**:

- *Cross-repo CI integration test*: would catch catalog/CLI desync end-to-end but requires CI infrastructure changes. Defer; the in-process render test is sufficient for this change.

### Decision 3: Single signature change to `ProcessModuleRelease`

**Choice**: change the existing `runtimeLabelsOverride map[string]string` parameter (or the equivalent named call) to `runtimeName string`. One signature change, one call site to update (the CLI's `cmd/.../apply.go` or wherever `mod apply` calls into the render package).

**Why**: trying to preserve backward-compat via deprecated parameters or shims accumulates complexity for no upside in a single-binary CLI. The catalog change is breaking; the runtime change should be too.

## Risks / Trade-offs

- **Risk**: the CLI ships before the catalog version it depends on is available in the OCI registry. → **Mitigation**: task 1.x explicitly waits on the sister change's publish; CI will fail to resolve the new catalog version if attempted prematurely.
- **Risk**: external tooling that matches CLI-applied resources by the legacy literal stops working. → **Mitigation**: documented in proposal Impact. Internal logic uses `core.IsOPMManagedBy` which accepts the legacy value indefinitely.
- **Risk**: `task update-deps` updates more than just the catalog version. → **Mitigation**: review the diff before commit; restrict the commit to the catalog version bump if other deps changed unintentionally.

## Migration Plan

1. Sister change merges in `poc-controller/`, publishing the new catalog version.
2. Pin to that version in this CLI change (task 1.x).
3. Update `cli/pkg/render/execute.go` and the `ProcessModuleRelease` signature (task 2.x).
4. Run the render-and-check test (task 3.x).
5. Validation gates (task 4.x).

Rollback: revert this change. Pin reverts to the previous catalog version. CLI behavior reverts to the legacy `"open-platform-model"` value (which was never correct, but matches prior behavior).

## Open Questions

None. Catalog version string and exact field path will be final once the sister change publishes.
