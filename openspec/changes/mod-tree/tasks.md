## 1. Core Data Model (internal/kubernetes/tree.go)

- [x] 1.1 Define `TreeResult` struct with Release, Components fields
- [x] 1.2 Define `ReleaseInfo` struct with Name, Namespace, Module, Version fields
- [x] 1.3 Define `Component` struct with Name, ResourceCount, Status, Resources fields
- [x] 1.4 Define `ResourceNode` struct with Kind, Name, Namespace, Status, Replicas, Children fields
- [x] 1.5 Add `TreeOptions` struct with ReleaseInfo, InventoryLive, ComponentMap,
          Depth, and OutputFormat fields
          (mirrors StatusOptions pattern — discovery done in command layer, not here)

## 2. Component Grouping Logic (internal/kubernetes/tree.go)

- [x] 2.1 Implement `groupByComponent()` function to group resources using ComponentMap
          (key: Kind+"/"+Namespace+"/"+Name → component name, same key format as status.go)
- [x] 2.2 Resources absent from ComponentMap (empty string value) go into `(no component)`
- [x] 2.3 Implement component sorting (alphabetical, with `(no component)` last)
- [x] 2.4 Implement resource ordering within components (inventory order preserved from InventoryLive)
- [x] 2.5 Write unit tests for component grouping with various ComponentMap scenarios

## 3. Ownership Walking (internal/kubernetes/tree.go)

- [x] 3.1 Implement `walkOwnership()` dispatcher based on resource Kind
- [x] 3.2 Implement `walkDeployment()`:
          use Clientset.AppsV1().ReplicaSets(ns).List() filtered by ownerReference UID,
          then for each RS call walkReplicaSet()
- [x] 3.3 Implement `walkReplicaSet()`:
          use Clientset.CoreV1().Pods(ns).List() filtered by ownerReference UID,
          extract pod status via extractPodInfoFromPod() (already in pods.go)
- [x] 3.4 Implement `walkStatefulSet()`:
          use Clientset.CoreV1().Pods(ns).List() filtered by ownerReference UID
          (no ReplicaSet layer for StatefulSets)
- [x] 3.5 Implement `walkDaemonSet()`:
          use Clientset.CoreV1().Pods(ns).List() filtered by ownerReference UID
- [x] 3.6 Implement `walkJob()`:
          use Clientset.CoreV1().Pods(ns).List() filtered by ownerReference UID
- [x] 3.7 Implement `filterPodsByOwnerUID()` helper:
          (implemented as hasOwnerWithUID + walkPodsOwnedBy)
- [x] 3.8 Implement `filterReplicaSetsByOwnerUID()` helper:
          (implemented inline in walkDeployment via hasOwnerWithUID)
- [x] 3.9 Add error handling for RBAC/API failures (log debug, return partial node without children)
- [x] 3.10 Write unit tests for ownership walking with mocked Clientset

## 4. Replica Count Extraction (internal/kubernetes/tree.go)

- [x] 4.1 Implement `getReplicaCount()` function with Kind-based dispatch
- [x] 4.2 Deployment: `status.readyReplicas / status.replicas` → "N/M"
- [x] 4.3 StatefulSet: `status.readyReplicas / spec.replicas` → "N/M"
- [x] 4.4 DaemonSet: `status.numberReady / status.desiredNumberScheduled` → "N/M"
- [x] 4.5 Job: `status.succeeded / spec.completions` → "N/M"
- [x] 4.6 ReplicaSet: `status.replicas` → "N pods" (rendered in walkDeployment)
- [x] 4.7 Return empty string for non-workload resources
- [x] 4.8 Write unit tests for replica count extraction across all workload kinds

## 5. Tree Building (internal/kubernetes/tree.go)

- [x] 5.1 Implement `BuildTree()` main function with depth filtering
- [x] 5.2 Handle depth=0: return component summary only (no resource iteration, no K8s queries)
- [x] 5.3 Handle depth=1: build resource nodes without calling walkOwnership()
- [x] 5.4 Handle depth=2: build full tree — for each resource call walkOwnership()
- [x] 5.5 Populate ReleaseInfo from TreeOptions.ReleaseInfo
          (no label extraction needed — caller provides from inventory)
