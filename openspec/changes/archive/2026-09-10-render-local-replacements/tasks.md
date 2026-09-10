# Tasks: render-local-replacements (cli)

## 1. Library bump

- [x] 1.1 `go.mod`: bump `github.com/open-platform-model/library` to the release carrying `RenderInput.LocalReplacements` and `RenderDiagnostics.Replacements`; verify `go build ./...` and `task test` are green before any other edit (an unrelated API drift shows up here, not later).

## 2. `pkg/loader`

- [x] 2.1 `pkg/loader/provenance.go`: add `LocalReplacement{Path, ReplaceWith}` and `LocalReplacements(moduleRoot) ([]LocalReplacement, error)` built on `modfile.ParseLocal` against the root's `module.cue`; reimplement `HasLocalModuleReplacement` on it, keeping "malformed file counts as local"; verify `provenance_test.go` covers absent file, one directory entry, one module-path entry, malformed file, and that `local_module_resolution_test.go` still passes.

## 3. `internal/workflow/render`

- [x] 3.1 `render.go` `renderInstance`: set `LocalReplacements: true` on the `RenderInput`; verify a unit test asserting the field is set (the `renderEnv` fake or a recorded input), and that `TestWarnLocalReplacement` still passes.
- [x] 3.2 New `replacements.go`: `replacementWarnings(rows []kernel.Replacement, moduleRoot string) []string` per design (honoured rows worded with path, target and source; inert module-context entries worded with the platform-file guidance); verify `replacements_test.go` covers: no rows and no file (empty), one honoured platform row, one honoured instance row, one inert module entry, mixed, and stable ordering.
- [x] 3.3 `render.go` and `module.go`: replace `warnLocalReplacement(bool)` at both entry points with emitting `replacementWarnings(out.Diagnostics.Replacements, moduleRoot)` after the render; keep the `SourceLocal` computation as is; delete `localReplacementWarning`; verify `grep -rn localReplacementWarning internal` is empty and the D19 tests are rewritten against the new builder.

## 4. End-to-end

- [x] 4.1 `tests/e2e/instance_build_test.go`: seed a platform with `config.WritePlatformModule`, add a temp never-published module `test.example/lib@v0` whose definition sets a distinctive label, an instance module listing it version-less in `module.cue` and redirecting it in `local-module.cue`, run `opm instance build --platform <dir>`; verify the rendered output carries the label and stderr carries the "in effect" warning naming the path and directory.
- [x] 4.2 Same file: an instance module redirecting `opmodel.dev/catalogs/opm@v4` to a temp directory; verify the render uses the platform's pinned catalog (output unchanged from a clean run) and stderr carries the "ignored … platform names that path" warning.
- [x] 4.3 Same file: the seeded platform's `cue.mod/local-module.cue` redirecting `opmodel.dev/catalogs/opm@v4` to a copy of the resolved catalog from the CUE cache with one transformer label changed; verify the rendered output carries the changed label and stderr names the platform as the source. If copying the cached catalog proves brittle, record that in `design.md` and cover the platform case in the library's tests only.

## 5. Validation gates

- [x] 5.1 `task fmt`, `task lint`, `task test` green; `task e2e` (or the repo's e2e task) green; verify `openspec validate --changes` passes.
