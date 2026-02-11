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

- [x] Fix the "initializing CLI" output. Even if environment variables or flags are not set it should show what value is being used. The default values will always be found in .opm/config.cue.
  - Also what is the point of 'registry_flag=""', this seems redundant.

  ```bash
  12:33:38 DEBU <output/log.go:70> initializing CLI kubeconfig="" context="" namespace="" config="" output=yaml registry_flag="" resolved_registry=localhost:5000
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

- [x] ~~Update the CLI kubernetes SDK to 1.34+~~
  - **Resolved:** SDK already at v0.35.0 (Kubernetes 1.35). Implemented in `improve-k8s-api-warnings-and-discovery` change:
    - Custom `rest.WarningHandler` routes K8s API warnings through charmbracelet/log (formatted consistently)
    - Config option `log.kubernetes.apiWarnings` with values `"warn"` (default), `"debug"`, or `"suppress"`
    - Switched discovery to `ServerPreferredResources()` to reduce API calls and avoid deprecated versions
    - Added `ExcludeOwned` option to skip controller-managed resources (prevents 404 errors on auto-managed Endpoints/EndpointSlice)
- [x] Missing values in fields during log:
  - [x] Missing values during config loading "initializing CLI kubeconfig="" context="" namespace="" config="" output=yaml registry_flag="" resolved_registry=localhost:5000". Should show at least default values (from .opm/config.cue). Example "initializing CLI kubeconfig="~/.kube/config" context="kind-opm-dev" namespace="default" config="~/.opm/config.cue" output=yaml registry_flag="" resolved_registry=localhost:5000"
    - The log "initializing CLI" should display the final config values that was choosen.
  - [x] Keep this "resolved config path path=/var/home/emil/.opm/config.cue source=default"
  - [x] Keep this "resolved registry registry=localhost:5000 source=env"
  - [x] This becomes redundant "bootstrap: extracted registry from config registry=localhost:5000 path=/var/home/emil/.opm/config.cue"
  - [x] This becomes redundant "setting CUE_REGISTRY for config load registry=localhost:5000"
  - [x] This becomes redundant "extracted provider from config name=kubernetes"
  - [x] This becomes redundant "extracted providers from config count=1"
  - [x] This becomes redundant "loading provider name=kubernetes"
  - [x] This needs to display provider "rendering module module=. namespace=default provider="""
  - [x] We only need one of these "release built successfully name=jellyfin namespace=default components=1", "release built name=jellyfin namespace=default components=1"

    ```bash
    ❯ opm mod apply . --name jellyfin --namespace default --verbose
    10:49:46 DEBU <output/log.go:70> initializing CLI kubeconfig="" context="" namespace="" config="" output=yaml registry_flag="" resolved_registry=localhost:5000
    10:49:46 DEBU <output/log.go:70> resolved config path path=/var/home/emil/.opm/config.cue source=default
    10:49:46 DEBU <output/log.go:70> bootstrap: extracted registry from config registry=localhost:5000 path=/var/home/emil/.opm/config.cue
    10:49:46 DEBU <output/log.go:70> resolved registry registry=localhost:5000 source=env
    10:49:46 DEBU <output/log.go:70> setting CUE_REGISTRY for config load registry=localhost:5000
    10:49:46 DEBU <output/log.go:70> extracted provider from config name=kubernetes
    10:49:46 DEBU <output/log.go:70> extracted providers from config count=1
    10:49:46 DEBU <output/log.go:70> rendering module module=. namespace=default provider=""
    10:49:46 DEBU <output/log.go:70> inspected module via AST pkgName=main name=jellyfin defaultNamespace=jellyfin
    10:49:46 DEBU <output/log.go:70> building release path=/var/home/emil/Dev/open-platform-model/cli/examples/jellyfin name=jellyfin namespace=default
    10:49:46 DEBU <output/log.go:70> release built successfully name=jellyfin namespace=default components=1
    10:49:46 DEBU <output/log.go:70> release built name=jellyfin namespace=default components=1
    10:49:46 DEBU <output/log.go:70> loading provider name=kubernetes
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#DeploymentTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#DeploymentTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#StatefulsetTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#StatefulsetTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#DaemonsetTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#DaemonsetTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#JobTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#JobTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[opmodel.dev/traits/workload@v0#JobConfig]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#CronjobTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#CronjobTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[opmodel.dev/traits/workload@v0#CronJobConfig]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#ServiceTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#ServiceTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[opmodel.dev/traits/network@v0#Expose]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#PvcTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#PvcTransformer requiredResources=[opmodel.dev/resources/storage@v0#Volumes] requiredTraits=[]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#ConfigmapTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#ConfigmapTransformer requiredResources=[opmodel.dev/resources/config@v0#ConfigMaps] requiredTraits=[]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#SecretTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#SecretTransformer requiredResources=[opmodel.dev/resources/config@v0#Secrets] requiredTraits=[]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#ServiceaccountTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#ServiceaccountTransformer requiredResources=[] requiredTraits=[opmodel.dev/traits/security@v0#WorkloadIdentity]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#HpaTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#HpaTransformer requiredResources=[] requiredTraits=[opmodel.dev/traits/workload@v0#Scaling]
    10:49:46 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#IngressTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#IngressTransformer requiredResources=[] requiredTraits=[opmodel.dev/traits/network@v0#HttpRoute]
    10:49:46 DEBU <output/log.go:70> loaded provider name=kubernetes version="" transformers=12
    10:49:46 DEBU <output/log.go:70> executing jobs count=4
    10:49:46 DEBU <output/log.go:70> execution complete resources=5 errors=0
    10:49:46 INFO <cmd/mod_apply.go:177> m:jellyfin >: applying 5 resources
    10:49:46 INFO <kubernetes/apply.go:82> m:jellyfin >: r:PersistentVolumeClaim/default/config            unchanged
    10:49:46 INFO <kubernetes/apply.go:82> m:jellyfin >: r:PersistentVolumeClaim/default/tvshows           unchanged
    10:49:46 INFO <kubernetes/apply.go:82> m:jellyfin >: r:PersistentVolumeClaim/default/movies            unchanged
    10:49:46 INFO <kubernetes/apply.go:82> m:jellyfin >: r:Service/default/jellyfin                        unchanged
    10:49:46 INFO <kubernetes/apply.go:82> m:jellyfin >: r:StatefulSet/default/jellyfin                    unchanged
    10:49:46 INFO <cmd/mod_apply.go:200> m:jellyfin >: applied 5 resources successfully
    ✔ Module applied
    ```

- [x] Remove "fqn=" from transformer output and use the fqn in "name=" instead. This will make the line entry shorter.

  ```bash
  09:54:17 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#DeploymentTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#DeploymentTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[]
  09:54:17 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#StatefulsetTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#StatefulsetTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[]
  09:54:17 DEBU <output/log.go:70> extracted transformer name=opmodel.dev/providers/kubernetes/transformers@v0#DaemonsetTransformer fqn=kubernetes#opmodel.dev/providers/kubernetes/transformers@v0#DaemonsetTransformer requiredResources=[opmodel.dev/resources/workload@v0#Container] requiredTraits=[]
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
