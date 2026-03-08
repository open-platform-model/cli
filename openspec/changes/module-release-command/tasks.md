## 1. Release File Loader (`pkg/loader/`)

- [ ] 1.1 Create `pkg/loader/release_file.go` with `LoadReleaseFile(ctx *cue.Context, filePath string, registry string) (cue.Value, string, error)` — loads a standalone `.cue` file via `load.Instances()` with the file's parent directory for `cue.mod` resolution; sets `CUE_REGISTRY` env var if registry is non-empty; returns evaluated CUE value + resolve directory
- [ ] 1.2 Add `LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, error)` to `pkg/loader/release_file.go` — loads a module CUE package from a local directory for `--module` flag injection; replaces the now-deleted `internal/loader.LoadModule()`
- [ ] 1.3 Add unit tests for `LoadReleaseFile()` — valid `#ModuleRelease` file, valid `#BundleRelease` file, invalid CUE, unrecognised `kind`, registry import (fixture only, no live registry)
- [ ] 1.4 Add unit tests for `LoadModulePackage()` — valid module directory, missing directory, missing CUE files
- [ ] 1.5 Verify: `go test ./pkg/loader/...` passes (all existing + new tests)

> Note: `DetectReleaseKind()` already exists in `pkg/loader/module_release.go` and works as-is. `LoadModuleReleaseFromValue()` accepts any `cue.Value` regardless of how it was loaded. No changes needed to either.

## 2. debugValues Support (`internal/cmdutil/`)

- [ ] 2.1 Add `DebugValues bool` field to `RenderReleaseOpts` in `internal/cmdutil/render.go`
- [ ] 2.2 Implement debugValues extraction in `RenderRelease()` — when `DebugValues: true`, call `loader.LoadModulePackage()` to get the module CUE value, extract `debugValues` field, validate it is not `_` (unconstrained); error clearly if absent or open
- [ ] 2.3 Wire the extracted `debugValues` CUE value as the values source — add `LoadReleasePackageWithValue(ctx, releaseFile string, valuesVal cue.Value) (cue.Value, string, error)` to `pkg/loader/release_file.go` that accepts a pre-loaded CUE value instead of a values file path; avoids temp file creation
- [ ] 2.4 Update `opm mod vet` command in `internal/cmd/mod/vet.go` — set `DebugValues: len(rf.Values) == 0` on `RenderReleaseOpts`; update the vet summary output to say "debugValues" when no `-f` flag was provided
- [ ] 2.5 Add unit tests — debugValues used when option set, error when debugValues is `_`, `-f` flag overrides debugValues (DebugValues: false)
- [ ] 2.6 Verify: `go test ./internal/cmd/mod/...` passes

## 3. cmdutil: Release Orchestration (`internal/cmdutil/`)

- [ ] 3.1 Add `ReleaseFileFlags` struct to `internal/cmdutil/flags.go` — `Module string` (`--module`), `Provider string` (`--provider`), with `AddTo(*cobra.Command)` method
- [ ] 3.2 Add `RenderFromReleaseFileOpts` struct and `RenderFromReleaseFile(ctx, opts) (*RenderResult, error)` to `internal/cmdutil/render.go` — loads release file via `loader.LoadReleaseFile()`, detects kind (errors on BundleRelease), optionally calls `loader.LoadModulePackage()` + `FillPath` for `--module`, then calls `loader.LoadModuleReleaseFromValue()`, `loader.LoadProvider()`, `engine.ModuleRenderer.Render()`, converts resources to `Unstructured`
- [ ] 3.3 Add `ResolveReleaseIdentifier(arg string) (name string, uuid string)` to `internal/cmdutil/` — UUID pattern detection (`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`) vs release name
- [ ] 3.4 Add unit tests for `RenderFromReleaseFile()` — valid release file, BundleRelease rejection, `--module` injection, missing `#module` error
- [ ] 3.5 Add unit tests for `ResolveReleaseIdentifier()` — UUID format, name format, edge cases
- [ ] 3.6 Verify: `go test ./internal/cmdutil/...` passes

## 4. Release Command Group: Scaffold

