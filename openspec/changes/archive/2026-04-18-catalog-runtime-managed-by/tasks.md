## 1. Wait for catalog publish + pin

- [x] 1.1 Confirm the sister change (`poc-controller/openspec/changes/catalog-runtime-managed-by`) has published the new catalog version. Capture the version string.
- [x] 1.2 From the workspace root, run `task update-deps` to bump all CUE module pins (`cli/cue.mod/module.cue`, any subpackage `cue.mod/`). Verify the diff is scoped to the catalog version.
- [x] 1.3 Run `task fmt && task vet` in `cli/`. Render and CUE evaluation may break here because `#runtimeLabels` no longer exists on `#TransformerContext` and `#runtimeName` is mandatory. This is expected — task 2.x fixes it.

## 2. Update CLI render path

- [x] 2.1 In `cli/pkg/render/execute.go:233-243`, replace the `runtimeLabels` `map[string]string` block with a `runtimeName` string. Change the `FillPath` target from `cue.MakePath(cue.Def("context"), cue.Def("runtimeLabels"))` to `cue.MakePath(cue.Def("context"), cue.Def("runtimeName"))`. The injected value is `core.LabelManagedByValue` (`"opm-cli"`).
- [x] 2.2 Update `ProcessModuleRelease` signature: replace the `runtimeLabelsOverride map[string]string` parameter with `runtimeName string` (or rename the existing parameter and change its type). Match the controller's signature shape for consistency — confirm by reading the merged controller change at `poc-controller/internal/render/module.go` and `poc-controller/pkg/render/execute.go` once it lands.
- [x] 2.3 Update the CLI's call site for `ProcessModuleRelease` (likely in `cli/cmd/.../apply.go` or `cli/cmd/.../mod_apply.go` — grep for the call). Pass `core.LabelManagedByValue` instead of building a label map.
- [x] 2.4 Remove the now-unused `core.LabelModuleReleaseNamespace` map entry from the CLI render path, if it exists. The catalog never iterated `#runtimeLabels` so the entry was dead; deletion is mechanical.
- [x] 2.5 Run `task fmt && task vet && task lint && task test` in `cli/`. Address any remaining compilation errors. Tests that hand-construct `#TransformerContext` need `#runtimeName` filled.

## 3. Render-and-check test

- [x] 3.1 Add a unit test (`cli/pkg/render/render_runtime_label_test.go`): construct a minimal `#ModuleRelease` (one component, one resource — a ConfigMap is sufficient). Render via the production pipeline with `runtimeName = core.LabelManagedByValue`.
- [x] 3.2 Assert the rendered resource's `metadata.labels["app.kubernetes.io/managed-by"]` exactly equals `core.LabelManagedByValue`.
- [x] 3.3 Assert the rendered resource's `metadata.labels["module-release.opmodel.dev/uuid"]` is non-empty (sanity check ownership labels still flow).
- [x] 3.4 Add a negative test: invoke the render path without filling `#runtimeName` (may require a lower-level helper). Assert CUE evaluation returns an error mentioning the missing required field.
- [x] 3.5 Run the new tests in isolation: `go test ./pkg/render -run TestRender_RuntimeName -v`.

## 4. Validation gates

- [x] 4.1 `task build` — must succeed.
- [x] 4.2 `task fmt && task vet` — must pass.
- [x] 4.3 `task lint` — must pass.
- [x] 4.4 `task test` — full suite must pass.
- [x] 4.5 `task check` — repo-level check must pass.
- [ ] 4.6 Manual smoke test (optional): render a sample module via `opm mod render -f sample.cue` and inspect a resource's labels — `managed-by` value should be `opm-cli`.
- [x] 4.7 `cd cli && openspec validate catalog-runtime-managed-by --strict` — must pass.
