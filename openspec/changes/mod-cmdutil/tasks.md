## 1. Scaffold cmdutil package and relocate ExitError

- [x] 1.1 Create `internal/cmdutil/` package directory
- [x] 1.2 Move `ExitError` struct (and its `Error()`, `Unwrap()` methods) from `mod_build.go` to `exit.go`, alongside the existing exit code constants and `exitCodeFromK8sError`
- [x] 1.3 Verify all existing references to `ExitError` still compile (`go build ./...`)

## 2. Implement flag group structs

- [x] 2.1 Create `internal/cmdutil/flags.go` with `RenderFlags` struct (`Values []string`, `Namespace string`, `ReleaseName string`, `Provider string`) and `AddTo(*cobra.Command)` method
- [x] 2.2 Add `K8sFlags` struct (`Kubeconfig string`, `Context string`) with `AddTo(*cobra.Command)` method to the same file
- [x] 2.3 Add `ReleaseSelectorFlags` struct (`ReleaseName string`, `ReleaseID string`, `Namespace string`) with `AddTo(*cobra.Command)`, `Validate() error`, and `LogName() string` methods
- [x] 2.4 Write unit tests for flag registration (`TestRenderFlags_AddTo`, `TestK8sFlags_AddTo`) — verify flag names, types, defaults
- [x] 2.5 Write unit tests for `ReleaseSelectorFlags.Validate()` — both-set, neither-set, one-set cases
- [x] 2.6 Write unit tests for `ReleaseSelectorFlags.LogName()` — name-set and ID-only cases
- [x] 2.7 Write test for flag group composition — `RenderFlags` + `K8sFlags` on the same command without conflicts

## 3. Implement render pipeline helpers

- [x] 3.1 Create `internal/cmdutil/render.go` with `RenderModuleOpts` struct and `RenderModule(ctx, opts) (*build.RenderResult, error)` function — resolves module path from args, validates OPM config, resolves K8s config, builds RenderOptions, validates, creates pipeline, executes Render, handles errors with PrintValidationError
- [x] 3.2 Implement `ShowRenderOutput(result *build.RenderResult, opts ShowOutputOpts) error` — checks HasErrors (calls PrintRenderErrors), shows transformer matches (compact/verbose/verboseJSON), logs warnings via module logger
- [x] 3.3 Write unit test for `RenderModule` with nil OPMConfig — verify ExitError with ExitGeneralError code
- [x] 3.4 Write unit test for module path resolution — empty args yields `"."`, single arg yields that path

## 4. Implement K8s client factory

- [x] 4.1 Create `internal/cmdutil/k8s.go` with `NewK8sClient(opts K8sClientOpts) (*kubernetes.Client, error)` function — accepts kubeconfig, context, apiWarnings; returns client or ExitError with ExitConnectivityError
- [x] 4.2 Write unit test verifying that a failed client creation returns `*ExitError` with `ExitConnectivityError` code

## 5. Move shared output formatters

- [x] 5.1 Move `PrintValidationError` (currently `printValidationError` in `mod_build.go`) to `internal/cmdutil/output.go` — export it
- [x] 5.2 Move `PrintRenderErrors` (currently `printRenderErrors` in `mod_build.go`) to `internal/cmdutil/output.go` — export it
- [x] 5.3 Move `writeTransformerMatches`, `writeVerboseMatchLog` to `internal/cmdutil/output.go` (used by ShowRenderOutput)
- [x] 5.4 Keep `writeBuildVerboseJSON` and `formatApplySummary` in `internal/cmd/` (build-specific and apply-specific)
- [x] 5.5 Verify all existing callers compile after moves (`go build ./...`)

## 6. Migrate mod vet (simplest render command)

- [x] 6.1 Refactor `NewModVetCmd` to use `cmdutil.RenderFlags` (local var + `AddTo`) instead of package-level `vetValuesFlags`, `vetNamespaceFlag`, `vetReleaseNameFlag`, `vetProviderFlag`
- [x] 6.2 Refactor `runVet` to call `cmdutil.RenderModule()` then `cmdutil.ShowRenderOutput()` instead of inline preamble
- [x] 6.3 Remove orphaned package-level vet flag variables
- [x] 6.4 Run `go test ./internal/cmd/ -run TestModVet` — all existing tests must pass

