## 1. Core Data Model (internal/kubernetes/tree.go)

- [ ] 1.1 Define `TreeResult` struct with Release, Components fields
- [ ] 1.2 Define `ReleaseInfo` struct with Name, Namespace, Module, Version fields
- [ ] 1.3 Define `Component` struct with Name, ResourceCount, Status, Resources fields
- [ ] 1.4 Define `ResourceNode` struct with Kind, Name, Namespace, Status, Replicas, Children fields
- [ ] 1.5 Add `TreeOptions` struct with Namespace, ReleaseName, ReleaseID, Depth, OutputFormat fields

## 2. Component Grouping Logic (internal/kubernetes/tree.go)

- [ ] 2.1 Implement `groupByComponent()` function to extract component label and group resources
- [ ] 2.2 Handle resources without component label → group under `(no component)`
- [ ] 2.3 Implement component sorting (alphabetical, with `(no component)` last)
- [ ] 2.4 Implement resource sorting within components (by weight ascending, then name)
- [ ] 2.5 Write unit tests for component grouping with various label scenarios

## 3. Ownership Walking (internal/kubernetes/tree.go)

- [ ] 3.1 Implement `walkOwnership()` function dispatcher based on resource Kind
- [ ] 3.2 Implement `walkDeployment()` to query ReplicaSets by ownerReference UID
- [ ] 3.3 Implement `walkReplicaSet()` to query Pods by ownerReference UID
- [ ] 3.4 Implement `walkStatefulSet()` to query Pods directly (no ReplicaSet)
- [ ] 3.5 Implement `walkDaemonSet()` to query Pods by ownerReference UID
- [ ] 3.6 Implement `walkJob()` to query Pods by ownerReference UID
- [ ] 3.7 Add error handling for RBAC failures (log debug, continue with partial data)
- [ ] 3.8 Write unit tests for ownership walking with mocked K8s client

## 4. Replica Count Extraction (internal/kubernetes/tree.go)

- [ ] 4.1 Implement `getReplicaCount()` function with Kind-based dispatch
- [ ] 4.2 Extract Deployment replica count: `status.readyReplicas / status.replicas`
- [ ] 4.3 Extract StatefulSet replica count: `status.readyReplicas / status.replicas`
- [ ] 4.4 Extract DaemonSet replica count: `status.numberReady / status.desiredNumberScheduled`
- [ ] 4.5 Extract Job completion count: `status.succeeded / spec.completions`
- [ ] 4.6 Extract ReplicaSet count: `status.replicas pods` format
- [ ] 4.7 Return empty string for non-workload resources
- [ ] 4.8 Write unit tests for replica count extraction across all workload kinds

## 5. Tree Building (internal/kubernetes/tree.go)

- [ ] 5.1 Implement `BuildTree()` main function with depth filtering
- [ ] 5.2 Handle depth=0: return component summary only (no resource iteration)
- [ ] 5.3 Handle depth=1: build resource tree without ownership walking
- [ ] 5.4 Handle depth=2: build full tree with ownership walking
- [ ] 5.5 Extract release metadata from first resource labels (name, version, module ID)
- [ ] 5.6 Compute component-level aggregate status (Ready if all resources Ready)
- [ ] 5.7 Write unit tests for BuildTree at each depth level

## 6. Tree Rendering - Colored Output (internal/kubernetes/tree.go)

- [ ] 6.1 Implement `FormatTree()` function with TTY detection via `lipgloss.IsTerminal()`
- [ ] 6.2 Render release header (bold white): `<name> (<module>@<version>)`
- [ ] 6.3 Render component names (cyan bold) with box-drawing prefix `├──` or `└──`
- [ ] 6.4 Render OPM resources (default white) with status colored (green/red/yellow)
- [ ] 6.5 Render K8s children (dim gray) with status colored
- [ ] 6.6 Implement tree chrome rendering (├── └── │) in dim gray
- [ ] 6.7 Handle depth=0 output format: component name, resource count, aggregate status
- [ ] 6.8 Implement indentation logic: 3 spaces per tree level
- [ ] 6.9 Write unit tests for colored rendering (verify ANSI codes present)

## 7. Tree Rendering - Plain Output (internal/kubernetes/tree.go)

- [ ] 7.1 Implement `formatPlainTree()` for non-TTY environments
- [ ] 7.2 Use same box-drawing characters without color codes
- [ ] 7.3 Write unit tests for plain rendering (verify no ANSI codes)

## 8. Structured Output Formats (internal/kubernetes/tree.go)

- [ ] 8.1 Implement `FormatTreeJSON()` to serialize TreeResult to JSON
- [ ] 8.2 Implement `FormatTreeYAML()` to serialize TreeResult to YAML
- [ ] 8.3 Ensure nested `children` arrays in JSON/YAML output
- [ ] 8.4 Write unit tests for JSON schema validation
- [ ] 8.5 Write unit tests for YAML output validation

## 9. GetModuleTree Function (internal/kubernetes/tree.go)

- [ ] 9.1 Implement `GetModuleTree()` as main entry point (similar to `GetModuleStatus`)
- [ ] 9.2 Call `DiscoverResources()` with TreeOptions selector
- [ ] 9.3 Return `noResourcesFoundError` if resources slice is empty
- [ ] 9.4 Call `BuildTree()` with discovered resources and depth
- [ ] 9.5 Return TreeResult or error
- [ ] 9.6 Write unit tests for GetModuleTree with various scenarios

