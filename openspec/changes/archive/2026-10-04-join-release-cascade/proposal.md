## Why

Phase 2 gave the cli `task -x deps:cascade` and its title and body tasks (`Taskfile.yml`, archived change `2026-10-04-add-deps-cascade-task`). Nothing runs them yet, and nothing tells the cli's downstreams that a cli release exists. Phase 3 of the rollout wires each repo into the release cascade (workspace RELEASING.md, "Rollout and changes" › "Phases", row 3, and the Changes row `join-release-cascade`).

The binding interface is the **Phase 3 wiring contract (version 3.1)**, merged with the `.github` change `add-release-cascade-workflows` (`open-platform-model/.github#9`, squash `2376ffae4bfc665f327d51581350dea694c01504`) and archived at `openspec/changes/archive/2026-10-04-add-release-cascade-workflows/contract.md` in `open-platform-model/.github`, including its changelogs 3.1.1 and 3.1.2 (cited below as "wiring §N"). §10.1 is this change's checklist. The other sources are workspace RELEASING.md on `main` (sections "The cascade", "Pinning the cascade code", "Two-job split", "Gates", "Stop switches", "Moving the cascade pin" and "Owner settings") and the `.github` README ("Cascade workflows", "Pinning and bumps").

This change was first built to contract version 2 (reusable notify and publish workflows called at `@main`). Version 3 moved the App key into caller-owned jobs that run SHA-pinned composite actions, because a reusable-workflow job cannot read the caller's Environment secret without `secrets: inherit` (sandbox E1). Owner decision 24 pins the cascade actions by SHA in all five repos, and the supervisor extended the pin to the two reusable workflows and the resolver checkout (wiring top item 2, §2.4). This change is rebuilt to version 3.1.

The cli has both roles:

- **Receiver.** It pins library, the opm-operator release and the opm catalog with core (RELEASING.md "What each repo's task moves", cli row). It accepts dispatches from catalog_opm, library and opm-operator (wiring §3.2).
- **Notifier.** It is the release tool that catalog_opm and opm-operator pin in `.opm-cli-version` (RELEASING.md "Notify after publish", cli row; wiring §3.1).

## What Changes

- **Notify** (`.github/workflows/release.yml`): a new last job, `notify-downstream`, exactly the cli block of wiring §4.6.
  - It is a caller-owned job: `runs-on: ubuntu-latest`, `environment: cascade`, `timeout-minutes: 20`, `permissions: {contents: read}`, and one step that runs `open-platform-model/.github/.github/actions/cascade-notify@<SHA> # .github main` with `tag`, `client-id: ${{ vars.CASCADE_APP_CLIENT_ID }}` and `private-key: ${{ secrets.CASCADE_APP_PRIVATE_KEY }}`. The action mints the token and dispatches to catalog_opm and opm-operator (release-tool edges, wiring §3.1).
  - It needs `release-please` and `goreleaser` and runs `if: ${{ !cancelled() && needs.goreleaser.result == 'success' && github.ref == 'refs/heads/main' && vars.CASCADE_NOTIFY != 'off' }}`, so it runs once per published release, also on the manual recovery path, and never for a branch run (wiring §4.5).
  - The runbook comment at the top of `release.yml` gains the notify recovery line, including the skipped case after a manual run from a branch other than `main`.
