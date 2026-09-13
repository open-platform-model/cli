## 1. Shared resolver (internal/workflow/render)

- [x] 1.1 Change `DebugValuesSource` to take the module package value (`pkg cue.Value`) instead of `*module.Module`, and export `ResolveModuleValues(k *kernel.Kernel, pkg cue.Value, moduleDir string, valuesFiles []string) ([]kernel.Source, error)` in place of `resolveModuleValues`, returning sources without validating them; update `FromModule` to pass `mod.Package` and to validate the `-f` sources against `mod.ConfigSchema()` before synthesis as it does today; verify `go test ./internal/workflow/render/...` passes and `opm module build <fixture> -f <bad-values>` still fails before "Building synthetic instance" is printed.

## 2. vet (internal/cmd/module/vet.go)

- [x] 2.1 Replace `resolveVetValues`, the per-value `Validate(cue.Concrete(true))` loop and the `validate.Config` call with `render.ResolveModuleValues(k, modVal, modulePath, rf.Values)` followed by `k.ValidateConfigDetailed(modVal.LookupPath(schema.Config), sources)`; on failure wrap as `module %q: values do not satisfy #config: %w`, print with `cmdutil.PrintValidationError("values do not satisfy #config", err)` and exit 2 with `Printed: true`; keep the values-detail string (`-f` basenames joined by ", ", else `debugValues`) for the "Values satisfy #config" line; drop the `pkg/loader` and `pkg/validate` imports; verify `go test ./internal/cmd/module/...` passes and the vet e2e cases (`tests/e2e`, the `validation-output` spec's `opm mod vet` scenarios) still print one grouped block with exit 2.
- [x] 2.2 Verify against a module whose `debugValues` is `_`: exit 2, one grouped block naming the incomplete `#config` field at its schema position (an unconstrained `debugValues` carries no position of its own, so the kernel reports the field where the schema declares it, exactly as `build` does), no "not concrete" line; and against a two-file stack where the base leaves a field open that the override fills: exit 0.

## 3. Delete the copies

- [x] 3.1 Delete `pkg/validate/` (package, tests, `testdata/`); delete `LoadValuesFile` and its test cases from `pkg/loader/instance_file.go` and `instance_file_test.go`; delete the `ConfigError` type, its methods and the `errors.As(*ConfigError)` branch in `internal/cmdutil/output.go`, keeping `GroupedError`, `GroupedErrorsFromError`, `groupCUEErrors` and `normalizeCUEPath`; update `pkg/errors/errors_test.go` and `internal/cmdutil` tests accordingly; verify `go build ./...` is green and `grep -rn 'pkg/validate"\|ConfigError\|LoadValuesFile' --include='*.go' .` is empty.

## 4. Gates

- [x] 4.1 Run `task fmt`, `task lint`, `task test`; verify all green (e2e: the two operator-owned tests fail on the local kind cluster, whose released alpha.14 operator cannot reconcile a core v2 platform; a documented operator release gap that reproduces on main, not a CLI regression).

## 5. Verify follow-ups

- [x] 5.1 Add the spec deltas the verify pass found missing: retire `validation-gates` (REMOVED), rewrite `module-synthetic-instance` values selection over the shared resolver, list the surviving types in the `errors-domain` package-location requirement, and in `mod-vet` drop the `debugValues: _` parenthetical from "No debugValues and no -f flag" and add the no-`#config` scenario; verify `openspec validate` is green.
- [x] 5.2 vet refuses values files against a module that declares no `#config`, the verdict build reaches (`validateVetValues` in `internal/cmd/module/vet.go`); verify with the e2e case `TestE2E_ModuleVet_ValuesFilesWithoutConfig` over the `no-config` testdata module.
- [x] 5.3 Drop the unused `TransformError` and `FieldError` types from `pkg/errors`, delete the duplicate `TestModVet_MultipleValuesAreMergedForValidation`, describe `pkg/loader` without the deleted values loader in CLAUDE.md, and trim the `ConfigError.FieldErrors()` note from the build spec; verify `task lint` and the module, errors, cmdutil and render unit suites are green.
