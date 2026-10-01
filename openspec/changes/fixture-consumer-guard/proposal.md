## Why

A fixture bump can leave a consumer on a stale core or catalog pin, and nothing fails. CUE v0.17.1 keeps a dependency the consumer's `module.cue` already lists at its listed version; it does not raise it to what the fixture requires. So a consumer that names podinfo `v0.1.11` (core `v2.0.0-beta.1`) next to core `v2.0.0-alpha.6` passes `cue mod tidy --check` and `cue export` really evaluates against alpha.6. That is exactly what happened in PR 254: the workspace `task deps:pins:fixtures` re-pinned only the fixture version in `tests/e2e/testdata/operator-owned` (commit `dd23e01`) and the core and catalog pins were fixed by hand (`a3d8b01`). `task deps:update` skips every `testdata` tree, so nothing else would ever have moved them.

The workspace task is being fixed to copy the fixture's pins into every consumer (workspace PR, outside this repo). This change adds the check that catches the drift whatever produced it: the workspace task, a hand bump (`opm module version set` plus a hand re-pin, the path `AGENTS.md` documents), or a later edit.

The check is the owner's own fix direction ("run cue mod get + cue mod tidy per consumer", workspace `tasks.md`, 2026-10-01), run in the one place the new fixture version resolves: PR CI, after `hack/fixtures.sh seed`. It lives in `hack/fixtures.sh` so opm-operator, whose modulepackages have the same exposure, runs the identical code (its own change, `fixture-consumer-guard` in opm-operator).

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
