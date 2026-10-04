# Tasks: join-release-cascade

The change ships as one PR, titled `ci: join the release cascade` (Phase 3 wiring contract §1). Workers never tag. They never create, edit or delete a GitHub Release, label, repository variable, Environment, ruleset or repo setting. They never touch a Kubernetes cluster. Items marked **SUPERVISOR** are not worker tasks.

The binding interface is the Phase 3 wiring contract, `openspec/changes/add-release-cascade-workflows/contract.md` in `open-platform-model/.github` (archived with that change on merge; cited as "wiring §N"), together with workspace RELEASING.md, sections "The cascade", "Gates", "Stop switches" and "Owner settings". Where the two disagree, stop and report the conflict to the supervisor. Do not pick one.

**Paths.**

- The workspace root is `<ws>` = `/var/home/emil/dev/open-platform-model`.
- `actionlint` is the scratchpad build, v1.7.12: `<scratchpad>/actionlint/bin/actionlint`.
- "A" is the `.github` change `add-release-cascade-workflows`, on branch `feat/add-release-cascade-workflows` until it merges.

**The local gate for every section:**

- `task lint`
- `task test:unit`
- `task openspec:check`
- `actionlint` on every workflow file the section touched. It runs with `-shellcheck=` and `-pyflakes=` empty when those tools are absent. A finding the section introduced fails the gate. A finding listed in the 1.4 baseline does not.

`task test` also runs `test:integration` and `test:e2e`, which need a cluster. This change adds no Go code, so report them as not run.

## Gates (merge-time checks; carried in the PR body, not ticked here)

- **G-shared.** Before merge, A is merged on `open-platform-model/.github` `main`, and `cascade-notify.yml`, `cascade-receive.yml` and `cascade-gates.yml` exist there.
  - Each input this change passes exists there with the same name and type: `tag`; `dry-run`, `gates-only`, `g2-mode`, `g3-mode`, `setup-go` and `labels-managed`; `g2-mode` and `g3-mode`.
  - No required input is left unpassed.
  - `cascade-receive.yml`'s `publish.if` reads `inputs.dry-run`, `inputs.gates-only` and `github.ref` directly, not only the compute job's outputs. The compute job runs repository code (on a release PR's gates-only run, the release head's), so its outputs cannot hold the stop switch.
  - Advisory, not blocking: the receiver's `setup-go` step passes `cache-dependency-path: repo/go.sum`, and its `setup-task` pin is v2.2.0, the version the workspace uses most.
  - Check: `git -C <ws>/.github fetch && git -C <ws>/.github show origin/main:.github/workflows/<name>.yml`.
  - Merge order: sections 2 and 3 add `@main` callers. GitHub resolves those when a run is created, so before A merges, every run of `release.yml` (release-please and goreleaser included) would fail at load. The per-section invariant (constitution VIII) therefore holds only once G-shared holds. This PR merges only after it.
- **G-releasing.** The workspace RELEASING.md amendments of wiring §14 are merged, or are merged in the same review round as A.
- **G-dry-run-var (SUPERVISOR).** Before merge, the cli repository variable `CASCADE_DRY_RUN` is `true` (owner decision 22). Check: `gh variable get CASCADE_DRY_RUN -R open-platform-model/cli` prints `true`.
- **G-ci.** On this PR, `Lint`, `Unit Tests`, `G4 operator-embed evidence`, `E2E (kind, embedded operator)` (not applicable: passes quickly), `Validate Conventional Commit title` and mention-guard are green. `Lint`, `G4 operator-embed evidence` and `E2E (kind, embedded operator)` are the `main` ruleset's required checks.
  - The new `Cascade gates` run is green too. It cannot run before A merges: `pull_request_target` runs `main`'s copy, which does not contain the file yet. Its first run is on the next PR after merge.
- **G-post-merge (SUPERVISOR, not a merge precondition)** (wiring §1 "Phase 3 gate", §10):
  - `gh workflow run deps-cascade.yml -R open-platform-model/cli -f dry_run=true`. The summary shows "DRY RUN", mode `fresh` and action `noop`, or the diff task 1.3 recorded.
  - On the next PR, `cascade/freshness` and `cascade/settled` appear as `n/a`.

