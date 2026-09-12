## 1. Constructor

- [ ] 1.1 Add `internal/config/kernel.go` with `NewKernel(registry string) *kernel.Kernel` returning `kernel.New(kernel.WithRegistry(registry))`, doc comment naming it the CLI's single construction site and stating that the schema loader is seeded from the same mapping by the library; add a unit test asserting a non-nil kernel with a non-nil `SchemaCache()`; verify `go test ./internal/config/...` passes.

## 2. Call sites

- [ ] 2.1 Replace the `kernel.New(...)` calls in `internal/workflow/render/kernel.go` (delete `NewKernel` there and update its callers inside the package), `internal/cmdutil/publish.go`, `internal/cmd/module/vet.go` (`identitySchemaForVet`), `internal/cmd/module/init.go` (`runInit` and `runRepair`) and `internal/config/platform.go` (`BuildPlatformModule`) with `config.NewKernel(...)`, dropping every `opm/schema` import that only served `WithSchemaLoader`; verify `go build ./...` is green and `grep -rn 'kernel.New(' internal/ --include='*.go' | grep -v _test` lists only `internal/config/kernel.go`.
- [ ] 2.2 Verify `grep -rn 'WithSchemaLoader' internal/ --include='*.go'` is empty and `go test ./internal/workflow/render/... ./internal/cmdutil/... ./internal/cmd/module/... ./internal/config/...` pass unchanged.

## 3. Gates

- [ ] 3.1 Run `task fmt`, `task lint`, `task test`; verify all green.