## 10. Command Implementation (internal/cmd/mod_tree.go)

- [ ] 10.1 Create `NewModTreeCmd()` function following existing command pattern
- [ ] 10.2 Add flag variables: `depthFlag int`, `outputFlag string`
- [ ] 10.3 Declare `ReleaseSelectorFlags` and `K8sFlags` instances
- [ ] 10.4 Register flags via `rsf.AddTo(cmd)` and `kf.AddTo(cmd)`
- [ ] 10.5 Add `--depth` flag with default 2, validation in [0, 1, 2]
- [ ] 10.6 Add `-o, --output` flag with values: table (default), json, yaml
- [ ] 10.7 Implement `runTree()` function to execute tree command
- [ ] 10.8 Validate release selector flags via `rsf.Validate()`
- [ ] 10.9 Resolve Kubernetes config via `cmdutil.ResolveKubernetes()`
- [ ] 10.10 Create K8s client via `cmdutil.NewK8sClient()`
- [ ] 10.11 Validate depth flag (0-2 range)
- [ ] 10.12 Parse and validate output format via `output.ParseFormat()`
- [ ] 10.13 Build `TreeOptions` from flags and resolved config
- [ ] 10.14 Call `kubernetes.GetModuleTree()` with options
- [ ] 10.15 Format result based on output format (table/json/yaml)
- [ ] 10.16 Handle errors and return appropriate exit codes (0, 1, 3)
- [ ] 10.17 Add command short/long help text with examples

## 11. Command Registration (internal/cmd/mod.go)

- [ ] 11.1 Import `NewModTreeCmd` in mod.go
- [ ] 11.2 Register tree subcommand: `modCmd.AddCommand(NewModTreeCmd())`

## 12. Command Tests (internal/cmd/mod_tree_test.go)

- [ ] 12.1 Write test for flag existence (--depth, --release-name, --release-id, -n, -o)
- [ ] 12.2 Write test for release selector mutual exclusivity
- [ ] 12.3 Write test for missing namespace flag error
- [ ] 12.4 Write test for depth validation (invalid value rejected)
- [ ] 12.5 Write test for output format validation (invalid format rejected)

## 13. Tree Tests (internal/kubernetes/tree_test.go)

- [ ] 13.1 Write tests for component grouping with various label scenarios
- [ ] 13.2 Write tests for component sorting (alphabetical + (no component) last)
- [ ] 13.3 Write tests for resource sorting within components (weight + name)
- [ ] 13.4 Write tests for ownership walking (Deployment→RS→Pod)
- [ ] 13.5 Write tests for ownership walking (StatefulSet→Pod)
- [ ] 13.6 Write tests for ownership walking error handling (RBAC failure)
- [ ] 13.7 Write tests for replica count extraction (all workload kinds)
- [ ] 13.8 Write tests for BuildTree at depth=0 (summary only)
- [ ] 13.9 Write tests for BuildTree at depth=1 (no children)
- [ ] 13.10 Write tests for BuildTree at depth=2 (full tree)
- [ ] 13.11 Write tests for FormatTree with colors (verify ANSI codes)
- [ ] 13.12 Write tests for formatPlainTree (verify no ANSI codes)
- [ ] 13.13 Write tests for JSON output schema
- [ ] 13.14 Write tests for YAML output format
- [ ] 13.15 Write tests for GetModuleTree with no resources found

## 14. Integration Tests

- [ ] 14.1 Add integration test: deploy test module with multiple components
- [ ] 14.2 Add integration test: verify tree output at depth=0 (component summary)
- [ ] 14.3 Add integration test: verify tree output at depth=1 (resources only)
- [ ] 14.4 Add integration test: verify tree output at depth=2 (full hierarchy)
- [ ] 14.5 Add integration test: verify JSON output structure matches schema
- [ ] 14.6 Add integration test: verify no resources found error (exit code 3)
- [ ] 14.7 Add integration test: verify component grouping with real resources
- [ ] 14.8 Add integration test: verify ownership walking (Deployment→Pod visible)

## 15. Documentation

- [ ] 15.1 Add help text and examples to `NewModTreeCmd()` long description
- [ ] 15.2 Update CLI README with `opm mod tree` command
- [ ] 15.3 Add tree command to command reference docs (if exists)
- [ ] 15.4 Document depth flag behavior and use cases

## 16. Validation & Polish

- [ ] 16.1 Run `task fmt` and fix any formatting issues
- [ ] 16.2 Run `task lint` and fix any linter warnings
- [ ] 16.3 Run `task test` and ensure all tests pass
- [ ] 16.4 Run integration tests against kind cluster
- [ ] 16.5 Test against real multi-component module deployment
- [ ] 16.6 Verify exit codes match spec (0=success, 1=error, 3=not found)
- [ ] 16.7 Verify colored output in terminal
- [ ] 16.8 Verify plain output in non-TTY (e.g., `| cat`)
- [ ] 16.9 Verify JSON output is valid and parseable
- [ ] 16.10 Verify YAML output is valid and parseable
