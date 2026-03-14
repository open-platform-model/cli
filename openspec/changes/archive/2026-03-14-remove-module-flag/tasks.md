## 1. Remove `--module` flag registration and plumbing

- [x] 1.1 Remove `Module` field from `ReleaseFileFlags` struct in `internal/cmdutil/flags.go`
- [x] 1.2 Remove `--module` flag registration from `ReleaseFileFlags.AddTo()` in `internal/cmdutil/flags.go`
- [x] 1.3 Remove `ModulePath` field from `ReleaseFileOpts` struct in `internal/workflow/render/types.go`

## 2. Remove `--module` injection logic from render workflow

- [x] 2.1 Remove the entire `--module` injection block (lines 56-77) in `internal/workflow/render/render.go`
- [x] 2.2 Simplify the `#module` filled check (lines 79-84) to a hard error: `"#module is not filled in the release file — import a module to fill it"`
- [x] 2.3 Remove `loader` import from `render.go` if no longer used (check for `loader.LoadModulePackage` removal)
- [x] 2.4 Remove `ModulePath: rff.Module` from `ReleaseFileOpts` construction in `internal/cmd/release/vet.go`
- [x] 2.5 Remove `ModulePath: rff.Module` from `ReleaseFileOpts` construction in `internal/cmd/release/build.go`
- [x] 2.6 Remove `ModulePath: rff.Module` from `ReleaseFileOpts` construction in `internal/cmd/release/apply.go`
- [x] 2.7 Remove `ModulePath: rff.Module` from `ReleaseFileOpts` construction in `internal/cmd/release/diff.go`

## 3. Remove `--module` from command help examples

- [x] 3.1 Remove `--module ./my-module` from example in `internal/cmd/release/vet.go`
- [x] 3.2 Remove `--module ./my-module` from example in `internal/cmd/release/build.go`
- [x] 3.3 Remove `--module ./my-module` from example in `internal/cmd/release/apply.go`
- [x] 3.4 Remove `--module ./my-module` from example in `internal/cmd/release/diff.go`

## 4. Update doc comments

- [x] 4.1 Remove `--module` reference from `LoadReleaseFile` doc comment in `pkg/loader/release_file.go`
- [x] 4.2 Update `LoadModulePackage` doc comment in `pkg/loader/release_file.go` — remove `--module` reference, describe it as used by `opm module vet`
- [x] 4.3 Remove `--module` reference from `FromReleaseFile` doc comment in `internal/workflow/render/render.go`

## 5. Update tests

- [x] 5.1 Remove `--module` flag registration assertions from `internal/cmd/release/release_test.go` (3 assertions)
- [x] 5.2 Remove or update the `--module` workflow test in `internal/workflow/render/render_test.go` (the test using `ModulePath`)
- [x] 5.3 Restructure `tests/e2e/vet_output_test.go` to use CUE import instead of `--module` for module injection (add `cue.mod/` to test fixture)

## 6. Validation

- [x] 6.1 Run `task build` — clean build
- [x] 6.2 Run `task test` — all tests pass
- [x] 6.3 Run `task lint` — no lint issues
