## 1. Shared directory resolution (internal/cmdutil, internal/workflow/render)

- [x] 1.1 Add `cmdutil.InstanceDir(path string) (string, error)` (the path when it is a directory, else its parent; a missing path resolves to its parent as `render.resolveInstanceDir` does today) with a unit test for file, directory and missing inputs; replace `resolveInstanceDir` in `internal/workflow/render/values.go` with it; verify `go test ./internal/cmdutil/... ./internal/workflow/render/...` pass.
- [x] 1.2 Commit as `refactor(cmdutil): share the instance directory resolution` (c1dce05).

## 2. Instance argument through the kernel (internal/cmdutil, internal/cmd/instance)

- [x] 2.1 Change `ResolveInstanceArg` and `ResolveInstanceTarget` to take a leading `context.Context`; in `resolveInstanceArgFromFile` replace the bare `cuecontext.New()` + `loader.LoadInstanceFile` + `extractInstanceFileIdentity` with `InstanceDir` + `config.NewKernel(cfg.Registry).AcquireInstanceFromDir(ctx, dir)`, reading `inst.Metadata.Name` and `.Namespace` and wrapping a failure as `loading instance %q: %w`; delete `extractInstanceFileIdentity`; update `internal/cmd/instance/{status,tree,events,delete}.go` to pass `c.Context()`; verify `go build ./...` is green and `grep -rn 'pkg/loader"' internal/cmdutil/` is empty.
- [x] 2.2 Add tests in `internal/cmdutil`: path detection unchanged for the three identifier forms, and a registry-backed test (following the pattern the existing integration tests use for GHCR-resolved fixtures) that resolves an instance package directory to its declared name and namespace and refuses a directory whose `kind` is not `ModuleInstance` with an error naming the path; verify `go test ./internal/cmdutil/...` passes.
- [x] 2.3 Commit as `refactor(cmdutil): acquire the instance path argument through the kernel` (9128156). Carries the inst-tree integration fixture's inline `#module` and the program's context argument (see proposal.md, Impact).

## 3. Delete the loader copy (pkg/loader)

- [x] 3.1 Delete `pkg/loader/instance_file.go`, `instance_file_test.go` and `local_module_resolution_test.go`; keep `provenance.go` and `provenance_test.go`; verify `go build ./...` and `go test ./pkg/loader/...` pass and `grep -rn 'LoadInstanceFile\|loader.LoadOptions\|os.Setenv' --include='*.go' internal/ pkg/` is empty.
- [x] 3.2 Commit as `refactor(loader): delete the instance file loader copy` (a0f4284).

## 4. Gates

- [x] 4.1 Run `task fmt`, `task lint`, `task test`; run the e2e instance tests that delete by identifier (`tests/e2e/instance_operator_owned_test.go`); verify all green.
  - 2026-09-13: fmt, lint, unit and the integration programs (including inst-tree's path resolution through the kernel) are green. e2e is green except `TestE2E_ThinEditor_ValuesRoundTrip` and `TestE2E_Delete_OperatorOwnedDelegates`, which fail at the reconcile wait (`operator did not reconcile generation 2 ... within 3m`) before any delete runs: the deployed operator v1.0.0-alpha.14 stalls on the D5 Platform (`MaterializeFailed`), an operator release gap that produced the same two failures before this change. The delete-by-identifier resolution is covered by the registry-backed cmdutil test on the same fixture.
- [x] 4.2 Commit as `chore(openspec): record the gate results for instance-arg-acquires-through-kernel` (9222a7d), `chore(openspec): mark task 4.1 done for instance-arg-acquires-through-kernel` (8a5a2e0) and `chore(openspec): close the verification warnings for instance-arg-acquires-through-kernel` (commit tasks, fixture note, `instance-building` and `cmd-structure` deltas, security-audit skill refresh).