## 1. Spike: confirm the assumptions in design.md

All checks are read-only. Record each result as an indented "Done:" note. Then write a short "Spike findings" subsection at the end of design.md's "Research & Decisions".

- [x] 1.1 Read A's three reusable workflows:
  - from `.github` `origin/main` if A has merged;
  - otherwise from `origin/feat/add-release-cascade-workflows`, or from the local worktree `<ws>/.github/.claude/worktrees/add-release-cascade-workflows`.

  List each `workflow_call` input with its type, default and required flag. Compare them with design D1 to D3 and wiring §4.1, §6.1 and §8.3. If A is not written yet, record that and use the wiring sections as the interface. A mismatch between A and the wiring contract is reported to the supervisor, not resolved here.
  Done: A is written but not merged (local worktree, branch not on `origin`; `.github` `main` is `6a18e7d`). Inputs: `cascade-notify.yml`: `tag` (string, required), `org-github-ref` (string, default `main`). `cascade-receive.yml`: `dry-run` (boolean, required), `gates-only` (boolean, false), `g2-mode` and `g3-mode` (string, `warn`), `setup-go` (boolean, false), `setup-cue` (boolean, true), `cue-version` (string, `v0.17.1`), `labels-managed` (boolean, false), `org-github-ref`. `cascade-gates.yml`: `g2-mode`, `g3-mode` (string, `warn`), `org-github-ref`. Every input D1 to D3 passes exists with that name and type, the only required ones (`tag`, `dry-run`) are passed, and the job permissions A requests (notify `contents: read`; receive up to `contents: read`, `pull-requests: read`, `statuses: write`; gates `statuses: write`, `actions: write`) equal what the callers grant. The gates job dispatches `deps-cascade.yml --ref main -f gates_only=true`, matching D2's input name. Reported to the supervisor: `publish.if` reads `inputs.dry-run` and `github.ref` but not `inputs.gates-only` (contract §6.2 step 5 relies on compute's `action=gates-only` output), so G-shared does not hold yet; `setup-go` has no `cache-dependency-path` and `setup-task` is v2.0.0 (advisory).
- [x] 1.2 Confirm the Phase 0 state the callers rely on, read-only:
  - `gh api repos/open-platform-model/cli/environments/cascade --jq '.name, .deployment_branch_policy'`: the Environment exists with custom branch policies;
  - `gh api repos/open-platform-model/cli/environments/cascade/deployment-branch-policies --jq '.branch_policies[].name'`: lists only `main`;
  - `gh api repos/open-platform-model/cli/environments/cascade/variables --jq '.variables[].name'`: lists `CASCADE_APP_CLIENT_ID`;
  - `gh label list -R open-platform-model/cli --search deps-cascade` and `--search need-human-review`: all five bot-relevant labels exist with the RELEASING.md "Labels" colours.

  Report a missing item to the supervisor. Never create it.
  Done: Environment `cascade` exists with custom branch policies and only `main`; it holds the secret `CASCADE_APP_PRIVATE_KEY` and the variable `CASCADE_APP_CLIENT_ID`. All five bot-relevant labels exist with the RELEASING.md colours (`deps-cascade` 0366d6, `:conflict` b60205, `:hold` fbca04, `:breaking` d93f0b, `need-human-review` e99695). No repository variable is set, so `CASCADE_DRY_RUN` is still the supervisor's G-dry-run-var step (the receiver dry-runs with it unset too).
