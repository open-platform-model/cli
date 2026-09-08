# Tasks: migrate-kernel-api-and-verdicts

Every task before 6.1 runs against a local `replace github.com/open-platform-model/library => ../library` (added in 1.1, removed in 6.1). `task build` means `go build ./...` including `tests/integration`.

## 1. Acquire calls drop their load options

- [ ] 1.1 Add `replace github.com/open-platform-model/library => ../library` to `go.mod`, then `go build ./...`; verify the only failures reported are the `opm/helper/loader/file` and `opm/helper/synth` import lines in the eleven files that name them (record the list — it is the work of tasks 1.2 to 2.3) plus the `opm/compat` import lines in `internal/publish/compat.go` and `internal/scaffold/scaffold.go`, which task 5 clears, plus the three `CueContext()` calls in `internal/cmd/module/vet.go` and `internal/cmdutil/publish.go`, which task 2.4 clears.
- [ ] 1.2 `internal/workflow/render/kernel.go`, `render.go`: drop the `loaderfile.LoadOptions{Registry: ...}` argument from `AcquirePlatformFromDir` and `AcquireInstanceFromDir`, and pass the values sources as the trailing variadic arguments instead of `kernel.WithValues(sources...)`; drop the `loaderfile` import from both; verify `go build ./internal/workflow/render/` is green and `go test ./internal/workflow/render/...` passes.
- [ ] 1.3 `internal/workflow/render/module.go`: replace `LoadModulePackage` + `NewModuleFromValue` with one `AcquireModuleFromDir(ctx, opts.ModulePath)`, and `synth.InstanceInput` with `kernel.InstanceInput`; drop both helper imports; verify `opm module build` against a fixture module directory renders the same resource set as before the edit.
- [ ] 1.4 `internal/config/platform.go`: replace `LoadPlatformPackage` + `NewPlatformFromValue` with `AcquirePlatformFromDir(ctx, dir)` and the two `loaderfile.Err*` comparisons with `liberrors.Err*`; verify `go test ./internal/config/...` passes and a wrong-kind platform directory still produces the CLI's existing refusal message.

## 2. The publish gate and the scaffold acquire through a kernel

