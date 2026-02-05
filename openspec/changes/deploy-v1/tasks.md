# Tasks: CLI Deploy Commands

## Prerequisites

- [ ] **build-v1 must be implemented first** (provides Pipeline.Render())
- [ ] **render-pipeline-v1 types** must be available in `internal/build/`

---

## Phase 1: Kubernetes Client Package

### 1.1 Client Infrastructure

- [ ] 1.1.1 Create `internal/kubernetes/` package directory
- [ ] 1.1.2 Create `internal/kubernetes/client.go` with client initialization
- [ ] 1.1.3 Implement kubeconfig resolution (flag > env > default)
- [ ] 1.1.4 Implement context selection support
- [ ] 1.1.5 Add client caching for reuse within command

### 1.2 Resource Discovery

- [ ] 1.2.1 Create `internal/kubernetes/discovery.go`
- [ ] 1.2.2 Implement label selector builder for OPM labels
- [ ] 1.2.3 Implement resource listing by label
- [ ] 1.2.4 Handle namespace vs cluster-scoped resources

---

## Phase 2: Apply Command

### 2.1 Apply Implementation

- [ ] 2.1.1 Create `internal/kubernetes/apply.go`
- [ ] 2.1.2 Implement server-side apply with force conflicts
- [ ] 2.1.3 Implement OPM label injection
- [ ] 2.1.4 Implement ordered apply (resources already ordered in RenderResult)
- [ ] 2.1.5 Implement dry-run support

### 2.2 CLI Command

- [ ] 2.2.1 Create `internal/cmd/mod/apply.go`
- [ ] 2.2.2 Replace stub with implementation
- [ ] 2.2.3 Add `--values` / `-f` flag (repeatable)
- [ ] 2.2.4 Add `--namespace` / `-n` flag
- [ ] 2.2.5 Add `--name` flag
- [ ] 2.2.6 Add `--provider` flag
- [ ] 2.2.7 Add `--dry-run` flag
- [ ] 2.2.8 Add `--wait` flag
- [ ] 2.2.9 Add `--timeout` flag
- [ ] 2.2.10 Add `--kubeconfig` flag
- [ ] 2.2.11 Add `--context` flag

### 2.3 Integration

- [ ] 2.3.1 Call `build.NewPipeline().Render()` to get resources
- [ ] 2.3.2 Check RenderResult.HasErrors() before applying
- [ ] 2.3.3 Pass resources to kubernetes.Apply()
- [ ] 2.3.4 Handle and display apply errors

---

## Phase 3: Delete Command

### 3.1 Delete Implementation

- [ ] 3.1.1 Create `internal/kubernetes/delete.go`
- [ ] 3.1.2 Implement resource discovery via labels
- [ ] 3.1.3 Implement reverse weight ordering for deletion
- [ ] 3.1.4 Implement delete with foreground propagation
- [ ] 3.1.5 Implement dry-run support

### 3.2 CLI Command

- [ ] 3.2.1 Create `internal/cmd/mod/delete.go`
- [ ] 3.2.2 Replace stub with implementation
- [ ] 3.2.3 Add `--namespace` / `-n` flag (required)
- [ ] 3.2.4 Add `--name` flag (required)
- [ ] 3.2.5 Add `--force` flag
- [ ] 3.2.6 Add `--dry-run` flag
- [ ] 3.2.7 Add `--wait` flag
- [ ] 3.2.8 Add `--kubeconfig` flag
- [ ] 3.2.9 Add `--context` flag
- [ ] 3.2.10 Implement confirmation prompt

---

## Phase 4: Diff Command

### 4.1 Diff Implementation

- [ ] 4.1.1 Create `internal/kubernetes/diff.go`
- [ ] 4.1.2 Implement live state fetching
- [ ] 4.1.3 Integrate dyff for semantic diff
- [ ] 4.1.4 Handle missing resources (additions)
- [ ] 4.1.5 Handle extra resources (deletions)
- [ ] 4.1.6 Implement colorized output

### 4.2 CLI Command

- [ ] 4.2.1 Create `internal/cmd/mod/diff.go`
- [ ] 4.2.2 Replace stub with implementation
- [ ] 4.2.3 Add `--values` / `-f` flag
- [ ] 4.2.4 Add `--namespace` / `-n` flag
- [ ] 4.2.5 Add `--name` flag
- [ ] 4.2.6 Add `--kubeconfig` flag
- [ ] 4.2.7 Add `--context` flag

### 4.3 Integration

- [ ] 4.3.1 Call `build.NewPipeline().Render()` to get resources
- [ ] 4.3.2 Allow partial results (with warnings)
- [ ] 4.3.3 Compare each resource with live state

---

## Phase 5: Status Command

### 5.1 Health Implementation

- [ ] 5.1.1 Create `internal/kubernetes/health.go`
- [ ] 5.1.2 Implement workload health check (Ready condition)
- [ ] 5.1.3 Implement job health check (Complete condition)
- [ ] 5.1.4 Implement passive resource health (always healthy)
- [ ] 5.1.5 Implement custom resource health (Ready if present)

### 5.2 Status Implementation

- [ ] 5.2.1 Create `internal/kubernetes/status.go`
- [ ] 5.2.2 Implement resource discovery via labels
- [ ] 5.2.3 Implement health evaluation per resource
- [ ] 5.2.4 Implement table output
- [ ] 5.2.5 Implement YAML/JSON output
- [ ] 5.2.6 Implement watch mode

### 5.3 CLI Command

- [ ] 5.3.1 Create `internal/cmd/mod/status.go`
- [ ] 5.3.2 Replace stub with implementation
- [ ] 5.3.3 Add `--namespace` / `-n` flag (required)
- [ ] 5.3.4 Add `--name` flag (required)
- [ ] 5.3.5 Add `--output` / `-o` flag (table, yaml, json)
- [ ] 5.3.6 Add `--watch` flag
- [ ] 5.3.7 Add `--kubeconfig` flag
- [ ] 5.3.8 Add `--context` flag

---

## Phase 6: Resource Labeling

- [ ] 6.1 Verify OPM labels are set by TransformerContext (build-v1)
- [ ] 6.2 Implement label addition in apply if missing
- [ ] 6.3 Implement Resource.Component and Resource.Transformer access

---

## Phase 7: Testing

### 7.1 Unit Tests

- [ ] 7.1.1 Add tests for client initialization
- [ ] 7.1.2 Add tests for label selector building
- [ ] 7.1.3 Add tests for weight ordering
- [ ] 7.1.4 Add tests for health evaluation

### 7.2 Integration Tests

- [ ] 7.2.1 Set up envtest for integration testing
- [ ] 7.2.2 Add apply/delete round-trip test
- [ ] 7.2.3 Add diff accuracy test
- [ ] 7.2.4 Add status reporting test
- [ ] 7.2.5 Add idempotency test

---

## Phase 8: Validation Gates

- [ ] 8.1 Run `task fmt` - verify Go files formatted
- [ ] 8.2 Run `task lint` - verify golangci-lint passes
- [ ] 8.3 Run `task test` - verify all tests pass
- [ ] 8.4 Manual testing with test module on real cluster
