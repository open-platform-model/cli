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

## Phase 4: Resource Labeling

- [ ] 4.1 Verify OPM labels are set by TransformerContext (build-v1)
- [ ] 4.2 Implement label addition in apply if missing
- [ ] 4.3 Implement Resource.Component and Resource.Transformer access

---

## Phase 5: Testing

### 5.1 Unit Tests

- [ ] 5.1.1 Add tests for client initialization
- [ ] 5.1.2 Add tests for label selector building
- [ ] 5.1.3 Add tests for weight ordering

### 5.2 Integration Tests

- [ ] 5.2.1 Set up envtest for integration testing
- [ ] 5.2.2 Add apply/delete round-trip test
- [ ] 5.2.3 Add idempotency test

---

## Phase 6: Validation Gates

- [ ] 6.1 Run `task fmt` - verify Go files formatted
- [ ] 6.2 Run `task lint` - verify golangci-lint passes
- [ ] 6.3 Run `task test` - verify all tests pass
- [ ] 6.4 Manual testing with test module on real cluster
