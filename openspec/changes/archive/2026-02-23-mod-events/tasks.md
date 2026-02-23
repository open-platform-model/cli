## 1. Child Resource Discovery

- [x] 1.1 Create `internal/kubernetes/children.go` with `DiscoverChildren` function that accepts parent resources and returns Kubernetes-owned children (as `[]*unstructured.Unstructured`) via ownerReference traversal. Note: `internal/kubernetes/tree.go` already implements equivalent ownership walking (`walkOwnership`, `walkDeployment`, `walkStatefulSet`, etc.) returning `[]ResourceNode` — follow those same traversal patterns, returning child resources rather than display nodes.
- [x] 1.2 Implement targeted traversal for each workload kind: Deployment→ReplicaSet→Pod, StatefulSet→Pod, DaemonSet→Pod, Job→Pod, CronJob→Job→Pod
- [x] 1.3 Implement ownerReference UID matching logic — list candidate children by kind, filter by `ownerReferences[].uid` against parent UID. Reuse the `hasOwnerWithUID` helper from `tree.go` (or duplicate if keeping packages clean).
- [x] 1.4 Handle non-fatal API errors during child listing (log warning, continue with other parents)
- [x] 1.5 Create `internal/kubernetes/children_test.go` with table-driven tests: Deployment children, StatefulSet children, non-workload parents skipped, no children exist, API errors non-fatal

## 2. Event Collection and Filtering

- [x] 2.1 Create `internal/kubernetes/events.go` with `EventsOptions` struct (Namespace, ReleaseName, ReleaseID, Since, EventType, OutputFormat)
- [x] 2.2 Implement `GetModuleEvents` function: discover resources → discover children → collect UIDs → bulk fetch events → filter by UID + since + type → sort by lastTimestamp ascending
- [x] 2.3 Implement `parseSince` function for `--since` flag parsing — Go duration syntax plus `d` day extension (e.g., `30m`, `1h`, `2h30m`, `1d`, `7d`)
- [x] 2.4 Implement `EventsResult` and `EventEntry` types for structured output (JSON/YAML fields: lastSeen, type, kind, name, reason, message, count, firstSeen)
- [x] 2.5 Create `internal/kubernetes/events_test.go` with tests: UID filtering, since filtering, type filtering, parseSince edge cases, empty results, sort order

## 3. Event Formatting

- [x] 3.1 Implement `FormatEventsTable` — table with columns LAST SEEN, TYPE, RESOURCE, REASON, MESSAGE using `output.NewTable`
- [x] 3.2 Implement color coding in table output: Warning=yellow (ColorYellow), Normal=dim gray (colorDimGray), Resource=cyan (ColorCyan/styleNoun)
- [x] 3.3 Implement `FormatEvents` dispatcher for table/json/yaml output formats using the same pattern as `status.go:FormatStatus`
- [x] 3.4 Add tests for formatting: table column content, JSON structure validity, YAML structure validity

## 4. Command Definition

- [x] 4.1 Create `internal/cmd/mod/events.go` with `NewModEventsCmd` using cobra, `ReleaseSelectorFlags`, `K8sFlags`, and events-specific flags (--since, --type, --watch, --output, --ignore-not-found). Follow the structure of `internal/cmd/mod/tree.go` or `status.go` as a reference — use `cmdutil.ResolveInventory` for resource discovery.
- [x] 4.2 Implement `runEvents` function: validate flags → resolve K8s config → create client → call `cmdutil.ResolveInventory` → call `GetModuleEvents` → format and print
- [x] 4.3 Implement `--type` flag validation (reject values other than `Normal`, `Warning`)
- [x] 4.4 Implement `--output` flag validation (reject values other than `table`, `json`, `yaml`)
- [x] 4.5 Register `NewModEventsCmd` in `internal/cmd/mod/mod.go` via `c.AddCommand`
- [x] 4.6 Create `internal/cmd/mod/events_test.go` with flag validation tests: mutual exclusivity of selectors, invalid --type, invalid --output, invalid --since, flags exist

## 5. Watch Mode

- [x] 5.1 Implement `runEventsWatch` function: discover resources + children → collect UID set → start Kubernetes Watch on Events(namespace) → filter incoming events by UID set + type + since → append formatted output
- [x] 5.2 Implement signal handling for clean shutdown (SIGINT/SIGTERM → context cancellation → exit code 0), matching existing `runStatusWatch` pattern
- [x] 5.3 Add watch mode test for signal handling and clean exit

## 6. Validation Gates

- [x] 6.1 Run `task fmt` — verify all new files pass gofmt
- [x] 6.2 Run `task test` — verify all new and existing tests pass
- [x] 6.3 Run `task check` — verify fmt + vet + test all pass
