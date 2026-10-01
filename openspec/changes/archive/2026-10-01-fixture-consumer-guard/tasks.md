# Tasks: fixture-consumer-guard

One PR, title `ci(fixtures): check fixture consumers against what cue resolves`. Sections 1 and 2 land in it; section 3 (archive) is a follow-up PR after merge. The opm-operator change `fixture-consumer-guard` carries the same `hack/fixtures.sh`; the two PRs merge back to back.

Environment for every Go/CUE command, exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`task test` includes `task test:integration`, which needs a live kind cluster; when none is available, run `task test:unit` and `task test:e2e` (with `env -u OPM_CONFIG`) and report integration as skipped instead of starting a cluster unasked.

Design.md "Research & Decisions" records the scratch proof; no assumption is unverified, so there is no spike section.

## 1. `hack/fixtures.sh consumers` (test-fixture-lineage)

- [x] 1.1 Add the `consumers` subcommand to `hack/fixtures.sh` exactly as design.md D1 specifies (header doc lines for the subcommand and `FIX`, `fixture_deps`, `dep_version`, `consumer_fail`, `cmd_consumers`, the dispatch entry and the usage list). Apply the diff in design.md "Appendix: reference implementation" as is (`git apply`); do not re-derive it. Verify: `shellcheck hack/fixtures.sh` is clean; once opm-operator's section 1 is committed, `git -C <workspace>/opm-operator show test/fixture-consumer-guard:hack/fixtures.sh | cmp - hack/fixtures.sh` reports no difference.
- [x] 1.2 Verify the positive and failure paths against GHCR, fresh cache each run: `CUE_CACHE_DIR=$(mktemp -d) hack/fixtures.sh consumers examples tests/e2e/testdata/operator-owned` prints `ok` twice and exits 0; with only `examples` named it fails naming `tests/e2e/testdata/operator-owned/cue.mod/module.cue` as unlisted; with operator-owned's core set to `v2.0.0-alpha.6` it prints the diff and a `FAIL` line and exits 1 (restore the file afterwards); with operator-owned's podinfo pin set to `v0.1.99` it prints a `FAIL ... cue mod get` line, still prints `ok` for `examples`, exits 1, and `TMPDIR=$(mktemp -d)` is empty afterwards. `git status` is clean at the end.
- [x] 1.3 `task fmt`, `task lint`, `task test` (see the integration note) and `task openspec:check` green, then commit `test(fixtures): check fixture consumers against what cue resolves`.

## 2. Wire it into the `fixtures` job (pr-workflow)

- [x] 2.1 `.github/workflows/pr.yml` `fixtures` job: a step `Fixture consumers follow their fixtures` between "Seed the job-local registry from the tree" and "Render parity", with the same `CUE_REGISTRY`/`OPM_REGISTRY` env as the seed step, `run: hack/fixtures.sh consumers examples tests/e2e/testdata/operator-owned`. Extend the job's header comment by one line naming the step. Verify: `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/pr.yml` is clean.
- [x] 2.2 `Taskfile.yml` `test:fixtures`: the same command after the seed line, with `CUE_REGISTRY='{{.MIXED_CUE_REGISTRY}}'`; the `desc` names the step. Verify: `task --dry test:fixtures --verbose` lists it after the seed.
- [x] 2.3 `AGENTS.md`, the fixture paragraph under the registry rules: the sentence "A fixture bump is `opm module version set` on the fixture plus the cue.mod pins in ..." gains that the consumers' core and catalog pins follow the fixture's, that the `fixtures` job checks it with `hack/fixtures.sh consumers`, and that `FIX=1` against a seeded registry writes the fix. No other prose.
- [x] 2.4 Seeded proof (Registry Policy rule 3 opt-in, local only): as run, a throwaway `registry:2` on `127.0.0.1:5593` (not the workspace `task registry:start` registry), scratch copies, empty `HOME` and `DOCKER_CONFIG` (design.md "Research & Decisions"). Bump podinfo with `opm module version set 0.1.900 tests/fixtures/modules/podinfo`, text-re-pin both consumers to `v0.1.900`, set operator-owned's core to `v2.0.0-alpha.6`, seed, and run the consumers step under the mixed mapping pointing `testing.opmodel.dev` at the throwaway registry: it fails with the core diff. `FIX=1` with the same mapping then repairs operator-owned. Every file change from this task is discarded; nothing from it is committed. The throwaway registry, and the `0.1.900` tag with it, was removed afterwards.
- [x] 2.5 `task fmt`, `task lint`, `task test` (see the integration note) and `task openspec:check` green, then commit `ci(fixtures): run the fixture consumer check in the pr fixtures job`.

## 3. Archive (after merge)

- [x] 3.1 Precondition: the PR merged and the `fixtures` job on it showed the new step green. Evidence: merged as #263 (`75936ca`); the PR run 36906266085 "Fixtures (gates, seed, render parity)" job ran "Fixture consumers follow their fixtures" green; main CI run 36909912355 on `75936ca` green. `openspec archive fixture-consumer-guard --yes`. Verify: `task openspec:check` green; the main `pr-workflow` and `test-fixture-lineage` specs carry every MODIFIED scenario heading unchanged and the ADDED requirement.
- [x] 3.2 `task fmt` and `task openspec:check` green, then commit `chore(openspec): archive fixture-consumer-guard`.
