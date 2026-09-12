## 1. Result carries the library types (internal/workflow/render)

- [ ] 1.1 Change `Result.Instance` to `module.InstanceMetadata` and `Result.Module` to `module.ModuleMetadata` in `types.go`; in `render.go` assign `result.Instance = *inst.Metadata` when non-nil (namespace override unchanged after it) and change `decodeModuleMetadata` to decode into `module.ModuleMetadata`; verify `go test ./internal/workflow/render/...` passes and `opm module build <fixture>` prints the same instance and version lines as before.
- [ ] 1.2 Add `moduleref.go` with `CanonicalModuleRef(m module.ModuleMetadata) (path, version string)` and `ensureVPrefix`, carrying the existing doc comments; move the tests from `pkg/module/module_test.go` beside it; update `internal/workflow/apply/apply.go` and `thineditor.go` to call `render.CanonicalModuleRef(result.Module)`; verify `go test ./internal/workflow/...` passes and the apply workflow's CR test still writes `spec.module` with a `v`-prefixed version.

## 2. Delete the copy (pkg/module)

- [ ] 2.1 Delete `pkg/module/`; verify `go build ./...` is green and `grep -rn 'cli/pkg/module"' --include='*.go' .` is empty.

## 3. Gates

- [ ] 3.1 Run `task fmt`, `task lint`, `task test`; verify all green.
