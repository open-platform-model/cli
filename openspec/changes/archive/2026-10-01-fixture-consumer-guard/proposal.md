## Why

A fixture bump can leave a consumer on a stale core or catalog pin, and nothing fails. CUE v0.17.1 keeps a dependency the consumer's `module.cue` already lists at its listed version, so `cue mod tidy --check` passes and `cue export` evaluates against the stale core. That happened in PR 254: `task deps:pins:fixtures` re-pinned only the fixture version in `tests/e2e/testdata/operator-owned` and the core and catalog pins were fixed by hand; `task deps:update` skips every `testdata` tree.

The workspace task is being fixed separately. This change adds the check that catches the drift whatever produced it: run `cue mod get` plus `cue mod tidy` per consumer (the owner's fix direction) in PR CI after `hack/fixtures.sh seed`, the one place the new fixture version resolves. It lives in `hack/fixtures.sh` so opm-operator runs the identical code.

## What Changes

- **`hack/fixtures.sh consumers <dir>...`** (new subcommand, byte-identical with opm-operator): for each consumer, in a scratch copy, `cue mod get <fixture>@<pinned version>` for every `testing.opmodel.dev` pin, then `cue mod tidy`, then a diff against the committed `module.cue`. Any difference, any `cue` error and any tracked `cue.mod` outside the fixtures dir that pins a fixture but is not listed prints a `FAIL <dir>: <reason>` line and fails the run after every consumer was checked. The scratch dir is removed by an EXIT trap. `FIX=1` writes the resolved `module.cue` back instead of failing.
- **`pr.yml` `fixtures` job**: a step after the seed runs `hack/fixtures.sh consumers examples tests/e2e/testdata/operator-owned` under the mixed mapping.
- **`task test:fixtures`**: the same step after its seed.
- **`AGENTS.md`**: the fixture-bump sentence says the consumers' core and catalog pins follow the fixture and that the `fixtures` job checks it.

SemVer class: none. Every commit is `test` or `ci`; nothing in the `opm` binary changes.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `test-fixture-lineage`: consumers follow the fixture's shared pins; a drift check exists and what it reports.
- `pr-workflow`: the `fixtures` job runs the consumer check; the registry-mapping requirement stops saying only `fixtures` runs a registry (stale since PR 254: `unit` and `e2e` seed their own).

## Impact

- Files: `hack/fixtures.sh`, `.github/workflows/pr.yml`, `Taskfile.yml`, `AGENTS.md`, `openspec/specs/{pr-workflow,test-fixture-lineage}/spec.md` (on archive).
- Cross-repo: `hack/fixtures.sh` must stay byte-identical with opm-operator's (workspace `task fixtures:lint`). The opm-operator change carries the same file; merge the two PRs back to back. `task fixtures:lint` is a local task, so the window between merges breaks no CI.
- Cost: a few seconds in a job that already has the seeded registry, `cue` and a fresh `CUE_CACHE_DIR`.
- Risk: the check needs GHCR for core and the catalogs, like every other step of the job.