- [ ] 2.1 `internal/publish/kernel_gate.go`: add a `*kernel.Kernel` field to the gate's options struct, drop its now-redundant `Registry` field, call `Kernel.AcquireModuleFromDir` in place of the free-function `loaderfile.LoadModulePackage`, and rename the four sentinel comparisons to `liberrors.Err*`; verify `go test ./internal/publish/...` passes.
- [ ] 2.2 Wire the publish command's existing kernel into the gate options at every construction site; verify `grep -rn 'kernel.New(' internal/ | wc -l` is unchanged from before this change (no second kernel per invocation, per the `kernel-render` spec) and `opm module publish --dry-run` against a fixture module reports the same gate outcome.
- [ ] 2.3 `internal/scaffold/scaffold.go`, `repair.go`: replace `LoadModulePackage` with `AcquireModuleFromDir` (both discard the value; they route on the sentinels), rename the sentinel comparisons, drop the `loaderfile` imports; verify `go test ./internal/scaffold/...` passes and `go build ./...` reports no remaining `opm/helper` import error.
- [ ] 2.4 `internal/cmd/module/vet.go` (`identitySchemaForVet`) and `internal/cmdutil/publish.go` (`RunPublish`): fetch the schema with `k.SchemaCache().Get()` and take the context from the value (`schemaVal.Context()`), passing it as `publish.Options.Context`; `internal/workflow/render/module_test.go`, `internal/publish/realtree_test.go`: build test values with `cuecontext.New()`, or the schema value's context where the test unifies with it; verify `go test ./internal/cmd/module/... ./internal/cmdutil/... ./internal/publish/... ./internal/workflow/render/...` pass, `opm module publish --dry-run` against a fixture module reports the same gate outcome, and `grep -rn 'CueContext()' --include=*.go .` is empty.

## 3. Verdict rows replace the reshaped types

- [ ] 3.1 `internal/cmdutil/output.go`: change `FormatUnresolvedDemands` to take `[]liberrors.UnresolvedDemand` and update the `errors.As` branch in `PrintValidationError` to pass `demandsErr.Demands`; verify `go test ./internal/cmdutil/...` passes and the rendered block is byte-identical to the pre-change output for a fixture with one alternatives-bearing and one bare demand.
- [ ] 3.2 `internal/workflow/render/validation.go`: call `cmdutil.FormatUnresolvedDemands(d.Unresolved)` with no aggregate reconstruction; iterate `d.Unmatched` as rows printing `.Component` (default) and, in verbose mode, each `.Candidates` entry with its `.Transformer` and `.MissingLabels`; format `d.OverSubscribed` rows unchanged; verify a refusal on a label-predicate mismatch prints the missing label in verbose mode and the same single line as before in default mode.
- [ ] 3.3 Update every `errors.As` target and field read the library reshaped: `*liberrors.OverSubscribedContractsError` for the over-subscription cause, `.Component` / `.Transformer` on `liberrors.TransformError`; verify `grep -rn 'OverSubscribedContractError\|ComponentName\|TransformerFQN\|liberrors.UnifyError\|liberrors.MatchResult' --include=*.go .` is empty and `go build ./...` is green.

## 4. The CLI words its own warnings

- [ ] 4.1 `internal/workflow/render/render.go`: add a private formatter producing `Result.Warnings` from `out.Diagnostics.UnhandledTraits` and the `Newer` rows of `out.Diagnostics.ResolvedVersions`, replacing `out.Warnings`; keep the skew sentence's current wording (path, module version, platform version) and the unhandled-trait sentence's; verify `internal/cmd/module/verbose_output_test.go` still compiles unchanged and a warn-policy skew render prints the same warning text as before this change.
- [ ] 4.2 `internal/workflow/render/types.go`: reword the `Warnings` field doc to say the CLI composes it from the diagnostics rows; verify `grep -rn 'out.Warnings\|RenderResult.Warnings' --include=*.go .` is empty.
- [ ] 4.3 Add a test in `internal/workflow/render` covering the formatter against a diagnostics value with one unhandled optional trait and one newer resolved-versions row; verify it asserts both lines and fails when either row is dropped.

## 5. Catalog compatibility moves in from the library

- [x] 5.1 Copy `compat.go`, `level.go`, `compat_test.go` and `level_test.go` from the library's deleted `opm/compat` into `internal/compat/`, package name unchanged; edit the package doc so its consumer list names the publish gate and `opm catalog registry check --compat` only (library-matching no longer exists) and no longer advertises `HighestStable`; verify `go build ./internal/compat/` and `go test ./internal/compat/...` pass, including `TestAPIVersionPatternCoreParity`.
- [x] 5.2 Declare `highestStable` in `internal/scaffold/predecessor.go` with `predecessor.go`'s doc comment (re-aiming its pointer at the gate-side rule from `the CLI` to `internal/publish`), add its four `predecessor_test.go` cases as `internal/scaffold/predecessor_test.go`, and change `scaffold.go:109` to call it, dropping the `compat` import and re-aiming the `ResolveTemplateVersion` doc comment's `compat.HighestStable` reference; verify `go test ./internal/scaffold/...` passes and `grep -n 'compat' internal/scaffold/*.go` is empty.
- [x] 5.3 Repoint `internal/publish/compat.go`'s import to `github.com/open-platform-model/cli/internal/compat`, then `go mod tidy`; verify `go build ./...`, `go vet ./...` and `go test ./internal/publish/...` pass, `github.com/Masterminds/semver/v3` sits in the direct require block of `go.mod`, and `grep -rn 'library/opm/compat' --include='*.go' .` is empty.

## 6. Integration programs and the gate

- [ ] 6.1 `tests/integration/platform-build/main.go`, `tests/integration/render-parity/main.go`: apply the same acquire-surface edits (drop load options, `synth.InstanceInput` -> `kernel.InstanceInput`); verify both build and `go run ./tests/integration/render-parity` reports identical CLI and operator-sequence render digests.
- [ ] 6.2 `task fmt`, `task lint`, `task test` green; verify `go vet ./...` is clean and no file under `internal/` or `tests/` imports `github.com/open-platform-model/library/opm/helper/loader/file`, `.../opm/helper/synth`, `.../opm/core` or `.../opm/compat`, and none calls `CueContext()`.

## 7. Pin the released library

- [ ] 7.1 Remove the `replace` directive and bump `github.com/open-platform-model/library` in `go.mod` to the published alpha carrying `one-api-tier`, `cue-owned-verdicts`, `move-compat-to-cli` and `kernel-owns-no-build-context`, then `task tidy`; verify `task check` is green with no `replace` directive present and `openspec validate migrate-kernel-api-and-verdicts` passes.
