## Why

The catalog's `#TransformerContext` is being changed (in the sister change `poc-controller/openspec/changes/catalog-runtime-managed-by`) to require a mandatory `#runtimeName` field. The catalog edits and publish are owned by that change so neither runtime races the other. Once the new catalog version is published, the CLI's render path needs a one-line shape change: stop filling the deprecated `#runtimeLabels` field, start filling `#runtimeName` with `core.LabelManagedByValue` (`"opm-cli"`).

This change exists so the CLI's update lives next to its own tests, lint config, and release cadence. The actual contract — what `#runtimeName` is, why it is mandatory, and what value the runtime must inject — is documented in the catalog change. This change pins to the new catalog version and updates the CLI render bridge.

## What Changes

- **Pin new catalog version**: bump `cli/cue.mod/module.cue` (and any subpackage `cue.mod/` files) to the catalog version published by the sister change. Use the workspace `task update-deps` command from the workspace root to keep all CUE module pins consistent.
- **Update CLI render path**: in `cli/pkg/render/execute.go:233-243`, replace the existing `runtimeLabels` `map[string]string` injection with a single `runtimeName string` injection. The `FillPath` target moves from `cue.MakePath(cue.Def("context"), cue.Def("runtimeLabels"))` to `cue.MakePath(cue.Def("context"), cue.Def("runtimeName"))`. The value injected is `core.LabelManagedByValue` (`"opm-cli"`).
- **Update `pkg/render/ProcessModuleRelease` signature** if needed to take a `runtimeName string` instead of (or in addition to) the existing `runtimeLabelsOverride` map. Trace the controller's matching change for the exact signature shape; the CLI's call site is `cli/cmd/...` (single call site, easy to update).
- **Render-and-check test**: add a unit test in `cli/pkg/render/` that renders a minimal `#ModuleRelease` and asserts `metadata.labels["app.kubernetes.io/managed-by"] == core.LabelManagedByValue`. Mirrors the controller-side test; both runtimes get the same contract enforcement.
- **Negative test**: assert CUE evaluation errors when `#runtimeName` is not filled (drift detection — guards against future bugs that bypass the helper).

This is a PATCH for the CLI: the user-visible value of `app.kubernetes.io/managed-by` on resources applied via `opm mod apply` transitions from the legacy `"open-platform-model"` literal to `"opm-cli"`. External consumers matching the legacy value need to accept either; internal consumers via `core.IsOPMManagedBy` are unaffected.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-identity-labeling`: the requirement set gains a scenario stating the CLI MUST fill the catalog's mandatory `#runtimeName` field with `core.LabelManagedByValue`. Existing identity-label scenarios (`module-release.opmodel.dev/uuid`, `module.opmodel.dev/uuid`) are unchanged — those still flow via the catalog's `moduleLabels` block.

## Impact

- **Catalog (consumed via OCI)**: pin to the new version published by the sister change. No catalog edits in this repo.
- **CLI code**: small diff in `cli/pkg/render/execute.go` (one map → string), `cli/pkg/render/ProcessModuleRelease` signature, one call site in `cli/cmd/`. New test file.
- **External consumers**: anyone matching CLI-applied resources by `managed-by=open-platform-model` will need to accept `opm-cli`. `core.IsOPMManagedBy` already accepts both old and new values for internal logic.
- **API**: no CLI command surface changes.
- **SemVer**: PATCH (bug fix; previously wrong value of an existing label).
- **Sequencing**: this change is BLOCKED by the sister change in `poc-controller/` until the new catalog version is published. Once available, this change can land independently.
