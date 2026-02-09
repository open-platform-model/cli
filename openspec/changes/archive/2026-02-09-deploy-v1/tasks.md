# Tasks: CLI Deploy Commands

## Prerequisites

- [x] **build-v1 must be implemented first** (provides Pipeline.Render())
- [x] **render-pipeline-v1 types** must be available in `internal/build/`

---

## Phase 1: Kubernetes Client Package

### 1.1 Client Infrastructure

- [x] 1.1.1 Create `internal/kubernetes/` package directory
- [x] 1.1.2 Create `internal/kubernetes/client.go` with client initialization
- [x] 1.1.3 Implement kubeconfig resolution (flag > env > default)
- [x] 1.1.4 Implement context selection support
- [x] 1.1.5 Add client caching for reuse within command

### 1.2 Resource Discovery

- [x] 1.2.1 Create `internal/kubernetes/discovery.go`
- [x] 1.2.2 Implement label selector builder for OPM labels
- [x] 1.2.3 Implement resource listing by label
- [x] 1.2.4 Handle namespace vs cluster-scoped resources

---

## Phase 2: Apply Command

### 2.1 Apply Implementation

- [x] 2.1.1 Create `internal/kubernetes/apply.go`
- [x] 2.1.2 Implement server-side apply with force conflicts
- [x] 2.1.3 Implement OPM label injection
- [x] 2.1.4 Implement ordered apply (resources already ordered in RenderResult)
- [x] 2.1.5 Implement dry-run support

### 2.2 CLI Command

- [x] 2.2.1 Create `internal/cmd/mod/apply.go`
- [x] 2.2.2 Replace stub with implementation
- [x] 2.2.3 Add `--values` / `-f` flag (repeatable)
- [x] 2.2.4 Add `--namespace` / `-n` flag
- [x] 2.2.5 Add `--name` flag
- [x] 2.2.6 Add `--provider` flag
- [x] 2.2.7 Add `--dry-run` flag
- [x] 2.2.8 Add `--wait` flag
- [x] 2.2.9 Add `--timeout` flag
- [x] 2.2.10 Add `--kubeconfig` flag
- [x] 2.2.11 Add `--context` flag

### 2.3 Integration

- [x] 2.3.1 Call `build.NewPipeline().Render()` to get resources
- [x] 2.3.2 Check RenderResult.HasErrors() before applying
- [x] 2.3.3 Pass resources to kubernetes.Apply()
- [x] 2.3.4 Handle and display apply errors

---

## Phase 3: Delete Command

### 3.1 Delete Implementation

- [x] 3.1.1 Create `internal/kubernetes/delete.go`
- [x] 3.1.2 Implement resource discovery via labels
- [x] 3.1.3 Implement reverse weight ordering for deletion
- [x] 3.1.4 Implement delete with foreground propagation
- [x] 3.1.5 Implement dry-run support

### 3.2 CLI Command

- [x] 3.2.1 Create `internal/cmd/mod/delete.go`
- [x] 3.2.2 Replace stub with implementation
- [x] 3.2.3 Add `--namespace` / `-n` flag (required)
- [x] 3.2.4 Add `--name` flag (required)
- [x] 3.2.5 Add `--force` flag
- [x] 3.2.6 Add `--dry-run` flag
- [x] 3.2.7 Add `--wait` flag
- [x] 3.2.8 Add `--kubeconfig` flag
- [x] 3.2.9 Add `--context` flag
- [x] 3.2.10 Implement confirmation prompt

---

## Phase 4: Resource Labeling

- [x] 4.1 Verify OPM labels are set by TransformerContext (build-v1)
- [x] 4.2 Implement label addition in apply if missing
- [x] 4.3 Implement Resource.Component and Resource.Transformer access

---

## Phase 5: Testing

### 5.1 Unit Tests

- [x] 5.1.1 Add tests for client initialization
- [x] 5.1.2 Add tests for label selector building
- [x] 5.1.3 Add tests for weight ordering

### 5.2 Integration Tests

- [x] 5.2.1 Set up envtest for integration testing
- [x] 5.2.2 Add apply/delete round-trip test
- [x] 5.2.3 Add idempotency test

---

## Phase 6: Validation Gates

- [x] 6.1 Run `task fmt` - verify Go files formatted
- [x] 6.2 Run `task lint` - verify golangci-lint passes
- [x] 6.3 Run `task test` - verify all tests pass
- [x] 6.4 Manual testing with test module on real cluster
