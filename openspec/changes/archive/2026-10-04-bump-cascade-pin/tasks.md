# Tasks: bump-cascade-pin

One PR, titled `ci(deps): pin the cascade to .github 7b9ad1b`. Workers never touch tags, releases, rulesets, repository settings, Environments, variables or secrets.

**Local gate for every section:** `actionlint` on every workflow, `bash .tasks/cascade/wiring-check.sh --pin-on-main`, `task -x deps:cascade:test` (offline set), `task openspec:check`.

## 1. Pin and canonical wiring check

- [x] 1.1 Move all five `.github` references (`release.yml`, `deps-cascade.yml` twice, `cascade-gates.yml`, the `cascade-task.yml` resolver `ref:`) to `7b9ad1bea132f7a3f053a5db61ac3933b59ee226 # .github main`; `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows no other SHA.
- [x] 1.2 Copy `.github/scripts/cascade/wiring-check.sh` at that SHA to `.tasks/cascade/wiring-check.sh` and prove it with `cmp` against both the local `.github` object and the `gh api` raw content.
- [x] 1.3 Add `.tasks/cascade/wiring-check.yaml` with the cli's README-table values; re-derive notify `needs`, `if:` and `tag` from `release.yml` and `publish-workflows` from the workflows that hold a write grant (`release.yml`, `publish-fixtures.yml`, `docs.yml`).
- [x] 1.4 `deps-cascade.yml` `publish`: `if:` gains `&& inputs.gates_only != true`; the `Publish` step passes `gates-only: ${{ inputs.gates_only == true }}`. Confirm `cascade-receive.yml` at the SHA needs no other caller input (no `cue-version`: default `v0.17.1`).
- [x] 1.5 `pr.yml` `lint`: "Verify the cascade wiring" runs `bash .tasks/cascade/wiring-check.sh --pin-on-main` with `GH_TOKEN: ${{ github.token }}`; `task cascade:wiring:check` stays the offline entry.
- [x] 1.6 `.github/dependabot.yml`: `cooldown: {default-days: 7}` on the `github-actions` entry.
- [x] 1.7 Check the `.github` mirror of the cli (`pins.sh`, `classes`, publish allow-list, read at `5f00930`) still matches `main`.
- [x] 1.8 Commit `ci(deps): pin the cascade to .github 7b9ad1b`.

## 2. Docs and verification

- [x] 2.1 `AGENTS.md` release cascade note and the `cascade:wiring:check` task description: the canonical copy, its config, `--pin-on-main`, the gates-only switch and the release key rule.
- [x] 2.2 Negative checks: the wiring check fails on a moved single reference, a missing `gates-only` input, a dropped `cache: false` in a publishing workflow, and with `--pin-on-main` on a SHA not on `.github` `main`.
- [x] 2.3 Run the gates: `task lint`, `task openspec:check`, `task -x deps:cascade:test` (offline set), `actionlint`, `bash .tasks/cascade/wiring-check.sh --pin-on-main`.
- [x] 2.4 Commit `docs(agents): describe the canonical cascade wiring check`.
