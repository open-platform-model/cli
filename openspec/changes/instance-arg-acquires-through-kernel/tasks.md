## 1. Shared directory resolution (internal/cmdutil, internal/workflow/render)

- [ ] 1.1 Add `cmdutil.InstanceDir(path string) (string, error)` (the path when it is a directory, else its parent; a missing path resolves to its parent as `render.resolveInstanceDir` does today) with a unit test for file, directory and missing inputs; replace `resolveInstanceDir` in `internal/workflow/render/values.go` with it; verify `go test ./internal/cmdutil/... ./internal/workflow/render/...` pass.

## 2. Instance argument through the kernel (internal/cmdutil, internal/cmd/instance)

- [ ] 2.1 Change `ResolveInstanceArg` and `ResolveInstanceTarget` to take a leading `context.Context`; in `resolveInstanceArgFromFile` replace the bare `cuecontext.New()` + `loader.LoadInstanceFile` + `extractInstanceFileIdentity` with `InstanceDir` + `config.NewKernel(cfg.Registry).AcquireInstanceFromDir(ctx, dir)`, reading `inst.Metadata.Name` and `.Namespace` and wrapping a failure as `loading instance %q: %w`; delete `extractInstanceFileIdentity`; update `internal/cmd/instance/{status,tree,events,delete}.go` to pass `c.Context()`; verify `go build ./...` is green and `grep -rn 'pkg/loader"' internal/cmdutil/` is empty.
- [ ] 2.2 Add tests in `internal/cmdutil`: path detection unchanged for the three identifier forms, and a registry-backed test (following the pattern the existing integration tests use for GHCR-resolved fixtures) that resolves an instance package directory to its declared name and namespace and refuses a directory whose `kind` is not `ModuleInstance` with an error naming the path; verify `go test ./internal/cmdutil/...` passes.

## 3. Delete the loader copy (pkg/loader)

- [ ] 3.1 Delete `pkg/loader/instance_file.go`, `instance_file_test.go` and `local_module_resolution_test.go`; keep `provenance.go` and `provenance_test.go`; verify `go build ./...` and `go test ./pkg/loader/...` pass and `grep -rn 'LoadInstanceFile\|loader.LoadOptions\|os.Setenv' --include='*.go' internal/ pkg/` is empty.

## 4. Gates

- [ ] 4.1 Run `task fmt`, `task lint`, `task test`; run the e2e instance tests that delete by identifier (`tests/e2e/instance_operator_owned_test.go`); verify all green.
