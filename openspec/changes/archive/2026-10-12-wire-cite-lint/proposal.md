## Why

The comment rule (an enhancement id only beside a reason in words, never a decision number, never a lone id) is now one shared tool. `.github` PR 22 (`2b66da51`) holds the reusable workflow `cite-lint.yml` and teaches `wiring-check.sh` a `cite-lint` reference kind. The cli must run the same rule locally and in CI, so a later citation by decision number fails the pull request that adds it.

## What Changes

- **Pin.** The five `.github` references move from `6df4000f` to `2b66da51fede4332969c324b069012632d8edea9 # .github main`. `.tasks/cascade/wiring-check.sh` is replaced by the file at that SHA. Between the two SHAs only `wiring-check.sh`, the workflow `cite-lint.yml`, the wiring tests and the README changed; no cascade action or workflow changed, so no cascade behaviour of the cli changes.
- **CI call.** A new workflow `.github/workflows/cite-lint.yml` runs on `pull_request` to `main` and on `push` to `main`. It has one plain call job of `.github`'s `cite-lint.yml` at the pinned SHA, with `paths: .` and the exclude `tests/fixtures/`. `wiring-check.yaml` declares the file under `extra-references` as kind `cite-lint`.
- **Local task.** `task cite-lint` runs the tool at `e0335b2ec44e659c0191bc3523d79d17f26f8e7d` (the commit the reusable workflow defaults to) with the same exclude. The workflow and the task each name the other in a comment, because the exclude lives in both.
- **Excludes.** `tests/fixtures/` only: the fixture modules keep six old citations on purpose (the podinfo module is published); they are rewritten in the next fixture bump that republishes them.
- **OpenSpec CLI pin.** Stays at 1.12.0. Under 1.14.1, `openspec validate --all --strict` fails on 59 of 68 specs on `main` (checked 2026-10-12).

Release class: none. CI and task files only.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-cascade-wiring`: one added requirement for the cite-lint call and task.

## Impact

- **Files:** `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task}.yml`, `.github/workflows/cite-lint.yml`, `.tasks/cascade/wiring-check.{sh,yaml}`, `Taskfile.yml`.
- **Risks:** the tool is fetched by `go run` at a commit, so `task cite-lint` needs the Go module proxy. A later `.github` pin bump moves the CI tool commit while the task keeps its own; the two can differ until the task is bumped.
