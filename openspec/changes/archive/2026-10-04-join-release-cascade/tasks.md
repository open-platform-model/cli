# Tasks: join-release-cascade

The change ships as one PR, titled `ci: join the release cascade` (Phase 3 wiring contract §1). Workers never tag. They never create, edit or delete a GitHub Release, label, repository variable, Environment, ruleset or repo setting. They never touch a Kubernetes cluster. Items marked **SUPERVISOR** are not worker tasks.

The binding interface is the Phase 3 wiring contract (version 3.1, with changelogs 3.1.1 and 3.1.2), archived in `open-platform-model/.github` at `openspec/changes/archive/2026-10-04-add-release-cascade-workflows/contract.md` and cited as "wiring §N"; its §10.1 is this change's checklist. With it: workspace RELEASING.md on `main` and the `.github` README ("Cascade workflows", "Pinning and bumps"). Where they disagree, stop and report the conflict to the supervisor. Do not pick one.

This change was first built to wiring version 2 (reusable notify and publish workflows at `@main`; commits `e82dfa97`, `182d2003`, `ede24684`). Sections 2 to 5 below are the version 3.1 rebuild; the version 2 task records they replace are in git history.

`<SHA>` is `2376ffae4bfc665f327d51581350dea694c01504`, the `.github` `main` squash of PR #9, written `@<SHA> # .github main` (or `ref: <SHA> # .github main`).

**Tools.** `actionlint` v1.7.12 is a scratch build outside this tree; any v1.7.12 binary gives the same result. ShellCheck 0.11.0 and mikefarah yq v4 are on PATH.

**The local gate for every section:**

- `task lint`
- `task test:unit`
- `task openspec:check`
- `actionlint` v1.7.12 on every workflow file, with shellcheck. A finding the section introduced fails the gate.
- From section 4 on, `task cascade:wiring:check`.

`task test` also runs `test:integration` and `test:e2e`, which need a cluster. This change adds no Go code, so report them as not run.

## Gates (merge-time checks; carried in the PR body, not ticked here)