- [x] 5.6 Compute component-level aggregate status (Ready if all resources Ready)
- [x] 5.7 Write unit tests for BuildTree at each depth level

## 6. Tree Rendering - Colored Output (internal/kubernetes/tree.go)

- [x] 6.1 Implement `FormatTree()` function dispatching by format (table/json/yaml)
- [x] 6.2 Render release header: `<name> (<module>@<version>)` or `<name>` if no module
- [x] 6.3 Render component names (cyan, output.StyleNoun) with box-drawing prefix
- [x] 6.4 Render OPM resources with status via output.FormatHealthStatus()
- [x] 6.5 Render K8s children (dim, output.Dim()) with status colored
- [x] 6.6 Implement tree chrome rendering (├── └── │) via output.Dim()
- [x] 6.7 Handle depth=0 output: component name, resource count, aggregate status per component
- [x] 6.8 Implement indentation logic tracking "isLast" at each level for correct chrome
- [x] 6.9 Write unit tests for colored rendering

## 7. Tree Rendering - Plain Output (internal/kubernetes/tree.go)

- [x] 7.1 Implement `formatPlainTree()` for non-TTY environments (no ANSI codes)
- [x] 7.2 Use same box-drawing characters, same structure, without lipgloss styling
- [x] 7.3 Write unit tests for plain rendering (verify no ANSI escape sequences)

## 8. Structured Output Formats (internal/kubernetes/tree.go)

- [x] 8.1 Implement `FormatTreeJSON()` to serialize TreeResult to JSON
- [x] 8.2 Implement `FormatTreeYAML()` to serialize TreeResult to YAML
- [x] 8.3 Ensure nested `children` arrays in JSON/YAML output
- [x] 8.4 Write unit tests for JSON schema validation
- [x] 8.5 Write unit tests for YAML output validation

## 9. GetModuleTree Function (internal/kubernetes/tree.go)

- [x] 9.1 Implement `GetModuleTree()` as main entry point (mirrors GetReleaseStatus pattern)
          Signature: GetModuleTree(ctx, client, opts TreeOptions) (*TreeResult, error)
- [x] 9.2 Return `noResourcesFoundError` when InventoryLive is empty
          (0 entries in inventory = invalid state, same error as no inventory found)
- [x] 9.3 Call `BuildTree()` with opts and client
- [x] 9.4 Return TreeResult or error
- [x] 9.5 Write unit tests for GetModuleTree with empty InventoryLive (expect error)
- [x] 9.6 Write unit tests for GetModuleTree with valid inventory at each depth

## 10. Command Implementation (internal/cmd/mod/tree.go)

- [x] 10.1 Create `NewModTreeCmd()` function following existing command pattern
- [x] 10.2 Declare `var rsf cmdutil.ReleaseSelectorFlags` and `var kf cmdutil.K8sFlags`
- [x] 10.3 Add local flag variables: `depthFlag int`, `outputFlag string`
- [x] 10.4 Register flags via `rsf.AddTo(cmd)` and `kf.AddTo(cmd)`
- [x] 10.5 Add `--depth` flag with default 2, description including valid range [0,1,2]
- [x] 10.6 Add `-o, --output` flag with default "table", values: table, json, yaml
- [x] 10.7 Implement `runTree()` function:
- [x] 10.8   Validate release selector via `rsf.Validate()`
- [x] 10.9   Resolve K8s config via `config.ResolveKubernetes(config.ResolveKubernetesOptions{...})`
- [x] 10.10  Create K8s client via `cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)`
- [x] 10.11  Resolve inventory via `cmdutil.ResolveInventory(ctx, client, &rsf, namespace, false, log)`
- [x] 10.12  Validate depth flag is in range [0, 2]; return ExitGeneralError if not
- [x] 10.13  Parse output format via `output.ParseFormat(outputFlag)`; reject FormatWide and FormatDir
- [x] 10.14  Build ComponentMap from `inv.Changes[inv.Index[0]].Inventory.Entries`
- [x] 10.15  Build ReleaseInfo from inventory metadata
- [x] 10.16  Build `TreeOptions` and call `kubernetes.GetModuleTree(ctx, client, opts)`
- [x] 10.17  Format result via FormatTree based on outputFormat
- [x] 10.18  Handle errors with appropriate exit codes
- [x] 10.19  Add command short/long help text with examples (depth=0/1/2, json output)

