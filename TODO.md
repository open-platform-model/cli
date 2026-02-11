# TODO

## Feature

- [ ] Add "opm mod tidy" to tidy up CUE module dependencies in an OPM module. Investigate how to implement tidy without access to the CUE binary.
- [ ] Add "opm mod vet" to validate an OPM module. Take inspiration from how Timoni solved it.
  - Include a "-c or --concrete" flag to force concreteness during validation
- [ ] Add "opm mod eval" to evaluate the module printing the raw CUE code of the module.
- [ ] Rework "opm mod diff" to ignore fields not "owned" by the OPM cli. For example the "managedFields"
  - This cannot be hardcoded, we need to somehow keep track of the OPM "owned" fields. Maybe secret (like helm) or a Custom Resource.
- [ ] Ensure "opm config init" also run "cue mod tidy" or similar to discover and download all dependencies.
- [ ] Rework the "opm mod init" to use cue.AST() instead of string template.
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

## Investigation

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