- [x] 1.3 Predict the Phase 3 gate. Add a detached scratch worktree at a literal path in the supervisor's scratchpad, `git worktree add --detach <scratchpad>/p3-cli-join-spike origin/main`, never inside this tree or under `.git/` (the worktree guard refuses computed paths). In it, run `CASCADE_RESOLVER=<ws>/.github/.github/scripts/cascade/cascade-resolve.sh task -x deps:cascade` (refresh the `.github` checkout with a plain `git fetch` run from inside `<ws>/.github`, and read the resolver from `origin/main` if the local checkout is behind).
  - Record the exit code: 3 means the post-merge dry run must show `noop`; 0 means record `git diff --stat`.
  - Remove the scratch worktree with `git worktree remove --force`.
  Done: exit 0 against cli `origin/main` `19f19cd2` with the real resolver from `.github` `main` `6a18e7d`: library v1.0.0-beta.3 and opm-operator v1.0.0-beta.5 current, opm catalog v4.4.4 -> v4.5.2. `git diff --stat`: 16 files, 18 insertions, 18 deletions (twelve `cue.mod/module.cue`, `hack/kind-platform.yaml`, four template and fixture `identity.cue` version bumps). Title `fix(deps): bump opm catalog to v4.5.2`. The post-merge dry run should show this diff unless the cli bumps the catalog first. Scratch worktree removed.
- [x] 1.4 Run `actionlint` on the current `.github/workflows/*.yml` and record the baseline findings (expected: none, or only pre-existing shellcheck notes).
  Done: actionlint v1.7.12 with shellcheck 0.11.0 (pyflakes off) on all current workflows: no findings. The baseline is empty.
- [x] 1.5 `task openspec:check` green, then commit `docs(openspec): record the join-release-cascade spike findings`. The commit touches only `openspec/changes/join-release-cascade/`.

## 2. Notify downstream after a release publishes

- [x] 2.1 In `.github/workflows/release.yml`, add the job `notify-downstream` after `publish-docs`, exactly as design D1 shows:
  - `needs: [release-please, goreleaser]`;
  - `if: ${{ !cancelled() && needs.goreleaser.result == 'success' && github.ref == 'refs/heads/main' && vars.CASCADE_NOTIFY != 'off' }}`;
  - `permissions: contents: read`;
  - `uses: open-platform-model/.github/.github/workflows/cascade-notify.yml@main`;
  - `with: tag: ${{ needs.release-please.outputs.tag_name || inputs.tag }}`.

  Add no `secrets:`, no `environment:` and no `org-github-ref`. The job comment says why it waits for goreleaser, why it never notifies twice, and why it has the `main` guard. It cites no wiring section number.
  Done: `notify-downstream` appended after `publish-docs` with D1's needs, `if`, permissions, `uses` and `with`; no `secrets:`, `environment:` or `org-github-ref`.
- [x] 2.2 Add one recovery line to the runbook comment at the top of `release.yml` (`:14-39`): "Published release whose downstreams were not notified (`Notify downstream` failed): re-run the failed jobs; the downstream receivers' daily sweep also picks the release up. `CASCADE_NOTIFY=off` stops the notify."
  Done: Recovery line added at the end of the runbook comment.
- [x] 2.3 Check the run matrix of design D1 by reading the edited file: every row's `needs` and `if` gives the stated outcome. Record any row that does not.
  Done: All seven rows hold. `!cancelled()` replaces the implicit `success()`, so a skipped `release-please` on a manual run does not skip notify; on a template failure `goreleaser` still runs under `always()` and fails its template step, so notify is skipped.
- [x] 2.4 The local gate (with `actionlint .github/workflows/release.yml`) is green, then commit `ci(release): notify downstream repos after a release is published`.
  Done: `task lint` (0 issues), `task test:unit`, `task openspec:check` (68/0) green. actionlint v1.7.12 with shellcheck, with the `@main` ref resolved to A's branch copy (`p3-cli-join-lint.sh`): clean; a renamed input is reported, so the input check bites.

## 3. Receiver and gate callers

- [x] 3.1 Create `.github/workflows/deps-cascade.yml` exactly as design D2 shows: cron `17 6 * * *`, `setup-go: true`, `labels-managed: true`, `permissions: {}` at the top, and the job permissions `contents: read`, `pull-requests: read` and `statuses: write`. Put a header comment above `on:` that names:
  - RELEASING.md "The receiver";
  - the fail-closed `CASCADE_DRY_RUN` (live only at exactly `false`);
  - that the shared workflow at `main` does the work.
  Done: Created as D2 with the wiring §5 template and the cli §5.1 values; header comment names RELEASING.md "The receiver", the fail-closed `CASCADE_DRY_RUN` and the shared workflow at `main`.