- [ ] 4.1 Create `internal/cmd/release/release.go` with `NewReleaseCmd(*config.GlobalConfig)` — group container with `Use: "release"`, `Aliases: []string{"rel"}`, `Short: "Release operations"`
- [ ] 4.2 Register `opm release` in `internal/cmd/root.go` via `rootCmd.AddCommand(cmdrelease.NewReleaseCmd(&cfg))`

## 5. Release Render Commands (`internal/cmd/release/`)

- [ ] 5.1 Implement `opm release vet <release.cue>` — requires positional arg, `ReleaseFileFlags`, calls `RenderFromReleaseFile()`, outputs per-resource validation lines, summary "Release valid (N resources)"
- [ ] 5.2 Implement `opm release build <release.cue>` — requires positional arg, `ReleaseFileFlags`, output flags (`-o yaml/json`, `--split`, `--out-dir`), calls `RenderFromReleaseFile()`, writes manifests
- [ ] 5.3 Implement `opm release apply <release.cue>` — requires positional arg, `ReleaseFileFlags`, `K8sFlags`, `--dry-run`, calls `RenderFromReleaseFile()`, SSA apply with inventory
- [ ] 5.4 Implement `opm release diff <release.cue>` — requires positional arg, `ReleaseFileFlags`, `K8sFlags`, calls `RenderFromReleaseFile()`, compares against live cluster state
- [ ] 5.5 Wire `--module` flag gating: if `kind == BundleRelease` and `--module` provided, error early rather than silently ignoring

## 6. Release Cluster-Query Commands (`internal/cmd/release/`)

- [ ] 6.1 Implement `opm release status <name|uuid>` — positional arg via `ResolveReleaseIdentifier()`, display release health; delegate to same logic as existing `opm mod status`
- [ ] 6.2 Implement `opm release tree <name|uuid>` — positional arg, display resource hierarchy
- [ ] 6.3 Implement `opm release events <name|uuid>` — positional arg, display K8s events
- [ ] 6.4 Implement `opm release delete <name|uuid>` — positional arg, `--dry-run`, delete release from cluster
- [ ] 6.5 Implement `opm release list` — list all releases in namespace, no positional arg

## 7. Module Command Migration

- [ ] 7.1 Update `opm mod vet` in `internal/cmd/mod/vet.go` — pass `DebugValues: len(rf.Values) == 0` to `RenderReleaseOpts`; update values detail string in output ("debugValues" when no `-f` flag)
- [ ] 7.2 Add Cobra `Deprecated` field to `opm mod status` — `"use 'opm release status <name>' instead"` — delegate to shared run function
- [ ] 7.3 Add Cobra `Deprecated` field to `opm mod tree` — `"use 'opm release tree <name>' instead"`
- [ ] 7.4 Add Cobra `Deprecated` field to `opm mod events` — `"use 'opm release events <name>' instead"`
- [ ] 7.5 Add Cobra `Deprecated` field to `opm mod delete` — `"use 'opm release delete <name>' instead"`
- [ ] 7.6 Add Cobra `Deprecated` field to `opm mod list` — `"use 'opm release list' instead"`

> Note: `opm mod build` and `opm mod apply` require no changes — they already call `cmdutil.RenderRelease()` which is the unified pipeline.

## 8. Testing

- [ ] 8.1 Add unit tests for `release` render commands (vet, build with release file, build with --module)
- [ ] 8.2 Add unit tests for `release` cluster-query commands (positional arg parsing, name vs UUID detection)
- [ ] 8.3 Add unit tests for `opm mod vet` with debugValues (no `-f`, uses debugValues; with `-f`, ignores debugValues)
- [ ] 8.4 Create test fixture: `testdata/jellyfin_release.cue` with `kind: "ModuleRelease"` and concrete metadata/values
- [ ] 8.5 Create test fixture: `testdata/bundle_release.cue` with `kind: "BundleRelease"` for rejection testing
- [ ] 8.6 Add unit test: `BundleRelease` file passed to render command returns clear unsupported error
- [ ] 8.7 Add unit test: `LoadReleaseFile()` correctly loads and evaluates a `ModuleRelease` file

## 9. Validation Gates

- [ ] 9.1 Run `task fmt` — all files formatted
- [ ] 9.2 Run `task lint` — golangci-lint passes
- [ ] 9.3 Run `task test` — all unit tests pass
- [ ] 9.4 Run `task test:e2e` — end-to-end tests pass (if applicable)
