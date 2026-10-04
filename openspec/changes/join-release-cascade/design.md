## Context

The cli is the last tier of the release cascade and also the release tool for two of its upstreams (workspace RELEASING.md, "Release order" and "Pin classes"). Phase 2 gave it the task the receiver runs. Phase 3 adds three shared reusable workflows to `open-platform-model/.github` (change `add-release-cascade-workflows`, "A" below):

- `cascade-notify.yml`
- `cascade-receive.yml`
- `cascade-gates.yml`

Each repo then joins with thin callers (Phase 3 wiring contract, A's `openspec/changes/add-release-cascade-workflows/contract.md`, archived with A on merge; cited as "wiring §N"). This design covers the cli's callers only. Everything the shared workflows do (payload validation, modes, the workflows guard, the title rule, the App token) belongs to A and is not repeated here.

State of the files this change edits, on `main` at `19f19cd2`:

- `.github/workflows/release.yml`:
  - workflow permissions `contents: write`, `pull-requests: write` (`:56-58`);
  - `release-please` exposes `releases_created` and `tag_name` (`:65-67`) and is skipped on `workflow_dispatch` (`:63`);
  - `goreleaser` needs `release-please` and `publish-templates` (`:91`), runs under `always() && (...)` (`:94`), and refuses a published release in its first step (`:108-141`);
  - `publish-docs` already uses the guard shape this change needs (`:235-239`).
- `.github/labels.yml:117-144`: declares all six cascade labels; `labels.yml` workflow syncs with `skip-delete: false`.
- `.github/scripts/e2e-cluster-applies.sh:31-33`, `:61-63`: the E2E job applies on `deps/cascade` and on the `deps-cascade` label.
- `.github/workflows/cascade-task.yml:34-46`: the repo-at-`repo/`, `.github`-at-`org-github/` layout (Phase 2 contract v1.1, C4), which the shared receiver reuses.
- `AGENTS.md:360`: the "Release cascade" bullet.

## Goals / Non-Goals

**Goals:**

- A cli release that publishes notifies catalog_opm and opm-operator once, including when a manual recovery run finishes a draft.
- An upstream release, the daily sweep or a manual run reaches the shared receiver, which computes the cli's cascade diff in dry run.
- Every cli PR carries `cascade/freshness` and `cascade/settled`, as warnings.
- Nothing in the cli holds or passes the App key.

**Non-Goals:**

- Going live, which is Phase 4.
- Requiring G2 or G3, which is Phase 5.
- Changing the shared workflows, the resolver or `task deps:cascade`.
- Changing the E2E job, `labels.yml` or `pr-title.yml`. They already cover cascade PRs.

## Decisions

### D1. The notify job

```yaml
  notify-downstream:
    name: Notify downstream
    # Runs only after goreleaser published the release (publish-templates is
    # already one of goreleaser's needs), also on a manual run that finishes a
    # draft. A manual run on a published release fails goreleaser's draft
    # check, so a release never notifies twice. The main guard keeps a branch
    # dispatch away from the main-only cascade Environment.
    needs: [release-please, goreleaser]
    if: ${{ !cancelled() && needs.goreleaser.result == 'success' && github.ref == 'refs/heads/main' && vars.CASCADE_NOTIFY != 'off' }}
    permissions:
      contents: read
    uses: open-platform-model/.github/.github/workflows/cascade-notify.yml@main
    with:
      tag: ${{ needs.release-please.outputs.tag_name || inputs.tag }}
```

- The job MUST be a caller job (`uses:`). It MUST NOT declare `environment:`, because a caller job cannot, and it MUST NOT pass `secrets:`. The reusable job declares `environment: cascade` and mints the token itself (wiring §2.2, §4.3; RELEASING.md "Notify after publish").
- It MUST NOT need `publish-docs`. A docs-bundle failure does not unpublish the release, and the downstream task pins the release, not the bundle. The same holds for `core` (wiring §4.5).
- The workflow-level `contents: write` and `pull-requests: write` (`release.yml:56-58`) MUST NOT reach the notify job. Its own `permissions: contents: read` replaces them, and the reusable job asks for no more than that.

**Run matrix** (each row follows from `release.yml` as it stands):

| Trigger | `release-please` | `goreleaser` | notify |
| --- | --- | --- | --- |
| push, no release | runs, `releases_created` empty | skipped (`:94`) | skipped |
| push, release cut, everything green | runs | success | runs with `tag_name` |
| push, release cut, templates fail | runs | failure (`:145-155`) | skipped; the release stays a draft |
| manual run on main, draft tag | skipped | success | runs with `inputs.tag` |
| manual run on main, published tag | skipped | failure (`:121-125`) | skipped |
| manual run on another branch | skipped | as above | skipped (main guard) |
| any of the above with `CASCADE_NOTIFY=off` | | | skipped |

A notify that fails (for example a dispatch that still fails after the retries in wiring §4.1) leaves the release published and the run red. "Re-run failed jobs" re-runs only notify. The daily sweep in each target covers a dispatch that is never re-sent.

### D2. The receiver caller

`.github/workflows/deps-cascade.yml` is the wiring §5 template with the cli values from wiring §5.1:

```yaml
name: Deps cascade

on:
  repository_dispatch:
    types: [upstream-released]
  schedule:
    - cron: '17 6 * * *'
  workflow_dispatch:
    inputs:
      dry_run:
        description: Compute and show the diff in the job summary; push nothing
        type: boolean
        default: false
      gates_only:
        description: Evaluate and post the release-PR gates only (sent by Cascade gates)
        type: boolean
        default: false

permissions: {}

concurrency:
  group: ${{ github.ref != 'refs/heads/main' && format('deps-cascade-{0}', github.ref) || (inputs.gates_only && 'deps-cascade-gates' || 'deps-cascade') }}
  cancel-in-progress: false

jobs:
  cascade:
    name: Deps cascade
    permissions:
      contents: read
      pull-requests: read
      statuses: write
    uses: open-platform-model/.github/.github/workflows/cascade-receive.yml@main
    with:
      dry-run: ${{ inputs.dry_run == true || vars.CASCADE_DRY_RUN != 'false' }}
      gates-only: ${{ inputs.gates_only == true }}
      g2-mode: ${{ vars.CASCADE_G2_MODE || 'warn' }}
      g3-mode: ${{ vars.CASCADE_G3_MODE || 'warn' }}
      setup-go: true
      labels-managed: true
```

- **Dry run fails closed.** The receiver is live only when `CASCADE_DRY_RUN` is exactly `false` (wiring §5, §9.1). Unset, `true` or anything else is a dry run. The caller reads `vars` itself, so nothing depends on how a called workflow sees `vars`.
- **The stop switch is not in the compute job's hands.** The receiver's compute job runs this repository's `task -x deps:cascade`, which on a release PR's gates-only run is the release head's code. That code can write `$GITHUB_ENV` or `$GITHUB_PATH` and so control every later compute step and output. The fail-closed dry run therefore holds only if A's `publish.if` reads `inputs.dry-run`, `inputs.gates-only` and `github.ref` directly, which workflow-call inputs guarantee cannot be forged from inside compute. Gate G-shared checks this.
- **Toolchain.** `setup-go: true` gives Go from `repo/go.mod` (`go 1.26.0`, the version every cli workflow pins). `setup-cue` keeps its default, `v0.17.1`, which equals the cue `cascade-task.yml:55` installs. The task needs `cue` only when a `cue.mod` moves (`cascade.sh:486`). It sets its own registry mapping (`cascade.sh:90`). `task operator:sync` needs only `curl`, `perl` and `gofmt` (`Taskfile.yml:497-499`), which the runner image and setup-go provide.
- **Labels.** `labels-managed: true` makes the receiver check that the five bot-relevant labels exist and fail with "declare it in .github/labels.yml" if one is missing. It never creates a label in the cli (RELEASING.md "Labels"; wiring §6.4 step 4). `labels.yml:122-140` declares all five.
- **No `org-github-ref`.** It defaults to `main`. Production refuses any other value (wiring §2.4).
- **Concurrency.** The expression is copied verbatim from wiring §5. Real runs on `main` share `deps-cascade` (RELEASING.md "Concurrency"). Gates-only runs and branch runs get their own groups, so they never replace a pending real run.

### D3. The per-PR gate caller

`.github/workflows/cascade-gates.yml` is the wiring §8.3 template, verbatim:

```yaml
name: Cascade gates
on:
  pull_request_target:
    types: [opened, reopened, synchronize]
permissions: {}
concurrency:
  group: cascade-gates-${{ github.event.pull_request.number }}
  cancel-in-progress: true
jobs:
  gates:
    name: Cascade gates
    permissions:
      statuses: write
      actions: write
    uses: open-platform-model/.github/.github/workflows/cascade-gates.yml@main
    with:
      g2-mode: ${{ vars.CASCADE_G2_MODE || 'warn' }}
      g3-mode: ${{ vars.CASCADE_G3_MODE || 'warn' }}
```

- `pull_request_target` runs the base branch's workflow, and the reusable job checks nothing out. No PR code runs with `statuses: write` or `actions: write` (wiring §8.3).
- `actions: write` is what lets the job send `gh workflow run deps-cascade.yml --ref main -f gates_only=true` for a release PR. A `workflow_dispatch` sent with `GITHUB_TOKEN` still starts a run.
- The check `Cascade gates` and the two statuses are not added to the cli ruleset here (Phase 5).

### D4. No Dependabot ignore for the shared workflows

An earlier draft added `- dependency-name: "open-platform-model/.github*"` to the `github-actions` ignore list. It is dropped (plan review, finding 5):

- Dependabot proposes `github-actions` updates only for refs that are a version tag or a commit SHA. It never rewrites a branch ref such as `@main`, so the entry would change nothing.
- The docs-kit ignore is not a precedent: docs-kit's `publish.yml` is pinned by release tag (`@v0.4.0`), which Dependabot does update.
- A normative requirement in only one of the six callers, outside wiring §10, would drift from the other repos.

If the supervisor wants the intent recorded, it belongs in wiring §10 for all six callers.

### D5. AGENTS.md

The "Release cascade" bullet (`AGENTS.md:360`) gains three sentences:

- the receiver (`deps-cascade.yml`: dispatch, daily sweep, manual `dry_run`), its fail-closed `CASCADE_DRY_RUN`, and that it is computed by the shared `.github` workflow, never by a local script;
- the notify job and `CASCADE_NOTIFY=off`;
- the gate caller and `CASCADE_G2_MODE` and `CASCADE_G3_MODE`.

It names no section or decision number of the wiring contract, which lives in a `.github` change directory that moves to the archive when A merges (openspec config, "Comments"). It points to RELEASING.md "The cascade" and "Stop switches".

## Research & Decisions

### Where notify hooks in

**Context**: RELEASING.md "Notify after publish" says the cli notifies after `goreleaser` and `publish-templates`, also on the manual recovery path. The wiring research and wiring §4.5 fix the expression.

**Explored**: `release.yml:87-155` (goreleaser's guard, draft check and template requirement), `:186-200` (publish-templates), `:227-254` (publish-docs, the existing `!cancelled`-style guard with the main check).

**Options considered**:

1. `needs: [release-please, goreleaser]` with `!cancelled() && goreleaser.result == 'success' && main`. Runs once per published release on both paths. A published-release manual run cannot double-notify, because goreleaser fails on it.
2. `needs: [release-please, goreleaser, publish-docs]`. A docs failure would suppress the dispatch of a release that is already published. Rejected.
3. Notify on the `release: published` event in a separate workflow. A release published by `GITHUB_TOKEN` (goreleaser) starts no workflow, so this would never fire. Rejected.

**Decision**: Option 1, as wiring §4.5 states it.

**Rationale**: It is the only option that fires exactly once per published release and needs no new state.

### Toolchain inputs for the receiver

**Context**: The shared receiver installs Go from `go-version-file`, cue through setup-cue, and Task, and checks for mikefarah yq (wiring §5.1, §6.2 step 3). The cli's own `cascade-task.yml` installs cue with `go install` and Go by a literal `1.26.0`.

**Options considered**:

1. Use the receiver's defaults (`setup-go: true`, setup-cue `v0.17.1`). Same versions as `cascade-task.yml`, with no per-repo input.
2. Add a `cue-version` override read from `pr.yml`. The task's `language.version` check reads the pinned cue from its own source file (`cascade.sh:393-395`), not from PATH, so an override gains nothing.

**Decision**: Option 1.

**Rationale**: Both sources give the same versions today. A drift between them shows as a `language.version` warning in the cascade body, not as a wrong diff.

### Verification without a cluster or a live run

**Context**: The callers are thin. Their behaviour lives in the shared workflows, which A tests in the sandbox cycle (wiring §11). The cli has no `actionlint` in CI.

**Options considered**:

1. Add `actionlint` to the cli's `Lint` job. It is useful, but it is a new CI dependency outside this change's scope, and it cannot check a remote reusable workflow's inputs.
2. Run the scratchpad `actionlint` locally on the edited files, read the inputs against A's merged workflow files (gate G-shared), and let the supervisor's post-merge dry run be the end-to-end check (wiring §10).

**Decision**: Option 2. Adding `actionlint` to cli CI is a follow-up.

**Rationale**: It matches wiring §10 ("runs actionlint from the scratchpad binary on the new files") and adds no CI surface.

### Spike findings

- A's three reusable workflows exist on its branch and match D1 to D3 input by input, with the same job permissions the callers grant (task 1.1).
- A's receiver `publish.if` does not yet read `inputs.gates-only`. On a release PR's gates-only run, the compute job runs the release head's task, so G-shared blocks this PR until A adds it (reported to the supervisor).
- The cli's `cascade` Environment, App variables and five cascade labels are in place; `CASCADE_DRY_RUN` is not set yet (task 1.2).
- The first post-merge dry run is not a `noop`: it should show the opm catalog moving from v4.4.4 to v4.5.2 in 16 files, titled `fix(deps): bump opm catalog to v4.5.2` (task 1.3).
- actionlint has no baseline findings on the current workflows (task 1.4).

## Error handling

| Failure | Where it shows | Effect |
| --- | --- | --- |
| A not merged, or a `@main` workflow is later removed, renamed, or given a new required or renamed input | every run of the caller fails at creation, before any job starts | for `release.yml` this stops `release-please` and `goreleaser` too, so no cli release can be cut, and `CASCADE_NOTIFY=off` does not help. G-shared prevents it at merge. Afterwards: revert in `.github`, or a cli PR that removes the job. |
| notify dispatch fails after retries | red `Notify downstream` job on the release run | the release stays published. Re-run failed jobs, or the targets' sweep picks it up within a day. |
| `CASCADE_NOTIFY=off` | notify skipped | the release is not announced. The targets' sweep still finds it, because the sweep re-resolves "newest". |
| `CASCADE_DRY_RUN` unset or not `false` | receiver summary line "DRY RUN: nothing was pushed" | compute and gates run, and publish is skipped. |
| a cascade label missing from `labels.yml` | receiver `publish` fails with "declare it in .github/labels.yml" | only in a live run. `labels.yml` declares all five today. |
| gate dispatch fails (for example `deps-cascade.yml` disabled) | `Cascade gates` red; statuses `WARN:` success in `warn` mode | advisory only until Phase 5 (wiring §8.3, §9.2). |

## Risks / Trade-offs

- **[A changes its interface after this branch is written]** The inputs in D1 to D3 are the wiring §4.1, §5 and §8.3 inputs. → Gate G-shared re-reads the merged files before this PR merges, and task 4.2 compares them input by input.
- **[A later breaks its interface]** GitHub checks a called workflow and its inputs when the caller's run is created, so a breaking change to `cascade-notify.yml` on `.github` `main` stops every cli release run, not only the notify job. → The shared workflows' interface must stay append-only, with new inputs optional. That rule belongs in A's specs (supervisor). Recovery is a revert in `.github` or a cli PR that removes the `notify-downstream` job.
- **[This PR changes workflow files on `main`]** Under `WF_GUARD_RULE=strict`, an open bot-only cascade PR becomes `recreate` after this merges (wiring §7.6). → No cli cascade PR exists before this merges, because the receiver does not exist yet.
- **[Shared App key]** Any job in the cli's `cascade` Environment can reach all seven repos (wiring Facts, §11.5). → This change declares no Environment. Only the shared reusable jobs do, and only from `main`.
- **[A daily scheduled run in a repo that goes quiet]** GitHub disables schedules after 60 days without activity in a public repo. → The cli is active. If it is ever disabled, re-enable it with `gh workflow enable deps-cascade.yml`.

## Open Questions

- None. The Dependabot question is settled in D4.
