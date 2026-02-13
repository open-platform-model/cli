# Tasks: mod diff & mod status

## Prerequisites

- [x] **deploy-v1 Kubernetes client infrastructure must exist** (`internal/kubernetes/client.go`, `internal/kubernetes/discovery.go`)

---

## 1. Add dyff Dependency

- [x] 1.1 Run `go get github.com/homeport/dyff/...` to add dyff to go.mod
- [x] 1.2 Verify `go mod tidy` completes without errors

## 2. Diff Business Logic (`internal/kubernetes/diff.go`)

- [x] 2.1 Create `internal/kubernetes/diff.go` with `DiffOptions` struct (namespace, name, kubeconfig, context)
- [x] 2.2 Implement `FetchLiveState(ctx, client, resource)` to get a single resource from the cluster
- [x] 2.3 Implement `CompareResource(rendered, live)` using dyff for semantic YAML comparison
- [x] 2.4 Implement resource categorization into three states: modified, added, orphaned
- [x] 2.5 Implement orphan detection by comparing rendered resources against label-discovered resources
- [x] 2.6 Implement `DiffResult` struct with modified/added/orphaned counts and per-resource details
- [x] 2.7 Implement summary line formatting ("N modified, M added, K orphaned")
- [x] 2.8 Implement partial render handling — compare successful resources, collect warnings for failures
- [x] 2.9 Wrap dyff behind a thin adapter so the library can be swapped

## 3. Diff CLI Command (`internal/cmd/mod_diff.go`)

- [x] 3.1 Create `internal/cmd/mod_diff.go` with `NewModDiffCmd()` returning `*cobra.Command`
- [x] 3.2 Add `--values`/`-f` flag (repeatable string slice)
- [x] 3.3 Add `--namespace`/`-n` flag
- [x] 3.4 Add `--name` flag
- [x] 3.5 Add `--kubeconfig` flag
- [x] 3.6 Add `--context` flag
- [x] 3.7 Add `path` positional argument (default: current directory)
- [x] 3.8 Implement `RunE`: call `build.NewPipeline().Render()`, then pass resources to diff logic
- [x] 3.9 Handle connectivity errors — fail fast with exit code 3 and clear message
- [x] 3.10 Display "No differences found" when diff result is empty
- [x] 3.11 Replace `NewModDiffStubCmd()` registration in `mod.go` with `NewModDiffCmd()`
- [x] 3.12 Remove `NewModDiffStubCmd()` from `mod_stubs.go`

## 4. Health Evaluation (`internal/kubernetes/health.go`)

- [x] 4.1 Create `internal/kubernetes/health.go` with `HealthStatus` type (Ready, NotReady, Complete, Unknown)
- [x] 4.2 Implement `EvaluateHealth(resource)` dispatcher that routes to category-specific logic
- [x] 4.3 Implement workload health check — Deployment, StatefulSet, DaemonSet: Ready condition True
- [x] 4.4 Implement job health check — Job: Complete condition True
- [x] 4.5 Implement cronJob health — always healthy
- [x] 4.6 Implement passive resource health — ConfigMap, Secret, Service, PVC: healthy on creation
- [x] 4.7 Implement custom resource health — Ready condition if present, else passive

## 5. Status Business Logic (`internal/kubernetes/status.go`)

- [x] 5.1 Create `internal/kubernetes/status.go` with `StatusOptions` struct (namespace, name, output format, watch)
- [x] 5.2 Implement `GetModuleStatus(ctx, client, opts)` — discover resources by OPM labels, evaluate health per resource
- [x] 5.3 Implement `StatusResult` struct with resource list, per-resource health, and aggregate status
- [x] 5.4 Implement table output formatting (KIND, NAME, NAMESPACE, STATUS, AGE columns)
- [x] 5.5 Implement JSON output formatting
- [x] 5.6 Implement YAML output formatting
- [x] 5.7 Implement "No resources found" message when label query returns empty

## 6. Status CLI Command (`internal/cmd/mod_status.go`)

- [x] 6.1 Create `internal/cmd/mod_status.go` with `NewModStatusCmd()` returning `*cobra.Command`
- [x] 6.2 Add `--namespace`/`-n` flag (required)
- [x] 6.3 Add `--name` flag (required)
- [x] 6.4 Add `--output`/`-o` flag (table, yaml, json; default: table)
- [x] 6.5 Add `--watch` flag
- [x] 6.6 Add `--kubeconfig` flag
- [x] 6.7 Add `--context` flag
- [x] 6.8 Implement required flag validation — exit code 1 with usage error if `--name` or `-n` missing
- [x] 6.9 Implement `RunE`: build label selector, call status logic, format output
- [x] 6.10 Implement watch mode — poll every 2s, clear and redraw table, exit cleanly on Ctrl+C
- [x] 6.11 Handle connectivity errors — fail fast with exit code 3 and clear message
- [x] 6.12 Replace `NewModStatusStubCmd()` registration in `mod.go` with `NewModStatusCmd()`
- [x] 6.13 Remove `NewModStatusStubCmd()` from `mod_stubs.go`

## 7. Unit Tests

- [x] 7.1 Add table-driven tests for `CompareResource` — identical resources, modified fields, field reordering
- [x] 7.2 Add table-driven tests for resource categorization — modified, added, orphaned states
- [x] 7.3 Add table-driven tests for `EvaluateHealth` — each resource category (workload, job, cronJob, passive, custom)
- [x] 7.4 Add table-driven tests for summary line formatting
- [x] 7.5 Add tests for required flag validation on mod status
- [x] 7.6 Add tests for output format selection (table, json, yaml)

## 8. Integration Tests

- [x] 8.1 Set up envtest fixture for diff/status integration tests
- [x] 8.2 Add integration test: deploy resources, modify locally, verify diff shows modifications
- [x] 8.3 Add integration test: deploy resources, verify status reports correct health
- [x] 8.4 Add integration test: diff with no prior deployment shows all as additions
- [x] 8.5 Add integration test: status with no matching resources returns empty message

## 9. Validation Gates

- [x] 9.1 Run `task fmt` — verify all Go files formatted
- [x] 9.2 Run `task lint` — verify golangci-lint passes
- [x] 9.3 Run `task test` — verify all tests pass