- [x] 3.2 Create `.github/workflows/cascade-gates.yml` exactly as design D3 shows. Its header comment says:
  - that it posts `cascade/freshness` and `cascade/settled` on every PR;
  - that it checks out no PR code;
  - that neither status is required until Phase 5.
  Done: Created as the wiring §8.3 template; header comment covers both statuses on every PR, no PR checkout, and not required until Phase 5.
- [x] 3.3 Grep check: `grep -n -E 'secrets:|environment:' .github/workflows/deps-cascade.yml .github/workflows/cascade-gates.yml` prints nothing, and `grep -c 'open-platform-model/.github/.github/workflows/cascade-.*\.yml@main' .github/workflows/{release,deps-cascade,cascade-gates}.yml` prints 1 for each file.
  Done: No `secrets:` or `environment:` in either new file; each of the three files has exactly one `cascade-*.yml@main` reference.
- [x] 3.4 The local gate (with `actionlint` on both new files) is green, then commit `ci(cascade): add the cascade receiver and gate callers`.
  Done: `task lint` (0 issues), `task test:unit`, `task openspec:check` (68/0) green; actionlint with shellcheck on all three callers, with `@main` resolved to A's branch copies: clean.

## 4. Documentation and final cross-check

- [x] 4.1 In `AGENTS.md`, extend the "Release cascade" bullet (`:360`) as design D5 says:
  - `deps-cascade.yml` (dispatch, daily sweep, manual `dry_run`) runs the shared receiver, a dry run unless `CASCADE_DRY_RUN` is exactly `false`;
  - `release.yml`'s `Notify downstream` dispatches to catalog_opm and opm-operator after a release publishes, and `CASCADE_NOTIFY=off` stops it;
  - `cascade-gates.yml` posts `cascade/freshness` and `cascade/settled` on every PR, as warnings set by `CASCADE_G2_MODE` and `CASCADE_G3_MODE`.

  Point to RELEASING.md "The cascade" and "Stop switches". Cite no wiring section.
  Done: The bullet now names `deps-cascade.yml` (dispatch, 06:17 UTC sweep, `dry_run`), the fail-closed `CASCADE_DRY_RUN`, `Notify downstream` and `CASCADE_NOTIFY=off`, `cascade-gates.yml` and the two mode variables, and points to RELEASING.md "The cascade" and "Stop switches". No wiring section is cited.
- [x] 4.2 Repeat the 1.1 comparison against A's newest state, input by input. Fix any drift in the callers in its own `ci(cascade)` commit, never inside the 4.4 `docs(agents)` commit, or report it to the supervisor when A and the wiring contract disagree.
  Done: A unchanged since 1.1 (`04bc25d`, still not on `origin`); every caller input matches and the actionlint input check against A's copies is clean. No drift fix needed. The open G-shared item is still A's `publish.if` lacking `inputs.gates-only` (reported).
- [x] 4.3 Run `openspec validate join-release-cascade --strict` and `task openspec:check`. Re-read the three spec deltas against the final files, so every scenario matches the YAML as written.
  Done: `openspec validate join-release-cascade --strict` valid, `task openspec:check` 68/0. Each delta scenario matches the YAML as written; the scenario "A release head's code cannot unlock publishing" depends on A reading `inputs.gates-only` in `publish.if`, which G-shared checks.
- [x] 4.4 The local gate is green, then commit `docs(agents): describe the cascade receiver, notify job and switches`.
  Done: `task lint` (0 issues), `task test:unit`, `task openspec:check` green; no `environment: cascade`, `CASCADE_APP_PRIVATE_KEY` or `secrets: inherit` anywhere under `.github/`.

## 5. Verify and archive

- [ ] 5.1 Run the OpenSpec verify skill (`opsx:verify`) on `join-release-cascade` and resolve or report every CRITICAL and WARNING.
- [ ] 5.2 Archive with `openspec archive join-release-cascade` and commit it in this PR (`docs(openspec): archive join-release-cascade`). The archive rides the implementing PR. Under the supervised swarm protocol, the supervisor decides when this runs, after review.