## 11. Command Registration (internal/cmd/mod/mod.go)

- [x] 11.1 Add `c.AddCommand(NewModTreeCmd(cfg))` to NewModCmd()

## 12. Command Tests (internal/cmd/mod/tree_test.go)

- [x] 12.1 Write test for flag existence (--depth, --release-name, --release-id, -n, -o)
- [x] 12.2 Write test for release selector mutual exclusivity (both set → error)
- [x] 12.3 Write test for release selector requirement (neither set → error)
- [x] 12.4 Write test for depth validation: values 0,1,2 accepted; -1 and 3 rejected
- [x] 12.5 Write test for output format validation: table/json/yaml accepted; wide/dir rejected

## 13. Tree Tests (internal/kubernetes/tree_test.go)

- [x] 13.1 Write tests for component grouping with ComponentMap (normal, missing key, empty string)
- [x] 13.2 Write tests for component sorting (alphabetical + (no component) last)
- [x] 13.3 Write tests for resource ordering within components (inventory order preserved)
- [x] 13.4 Write tests for walkDeployment() → RS nodes with pod children
- [x] 13.5 Write tests for walkStatefulSet() → direct pod children (no RS layer)
- [x] 13.6 Write tests for walkOwnership() RBAC failure (log debug, return node without children)
- [x] 13.7 Write tests for getReplicaCount() across all workload kinds
- [x] 13.8 Write tests for BuildTree at depth=0 (component summaries only, no K8s queries)
- [x] 13.9 Write tests for BuildTree at depth=1 (resource nodes, no ownership walking)
- [x] 13.10 Write tests for BuildTree at depth=2 (full tree with children)
- [x] 13.11 Write tests for FormatTree colored output
- [x] 13.12 Write tests for formatPlainTree (no ANSI codes)
- [x] 13.13 Write tests for JSON output schema (nested children array)
- [x] 13.14 Write tests for YAML output
- [x] 13.15 Write tests for GetModuleTree with empty InventoryLive (expect noResourcesFoundError)

## 14. Integration Tests

- [ ] 14.1 Add integration test: deploy test module with multiple components
- [ ] 14.2 Add integration test: verify tree output at depth=0 (component summary)
- [ ] 14.3 Add integration test: verify tree output at depth=1 (resources only)
- [ ] 14.4 Add integration test: verify tree output at depth=2 (Deployment→RS→Pod visible)
- [ ] 14.5 Add integration test: verify JSON output structure matches schema
- [ ] 14.6 Add integration test: verify no-release-found error (exit code matches ExitNotFound)
- [ ] 14.7 Add integration test: verify component grouping with real resources
- [ ] 14.8 Add integration test: verify StatefulSet→Pod chain at depth=2

## 15. Documentation

- [x] 15.1 Add help text and examples to `NewModTreeCmd()` long description
- [ ] 15.2 Update CLI README with `opm mod tree` command if a command reference exists

## 16. Validation & Polish

- [x] 16.1 Run `task fmt` and fix any formatting issues
- [x] 16.2 Run `task lint` and fix any linter warnings (new files lint-clean)
- [x] 16.3 Run `task test` and ensure all tests pass
- [ ] 16.4 Run integration tests against kind cluster (`task cluster:create` first)
- [ ] 16.5 Test against a real multi-component module (e.g. examples/jellyfin)
- [ ] 16.6 Verify exit codes: 0=success, ExitNotFound=no resources, ExitGeneralError=other
- [ ] 16.7 Verify colored output in terminal
- [ ] 16.8 Verify plain output in non-TTY (`opm mod tree ... | cat`)
- [ ] 16.9 Verify JSON output is valid (`opm mod tree ... -o json | jq .`)
- [ ] 16.10 Verify YAML output is valid
