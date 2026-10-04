## Context

The cli is the last tier of the release cascade and also the release tool for two of its upstreams (workspace RELEASING.md, "Release order" and "Pin classes"). Phase 2 gave it the task the receiver runs. Phase 3 added the shared cascade code to `open-platform-model/.github` (change `add-release-cascade-workflows`, PR #9, squash `2376ffae4bfc665f327d51581350dea694c01504` on `main`):

- two reusable workflows: `cascade-receive.yml` (jobs `compute` and `gates`, no secret) and `cascade-gates.yml` (the per-PR gate caller's job);
- two composite actions: `cascade-notify` and `cascade-publish`, which a repo's own `cascade` Environment job runs with the App key as an input.

Each repo joins with callers fixed by the Phase 3 wiring contract (version 3.1, with changelogs 3.1.1 and 3.1.2), archived in `.github` at `openspec/changes/archive/2026-10-04-add-release-cascade-workflows/contract.md` and cited as "wiring §N". This design covers the cli's callers only. Everything the shared code does (payload validation, modes, the workflows guard, the title rule, minting the App token) belongs to `.github` and is not repeated here.

This change was first built to wiring version 2: reusable `cascade-notify.yml` and `cascade-receive.yml` called at `@main`, with the reusable jobs declaring the `cascade` Environment. The sandbox cycle showed that a reusable-workflow job does not see the caller's Environment secrets unless the caller passes `secrets: inherit` (E1), so version 3 moved the key-holding work into caller-owned jobs, and owner decision 24 plus the supervisor's extension pin every cascade reference by SHA (wiring top items 1 and 2). This design is the version 3.1 rebuild.

State of the files this change edits, on `main` at `30eac4f0`:

- `.github/workflows/release.yml`: workflow permissions `contents: write`, `pull-requests: write`; no workflow-level `env`; `release-please` is skipped on `workflow_dispatch`; `goreleaser` needs `release-please` and `publish-templates`, runs under `always() && (...)`, and refuses a published release in its first step; `publish-docs` already uses the `main` guard this change needs.
- `.github/workflows/cascade-task.yml`: checks the resolver out from `open-platform-model/.github` at `ref: main`, beside the repo (Phase 2 contract v1.1, C4).
- `.github/workflows/pr.yml`: job `lint` (context `Lint`, required on `main`) installs Task with `go-task/setup-task`.
- `.github/dependabot.yml`: the `github-actions` entry ignores `open-platform-model/docs-kit*`.
- `.github/labels.yml`: declares all six cascade labels; the label sync deletes undeclared ones.
- `AGENTS.md`: the "Release cascade" bullet.

## Goals / Non-Goals

**Goals:**

- A cli release that publishes notifies catalog_opm and opm-operator once, including when a manual recovery run finishes a draft.
- An upstream release, the daily sweep or a manual run reaches the shared receiver, which computes the cli's cascade diff in dry run.
- Pull requests into `main` carry `cascade/freshness` and `cascade/settled`, as warnings.
- The App key is read only in two caller-owned jobs that run nothing but a SHA-pinned cascade action, and CI refuses a PR that breaks that shape by mistake.
- Every cascade reference names one `.github` `main` SHA.

**Non-Goals:**

- Going live, which is Phase 4.
- Requiring G2 or G3, which is Phase 5.
- Changing the shared workflows, the actions, the resolver or `task deps:cascade`.
- Changing the E2E job, `labels.yml` or `pr-title.yml`. They already cover cascade PRs.

## Decisions

### D1. The notify job (wiring §4.5, §4.6 cli block)

Appended as the last job of `release.yml`, byte for byte except the comment above it:

```yaml
  notify-downstream:
    name: Notify downstream
    needs: [release-please, goreleaser]
    if: ${{ !cancelled() && needs.goreleaser.result == 'success' && github.ref == 'refs/heads/main' && vars.CASCADE_NOTIFY != 'off' }}
    runs-on: ubuntu-latest
    environment: cascade
    timeout-minutes: 20
    permissions:
      contents: read
    steps:
      - name: Notify downstream
        uses: open-platform-model/.github/.github/actions/cascade-notify@2376ffae4bfc665f327d51581350dea694c01504 # .github main
        with:
          tag: ${{ needs.release-please.outputs.tag_name || inputs.tag }}
          client-id: ${{ vars.CASCADE_APP_CLIENT_ID }}
          private-key: ${{ secrets.CASCADE_APP_PRIVATE_KEY }}
```

- The job is the cli's own and declares `environment: cascade`. Its one step is the pinned action; the key reaches it only as `private-key`. It has no checkout, `run:`, `env:`, `container:` or `services:` (wiring §2.2, §4.3).
- Its own `permissions: contents: read` replaces the workflow-level `contents: write` and `pull-requests: write`. The App token does the dispatch.
- It does not need `publish-docs`. A docs-bundle failure does not unpublish the release, and the downstream task pins the release, not the bundle.
- The `main` guard keeps a branch dispatch away from the main-only Environment, which would otherwise show a failed deployment (wiring §4.5).
- The comment above the job says only that it is caller-owned, declares the Environment and passes the key as the pinned action's input (wiring §10.1 item 1). The reasons for `needs` and `if` move to the runbook comment at the top of the file.

**Run matrix** (each row follows from `release.yml` as it stands):

| Trigger | `release-please` | `goreleaser` | notify |
| --- | --- | --- | --- |
| push, no release | runs, no release | skipped | skipped |
| push, release cut, everything green | runs | success | runs with `tag_name` |
| push, release cut, templates fail | runs | failure | skipped; the release stays a draft |
| manual run on main, draft tag | skipped | success | runs with `inputs.tag` |
| manual run on main, published tag | skipped | failure (draft check) | skipped |
| manual run on another branch, draft tag | skipped | success, publishes the draft | skipped (main guard); nothing re-sends it, the targets' daily sweep picks the release up |
| any of the above with `CASCADE_NOTIFY=off` | | | skipped |

A notify that fails (a dispatch that still fails after the retries in wiring §4.1) leaves the release published and the run red. "Re-run failed jobs" re-runs only notify. The targets' daily sweep covers a dispatch that is never re-sent.

### D2. The receiver caller (wiring §5, §5.1 cli row, §5.2 cli block)

`.github/workflows/deps-cascade.yml` is wiring §5 with the cli `jobs:` map of §5.2: cron `17 6 * * *`, `setup-go: true` on the reusable `cascade` job, `labels-managed: true` on the `cascade-publish` step. A header comment between `name:` and `on:` and short comments above a `with:` value are the only additions; every other line is byte for byte, so the comment that used to sit above `concurrency:` moved into the header.

- **Two jobs.** `cascade` calls the reusable `cascade-receive.yml` at the pin (compute and gates, no secret; permissions `contents: read`, `pull-requests: read`, `statuses: write`). `publish` is the cli's own job in the `cascade` Environment, one step running `cascade-publish` at the pin, permissions `contents: read`, `pull-requests: read`, timeout 15 minutes.
- **Dry run fails closed.** The receiver is live only when `CASCADE_DRY_RUN` is exactly `false` (wiring §5, §9.1). Three places read the switch from `inputs` and `vars`, never from compute's outputs alone: the reusable `dry-run` input, `publish`'s `if:`, and `cascade-publish`'s required `dry-run` input, which refuses to mint unless the value is exactly `false` and fails on anything but `true` or `false`. The `if:` repeats the switches only so a dry run starts no `cascade` Environment job.
- **`labels-managed: true`** moved from the reusable job (where version 3 has no such input) to the `cascade-publish` step: `.github/labels.yml` owns the cli's labels, so publish checks that the five bot-relevant labels exist, fails with "declare it in .github/labels.yml" if one is missing, and never creates one (RELEASING.md "Labels"; wiring §6.4 step 4).
- **Toolchain.** `setup-go: true` gives Go from `repo/go.mod` (the task builds `opm`, runs `go get`, `go mod tidy` and `go run ./hack/docskit-dump`). `setup-cue` keeps its default, `v0.17.1`, the cue `cascade-task.yml` installs.
- **Concurrency.** Copied verbatim from wiring §5. Real runs on `main` share `deps-cascade`; gates-only runs use `deps-cascade-gates` and branch runs `deps-cascade-<ref>`, so neither replaces a pending real run (RELEASING.md "Concurrency").

### D3. The per-PR gate caller (wiring §8.3)

`.github/workflows/cascade-gates.yml` is wiring §8.3 byte for byte, with `uses: …/cascade-gates.yml@<SHA> # .github main` and a header comment between `name:` and `on:` (the comment that used to sit above `actions: write` moved there).

- `pull_request_target` runs the workflow file as it is on the pull request's base branch, and the reusable job checks nothing out. No PR code runs with `statuses: write` or `actions: write`.
- The statuses are promised for pull requests into `main` only. A pull request into another branch runs that branch's copy of the file, which may be absent.
- A Dependabot pull request's `pull_request_target` token had `statuses: write` in the sandbox (E7, a private repo; the cli is public, so it is a proxy, wiring §8.3). If a real Dependabot PR cannot post, it lacks the two warnings; that is harmless until Phase 5.
- `actions: write` lets the reusable job send `gh workflow run deps-cascade.yml --ref main -f gates_only=true` for a release PR.

### D4. The resolver pin in `cascade-task.yml` (wiring §10.1 item 3)

The `open-platform-model/.github` checkout's `ref:` becomes `2376ffae4bfc665f327d51581350dea694c01504 # .github main`, and its comment says the resolver comes from the repo's pinned commit. CI then tests `task deps:cascade` against the resolver the receiver runs. The cli already sets `CASCADE_RESOLVER_REAL` unconditionally, so it has no skip fallback to remove.

### D5. One SHA, moved only by a pin PR (wiring §2.4)

All five cascade references (notify, receive, publish, gates, the resolver `ref:`) carry `2376ffae4bfc665f327d51581350dea694c01504 # .github main`. A later `.github` cascade change reaches the cli only through a PR titled `ci(deps): pin the cascade to .github <sha7>` that replaces the SHA in every reference (found with `grep -rn -A1 'open-platform-model/.github' .github/workflows`), after the canary rule of wiring §2.4 step 2 (RELEASING.md "Moving the cascade pin").

### D6. The wiring check (wiring §10.1 item 6, with the 3.1.2 additions)

`.tasks/cascade/wiring-check.sh` is the contract's script with `RECEIVER=true` and `PIN_COMMENT='.github main'`, plus the two additions of changelog 3.1.2:

- `release.yml`'s workflow-level `env` keys must equal an allow-list, which is empty for the cli (`RELEASE_ENV_KEYS='[]'`). It replaces the contract script's `BASH_ENV`/`ENV`/`NODE_OPTIONS` deny-list, so any key fails.
- Every key-holding job (`notify-downstream`, `publish`) has `runs-on: ubuntu-latest`.

`task cascade:wiring:check` runs it. It is the step "Verify the cascade wiring" in `pr.yml`'s required `lint` job, right after `go-task/setup-task`, and the last command of the aggregate `task check`. It uses the runner's preinstalled mikefarah yq and refuses any other yq. `ci.yml`'s `lint` job (push only, no Task) is left alone.

The check guards against mistakes. It lives in the PR's own tree, so a deliberate edit can change it along with the workflows; review and the `main` ruleset guard against that.

### D7. Dependabot ignores the cascade references (wiring §10.1 item 7)

Version 2's design (old D4) dropped the ignore because Dependabot never rewrites a branch ref such as `@main`. Under the SHA pin that reason is gone: Dependabot can propose updates for SHA-pinned actions and reusable workflows, and a bump of one reference would break "one SHA per repo". The `github-actions` entry's `ignore:` list gains `open-platform-model/.github*`, after the docs-kit entry, with the contract's comment.

### D8. AGENTS.md

The "Release cascade" bullet describes: the receiver (`deps-cascade.yml`: dispatch, daily sweep, manual `dry_run`) with its reusable compute and gates and its own `publish` job; the fail-closed `CASCADE_DRY_RUN`; the notify job and `CASCADE_NOTIFY=off`; the gate caller and the two mode variables; the key rule; the one `.github` `main` SHA, moved only by a `ci(deps)` pin PR and ignored by Dependabot; and `task cascade:wiring:check` in the `Lint` job. It points to RELEASING.md "The cascade", "Pinning the cascade code" and "Stop switches", and names no wiring section number (the contract lives in another repo's archive).

## Research & Decisions

### Where notify hooks in

**Context**: RELEASING.md "Notify after publish" says the cli notifies after `goreleaser` and `publish-templates`, also on the manual recovery path, only on `main`. Wiring §4.5 fixes the expression.

**Options considered**:

1. `needs: [release-please, goreleaser]` with `!cancelled() && goreleaser.result == 'success' && main`. Runs once per published release on both paths. A published-release manual run cannot double-notify, because goreleaser fails on it.
2. `needs: [release-please, goreleaser, publish-docs]`. A docs failure would suppress the dispatch of a release that is already published. Rejected.
3. Notify on the `release: published` event in a separate workflow. A release published by `GITHUB_TOKEN` (goreleaser) starts no workflow, so this would never fire. Rejected.

**Decision**: Option 1, as wiring §4.5 states it.

### Verification without a cluster or a live run

**Context**: The callers are thin. The shared code is tested by `.github`'s offline suites and its sandbox cycle (wiring §11, §12). actionlint does not check a remote reusable workflow's or a composite action's inputs.

**Decision**: Run the scratchpad `actionlint` v1.7.12 on every workflow, and once more with the `.github` references rewritten to local copies of the pinned files (so actionlint checks every input this change passes); run the wiring check against mutations; let the supervisor's post-merge dry run be the end-to-end check (wiring §10). Adding actionlint to cli CI stays a follow-up.

### Spike findings

- Version 2 spike (`c513fa8`'s branch, before the rebuild): the cli's `cascade` Environment (deployments from `main` only), the App variable and secret, and the five bot-relevant labels are in place (task 1.2). actionlint had no baseline findings (task 1.4).
- The first post-merge dry run is not a `noop` unless the cli bumps the catalog first: against `main` `19f19cd2` it showed the opm catalog moving from v4.4.4 to v4.5.2 in 16 files, titled `fix(deps): bump opm catalog to v4.5.2` (task 1.3). Wiring §1's "today ... noop" does not hold for the cli.
- Version 3.1 re-check (task 1.6): at `2376ffa`, `cascade-notify` takes `tag`, `client-id`, `private-key`; `cascade-publish` takes `dry-run` (required), `labels-managed` (default `'false'`), `client-id`, `private-key`; `cascade-receive.yml` takes `dry-run` (required), `gates-only`, `g2-mode`, `g3-mode`, `setup-go`, `setup-cue`, `cue-version` and exposes `action`, `dry-run`, `compute-ok`; `cascade-gates.yml` takes `g2-mode`, `g3-mode`. Every input the callers pass exists, and every required one is passed.

## Error handling

| Failure | Where it shows | Effect |
| --- | --- | --- |
| a bad pin (a SHA without the cascade code, or not on `.github` `main`) | the caller's run fails at creation or in its first step; `cascade-task.yml` fails on a missing resolver | for `release.yml` a run that cannot load stops `release-please` and `goreleaser` too. The pre-merge `compare` check and the wiring check prevent it; the fix is a pin PR (wiring §10.1 item 10). A pinned SHA cannot go missing from `.github` `main` the way a renamed `@main` file could. |
| notify dispatch fails after retries | red `Notify downstream` job on the release run | the release stays published. Re-run the failed jobs, or the targets' sweep picks it up within a day. |
| `CASCADE_NOTIFY=off` | notify skipped | the release is not announced. The targets' sweep still finds it, because the sweep re-resolves "newest". |
| `CASCADE_DRY_RUN` unset or not `false` | receiver summary line "DRY RUN: nothing was pushed"; `Publish` skipped | compute and gates run; nothing is pushed. |
| a cascade label missing from `labels.yml` | `Publish` fails with "declare it in .github/labels.yml" | only in a live run. `labels.yml` declares all five today. |
| gate dispatch fails (for example `deps-cascade.yml` disabled) | `Cascade gates` red; statuses `WARN:` success in `warn` mode | advisory only until Phase 5 (wiring §8.3, §9.2). |
| a PR breaks a key-holding job's shape | "Verify the cascade wiring" fails in the required `Lint` job | the PR cannot merge until the shape is restored. |

## Risks / Trade-offs

- **[Shared App key]** Any job that runs in the cli's `cascade` Environment can mint a token for every repo the App is installed on (wiring Facts, §11.5). → The Environment deploys from `main` only, the cli `main` ruleset requires a PR, and the wiring check refuses `environment: cascade` or the key anywhere but `notify-downstream` and `publish`.
- **[Repo code can forge compute's outputs]** `compute` runs repo code (the task, and on a release PR's gates-only run, or G2 in any run, the release head's code). That code can write the runner's env, path and output files, so it can forge `action`, `dry-run` and `compute-ok` (wiring §6.2 step 14, m3). Today `CASCADE_DRY_RUN=true`, so `publish`'s `if:` and the action's `dry-run` input, which read `vars` directly, keep every run dry. Once the receiver is live (Phase 4):
  - a gates-only run's forged outputs can pass `publish`'s `if:`, which does not read `inputs.gates_only`; publish then stops only when it finds no valid `cascade-plan` (implementation review finding 1);
  - G2 runs release-head code in the same `compute` job before the plan is built, and the cli has no ruleset limiting who may push a `release-please--*` branch, so a write collaborator could shape the plan and bundle the App then pushes to `deps/cascade` (review finding 2; wiring §8.2 accepts this on the premise that only release-please writes those heads).
  → Both are `.github`- or owner-side fixes (a `gates-only` term in the §5 `if:` and `dry-run` expression for all four receivers, G2 in its own job, or a ruleset on `release-please--*`). The §5 templates and the wiring check are byte for byte, so this change does not alter them; the question is reported to the supervisor, to settle before Phase 4.
- **[Pin cost]** Every later `.github` cascade change needs a `ci(deps)` pin PR here (owner decision 24's stated cost). → RELEASING.md "Moving the cascade pin" and the canary rule.
- **[The wiring check is a lint, not a boundary]** It sits in the PR's own tree. → Review and the `main` ruleset are the control for a deliberate edit.
- **[This PR changes workflow files on `main`]** Under `WF_GUARD_RULE=tree` (wiring §7.6) a workflow change on `main` is merged or rebuilt into an open cascade PR in place. No cli cascade PR exists before this merges anyway.
- **[A daily scheduled run in a repo that goes quiet]** GitHub disables schedules after 60 days without activity in a public repo. → The cli is active. If it is ever disabled, re-enable it with `gh workflow enable deps-cascade.yml`.

## Open Questions

- For the supervisor, before Phase 4: whether to close the gates-only publish path and the release-head-code path above in `.github` (and in the §5 templates for all four receivers) or by a ruleset on `release-please--*`.
