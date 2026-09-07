# Tasks: migrate-kernel-api-and-verdicts

Every task before 6.1 runs against a local `replace github.com/open-platform-model/library => ../library` (added in 1.1, removed in 6.1). `task build` means `go build ./...` including `tests/integration`.

## 1. Acquire calls drop their load options

- [ ] 1.1 Add `replace github.com/open-platform-model/library => ../library` to `go.mod`, then `go build ./...`; verify the only failures reported are the `opm/helper/loader/file` and `opm/helper/synth` import lines in the eleven files that name them (record the list — it is the work of tasks 1.2 to 2.3).
- [ ] 1.2 `internal/workflow/render/kernel.go`, `render.go`: drop the `loaderfile.LoadOptions{Registry: ...}` argument from `AcquirePlatformFromDir` and `AcquireInstanceFromDir`, and pass the values sources as the trailing variadic arguments instead of `kernel.WithValues(sources...)`; drop the `loaderfile` import from both; verify `go build ./internal/workflow/render/` is green and `go test ./internal/workflow/render/...` passes.
- [ ] 1.3 `internal/workflow/render/module.go`: replace `LoadModulePackage` + `NewModuleFromValue` with one `AcquireModuleFromDir(ctx, opts.ModulePath)`, and `synth.InstanceInput` with `kernel.InstanceInput`; drop both helper imports; verify `opm module build` against a fixture module directory renders the same resource set as before the edit.
- [ ] 1.4 `internal/config/platform.go`: replace `LoadPlatformPackage` + `NewPlatformFromValue` with `AcquirePlatformFromDir(ctx, dir)` and the two `loaderfile.Err*` comparisons with `liberrors.Err*`; verify `go test ./internal/config/...` passes and a wrong-kind platform directory still produces the CLI's existing refusal message.

## 2. The publish gate and the scaffold acquire through a kernel

- [ ] 2.1 `internal/publish/kernel_gate.go`: add a `*kernel.Kernel` field to the gate's options struct, drop its now-redundant `Registry` field, call `Kernel.AcquireModuleFromDir` in place of the free-function `loaderfile.LoadModulePackage`, and rename the four sentinel comparisons to `liberrors.Err*`; verify `go test ./internal/publish/...` passes.
- [ ] 2.2 Wire the publish command's existing kernel into the gate options at every construction site; verify `grep -rn 'kernel.New(' internal/ | wc -l` is unchanged from before this change (no second kernel per invocation, per the `kernel-render` spec) and `opm module publish --dry-run` against a fixture module reports the same gate outcome.
- [ ] 2.3 `internal/scaffold/scaffold.go`, `repair.go`: replace `LoadModulePackage` with `AcquireModuleFromDir` (both discard the value; they route on the sentinels), rename the sentinel comparisons, drop the `loaderfile` imports; verify `go test ./internal/scaffold/...` passes and `go build ./...` reports no remaining `opm/helper` import error.

## 3. Verdict rows replace the reshaped types

- [ ] 3.1 `internal/cmdutil/output.go`: change `FormatUnresolvedDemands` to take `[]liberrors.UnresolvedDemand` and update the `errors.As` branch in `PrintValidationError` to pass `demandsErr.Demands`; verify `go test ./internal/cmdutil/...` passes and the rendered block is byte-identical to the pre-change output for a fixture with one alternatives-bearing and one bare demand.
- [ ] 3.2 `internal/workflow/render/validation.go`: call `cmdutil.FormatUnresolvedDemands(d.Unresolved)` with no aggregate reconstruction; iterate `d.Unmatched` as rows printing `.Component` (default) and, in verbose mode, each `.Candidates` entry with its `.Transformer` and `.MissingLabels`; format `d.OverSubscribed` rows unchanged; verify a refusal on a label-predicate mismatch prints the missing label in verbose mode and the same single line as before in default mode.
- [ ] 3.3 Update every `errors.As` target and field read the library reshaped: `*liberrors.OverSubscribedContractsError` for the over-subscription cause, `.Component` / `.Transformer` on `liberrors.TransformError`; verify `grep -rn 'OverSubscribedContractError\|ComponentName\|TransformerFQN\|liberrors.UnifyError\|liberrors.MatchResult' --include=*.go .` is empty and `go build ./...` is green.

## 4. The CLI words its own warnings

- [ ] 4.1 `internal/workflow/render/render.go`: add a private formatter producing `Result.Warnings` from `out.Diagnostics.UnhandledTraits` and the `Newer` rows of `out.Diagnostics.ResolvedVersions`, replacing `out.Warnings`; keep the skew sentence's current wording (path, module version, platform version) and the unhandled-trait sentence's; verify `internal/cmd/module/verbose_output_test.go` still compiles unchanged and a warn-policy skew render prints the same warning text as before this change.
- [ ] 4.2 `internal/workflow/render/types.go`: reword the `Warnings` field doc to say the CLI composes it from the diagnostics rows; verify `grep -rn 'out.Warnings\|RenderResult.Warnings' --include=*.go .` is empty.
- [ ] 4.3 Add a test in `internal/workflow/render` covering the formatter against a diagnostics value with one unhandled optional trait and one newer resolved-versions row; verify it asserts both lines and fails when either row is dropped.

## 5. Integration programs and the gate

- [ ] 5.1 `tests/integration/platform-build/main.go`, `tests/integration/render-parity/main.go`: apply the same acquire-surface edits (drop load options, `synth.InstanceInput` -> `kernel.InstanceInput`); verify both build and `go run ./tests/integration/render-parity` reports identical CLI and operator-sequence render digests.
- [ ] 5.2 `task fmt`, `task lint`, `task test` green; verify `go vet ./...` is clean and no file under `internal/` or `tests/` imports `github.com/open-platform-model/library/opm/helper/loader/file`, `.../opm/helper/synth` or `.../opm/core`.

## 6. Pin the released library

- [ ] 6.1 Remove the `replace` directive and bump `github.com/open-platform-model/library` in `go.mod` to the published alpha carrying both `one-api-tier` and `cue-owned-verdicts`, then `task tidy`; verify `task check` is green with no `replace` directive present and `openspec validate migrate-kernel-api-and-verdicts` passes.
