## 1. Spike and published-module resolver (`internal/modref`)

- [ ] 1.1 Spike the two unverified assumptions from design.md as tests and verify each passes: a `modregistrytest` in-memory registry serving `v0` and `v1` tags of one module path answers `ModuleVersions` on the major-free path with both majors' tags; `Kernel.AcquireInstanceFromDir` on a module package directory (reuse `tests/e2e/testdata/duplicate-identities` or a minimal module in `t.TempDir()`) returns an error matching `errors.Is(err, oerrors.ErrWrongKind)`. Record any contradiction in design.md (Risks) before continuing
- [ ] 1.2 Add `internal/modref` with `ParsePath` and `ParseSelector`; verify table tests cover every spec scenario of "A published module is named by a major-free module path" and the malformed-selector scenario (`@v1` suffix refused with the `--version v1` hint, `oci://` refused, `v1.0.4` refused showing `v1` and `1.0.4`, valid `v1`, `1.0.4`, `2.0.0-alpha.2`)
- [ ] 1.3 Move the dev-tag predicate from `internal/publish/compat.go` into `modref` and have `publish` call it; verify `go test ./internal/publish/...` passes unchanged
- [ ] 1.4 Add `Source`, `NewSource`, `Resolution` and `Resolve`; verify table tests on the in-memory registry cover: major float prefers stable, prerelease-only major, dev build never floated, exact pin (present, absent, dev tag), highest-compatible walk skipping a major on another core major, a major with no core dependency and a major with no selectable release, nothing compatible listing every major, and a `*publish.ConnectivityError` when the registry is unreachable
- [ ] 1.5 Add the report formatter (selection line plus one line per skipped major) as a pure function; verify a test pins the exact lines for each strategy
- [ ] 1.6 `task lint` and `task test` green, then commit `feat(cmd): add the published-module resolver`

## 2. `opm module build` and `opm module apply` from a published module

- [ ] 2.1 Add the argument classifier in `internal/cmdutil` (rules from design.md); verify table tests cover `.`, `./x`, `../x`, an absolute path, `my-module`, `opmodel.dev/modules/web_app`, and `web.app` existing as a directory (ambiguity refusal with the `./web.app` hint)
- [ ] 2.2 Extend `render.ModuleOpts` with the resolved published module and add the registry branch to `FromModule` (`AcquireModuleFromRegistry`, values origin labelled `<path>@<version>`, `renderInstance` with an empty module root and `sourceLocal` false); verify `replacementWarnings` with an empty module root emits only platform rows (unit test) and `go test ./internal/workflow/render/...` passes
- [ ] 2.3 Wire `opm module build`: `--version` flag, classification, resolution, report via `output.Info` to stderr, refusal of `--version` with a local directory, updated long help and examples; verify command tests assert the flag and its empty default and the local-directory refusal (exit 2)
- [ ] 2.4 Wire `opm module apply` the same way, resolving before any Kubernetes client is built; verify a command test with an unpublished pin exits 2 before cluster config is touched
- [ ] 2.5 Add e2e cases to `tests/e2e/mod_build_test.go` using the podinfo fixture coordinate from `tests/fixtures` (never a literal): build with `--version <major>` renders; build without `--version` resolves the fixture's major and stderr carries the selection line while stdout parses as YAML; the `@`-suffixed path exits 2; verify `task test:e2e` passes against GHCR
- [ ] 2.6 Update `README.md` (module build/apply examples with a published module) and the `AGENTS.md` package map (`internal/modref`); verify `task openspec:check` passes
- [ ] 2.7 `task lint` and `task test` green, then commit `feat(cmd): build and apply published modules by module path`

## 3. `opm instance build` decides by package kind (breaking)

- [ ] 3.1 In `FromInstanceFile`, compute the module context from the instance directory rather than `filepath.Dir(arg)`, drop the file-name heuristics from `ValidateInstanceInputPath`, and map an `ErrWrongKind` on a module package to the exit-2 refusal naming `opm module build <path>`; verify unit tests cover a directory argument with its own `cue.mod`, a file argument, and a module directory refused with the hint
- [ ] 3.2 Remove the directory branch and `--name` from `internal/cmd/instance/build.go` and rewrite its long help; verify a command test asserts `--name` is an unknown flag (exit 1) and the help lists only instance forms
- [ ] 3.3 Update `tests/e2e/instance_build_test.go`: `tests/e2e/testdata/operator-owned` builds identically as a directory and as its `instance.cue`; a module directory exits 2 naming `opm module build`; remove or retarget any case that built a module directory through `instance build`; verify `task test:e2e` passes
- [ ] 3.4 Update `README.md` instance examples and any help text or docs still showing `opm instance build <module-dir>` or `--name` on it (`rg -n "instance build" README.md docs internal`); verify the search finds no stale form
- [ ] 3.5 `task lint` and `task test` green, then commit `feat(cmd)!: decide opm instance build by package kind`
