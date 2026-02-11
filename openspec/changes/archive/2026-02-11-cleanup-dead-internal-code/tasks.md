## 1. Delete dead packages and files

- [x] 1.1 Delete `internal/testutil/` directory entirely
- [x] 1.2 Delete `internal/cmd/mod_stubs.go`
- [x] 1.3 Delete `internal/identity/` directory; inline UUID as unexported constant in `internal/build/release_builder.go` and update `build/release_builder_identity_test.go`

## 2. Clean up `internal/errors`

- [x] 2.1 Remove `NewConnectivityError`, `NewNotFoundError`, `NewPermissionError` functions and their test cases from `errors.go` and `errors_test.go`

## 3. Clean up `internal/config`

- [x] 3.1 Remove dead functions: `LogResolvedValues` from `resolver.go`, `PathsFromEnv`, `ExpandPath`, `EnsureDir` from `paths.go`, `CheckRegistryConnectivity` from `loader.go`; remove corresponding tests

## 4. Clean up `internal/cmd`

- [x] 4.1 Remove dead functions `GetOutputFormat`, `IsVerbose` from `root.go`
- [x] 4.2 Remove dead functions `Exit`, `ExitCodeFromError` and associated exit code constants from `exit.go`; remove `exit_test.go` if all its tests are for deleted code

## 5. Clean up `internal/output`

- [x] 5.1 Remove dead functions: `Fatal` from `log.go`, `WriteUnstructuredManifests` and `WriteResource` from `manifest.go`, `WriteSplitUnstructured` from `split.go`
- [x] 5.2 Remove `RenderStatusTable`, `ResourceStatus` from `table.go`; remove `Table.SetStyle`
- [x] 5.3 Unexport internal-only symbols: `StyleAction`, `StyleSummary`, `StatusFailed`, `StyleNoun`, `StyleDim`, color variables (`ColorCyan`, `ColorGreen`, `ColorYellow`, `ColorRed`, `ColorBoldRed`, `ColorGreenCheck`, `ColorDimGray`), `StatusStyle`, `DefaultTableStyle`, `TableStyle`, `Logger` variable
- [x] 5.4 Unexport internal-only verbose types: `VerboseResult`, `VerboseModule`, `VerboseMatchPlan`, `VerboseMatch`, `VerboseMatchDetail`, `VerboseResource`, `MatchDetailInfo`; remove the dead nil-only code path for match details
- [x] 5.5 Update `output` test files to use unexported names; remove tests for deleted functions

## 6. Clean up `internal/kubernetes`

- [x] 6.1 Unexport internal-only symbols in `discovery.go`: `LabelManagedByValue`, `LabelModuleNamespace`, `LabelModuleVersion`, `LabelReleaseID`, `LabelModuleID`, `FieldManagerName`, `ErrNoResourcesFound`, `NoResourcesFoundError`, `DiscoveryOptions`, `BuildModuleSelector`, `BuildReleaseIDSelector`, `SortByWeightDescending`
- [x] 6.2 Unexport internal-only symbols in `health.go`: `HealthStatus`, `HealthReady`, `HealthNotReady`, `HealthComplete`, `HealthUnknown`, `EvaluateHealth`
- [x] 6.3 Unexport/remove internal-only symbols in `diff.go`: remove unused `DiffOptions` type; unexport `ResourceState`, `ResourceUnchanged`, `ResourceDiff`, `DiffResult.IsEmpty`, `Comparer`, `FetchLiveState`
- [x] 6.4 Unexport internal-only symbols in `status.go`: `ResourceHealth`, `StatusResult`, `FormatStatusJSON`, `FormatStatusYAML`
- [x] 6.5 Unexport in `apply.go`: `ApplyResult`, `ResourceError`; in `delete.go`: `DeleteResult`
- [x] 6.6 Update `kubernetes` test files to use unexported names

## 7. Fix stale integration test

- [x] 7.1 Update `tests/integration/deploy/main.go` to use `DiscoveryOptions` struct instead of positional args for `DiscoverResources` calls (lines 121, 200)

## 8. Validation

- [x] 8.1 Run `go build ./...` — must pass
- [x] 8.2 Run `task test` — all tests must pass
- [x] 8.3 Run `task check` — fmt + vet + test must all pass
