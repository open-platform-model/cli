# TODO

## Feature

- [ ] Redesign "opm mod status" to print the summary status of all components and its resources. Should make use of <0001-release-inventory.md> entries.
- [ ] Add "opm mod tidy" to tidy up CUE module dependencies in an OPM module. Investigate how to implement tidy without access to the CUE binary.
- [ ] Add "opm mod vet" to validate an OPM module. Take inspiration from how Timoni solved it.
  - Include a "-c or --concrete" flag to force concreteness during validation
- [ ] Add "opm mod eval" to evaluate the module printing the raw CUE code of the module.
- [ ] Rework "opm mod diff" to ignore fields not "owned" by the OPM cli. For example the "managedFields"
  - This filtering has to be done on both manifests before comparing.
- [ ] Ensure "opm config init" also run "cue mod tidy" or similar to discover and download all dependencies.
- [ ] Rework all tests to use cue.AST() instead of strings for test data and comparison.
  - Use pure CUE files, packages and modules for testadata
  - Look into using pure CUE files for comparison data as well.
- [ ] Add `--ignore-not-found` flag to `opm mod delete` and `opm mod status` for idempotent operations. Currently these commands fail with an error when no resources match the selector. The flag would suppress this error and exit 0 instead.
- [ ] Find a way top create something similar like "timoni vendor crd" but fully in CUE. I MUST utilize "cue import openapi".
  - Example found here: <https://github.com/cue-lang/cue/issues/2691>
- [ ] Add #ModuleRelease.values to #Module.#config validation during processing. Meaning it should take the schema (#Module.#config) and validate it against the values (unified #ModuleRelease.values).
  - By leaning on the CUE evaluator we can allow developers to include mandatory fields (!) and optional fields (?) in #Module.#config. This was not possible before.
  - NOTE: Investigate wheter we should also allow for using default (*) in values.
  - Extract the schema and unified values separately and evaluate.
  - Ensure that the log output is referencing the corret files and line relative to the execution directory. Meaning it MUST give the user the correct path to the file and line that fails the evaluator.
- [ ] Add a "opm config update" command. It will extract the current values, initialize the latest config available, and reapply the values.
  - This is a helper command so that users can "upgrade" their configuration more easily.
- [ ] During "opm mod init" the module.cue in cue.mod should initialize as a blank slate, allowing opm to grab the latest versions of all OPM modules. Either by running "opm mod tidy" (internally) or by running something similar to "cue mod get" on each dependency.
- [ ] Add "opm mod list". It should list all modules in the defined namespace (default ns is, default). "-A" should list in all namespaces.
  - Note: Can now leverage `release-id` labels for discovery (see deterministic-release-identity).
- [ ] Add check during processing: Check if a module author has referenced "values" and not "#config" in a component. This will not work and should warn the user.
  - This can be utilized with the future "opm mod publish" command, so that an author cannot publish a module that is not valid.
- [x] ~~"opm mod delete --name blog --namespace default --verbose" proceeds but with no change, 0 resources deleted. We should add validation to first look for the module and inform the caller if not found.~~
  - **Resolved:** Implemented in `refine-resource-discovery` change. Commands now return `NoResourcesFoundError` when no resources match the selector.
- [ ] Add a flag to "opm mod apply" that will create the namespace if missing.
- [ ] Refactor and align the log output of "opm mod build" with the one from "opm mod apply"

  ```bash
  ❯ opm mod build . --verbose
  Module:
    Name:      jellyfin
    Namespace: jellyfin
    Version:   0.1.0
    Components: jellyfin

  Transformer Matching:
    jellyfin:
      ✓ kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#StatefulsetTransformer
        Matched: requiredLabels[core.opmodel.dev/workload-type=stateful], requiredResources[opmodel.dev/resources/workload@v0#Container]
      ✓ kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#ServiceTransformer
        Matched: requiredResources[opmodel.dev/resources/workload@v0#Container], requiredTraits[opmodel.dev/traits/network@v0#Expose]
      ✓ kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#PvcTransformer
        Matched: requiredResources[opmodel.dev/resources/storage@v0#Volumes]
      ✓ kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#HpaTransformer
        Matched: requiredTraits[opmodel.dev/traits/workload@v0#Scaling]

  Generated Resources:
    PersistentVolumeClaim/config [default] from jellyfin
    Service/jellyfin [default] from jellyfin
    StatefulSet/jellyfin [default] from jellyfin
  ```

## Chore

- [ ] Remove injectLabels(). It is redundant as we MUST rely on the transformers to properly apply all labels and annotations.
  - Update <docs/rfc/0001-release-inventory.md> when this change has been made.
- [x] Remove `BuildFromValue()` from `internal/build/release_builder.go:197-251`. It is a legacy path for modules that don't import `opmodel.dev/core@v0`. No production code calls it — only tests.
  - Also remove `extractMetadataFromModule()` at `release_builder.go:644-764` (sole caller is `BuildFromValue`).
  - Remove accompanying tests in `release_builder_test.go` (`TestReleaseBuilder_BuildFromValue` and related).
- [x] Remove `normalizeK8sResource()` and all helper functions from `internal/build/executor.go:257-443`. Transformers now output correct Kubernetes resources directly — this post-processing normalization layer is redundant.
  - Functions to remove: `normalizeK8sResource`, `normalizeContainers`, `mapToPortsArray`, `mapToEnvArray`, `mapToVolumeMountsArray`, `mapToVolumesArray`, `normalizeAnnotations` (~190 lines).
  - Remove the call site in `decodeResource()` at `executor.go:253`.
  - Remove accompanying tests in `executor_test.go`.