## 7. Migrate mod build

- [x] 7.1 Refactor `NewModBuildCmd` to use `cmdutil.RenderFlags` instead of package-level `buildValuesFlags`, `buildNamespaceFlag`, `buildReleaseNameFlag`, `buildProviderFlag`
- [x] 7.2 Refactor `runBuild` to call `cmdutil.RenderModule()` then `cmdutil.ShowRenderOutput()` — keep build-specific output logic (split, format, verboseJSON) in command
- [x] 7.3 Remove orphaned package-level build flag variables (render-related ones only; keep `buildOutputFlag`, `buildSplitFlag`, `buildOutDirFlag`, `buildVerboseJSONFlag` as local)
- [x] 7.4 Run `go test ./internal/cmd/ -run TestModBuild` — all existing tests must pass

## 8. Migrate mod apply

- [x] 8.1 Refactor `NewModApplyCmd` to use `cmdutil.RenderFlags` + `cmdutil.K8sFlags` instead of package-level flag variables
- [x] 8.2 Refactor `runApply` to call `cmdutil.RenderModule()`, `cmdutil.ShowRenderOutput()`, and `cmdutil.NewK8sClient()` — keep apply-specific logic (create-namespace, apply operation, result reporting)
- [x] 8.3 Convert apply-specific flags (`dryRun`, `wait`, `timeout`, `createNS`) to local variables captured by RunE closure
- [x] 8.4 Remove orphaned package-level apply flag variables
- [x] 8.5 Run `go test ./internal/cmd/ -run TestModApply` — all existing tests must pass

## 9. Migrate mod diff

- [x] 9.1 Refactor `NewModDiffCmd` to use `cmdutil.RenderFlags` + `cmdutil.K8sFlags` instead of package-level flag variables
- [x] 9.2 Refactor `runDiff` to call `cmdutil.RenderModule()` only (NOT `ShowRenderOutput` — diff handles HasErrors via DiffPartial), then `cmdutil.NewK8sClient()` — keep diff-specific comparison and output logic
- [x] 9.3 Remove orphaned package-level diff flag variables
- [x] 9.4 Verify `go build ./...` compiles (no dedicated diff tests exist yet)

## 10. Migrate mod delete

- [x] 10.1 Refactor `NewModDeleteCmd` to use `cmdutil.ReleaseSelectorFlags` + `cmdutil.K8sFlags` instead of package-level flag variables
- [x] 10.2 Refactor `runDelete` to call `rsf.Validate()`, `rsf.LogName()`, and `cmdutil.NewK8sClient()` — keep delete-specific logic (confirmation prompt, delete operation, result reporting)
- [x] 10.3 Convert delete-specific flags (`force`, `dryRun`, `wait`, `ignoreNotFound`) to local variables captured by RunE closure
- [x] 10.4 Remove orphaned package-level delete flag variables
- [x] 10.5 Verify `go build ./...` compiles

## 11. Migrate mod status

- [x] 11.1 Refactor `NewModStatusCmd` to use `cmdutil.ReleaseSelectorFlags` + `cmdutil.K8sFlags` instead of package-level flag variables
- [x] 11.2 Refactor `runStatus` to call `rsf.Validate()`, `rsf.LogName()`, and `cmdutil.NewK8sClient()` — keep status-specific logic (output format, watch mode)
- [x] 11.3 Refactor `runStatusOnce` and `displayStatus` to use `rsf.LogName()` instead of duplicated inline logName resolution
- [x] 11.4 Convert status-specific flags (`output`, `watch`, `ignoreNotFound`) to local variables captured by RunE closure
- [x] 11.5 Remove orphaned package-level status flag variables
- [x] 11.6 Run `go test ./internal/cmd/ -run TestModStatus` — all existing tests must pass

## 12. Cleanup and validation

- [x] 12.1 Search for any remaining orphaned package-level flag variables in `internal/cmd/` and remove them
- [x] 12.2 Verify no unused imports remain (`go vet ./...`)
- [x] 12.3 Run `task check` (fmt + vet + test) — all non-preexisting checks pass
- [x] 12.4 Verify `internal/cmdutil` has no import of `internal/cmd` (`go list -json ./internal/cmdutil | jq '.Imports'`)
- [x] 12.5 Update project structure tree in AGENTS.md to include `internal/cmdutil/`
