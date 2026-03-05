## 1. Release File Loader

- [ ] 1.1 Define `ReleaseType` enum (`ModuleRelease`, `BundleRelease`) in `internal/loader/release.go`
- [ ] 1.2 Create `internal/loader/release.go` with `LoadRelease(ctx *cue.Context, filePath string, registry string) (cue.Value, ReleaseType, string, error)` — loads a `.cue` file via `load.Instances()`, evaluates it, detects release type by reading the `kind` field
- [ ] 1.3 Add unit tests for `LoadRelease()` — valid `#ModuleRelease` file, valid `#BundleRelease` file (returns `BundleRelease` type), invalid CUE, unrecognised `kind`, release with registry import
- [ ] 1.4 Add `FillModule(releaseVal cue.Value, moduleRaw cue.Value) cue.Value` helper for `--module` flag injection via FillPath

## 2. Builder: debugValues Support

- [ ] 2.1 Add `DebugValues bool` option to builder's `BuildOptions` struct
- [ ] 2.2 Implement debugValues extraction from module's CUE value (`mod.Raw`) — extract `debugValues` field, validate it's not open/empty
- [ ] 2.3 Wire debugValues as values source in the builder when `DebugValues: true` — skip `values.cue` fallback and `--values` files
- [ ] 2.4 Add unit tests — debugValues used when option set, error when debugValues is `_`, `-f` flag overrides debugValues

## 3. Builder: Pre-filled Release Support

- [ ] 3.1 Add alternate `BuildFromRelease(ctx, releaseVal cue.Value, opts) (*modulerelease.ModuleRelease, error)` code path — accepts pre-evaluated CUE value, validates concreteness, extracts metadata/components/autoSecrets
- [ ] 3.2 Handle `--module` override for pre-filled releases — FillPath `#module` before validation
- [ ] 3.3 Add unit tests — build from pre-filled release value, build with --module override, concreteness errors

## 4. cmdutil: Release Shared Utilities

- [ ] 4.1 Add `ReleaseFileFlags` struct to `cmdutil/flags.go` with `--module` flag and `AddTo(*cobra.Command)` method
- [ ] 4.2 Add `RenderFromReleaseFile(ctx, opts) (*pipeline.RenderResult, error)` to `cmdutil/release.go` — loads release file, checks `ReleaseType` (errors on `BundleRelease`), optionally loads module via --module, calls builder, runs match+generate
- [ ] 4.3 Add `ResolveReleaseIdentifier(arg string) (name string, uuid string)` helper — detects UUID pattern vs release name from positional arg
- [ ] 4.4 Update `ReleaseSelectorFlags` or create `ReleaseIdentifierArg` for positional arg parsing in cluster-query commands

## 5. Release Command Group: Scaffold

- [ ] 5.1 Create `internal/cmd/release/release.go` with `NewReleaseCmd(*config.GlobalConfig)` — group container with `Use: "release"`, `Aliases: []string{"rel"}`, `Short: "Release operations"`
- [ ] 5.2 Register `opm release` in `internal/cmd/root.go` via `rootCmd.AddCommand(cmdrelease.NewReleaseCmd(&cfg))`

## 6. Release Render Commands

- [ ] 6.1 Implement `opm release vet <release.cue>` — load release file, check type (error on BundleRelease), render through pipeline, output per-resource validation
- [ ] 6.2 Implement `opm release build <release.cue>` — load release file, check type, render, output manifests (stdout or split files)
- [ ] 6.3 Implement `opm release apply <release.cue>` — load release file, check type, render, SSA apply to cluster with inventory
- [ ] 6.4 Implement `opm release diff <release.cue>` — load release file, check type, render, compare against live cluster state
- [ ] 6.5 Wire `--module`, `--provider`, `--kubeconfig`, `--context` flags to render commands; validate `--module` is not used with BundleRelease files

## 7. Release Cluster-Query Commands

- [ ] 7.1 Implement `opm release status <name|uuid>` — positional arg, auto-detect name vs UUID, display release health
- [ ] 7.2 Implement `opm release tree <name|uuid>` — positional arg, display resource hierarchy
- [ ] 7.3 Implement `opm release events <name|uuid>` — positional arg, display K8s events
- [ ] 7.4 Implement `opm release delete <name|uuid>` — positional arg, delete release from cluster
- [ ] 7.5 Implement `opm release list` — list all releases in namespace

## 8. Module Command Migration

- [ ] 8.1 Refactor `opm mod build` to construct ephemeral release and delegate to release pipeline
- [ ] 8.2 Refactor `opm mod apply` to construct ephemeral release and delegate to release pipeline
- [ ] 8.3 Update `opm mod vet` to use `debugValues` by default (pass `DebugValues: true` to builder when no `-f` flag)
- [ ] 8.4 Add deprecation aliases for `opm mod status/tree/events/delete/list` that delegate to `opm release` equivalents with deprecation notice (e.g., `use 'opm release status <name>' instead`)

## 9. Testing

- [ ] 9.1 Add unit tests for all `release` render commands (vet, build with release file, build with --module)
- [ ] 9.2 Add unit tests for `release` cluster-query commands (positional arg parsing, name vs UUID detection)
- [ ] 9.3 Add unit tests for `opm mod vet` with debugValues
- [ ] 9.4 Add unit tests for `opm mod build/apply` alias behavior
- [ ] 9.5 Create test fixture: example `<name>_release.cue` file with registry import pattern (`kind: "ModuleRelease"`)
- [ ] 9.6 Create test fixture: example `<name>_release.cue` file for --module flag pattern
- [ ] 9.7 Add unit test: `BundleRelease` file passed to render command returns clear unsupported error
- [ ] 9.8 Add unit test: `LoadRelease()` correctly detects `ModuleRelease` vs `BundleRelease` via `kind` field

## 10. Validation Gates

- [ ] 10.1 Run `task fmt` — all files formatted
- [ ] 10.2 Run `task lint` — golangci-lint passes
- [ ] 10.3 Run `task test` — all unit tests pass
- [ ] 10.4 Run `task test:e2e` — end-to-end tests pass (if applicable)