## Bugfix

- [ ] An incorrectly configured module should not be able to pass validation and apply only some of the resources. Only when all resources are valid should it apply the whole module

  ```bash
  ❯ opm mod apply . --name jellyfin --namespace default
  14:50:39 INFO m:jellyfin >: applying 3 resources
  14:50:39 INFO m:jellyfin >: r:PersistentVolumeClaim/default/config            created
  14:50:39 INFO m:jellyfin >: r:Service/default/jellyfin                        created
  14:50:39 WARN m:jellyfin >: applying StatefulSet/jellyfin: StatefulSet.apps "jellyfin" is invalid: [spec.template.spec.containers[0].volumeMounts[1].name: Not found: "tvshows", spec.template.spec.containers[0].volumeMounts[2].name: Not found: "movies"]
  14:50:39 WARN m:jellyfin >: 1 resource(s) had errors
  14:50:39 ERRO m:jellyfin >: StatefulSet/jellyfin in default: StatefulSet.apps "jellyfin" is invalid: [spec.template.spec.containers[0].volumeMounts[1].name: Not found: "tvshows", spec.template.spec.containers[0].volumeMounts[2].name: Not found: "movies"]
  14:50:39 INFO m:jellyfin >: applied 2 resources successfully
  1 resource(s) failed to apply
  ```

- [ ] Running "opm mod apply" on an already applied resource should not reply "✔ Module applied" it should reply something more approriate that there was NO change made.
  - Example:

    ```bash
    ❯ opm mod apply . --name jellyfin --namespace default
    10:56:47 INFO m:jellyfin >: applying 5 resources
    r:PersistentVolumeClaim/default/config            unchanged
    r:PersistentVolumeClaim/default/tvshows           unchanged
    r:PersistentVolumeClaim/default/movies            unchanged
    r:Service/default/jellyfin                        unchanged
    r:StatefulSet/default/jellyfin                    unchanged
    10:56:47 INFO m:jellyfin >: applied 5 resources successfully
    ✔ Module applied
    ```

- [ ] `opm mod build`/`opm mod apply` unconditionally requires `values.cue` in the module directory, even when `--values` (`-f`) flags are provided.
  - **Root cause:** `resolveModulePath()` in `internal/build/pipeline.go:194-197` checks for `values.cue` existence before the render pipeline considers `--values` flags.
  - **Expected behavior:** When `--values` flags are provided, `values.cue` on disk should be completely ignored. The external values files are unified (CUE merge) with `#config` from the Module, producing a concrete `values` struct for the ModuleRelease. When no `--values` flags are provided, `values.cue` remains required.
  - **Required changes:**
    1. `internal/build/pipeline.go:resolveModulePath()` — Remove `values.cue` existence check; move it into `Render()` conditioned on `len(opts.Values) == 0`.
    2. `internal/build/release_builder.go:Build()` — When `valuesFiles` is non-empty and `values.cue` exists on disk, overlay it with a minimal stub (`package <pkgName>`) to prevent default values from conflicting with provided values via CUE unification.
    3. `internal/build/release_builder.go:Build()` step 5 — Improve error message: "module missing 'values' field — provide values via values.cue or --values flag".
  - **Reproduction:**

    ```bash
    opm mod build . -f val.cue
    # ERROR: values.cue required but not found in ...
    ```

- [ ] `#config` injection in `release_builder.go:163-169` only surfaces the first CUE error when values have extra or invalid fields. CUE natively produces multi-error output with file, line, and row for each issue, but `concreteModule.Err()` is wrapped into a single `ReleaseValidationError` with a generic message. The same single-error pattern applies at the concreteness validation step (`release_builder.go:179-186`).
  - **Expected behavior:** All CUE validation errors should be collected and printed, each with file/line/row context, matching how `cue vet` reports errors.

## Investigation

- [ ] Investigate how to build a competent and reusable module validation pipeline. Something that can be used to validate the ModuleRelease in the finishing steps of the build pipeline but also can be used for "opm mod vet".
  - It should output the errors similar to how CUE does. Pointing to which file, line and row the error occured.
- [ ] When running "opm mod delete" is the build pipeline really required? Do we need to build the whole module to delete?
- [x] ~~Investigate how to redesign the "opm mod delete" workflow to be smarter. It should find ALL resources, even if the user has changed the module (and didn't apply the changes) before running "delete".~~
  - **Resolved:** Implemented across `deterministic-release-identity` and `refine-resource-discovery` changes. Each release now gets a deterministic UUID v5 identity (`module-release.opmodel.dev/uuid` label) computed by the CUE catalog schemas. The `opm mod delete` command now supports:
    - `--release-id <uuid>` flag for direct UUID-based deletion (mutually exclusive with `--name`)
    - Fails with an error when no resources match the selector (catches typos and misconfigurations)
    - Resources can be found even if module name changed, as long as release-id is known
- [ ] Test if the CLI has staged apply and delete. If not we must design a staged apply and delete system.
  - For resources that we MUST wait for status we need the CLI to wait for that resource to be reporting ok before moving on to the next.
  - Investigate if this should be configurable in the model. Either in the module as a Policy or in each component.

### Possible in controller?

- [ ] Add a smart delete workflow to "opm mod delete". **This is and advanced feature so will will pin it for now**
  - Should look for a Custom Resource called ModuleRelease in the namespace (or all namespaces), this CR contains all the information required and owns all the child resources.

env: {
 PUID: {
  name:  "PUID"
  value: "\(#config.puid)"
 }
}