- **G-pin (SUPERVISOR).** `gh api repos/open-platform-model/.github/compare/2376ffae4bfc665f327d51581350dea694c01504...main --jq .status` prints `identical` or `ahead`, and the wiring §10.1 "Pre-merge check" passes on the PR's final head.
- **G-releasing.** Done: the workspace RELEASING.md amendments of wiring §14 are on workspace `main` (`501594c`).
- **G-dry-run-var (SUPERVISOR).** The cli repository variable `CASCADE_DRY_RUN` reads back `true` (`gh variable get CASCADE_DRY_RUN -R open-platform-model/cli`).
- **G-ci.** On this PR, `Lint` (with its "Verify the cascade wiring" step printing `cascade wiring: ok, .github 2376ffae4bfc665f327d51581350dea694c01504 (.github main)`), `Unit Tests`, `G4 operator-embed evidence`, `E2E (kind, embedded operator)` (the full kind suite runs, because `Taskfile.yml` is one of the job's inputs in `.github/scripts/e2e-cluster-applies.sh`; it must be green), `Cascade task (network)` (the `cascade-task.yml` run at the pinned resolver), `Validate Conventional Commit title` and mention-guard are green. `Cascade gates` cannot run on this PR: `pull_request_target` runs `main`'s copy, which does not exist yet.
- **G-post-merge (SUPERVISOR, not a merge precondition)** (wiring §1 "Phase 3 gate", §10):
  - `gh workflow run deps-cascade.yml -R open-platform-model/cli -f dry_run=true`. The summary shows "DRY RUN", `Publish` is skipped, and the `Compute` log shows `scripts from open-platform-model/.github 2376ffae4bfc665f327d51581350dea694c01504`. The action is the diff `task -x deps:cascade` gives on `main` at that time, not necessarily `noop` (task 1.3).
  - On the next PR into `main`, `cascade/freshness` and `cascade/settled` appear as `n/a`.

## 1. Spike: confirm the assumptions in design.md

All checks are read-only. Each result is an indented "Done:" note; design.md "Spike findings" summarises them.

- [x] 1.1 Read the shared workflows' interface (version 2).
  Done: superseded by 1.6. Against the version 2 branch of `.github` (`04bc25d`) every input the version 2 callers passed existed; the review of that build is applied in the rebuild (design "Risks / Trade-offs", spec "Pull requests into main carry the cascade gate statuses").
- [x] 1.2 Confirm the Phase 0 state the callers rely on, read-only (`gh api repos/open-platform-model/cli/environments/cascade`, its `deployment-branch-policies` and `variables`, `gh label list`). Report a missing item to the supervisor; never create it.
  Done (2026-10-04): Environment `cascade` exists with custom branch policies and only `main`; it holds the secret `CASCADE_APP_PRIVATE_KEY` and the variable `CASCADE_APP_CLIENT_ID`. All five bot-relevant labels exist with the RELEASING.md colours (`deps-cascade` 0366d6, `:conflict` b60205, `:hold` fbca04, `:breaking` d93f0b, `need-human-review` e99695). `CASCADE_DRY_RUN` was unset then; the supervisor set it to `true` afterwards (wiring "Supervisor records").
- [x] 1.3 Predict the Phase 3 gate: run `task -x deps:cascade` with the real resolver in a detached scratch worktree of `origin/main` outside this tree, then remove it.
  Done: exit 0 against cli `origin/main` `19f19cd2`: library v1.0.0-beta.3 and opm-operator v1.0.0-beta.5 current, opm catalog v4.4.4 -> v4.5.2; 16 files, 18 insertions, 18 deletions; title `fix(deps): bump opm catalog to v4.5.2`. The post-merge dry run shows a diff, not `noop`, unless the cli bumps the catalog first.
- [x] 1.4 Run `actionlint` on the current `.github/workflows/*.yml` and record the baseline.
  Done: actionlint v1.7.12 with shellcheck 0.11.0 on all workflows of `main`: no findings.
- [x] 1.5 `task openspec:check` green, then commit `docs(openspec): record the join-release-cascade spike findings`.
- [x] 1.6 Re-read the merged interface at `<SHA>` (`git show <SHA>:.github/actions/<name>/action.yml` and `.github/workflows/<name>.yml` in a `.github` checkout) and compare it with design D1 to D3 and wiring §4.1, §6.1, §6.4 and §8.3. Report a mismatch; do not resolve it here.
  Done: `cascade-notify`: `tag`, `client-id`, `private-key` (all required). `cascade-publish`: `dry-run` (required), `labels-managed` (default `'false'`), `client-id`, `private-key`. `cascade-receive.yml`: `dry-run` (boolean, required), `gates-only`, `g2-mode`, `g3-mode`, `setup-go`, `setup-cue`, `cue-version`; outputs `action`, `dry-run`, `compute-ok`. `cascade-gates.yml`: `g2-mode`, `g3-mode`. Matches the wiring sections. Workspace RELEASING.md on `main` (`501594c`) carries the §14 amendments. `.github` `main` is `2376ffa`.
- [x] 1.7 Rewrite proposal, design, specs and this file to wiring version 3.1, then `openspec validate join-release-cascade --strict` and `task openspec:check`, and commit `docs(openspec): rebuild join-release-cascade to wiring contract v3.1`.

## 2. Notify downstream after a release is published (wiring §10.1 items 1, 2, 9)

- [x] 2.1 In `.github/workflows/release.yml`, replace the version 2 `notify-downstream` job with the cli block of wiring §4.6, byte for byte with `<SHA>`, as the last job. The comment above it says only that the job is caller-owned, declares `environment: cascade` and passes the key as the pinned action's input, keeping every clause of the wiring §10.1 item 9 text that applies to it.
  Done: the job is the §4.6 cli block with `cascade-notify@<SHA> # .github main`; the four-line comment above it says it is the only job that reads the key, declares `environment: cascade`, passes `secrets.CASCADE_APP_PRIVATE_KEY` only as `private-key` of the pinned action, and has no checkout, `run:`, `env:`, `container:` or `services:`.
- [x] 2.2 Rewrite the runbook's notify recovery line at the top of `release.yml`: re-run the failed jobs when `Notify downstream` failed; when it was skipped (a manual run from a branch other than `main` published the draft), nothing re-sends it and the daily sweep picks it up; the job's `needs` and `if` reasons; `CASCADE_NOTIFY=off`.
- [x] 2.3 Check design D1's run matrix against the edited file, row by row.
  Done: all seven rows hold. goreleaser's `if:` admits any `workflow_dispatch` ref, so a manual run from another branch can publish a draft while the `main` guard skips notify; that row and the runbook say the sweep covers it.
- [x] 2.4 The local gate is green, then commit `ci(release): run the pinned cascade-notify action in a cascade Environment job`.
  Done: `task lint` (0 issues), `task test:unit` (34 packages ok), `task openspec:check` (68/0), actionlint v1.7.12 with shellcheck on all workflows: clean.

## 3. Receiver, gate caller, resolver pin and Dependabot (wiring §10.1 items 2, 3, 4, 5, 7, 9)

- [x] 3.1 Make `.github/workflows/deps-cascade.yml` wiring §5 with the cli `jobs:` map of §5.2: `cascade-receive.yml@<SHA> # .github main`, `labels-managed` removed from the `cascade` job, the whole `publish` job added with `labels-managed: true` and the `.github/labels.yml` comment on the `cascade-publish` step. Only a header comment between `name:` and `on:` and short comments above a `with:` value are allowed; move the concurrency comment into the header. Rewrite the header's "no secret" text to the wiring §10.1 item 9 wording.
  Done: the file is wiring §5 with the cli §5.2 `jobs:` map (cron `17 6 * * *`, `setup-go: true`, `labels-managed: true` on `cascade-publish`); the only comments are the header and two above `with:` values. The header says compute and gates hold no secret and gives the item 9 key rule for `publish`.
- [x] 3.2 In `.github/workflows/cascade-gates.yml`, change the `uses:` line to `cascade-gates.yml@<SHA> # .github main`; make every other line wiring §8.3 byte for byte, with the comment above `actions: write` moved into the header. The header says the statuses are posted on pull requests into `main` and that `pull_request_target` runs the file as it is on the PR's base branch (main).
  Done: §8.3 byte for byte apart from the header and the pin; no blank lines, as in the template.
- [x] 3.3 In `.github/workflows/cascade-task.yml`, change the resolver checkout's `ref: main` to `ref: <SHA> # .github main`, and say in its comment that the resolver comes from the pinned commit. Keep its `actions/checkout` pin.
  Done: `ref: 2376ffae4bfc665f327d51581350dea694c01504 # .github main`; the `actions/checkout` pin stays `3d3c42e5` (v7.0.1). `CASCADE_RESOLVER_REAL` was already unconditional.
- [x] 3.4 In `.github/dependabot.yml`, add the wiring §10.1 item 7 entry and comment for `open-platform-model/.github*` after the docs-kit entry of the `github-actions` ignore list.
  Done: the entry sits after `open-platform-model/docs-kit*` at the same six-space indentation.
- [x] 3.5 actionlint again with the `.github` references rewritten to local copies of the pinned files (`git show <SHA>:…`), so every input the callers pass is checked; a renamed input must be reported.
  Done: with the four `uses:` rewritten to local copies of the files at `<SHA>`, actionlint v1.7.12 reports nothing on `release.yml`, `deps-cascade.yml` and `cascade-gates.yml`; a renamed `labels-managed` input on the publish step is reported (`input "labels-managedx" is not defined in action "Cascade publish"`), so the input check bites.
- [x] 3.6 The local gate is green, then commit `ci(cascade): run the receiver's publish job and pin the cascade callers to .github main`.
  Done: `task lint` (0 issues), `task test:unit`, `task openspec:check` (68/0), actionlint on all workflows: clean.

## 4. The wiring check in the required CI job (wiring §10.1 item 6)

- [x] 4.1 Add `.tasks/cascade/wiring-check.sh`, the wiring §10.1 item 6 script with `RECEIVER=true` and `PIN_COMMENT='.github main'`, plus the changelog 3.1.2 additions: `RELEASE_ENV_KEYS='[]'` replaces the `BASH_ENV`/`ENV`/`NODE_OPTIONS` deny-list with an exact allow-list of `release.yml`'s workflow `env` keys, and `key_job` asserts `runs-on: ubuntu-latest`. ShellCheck-clean.
  Done: the contract text with those two changes and a header line saying the check guards against mistakes while review and the `main` ruleset guard against a deliberate edit. ShellCheck 0.11.0: clean.
- [x] 4.2 Add the task `cascade:wiring:check` next to `docs:pins:check`, append `- task: cascade:wiring:check` to the aggregate `check` task, and add the step "Verify the cascade wiring" (`run: task cascade:wiring:check`) to `pr.yml`'s `lint` job right after `go-task/setup-task`.
  Done: task `cascade:wiring:check` after `docs:pins:check`; `check` ends with it (its `desc` names it); the `lint` step sits between `go-task/setup-task` and `Cascade task (offline)`.
- [x] 4.3 Test the check: it passes on the tree; it refuses every mutation of wiring §10.1 item 6 (the 13 version 3.1 and 11 version 3.1.1 ones), plus a `CUE_VERSION` in `release.yml`'s workflow `env` (allowed elsewhere, not in the cli), `runs-on` changed or turned into a list on either key-holding job, another secret as `private-key`, `cascade-publish` by tag, a short SHA, the Environment in object form on another job, and `secrets:` on the gates call; a header comment and a comment above a `with:` value still pass.
  Done (mikefarah yq v4, scratch copies of the tree): the tree passes (`cascade wiring: ok, .github 2376ffae4bfc665f327d51581350dea694c01504 (.github main)`); all 33 mutations are refused, each with a message naming the file and job (the 24 contract mutations and the 9 above); both allowed edits pass.
- [x] 4.4 The local gate (with `task cascade:wiring:check`) is green, then commit `ci(cascade): check the cascade wiring in the required Lint job`.
  Done: `task lint` (0 issues), `task test:unit`, `task openspec:check` (68/0), `task -x deps:cascade:test` offline set, actionlint on all workflows, `task cascade:wiring:check`: all green.

## 5. Documentation and final cross-check (wiring §10.1 items 8, 9, 10, 11)

- [x] 5.1 In `AGENTS.md`, rewrite the "Release cascade" bullet as design D8 says. Cite no wiring section.
  Done: the bullet names the receiver's reusable `cascade` job and own `publish` job, the fail-closed `CASCADE_DRY_RUN` read by the `if:` and the action, `Notify downstream` and `CASCADE_NOTIFY=off`, the gate caller on PRs into `main` with the two mode variables, the key rule, the one `.github` `main` SHA moved only by a `ci(deps)` pin PR and ignored by Dependabot, and `task cascade:wiring:check`; it points to RELEASING.md "The cascade", "Pinning the cascade code", "Moving the cascade pin" and "Stop switches".
- [x] 5.2 Run the wiring §10.1 item 11 re-grep over the files this branch changed plus `cascade-task.yml`, and fix every hit that is not an allowed one (`labels-managed` on or about the `cascade-publish` step; `@main` or `ref: main` about something else; "no secret" about `compute` or `gates` only; historical text that says it is superseded).
  Done: every hit is an allowed one: `labels-managed` on or about the `cascade-publish` step (`deps-cascade.yml`, `wiring-check.sh`, the change docs); `@main` and `ref: main` only in text about the version 2 build or `main`'s state before this change, and in the spec scenario that the check refuses `ref: main`; "no secret" only about `compute` and `gates`; `AGENTS.md:177` "holds no built platform value" is unrelated.
- [x] 5.3 `openspec validate join-release-cascade --strict` and `task openspec:check`; re-read the spec deltas against the final files.
  Done: `openspec validate join-release-cascade --strict` valid, `task openspec:check` 68/0; each scenario matches the YAML as written.
- [x] 5.4 The local gate is green, then commit `docs(agents): describe the pinned cascade callers and the wiring check`.
  Done: `task lint` (0 issues), `task test:unit`, `task openspec:check`, actionlint on all workflows, `task cascade:wiring:check`: green.

## 6. Verify and archive

- [x] 6.1 Run the OpenSpec verify skill on `join-release-cascade` and resolve or report every CRITICAL and WARNING.
  Done (2026-10-04, at `3f3ce756`): Completeness: every task but 6.2 is done; all seven requirements have their implementation (`release.yml` `notify-downstream`, `deps-cascade.yml` `cascade` and `publish`, `cascade-gates.yml`, `cascade-task.yml` `ref:`, `dependabot.yml`, `wiring-check.sh` with its task and `Lint` step). Correctness: the static scenarios are covered by the wiring check (33 mutations refused); no CRITICAL. Read-only pre-merge checks: `compare 2376ffa...main` prints `identical`, `RECEIVER=true`, `PIN_COMMENT='.github main'`, the `grep -A1` shows only the one SHA (4 `uses:` and the resolver `ref:`), and `CASCADE_DRY_RUN` reads back `true`. Cleanliness: no dead code. WARNINGs, reported to the supervisor:
  - the runtime scenarios (dispatch, sweep, a live publish, the label check, the gate statuses) have no test in this repo; they rest on the `.github` offline suites and sandbox cycle and on the post-merge dry run;
  - once `CASCADE_DRY_RUN=false`, a gates-only run or G2's release-head code can forge compute's outputs past `publish`'s `if:` (design "Risks / Trade-offs"); the fix belongs in `.github` and the §5 templates for all four receivers, or in a `release-please--*` ruleset;
  - the branch pins `.github` `main` from its first rebuild commit, because `.github` PR #9 merged before the rebuild, so there is no separate final `ci: pin the cascade to .github main` commit (wiring §10.1 pre-merge check step 1).
  SUGGESTION: the `release-workflow` main spec says every release job runs on `ubuntu-latest`; `notify-downstream` now does, and only the pre-existing `publish-docs` caller job does not.
- [x] 6.2 Archive with `openspec archive join-release-cascade` and commit it in this PR (`docs(openspec): archive join-release-cascade`). The archive rides the implementing PR. Under the supervised swarm protocol, the supervisor decides when this runs, after review.
  Done (2026-10-04): archived after review on the supervisor's instruction; the review fixes are commit `90aec645`.
