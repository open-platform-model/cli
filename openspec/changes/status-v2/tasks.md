## 1. Data Model & Types

- [ ] 1.1 Update `resourceHealth` struct in `internal/kubernetes/status.go`: add `Component string` field with `json:"component,omitempty" yaml:"component,omitempty"` tags
- [ ] 1.2 Add `wideInfo` struct in `internal/kubernetes/status.go` with `Replicas` and `Image` string fields, add `Wide *wideInfo` field to `resourceHealth`
- [ ] 1.3 Add `verboseInfo` struct with `Pods []podInfo` field and `podInfo` struct with `Name`, `Phase`, `Ready`, `Reason`, `Restarts` fields, add `Verbose *verboseInfo` field to `resourceHealth`
- [ ] 1.4 Update `StatusResult` struct: replace `ModuleID`/`ReleaseID` with `ReleaseName`, `Version`, `Namespace` fields, add `Summary statusSummary` struct with `Total`, `Ready`, `NotReady` int fields
- [ ] 1.5 Add `Wide bool`, `Verbose bool`, `Version string`, and `ComponentMap map[string]string` fields to `StatusOptions` struct
- [ ] 1.6 Update existing `status_test.go` tests for the new struct shapes (fix compilation)

## 2. Output Package

- [ ] 2.1 Add `FormatWide Format = "wide"` constant to `internal/output/format.go`, update `Valid()` switch and `ValidFormats()` slice to include it
- [ ] 2.2 Add `FormatHealthStatus(status string) string` function to `internal/output/styles.go` — maps Ready/Complete to green, NotReady to red, Unknown to yellow
- [ ] 2.3 Add `FormatComponent(name string) string` function to `internal/output/styles.go` — renders non-empty names in cyan via `styleNoun`, returns `-` unstyled for empty
- [ ] 2.4 Write unit tests for `FormatHealthStatus` and `FormatComponent` (verify non-empty output for each status value, verify `-` for empty component)

## 3. Wide Info Extraction

- [ ] 3.1 Create `internal/kubernetes/wide.go` with `extractWideInfo(resource *unstructured.Unstructured) *wideInfo` function
- [ ] 3.2 Implement Deployment/StatefulSet extraction: `status.readyReplicas`/`spec.replicas` for replicas, `spec.template.spec.containers[0].image` for image
- [ ] 3.3 Implement DaemonSet extraction: `status.numberReady`/`status.desiredNumberScheduled` for replicas, same image path
- [ ] 3.4 Implement PVC extraction: `status.capacity.storage` + `status.phase` for replicas column, no image
- [ ] 3.5 Implement Ingress extraction: `spec.rules[0].host` for image column, no replicas
- [ ] 3.6 Write unit tests for `extractWideInfo` with table-driven tests covering each kind and edge cases (missing fields, zero replicas, empty containers list, no rules)

## 4. Pod Diagnostics

- [ ] 4.1 Create `internal/kubernetes/pods.go` with `listWorkloadPods(ctx, client, resource *unstructured.Unstructured) ([]podInfo, error)` function
- [ ] 4.2 Implement label selector extraction from workload's `.spec.selector.matchLabels`
- [ ] 4.3 Implement pod listing via `client.Clientset.CoreV1().Pods(namespace).List()` with the extracted label selector
- [ ] 4.4 Implement pod status extraction: phase from `status.phase`, ready from conditions, waiting reason from `containerStatuses[].state.waiting.reason`, termination reason from `containerStatuses[].lastState.terminated.reason`, restart count summed across containers
- [ ] 4.5 Write unit tests for pod status extraction logic (use constructed pod objects, not live cluster): CrashLoopBackOff, OOMKilled, ImagePullBackOff, Pending, Running/Ready, zero restarts

## 5. Core Status Logic

