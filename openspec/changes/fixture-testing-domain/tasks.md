# Tasks — fixture-testing-domain

## 1. Fixture becomes a publishable artifact

- [x] 1.1 Add `tests/fixtures/modules/podinfo/identity/identity.cue` declaring
      `ModulePath: "testing.opmodel.dev/modules/cli/podinfo@v0"` and `Version: #VersionType | *"0.1.4"`,
      with no release-please marker (the fixture is hand-versioned).
- [x] 1.2 Rename `cue.mod/module.cue`'s `module:` to the same coordinate.
- [x] 1.3 Derive `metadata.{name,modulePath,version}` from the identity package in `module.cue`
      (the `templates/minimal` shape); rewrite the provenance header as a deliberate fork.
- [x] 1.4 `cue vet ./...` clean and `cue eval ./identity --out text -e Version` reports `0.1.4`.
- [x] 1.5 `opm module publish --dry-run` resolves with no refusals.

## 2. Publish pipeline

- [x] 2.1 Add `.github/scripts/publish-fixtures.sh` — identity-driven, repo derived from `ModulePath`,
      caller-side GHCR already-published filter, `--dry-run` mode tolerating only the
      already-published refusal.
- [x] 2.2 Add `.github/workflows/publish-fixtures.yml` — push to `main` on fixture paths plus
      `workflow_dispatch`, job-level `packages: write`, fork guard.
- [x] 2.3 Add the `fixture-gates` dry-run job to `pr.yml` and map `testing.opmodel.dev` in its
      registry env.
- [x] 2.4 Export `OPM_REGISTRY` alongside `CUE_REGISTRY` in both publish scripts. `opm` resolves
      `--registry` > `OPM_REGISTRY` > `~/.opm/config.cue` and never reads `CUE_REGISTRY`
      (`internal/config/resolver.go:44-74`), so exporting only the latter silently left the binary
      on the caller's personal config. Pre-existing in `publish-templates.sh`, fixed in the same
      pass; verified by driving the dry-run against a registry where the tag exists and confirming
      the already-published branch fires.

## 3. Consumers

- [x] 3.1 Re-point `examples/` (cue.mod deps, instance import, values comment, README prose).
- [x] 3.2 Re-point `tests/e2e/testdata/handoff/` (cue.mod deps, import, header comment).
- [x] 3.3 Re-point `tests/integration/{render-parity,ssa-ownership}/main.go`.
- [x] 3.4 Re-point `internal/workflow/apply/thineditor_test.go` and
      `internal/workflow/handoff/verify_test.go`.
- [x] 3.5 Leave the near-miss coordinates untouched: `modules/test_module@v0`,
      `opmodel.dev/modules/podinfo@v0` and its golden digest, `gates_test.go:208`'s expected refusal.

## 4. Collapse the registry overrides

- [x] 4.1 `Taskfile.yml`: `OPM_REGISTRY` maps both domains to GHCR; `KIND_CUE_REGISTRY` defaults to
      empty and is documented as the local-iteration opt-in.
- [x] 4.2 `cluster:operator`: drop the registry precondition, the network connect and the `--registry`
      patch from the default path; gate all three behind a non-empty `KIND_CUE_REGISTRY`; rewrite the
      summary and the failure hint.
- [x] 4.3 `hack/opm-config.cue` and `examples/Taskfile.yml` map both domains to GHCR; delete the
      dead `opmodel.dev/modules/test=localhost` prefix entry and its deviation comment.
- [x] 4.4 Refresh the stale localhost example in `internal/config/schema/config.cue`.

## 5. Policy and records

- [x] 5.1 Amend the root `CLAUDE.md` Registry Policy: canonical mapping, rule 1, rule 3 (testing
      domain is GHCR-backed; localhost is an opt-in override; never put a fixture on `opmodel.dev`),
      rule 5, and narrow the known-deviations note to opm-operator.
- [x] 5.2 Update the restatements: root `Taskfile.yml`, `.tasks/config.yml`,
      `.tasks/registry/docker.yml`, `.claude/settings.json`'s allowlisted export string.
- [x] 5.3 Update `cli/CLAUDE.md`'s registry section and add the fixture-authoring rule.
- [x] 5.4 Record the slice on `enhancements/0011` with a history event.

## 6. Verification

- [x] 6.1 `task fmt`, `go build ./...`, `go test ./internal/... ./pkg/...`.
- [x] 6.2 Bootstrap publish: dispatch `publish-fixtures.yml` against the branch, confirm the GHCR
      manifest for `testing.opmodel.dev/modules/cli/podinfo:v0.1.4` returns 200.
- [x] 6.3 With the local registry stopped: `cd examples && task deps:update` and
      `opm instance build ./examples/instances/podinfo/instance.cue`.
- [ ] 6.4 With the local registry stopped: `task cluster:create && task cluster:operator`,
      then `task test:integration` and `task test:e2e`.
      Run 2026-08-28 on a recreated `opm-dev` (main at f3d722f, registry stopped): 6.2 and 6.3 pass
      (podinfo v0.1.4 and v0.1.5 manifests 200; `examples` deps:update moved catalogs/opm to alpha.7,
      instance build renders). 6.4 blocked by three defects outside this change:
      (a) `task cluster:operator` refuses to seed a Platform because catalogs/opm@v2 has no stable
      release; installed with `--catalog-prerelease` instead. (b) integration `module-apply` and the
      five handoff/thin-editor/delete e2e tests fail inside
      `catalogs/opm/transformers/service-transformer@2.0.0-alpha.6|7` (`output.metadata.name:
      cannot reference optional field: name`), a catalog_opm defect. (c) the e2e harness writes
      `registry: "localhost:5000"` into its test config, so `TestE2E_ModuleVet_Output` and the
      operator-lifecycle install step need the local registry; the lifecycle test then uninstalls the
      operator and CRDs and cannot restore them via (a), leaving the cluster empty for later tests.
      18 e2e tests pass; 5 of 6 integration programs pass.
