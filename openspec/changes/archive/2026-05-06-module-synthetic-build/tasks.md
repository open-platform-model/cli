## 1. Synthesis function in `pkg/loader`

- [x] 1.1 Add `SynthesizeOptions` struct (`Name`, `Namespace`, fields needed by callers)
- [x] 1.2 Add helper `readModuleCatalogPin(modulePath) (string, error)` that parses `<modulePath>/cue.mod/module.cue` via `cuelang.org/go/mod/modfile.Parse` and returns the version string for `opmodel.dev/core/v1alpha1@v1`; emit a `DetailError` when the dep is absent
- [x] 1.3 Add `SynthesizeModuleReleaseFromPackage(ctx *cue.Context, modulePath string, opts SynthesizeOptions) (cue.Value, error)` returning a `#ModuleRelease`-shaped value with `#module` and `metadata` filled, `values` left for downstream filling
- [x] 1.4 Build `load.Config.Overlay` for the synth temp anchor: `cue.mod/module.cue` (declaring `opmodel.dev/core/v1alpha1@v1` at the copied pin, mirroring the shape of `releases/<env>/<module>/cue.mod/module.cue`) and `wrapper.cue` (`import mr "opmodel.dev/core/v1alpha1/modulerelease@v1"` followed by top-level `mr.#ModuleRelease`, mirroring real `release.cue` shape); use `load.FromString` for both; absolute paths via `filepath.ToSlash`
- [x] 1.5 Anchor created via `os.MkdirTemp`; `defer os.RemoveAll(anchor)` covering both success and error paths
- [x] 1.6 Load user's module via `load.Instances([]string{"."}, &load.Config{Dir: modulePath})` using the same `*cue.Context` as the synth load
- [x] 1.7 Compose with `Value.FillPath` directly on the synth package value (since `mr.#ModuleRelease` is at the top of `wrapper.cue`); fill `#module`, `metadata.name`, `metadata.namespace`. No string CUE generation.
- [x] 1.8 Reject inputs whose loaded package is bundle-shaped (`#Bundle`/`#BundleRelease`) with a clear error
- [x] 1.9 Wrap registry-resolution errors from the synth load with `fmt.Errorf("resolving catalog dep for synth wrapper: %w", err)` so users see a clear context note when the registry is unreachable

## 2. Synthesis unit tests

- [x] 2.1 Table-driven test: synthesise from each module under `examples/modules/` that defines `debugValues`; assert the returned value satisfies `#ModuleRelease`-shape after `LoadModuleReleaseFromValue`
- [x] 2.2 Test path: directory does not exist
- [x] 2.3 Test path: directory contains no CUE package
- [x] 2.4 Test path: bundle-shaped input rejected
- [x] 2.5 Test path: user module declares no `opmodel.dev/core/v1alpha1@v1` dep — assert the `DetailError` hint
- [x] 2.6 Test path: caller-supplied `--name`/`--namespace` overrides the defaults
- [x] 2.7 Test path: anchor temp dir removed after success and after error (assert via `os.Stat` returning `IsNotExist`)
- [x] 2.8 Test path: synth and user modfile pin different catalog versions if we tampered — sanity-check that the synth modfile actually copies, not hardcodes (use a fixture with a non-default pin)

## 3. Workflow function `internal/workflow/render.FromModule`

- [x] 3.1 Add `ModuleOpts` and `FromModule(ctx, opts) (*Result, error)` mirroring `FromReleaseFile` shape
- [x] 3.2 Resolve values: `-f`/`--values` files (via `loader.LoadValuesFile`) or fall back to `debugValues` lookup on the loaded module value (mirror `internal/cmd/module/vet.go` selection logic)
- [x] 3.3 Call `loader.SynthesizeModuleReleaseFromPackage`, then `loader.LoadModuleReleaseFromValue`, then `renderPreparedModuleRelease`
- [x] 3.4 Print synthesis banner before render output (`Building synthetic release "<name>" for module "<modName>"`)
- [x] 3.5 Apply namespace override semantics consistent with `FromReleaseFile`

## 4. Workflow unit tests

- [x] 4.1 Table-driven test through `FromModule` against fixtures from `examples/modules/`
- [x] 4.2 Test missing `debugValues` path returns the expected actionable error
- [x] 4.3 Test `-f` files override `debugValues`

## 5. Wire `opm release build` to branch on argument type

- [x] 5.1 Update `internal/cmd/release/build.go`: stat the positional argument and dispatch to `FromReleaseFile` (file) or `FromModule` (directory)
- [x] 5.2 Add `--name` flag; surface a warning when `--name` is set in file mode
- [x] 5.3 Update command help text and `Examples` section to cover the directory form
- [x] 5.4 Unit test the dispatcher: file vs dir vs missing path; flag warning emitted in file mode only

## 6. Add `opm module build` (alias `opm mod build`)

- [x] 6.1 Add `internal/cmd/module/build.go` with `NewModuleBuildCmd(*config.GlobalConfig)`, `Args: cobra.MaximumNArgs(1)`, default arg `"."`
- [x] 6.2 Register the subcommand in `internal/cmd/module/mod.go`
- [x] 6.3 Reject file arguments with the documented error message
- [x] 6.4 Wire the same flag set as `opm release build` directory mode (`-f`, `-n`, `--name`, `-o`, `--split`, `--out-dir`, `--provider`, `--verbose`)
- [x] 6.5 Unit test the reject-file path and default-cwd behaviour

## 7. Docs and examples

- [x] 7.1 Update `cli/QUICKSTART.md` with a "Render a module without writing a release.cue" section
- [x] 7.2 Add at least one `examples/modules/<one>/README.md` snippet showing `opm module build`
- [x] 7.3 Regenerate the auto-generated CLI reference (run docgen target if present in `cli/Taskfile.yml`)

## 8. Validation gates

- [x] 8.1 `task fmt`
- [x] 8.2 `task vet`
- [x] 8.3 `task lint`
- [x] 8.4 `task test:unit`
- [x] 8.5 `task test:e2e` for end-to-end coverage of `opm module build` against an `examples/modules/` fixture (registry pre-warmed in CI)