- **Receiver** (new `.github/workflows/deps-cascade.yml`): exactly wiring §5 with the cli `jobs:` map of §5.2.
  - Job `cascade` calls `cascade-receive.yml@<SHA> # .github main` (compute and gates, no secret) with the fail-closed `dry-run` expression, `gates-only`, `g2-mode`, `g3-mode` and `setup-go: true`.
  - Job `publish` is caller-owned: `environment: cascade`, the byte-for-byte §5 `if:`, and one step running `cascade-publish@<SHA> # .github main` with the same `dry-run` expression, `labels-managed: true` (`.github/labels.yml` owns the cli's labels), `client-id` and `private-key`.
- **Gate caller** (new `.github/workflows/cascade-gates.yml`): exactly wiring §8.3, calling `cascade-gates.yml@<SHA> # .github main`. Pull requests into `main` then carry `cascade/freshness` and `cascade/settled`, as warnings.
- **Resolver pin** (`.github/workflows/cascade-task.yml`): the `open-platform-model/.github` checkout moves from `ref: main` to `ref: <SHA> # .github main`, so CI tests the resolver the receiver runs (wiring §10.1 item 3).
- **Wiring check** (new `.tasks/cascade/wiring-check.sh`, task `cascade:wiring:check`): the wiring §10.1 item 6 script with the 3.1.2 additions (an allow-list of `release.yml` workflow `env` keys, empty for the cli, and `runs-on: ubuntu-latest` on every key-holding job). It runs as the step "Verify the cascade wiring" in `pr.yml`'s required `Lint` job and in the aggregate `task check`.
- **Dependabot** (`.github/dependabot.yml`): the `github-actions` entry ignores `open-platform-model/.github*`, so a cascade reference moves only through a `ci(deps): pin the cascade to .github <sha7>` PR (wiring §10.1 item 7).
- **`AGENTS.md`**: the "Release cascade" bullet describes the caller-owned notify and publish jobs, the receiver, the gate caller, the pin and the wiring check, and the switches `CASCADE_DRY_RUN`, `CASCADE_NOTIFY`, `CASCADE_G2_MODE` and `CASCADE_G3_MODE`.
- **Already in place, unchanged:** the E2E job applies to cascade PRs (`.github/scripts/e2e-cluster-applies.sh` matches `deps/cascade` and the `deps-cascade` label); `pr-title.yml` accepts the bot's `fix(deps)`, `test(fixtures)` and `ci(deps)` titles; `labels.yml` declares every cascade label (wiring §10).

`<SHA>` is `2376ffae4bfc665f327d51581350dea694c01504`, the `.github` `main` squash of PR #9, in all five cascade references.

Release class: none. This is CI wiring, titled `ci: join the release cascade` (wiring §1). No command, flag, output or shipped file changes.

## Depends on / gates

- **`.github` `add-release-cascade-workflows`**: merged (`2376ffa`). The pin must stay on `.github` `main`: `gh api repos/open-platform-model/.github/compare/2376ffae4bfc665f327d51581350dea694c01504...main --jq .status` prints `identical` or `ahead` (wiring §10.1 pre-merge check).
- **Workspace RELEASING.md amendments** (wiring §14): merged on workspace `main` (`501594c`, workspace PR #26).
- **`CASCADE_DRY_RUN=true`** in the cli: set by the supervisor before this change (wiring "Supervisor records"). The receiver merges dry.
- **Phase 0 settings** (read back by the supervisor 2026-10-04, recorded in wiring "Facts"): the cli has the `cascade` Environment (deployments from `main` only) with the secret `CASCADE_APP_PRIVATE_KEY` and the variable `CASCADE_APP_CLIENT_ID`, and the `opm-cascade` App is installed. The cli's own read-back is task 1.2.
- **Already merged:** cli `prepare-release-cascade` (labels, G1, G4), `add-deps-cascade-task` (the task), `add-embedded-operator-e2e-job` (the E2E job on cascade PRs).
- **Not part of this change:** going live (`CASCADE_DRY_RUN=false`, Phase 4); making G2 or G3 required (Phase 5); any change to the `.github` workflows, actions or resolver.

## Capabilities

### New Capabilities

- `release-cascade-wiring`: how the cli joins the release cascade: the receiver caller and its caller-owned publish job, the fail-closed dry run, the concurrency groups, the per-PR gate caller, the key rule, and the one `.github` `main` SHA every cascade reference carries, checked by `task cascade:wiring:check` in the required CI job.

### Modified Capabilities

- `release-workflow`: a new requirement for the caller-owned `notify-downstream` job (when it runs, the tag it sends, the recovery path, the `CASCADE_NOTIFY` switch, the pinned action and the key inputs). Existing requirements are unchanged.

## Impact

- **Commands and packages:** none. No Go code changes.
- **Files:**
  - new: `.github/workflows/deps-cascade.yml`, `.github/workflows/cascade-gates.yml`, `.tasks/cascade/wiring-check.sh`;
  - edited: `.github/workflows/release.yml`, `.github/workflows/cascade-task.yml`, `.github/workflows/pr.yml`, `.github/dependabot.yml`, `Taskfile.yml`, `AGENTS.md`.
- **Secrets:** the key is read only in the caller-owned `notify-downstream` and `publish` jobs, which declare `environment: cascade` and pass `secrets.CASCADE_APP_PRIVATE_KEY` only as the `private-key` input of the SHA-pinned cascade action; those jobs have no checkout or `run:` of their own, and no `env:`, `container:` or `services:` (the publish action checks out the repo but never runs it); no reusable call passes `secrets:` or `secrets: inherit` (wiring §2.2, §10.1 item 9).
- **Risks:** see design.md "Risks / Trade-offs". The main ones: the shared App key reaches every product repo from any `cascade` Environment job; repo code in `compute` (on a release PR's gates-only run, the release head's) can forge compute's outputs and, once the receiver is live, shape the plan; every later `.github` cascade change needs a pin PR here.
