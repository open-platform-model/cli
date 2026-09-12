## 1. Shared resolver (internal/workflow/render)

- [ ] 1.1 Change `DebugValuesSource` to take the module package value (`pkg cue.Value`) instead of `*module.Module`, and export `ResolveModuleValues(k *kernel.Kernel, pkg cue.Value, moduleDir string, valuesFiles []string) ([]kernel.Source, error)` in place of `resolveModuleValues`, returning sources without validating them; update `FromModule` to pass `mod.Package` and to validate the `-f` sources against `mod.ConfigSchema()` before synthesis as it does today; verify `go test ./internal/workflow/render/...` passes and `opm module build <fixture> -f <bad-values>` still fails before "Building synthetic instance" is printed.
- [ ] 1.2 `task fmt`, `task lint`, `task test` green, then commit `refactor(render): export ResolveModuleValues over the module package value`

## 2. vet (internal/cmd/module/vet.go)

- [ ] 2.1 Replace `resolveVetValues`, the per-value `Validate(cue.Concrete(true))` loop and the `validate.Config` call with `render.ResolveModuleValues(k, modVal, modulePath, rf.Values)` followed by `k.ValidateConfigDetailed(modVal.LookupPath(schema.Config), sources)`; on failure wrap as `module %q: values do not satisfy #config: %w`, print with `cmdutil.PrintValidationError("values do not satisfy #config", err)` and exit 2 with `Printed: true`; keep the values-detail string (`-f` basenames joined by ", ", else `debugValues`) for the "Values satisfy #config" line; drop the `pkg/loader` and `pkg/validate` imports; verify `go test ./internal/cmd/module/...` passes and the vet e2e cases (`tests/e2e`, the `validation-output` spec's `opm mod vet` scenarios) still print one grouped block with exit 2.
- [ ] 2.2 Verify against a module whose `debugValues` is `_`: exit 2, one grouped block naming `<module>/debugValues`, no "not concrete" line; and against a two-file stack where the base leaves a field open that the override fills: exit 0.
- [ ] 2.3 `task fmt`, `task lint`, `task test` green and the vet e2e cases green, then commit `fix(vet): validate values through the kernel`

## 3. Delete the copies

- [ ] 3.1 Delete `pkg/validate/` (package, tests, `testdata/`); delete `LoadValuesFile` and its test cases from `pkg/loader/instance_file.go` and `instance_file_test.go`; delete the `ConfigError` type, its methods and the `errors.As(*ConfigError)` branch in `internal/cmdutil/output.go`, keeping `GroupedError`, `GroupedErrorsFromError`, `groupCUEErrors` and `normalizeCUEPath`; update `pkg/errors/errors_test.go` and `internal/cmdutil` tests accordingly; verify `go build ./...` is green and `grep -rn 'pkg/validate"\|ConfigError\|LoadValuesFile' --include='*.go' .` is empty.
- [ ] 3.2 `task fmt`, `task lint`, `task test` green, then commit `refactor(validate): delete the pkg/validate copy and ConfigError`
