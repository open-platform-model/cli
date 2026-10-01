Every test command in this file runs with the CI registry mapping exported:
`CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'` and the same value for `OPM_REGISTRY`. Without it the registry-backed tests skip instead of failing (design.md, "Silent skip without a registry").

## 1. Record the deliberately old pins

- [ ] 1.1 Create the repo-root `.cascade-frozen` with the two entries of design.md D-a, `tests/e2e/instance_build_test.go` and `internal/cmd/platform/check_test.go`, each pinning `opmodel.dev/core@v2` with its one-sentence reason. Use the format in workspace RELEASING.md, section "Cascade files", exactly. Verify by parsing it, e.g. `yq '.frozen[] | [.path, .reason]' .cascade-frozen`: both paths exist and both reasons are non-empty.
- [ ] 1.2 Re-read the doc comments the reasons paraphrase (`tests/e2e/instance_build_test.go:247-253,285-286,316-319`; `internal/cmd/platform/check_test.go:351-356` and the comments above `:761` and `:785`) and confirm each reason states what the test needs the old pin for.
- [ ] 1.3 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `test(fixtures): list the deliberately old core pins in .cascade-frozen`

## 2. Bump the unit-test trees

- [ ] 2.1 In `tests/fixtures/valid/simple-module` and `tests/fixtures/valid/module-with-debug-values`, run `cue mod get opmodel.dev/core@v2.0.0-beta.1` then `cue mod tidy`. Verify `cue mod tidy --check` passes in each tree and the diff touches only the core `v:` line.
- [ ] 2.2 In `internal/instinit/testdata/initvalues` and `internal/workflow/render/testdata/skip-unprovided`, run `cue mod get opmodel.dev/core@v2.0.0-beta.1 opmodel.dev/catalogs/opm@v4.4.4` then `cue mod tidy`. Verify `cue mod tidy --check` passes in each tree and `initvalues` keeps `default: true` on the opm dependency.
- [ ] 2.3 Run `go test -count=1 -v ./internal/cmd/module/ ./internal/workflow/render/ ./internal/instinit/` and verify there is no FAIL and that `TestModVet_ValidModule`, `TestModEval_FixtureModule`, `TestPickValues_AcquiredModule` and the `TestSkipUnprovided_*` tests report PASS, not SKIP. Fix any failure under design.md D-d. If a tree has to be reverted, record it in `.cascade-frozen` in this commit.
- [ ] 2.4 With `kind-opm-dev` up (`task cluster:status`), run `go run tests/integration/skip-unprovided/main.go` and verify it exits 0. This program is not in CI (`.github/workflows/pr.yml:202-209`).
- [ ] 2.5 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `test(fixtures): bump unit-test trees to core v2.0.0-beta.1 and opm 4.4.4`

## 3. Bump the e2e and integration trees

- [ ] 3.1 In `tests/e2e/testdata/duplicate-identities` and `tests/integration/module-apply/testdata`, run `cue mod get opmodel.dev/core@v2.0.0-beta.1 opmodel.dev/catalogs/opm@v4.4.4` then `cue mod tidy`. Verify `cue mod tidy --check` passes in each tree.
- [ ] 3.2 Run `go test -count=1 -v ./tests/e2e/ -run 'TestE2E_ModBuild_RefusesDuplicateIdentities' -timeout 10m` and verify PASS, not SKIP.
- [ ] 3.3 With `kind-opm-dev` up, run `go run tests/integration/module-apply/main.go` and verify it exits 0. Fix any failure under design.md D-d.
- [ ] 3.4 Grep the tracked `cue.mod/module.cue` files under `tests/` and `internal/` for `v2.0.0-alpha` and `"v4.[0-3].`. Verify there is no match, which is the spec scenario "Every old core pin in a test module is accounted for": the frozen pins are Go literals, not `cue.mod` files.
- [ ] 3.5 `task fmt`, `task lint`, `task test`, `task openspec:check`, and the full `task test:e2e` against `kind-opm-dev` green, then commit `test(fixtures): bump e2e and integration testdata to core v2.0.0-beta.1 and opm 4.4.4`
