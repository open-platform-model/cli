# TODO

## Feature

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
- [ ] Add check during processing: Check if a module author has referenced "values" and not "#config" in a component. This will not work and should warn the user.
- [ ] "opm mod delete --name blog --namespace default --verbose" proceeds but with no change, 0 resources deleted. We should add validtion to first look for the module and inform the caller if not found. Look into more validation steps as well.

## Bugfix

- [ ] Update the CLI kubernetes SDK to 1.34+
  - Fix warnings like "Warning: v1 ComponentStatus is deprecated in v1.19+" and "Warning: v1 Endpoints is deprecated in v1.33+; use discovery.k8s.io/v1 EndpointSlice" while we are at it. This is caused by the transformers output is of an older k8s version.

    ```bash
    ❯ opm mod delete --name Blog -n default --verbose
    2026/02/06 11:43:01 DEBU <output/log.go:33> resolved config path path=/var/home/emil/.opm/config.cue source=default
    2026/02/06 11:43:01 DEBU <output/log.go:33> bootstrap: extracted registry from config registry=localhost:5000 path=/var/home/emil/.opm/config.cue
    2026/02/06 11:43:01 DEBU <output/log.go:33> resolved registry registry=localhost:5000 source=env
    2026/02/06 11:43:01 DEBU <output/log.go:33> setting CUE_REGISTRY for config load registry=localhost:5000
    2026/02/06 11:43:01 DEBU <output/log.go:33> extracted provider from config name=kubernetes
    2026/02/06 11:43:01 DEBU <output/log.go:33> extracted providers from config count=1
    2026/02/06 11:43:01 DEBU <output/log.go:33> initializing CLI kubeconfig="" context="" namespace="" config="" output=yaml registry_flag="" resolved_registry=localhost:5000
    Delete all resources for module "Blog" in namespace "default"? [y/N]: y
    2026/02/06 11:43:03 INFO <output/log.go:38> deleting resources for module "Blog" in namespace "default"
    I0206 11:43:03.107650 1902817 warnings.go:107] "Warning: v1 ComponentStatus is deprecated in v1.19+"
    I0206 11:43:03.110144 1902817 warnings.go:107] "Warning: v1 Endpoints is deprecated in v1.33+; use discovery.k8s.io/v1 EndpointSlice"
    2026/02/06 11:43:13 WARN <output/log.go:43> deleting Endpoints/web: the server could not find the requested resource
    2026/02/06 11:43:13 INFO <output/log.go:38>   EndpointSlice/web-n5gh4 in default deleted
    2026/02/06 11:43:13 INFO <output/log.go:38>   DaemonSet/api in default deleted
    2026/02/06 11:43:13 INFO <output/log.go:38>   DaemonSet/web in default deleted
    2026/02/06 11:43:13 INFO <output/log.go:38>   Deployment/api in default deleted
    2026/02/06 11:43:14 INFO <output/log.go:38>   Deployment/web in default deleted
    2026/02/06 11:43:14 INFO <output/log.go:38>   StatefulSet/api in default deleted
    2026/02/06 11:43:14 INFO <output/log.go:38>   StatefulSet/web in default deleted
    2026/02/06 11:43:14 INFO <output/log.go:38>   Service/web in default deleted
    2026/02/06 11:43:14 WARN <output/log.go:43> 1 resource(s) had errors
    2026/02/06 11:43:14 ERRO <output/log.go:48> Endpoints/web in default: the server could not find the requested resource
    2026/02/06 11:43:14 INFO <output/log.go:38> delete complete: 8 resources deleted
    1 resource(s) failed to delete
    ```

- [ ] Rendering log issues:
  - [ ] 'loading module path=/var/home/emil/Dev/open-platform-model/cli/testing/blog values_files=[]' should show all value files, including the default lookup of values.cue.
  - [ ] Investigate why initialization of the Config is happening multiple times.

    ```bash
    2026/02/06 10:49:53 DEBU <output/log.go:33> resolved config path path=/var/home/emil/.opm/config.cue source=default
    2026/02/06 10:49:53 DEBU <output/log.go:33> bootstrap: extracted registry from config registry=localhost:5000 path=/var/home/emil/.opm/config.cue
    2026/02/06 10:49:53 DEBU <output/log.go:33> resolved registry registry=localhost:5000 source=env
    2026/02/06 10:49:53 DEBU <output/log.go:33> setting CUE_REGISTRY for config load registry=localhost:5000
    2026/02/06 10:49:53 DEBU <output/log.go:33> extracted provider from config name=kubernetes
    2026/02/06 10:49:53 DEBU <output/log.go:33> extracted providers from config count=1
    2026/02/06 10:49:53 DEBU <output/log.go:33> initializing CLI kubeconfig="" context="" namespace="" config="" output=yaml registry_flag="" resolved_registry=localhost:5000
    2026/02/06 10:49:53 DEBU <output/log.go:33> resolved config path path=/var/home/emil/.opm/config.cue source=default
    2026/02/06 10:49:53 DEBU <output/log.go:33> bootstrap: extracted registry from config registry=localhost:5000 path=/var/home/emil/.opm/config.cue
    2026/02/06 10:49:53 DEBU <output/log.go:33> resolved registry registry=localhost:5000 source=env
    2026/02/06 10:49:53 DEBU <output/log.go:33> setting CUE_REGISTRY for config load registry=localhost:5000
    2026/02/06 10:49:53 DEBU <output/log.go:33> extracted provider from config name=kubernetes
    2026/02/06 10:49:53 DEBU <output/log.go:33> extracted providers from config count=1
    2026/02/06 10:49:53 DEBU <output/log.go:33> rendering module module=. namespace="" provider=""
    2026/02/06 10:49:53 DEBU <output/log.go:33> loading module path=/var/home/emil/Dev/open-platform-model/cli/testing/blog values_files=[]
    2026/02/06 10:49:53 DEBU <output/log.go:33> loaded module name=Blog namespace=default version=0.1.0
    2026/02/06 10:49:53 DEBU <output/log.go:33> building release name=Blog namespace=default
    2026/02/06 10:49:53 DEBU <output/log.go:33> release built successfully name=Blog namespace=default components=2
    2026/02/06 10:49:53 DEBU <output/log.go:33> release built name=Blog namespace=default components=2
    2026/02/06 10:49:53 DEBU <output/log.go:33> loading provider name=kubernetes
    ```

## Investigation

- [ ] Test if the CLI has staged apply and delete. If not we must design a staged apply and delete sytem.
  - For resources that we MUST wait for status we need the CLI to wait for that resource to be reporting ok before moving on to the next.
  - Investigate if this should be configurable in the model. Either in the module as a Policy or in each component.

### Possible in controller?

- [ ] Add a smart delete workflow to "opm mod delete". **This is and advanced feature so will will pin it for now**
  - Should look for a Custom Resource called ModuleRelease in the namespace (or all namespaces), this CR contains all the information required and owns all the child resources.
