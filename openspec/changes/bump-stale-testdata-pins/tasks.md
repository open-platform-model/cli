Every test command in this file runs with the canonical registry mapping exported, which includes the testing domain:
`CUE_REGISTRY='testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'` and the same value for `OPM_REGISTRY` (`.github/workflows/pr.yml:25-26`, `Taskfile.yml:18`). Without it the registry-backed tests skip instead of failing, and without the `testing.opmodel.dev` entry the module-apply program and the instance-build e2e tests fail (design.md D-b and "Silent skip without a registry").

## 1. Record the deliberately old pins

- [x] 1.1 Create the repo-root `.cascade-frozen` with the three entries of design.md D-a: `tests/e2e/instance_build_test.go` pinning `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4`, `internal/cmd/platform/check_test.go` pinning `opmodel.dev/core@v2`, and `internal/instinit/render_test.go` pinning `opmodel.dev/core@v2`, each with its one-sentence reason. Use the format in workspace RELEASING.md, section "Cascade files", exactly. Verify by parsing it, e.g. `yq '.frozen[] | [.path, .pins, .reason]' .cascade-frozen`: every path exists, every pin list is non-empty and every reason is non-empty.
- [x] 1.2 Re-read the doc comments the reasons paraphrase (`tests/e2e/instance_build_test.go:74-76,247-253,285-286,316-319`; `internal/cmd/platform/check_test.go:351-356` and the comments above `:761` and `:785`; `internal/instinit/render_test.go:18-50`) and confirm each reason states what the test needs the old pin for.
- [x] 1.3 `task fmt`, `task lint`, `task test` (against `kind-opm-dev`) and `task openspec:check` green, then commit `test(fixtures): list the deliberately old core and catalog pins in .cascade-frozen`

## 2. Bump the unit-test trees

- [x] 2.1 In `tests/fixtures/valid/simple-module` and `tests/fixtures/valid/module-with-debug-values`, run `cue mod get opmodel.dev/core@v2.0.0-beta.1` then `cue mod tidy`. Verify `cue mod tidy --check` passes in each tree and the diff touches only the core `v:` line.
- [x] 2.2 In `internal/instinit/testdata/initvalues` and `internal/workflow/render/testdata/skip-unprovided`, run `cue mod get opmodel.dev/core@v2.0.0-beta.1 opmodel.dev/catalogs/opm@v4.4.4` then `cue mod tidy`. Verify `cue mod tidy --check` passes in each tree and `initvalues` keeps `default: true` on the opm dependency.
- [x] 2.3 Run `go test -count=1 -v ./internal/cmd/module/ ./internal/workflow/render/ ./internal/instinit/` and verify there is no FAIL and that `TestModVet_ValidModule`, `TestModEval_FixtureModule`, `TestPickValues_AcquiredModule` and the `TestSkipUnprovided_*` tests report PASS, not SKIP. Then run `go test -count=1 -v ./tests/e2e/ -run 'TestE2E_ModBuild_SkipUnprovided|TestE2E_ModuleVet_OpenDebugValuesRefusedAtSynthesis' -timeout 10m` and verify both report PASS, not SKIP. Fix any failure under design.md D-d. If a tree has to be reverted, record it in `.cascade-frozen` in this commit.
- [x] 2.4 With `kind-opm-dev` up (`task cluster:status`), run `go run tests/integration/skip-unprovided/main.go` and verify it exits 0. This program is not in CI (`.github/workflows/pr.yml:202-209`).
- [x] 2.5 `task fmt`, `task lint`, `task test` (against `kind-opm-dev`) and `task openspec:check` green, then commit `test(fixtures): bump unit-test trees to core v2.0.0-beta.1 and opm 4.4.4`

## 3. Bump the e2e and integration trees

- [ ] 3.1 In `tests/e2e/testdata/duplicate-identities` and `tests/integration/module-apply/testdata`, run `cue mod get opmodel.dev/core@v2.0.0-beta.1 opmodel.dev/catalogs/opm@v4.4.4` then `cue mod tidy`. Verify `cue mod tidy --check` passes in each tree.
- [ ] 3.2 Run `go test -count=1 -v ./tests/e2e/ -run 'TestE2E_ModBuild_RefusesDuplicateIdentities' -timeout 10m` and verify PASS, not SKIP.
- [ ] 3.3 With `kind-opm-dev` up, run `go run tests/integration/module-apply/main.go` and verify it exits 0. Fix any failure under design.md D-d.
- [ ] 3.4 Run `git grep -nE '"v2\.0\.0-alpha|"v4\.([0-3]\.[0-9]+|4\.[0-3])"' -- 'tests/**/cue.mod/module.cue' 'internal/**/cue.mod/module.cue'` (or compare every core and catalog pin with `examples/cue.mod/module.cue`). Verify there is no match, which is the spec scenario "Every old core or catalog pin in a test module is accounted for": the frozen pins are Go literals, not `cue.mod` files.
- [ ] 3.5 `task fmt`, `task lint`, `task test` (includes `test:e2e`) against `kind-opm-dev`, and `task openspec:check` green, then commit `test(fixtures): bump e2e and integration testdata to core v2.0.0-beta.1 and opm 4.4.4`

## 4. Archive the change on this branch

- [ ] 4.1 Archive the change on this branch (openspec archive), so the archive rides the implementing PR; never push to main. Run `openspec archive bump-stale-testdata-pins --yes`.
- [ ] 4.2 Verify `task openspec:check` is green, `openspec/specs/test-fixture-lineage/spec.md` carries the requirement "Old test pins are current or frozen with a reason", and its requirement "Maintained fixtures track the current schema line" names `opmodel.dev/catalogs/opm@v4` with all four scenarios still present.
- [ ] 4.3 Commit `chore(openspec): archive bump-stale-testdata-pins` on this branch. Nothing is pushed to `main` directly (owner decision 2026-10-01 (RELEASING.md, "Owner settings")).