- [ ] 5.1 In the command layer (`runStatus`), build `StatusOptions.ComponentMap` from inventory entries: iterate `inv.Changes[inv.Index[0]].Inventory.Entries`, key = `entry.Kind+"/"+entry.Namespace+"/"+entry.Name`, value = `entry.Component`
- [ ] 5.2 In the command layer (`runStatus`), extract `StatusOptions.Version` from `inv.Changes[inv.Index[0]].Source.Version` (empty string if no changes or local module)
- [ ] 5.3 Update `GetReleaseStatus` to read component from `opts.ComponentMap[key]` for each live resource (key = `Kind/Namespace/Name`) and populate `resourceHealth.Component`; skip ComponentMap lookup for `MissingResource` entries
- [ ] 5.4 Update `GetReleaseStatus` to populate `StatusResult.ReleaseName` from `opts.ReleaseName`, `.Version` from `opts.Version`, `.Namespace` from `opts.Namespace`
- [ ] 5.5 Update `GetReleaseStatus` to compute `StatusResult.Summary` (total, ready, not ready counts); `MissingResource` entries count as not-ready
- [ ] 5.6 Update `GetReleaseStatus` to call `extractWideInfo()` for each **live** resource when `opts.Wide` is true; never call for `MissingResource` entries
- [ ] 5.7 Update `GetReleaseStatus` to call `listWorkloadPods()` for each unhealthy **live** workload when `opts.Verbose` is true; never call for `MissingResource` entries
- [ ] 5.8 Remove old `ModuleID`/`ReleaseID` population code from `GetReleaseStatus` (the `labels[core.LabelModuleUUID]` and `labels[core.LabelReleaseUUID]` reads)

## 6. Table Formatting

- [ ] 6.1 Update `FormatStatusTable` to render the new metadata header (Release, Version, Namespace, Status, Resources summary) with color
- [ ] 6.2 Update `FormatStatusTable` default table to include COMPONENT column (KIND, NAME, COMPONENT, STATUS, AGE), apply `FormatHealthStatus` and `FormatComponent` to cell values
- [ ] 6.3 Add wide table rendering path in `FormatStatus`: when `format == output.FormatWide`, render table with columns KIND, NAME, COMPONENT, STATUS, REPLICAS, IMAGE, AGE
- [ ] 6.4 Add verbose rendering path: after the table, render pod detail blocks for each resource with non-nil `Verbose` data, formatted as `Kind/Name (ready/total ready):` followed by indented pod lines
- [ ] 6.5 Update `formatStatusJSON` and `formatStatusYAML` to handle new fields (these should work automatically via struct tags, but verify output)
- [ ] 6.6 Write unit tests for `FormatStatusTable` covering: default table with component column, wide table with replicas/image, verbose output with pod detail blocks, header with ready/not-ready summary

## 7. Command Wiring

- [ ] 7.1 Add `--verbose` flag to `NewModStatusCmd` (bool, default false)
- [ ] 7.2 Update `-o` flag validation in `runStatus`: `output.ParseFormat` now accepts `wide`; when `outputFormat == output.FormatWide`, set `opts.Wide = true` and `opts.OutputFormat = output.FormatWide`
- [ ] 7.3 Wire `Wide` and `Verbose` flags through to `StatusOptions` in `runStatus`
- [ ] 7.4 Update `fetchAndPrintStatus` exit code logic: explicitly check `kubernetes.IsNoResourcesFound(err)` first and return `ExitNotFound(5)` (with `--ignore-not-found` override to 0); for aggregate NotReady return `ExitValidationError(2)`; for all healthy return `ExitSuccess(0)`
- [ ] 7.5 Update `runStatusWatch` (watch mode) to pass `Wide` and `Verbose` flags through to `StatusOptions`
- [ ] 7.6 Update command `Long` help text and examples to document `--verbose`, `-o wide`, and exit codes
- [ ] 7.7 Update existing `mod_status_test.go` flag tests: add test for `--verbose` flag existence, test for `-o wide` acceptance

## 8. Validation

- [ ] 8.1 Run `task fmt` — verify all files formatted
- [ ] 8.2 Run `task test` — verify all tests pass (existing + new)
- [ ] 8.3 Run `task check` — verify full check suite (fmt + vet + test)
