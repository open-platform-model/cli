## Why

Phase 2 gave the cli `task -x deps:cascade` and its title and body tasks (`Taskfile.yml:514-547`, archived change `2026-10-04-add-deps-cascade-task`). Nothing runs them yet, and nothing tells the cli's downstreams that a cli release exists. Phase 3 of the rollout wires each repo into the release cascade (workspace RELEASING.md, "Rollout and changes" › "Phases", row 3, and the Changes row `join-release-cascade`). The binding interface is the Phase 3 wiring contract (`p3-wiring-contract.md` in the supervisor's scratchpad, cited below as "wiring §N"), together with RELEASING.md, sections "The cascade", "Gates", "Stop switches" and "Owner settings".

The cli has both roles:

- **Receiver.** It pins library, the opm-operator release and the opm catalog with core (RELEASING.md "What each repo's task moves", cli row). It accepts dispatches from catalog_opm, library and opm-operator (wiring §3.2).
- **Notifier.** It is the release tool that catalog_opm and opm-operator pin in `.opm-cli-version` (RELEASING.md "Notify after publish", cli row: "`goreleaser` and `publish-templates`, also on the manual recovery path"; wiring §3.1).

Today a cli release reaches catalog_opm and opm-operator only when someone runs `task deps:pins:opm-cli` at the workspace root. An upstream release reaches the cli only when someone bumps it by hand. This change joins the cli to the shared workflows that the `.github` change `add-release-cascade-workflows` adds, with the receiver in dry run (`CASCADE_DRY_RUN=true`, owner decision 22, wiring §1 and §9.1).

## What Changes

- **Notify** (`.github/workflows/release.yml`): a new last job, `notify-downstream`, calls `open-platform-model/.github/.github/workflows/cascade-notify.yml@main` with the release tag (wiring §4.3, §4.5 cli row).
  - It needs `release-please` and `goreleaser`. `goreleaser` already needs `publish-templates` (`release.yml:91`), so notify runs only after the templates and every binary are published and the draft is public.
  - It runs `if: ${{ !cancelled() && needs.goreleaser.result == 'success' && github.ref == 'refs/heads/main' && vars.CASCADE_NOTIFY != 'off' }}`.
  - Its tag is `${{ needs.release-please.outputs.tag_name || inputs.tag }}`, the same expression `goreleaser` and `publish-docs` use (`release.yml:99`, `:252`). It therefore also fires on the manual `workflow_dispatch` recovery path (`release.yml:45-54`) when that run finishes a draft. A manual run on a published release fails the draft check (`release.yml:108-141`), so a release never notifies twice.
  - It grants only `contents: read`. The App token minted inside the reusable workflow, in the `cascade` Environment, does the dispatch. That workflow sends to catalog_opm and opm-operator as release-tool edges (wiring §3.1). The cli neither names nor chooses the targets.
  - The runbook comment at the top of `release.yml` (`:14-39`) gains the notify recovery line.
- **Receiver** (new `.github/workflows/deps-cascade.yml`): the per-repo caller of wiring §5. It runs on `repository_dispatch` (`upstream-released`), a daily sweep at `17 6 * * *` (cli tier, wiring §5.1) and `workflow_dispatch` with `dry_run` and `gates_only` inputs. It has `permissions: {}` at the top and the wiring §5 concurrency groups. It calls `cascade-receive.yml@main` with:
  - `dry-run: ${{ inputs.dry_run == true || vars.CASCADE_DRY_RUN != 'false' }}`, so the receiver fails closed: it is live only when the variable is exactly `false`;
  - `setup-go: true` (the task builds `opm`, runs `go get`, `go mod tidy` and `go run ./hack/docskit-dump`; `.tasks/cascade/cascade.sh:442`, `:490`, `:500-501`);
  - `labels-managed: true` (`.github/labels.yml:117-144` declares all six cascade labels, and the label sync deletes undeclared ones; RELEASING.md "Labels"), so the receiver only checks that the labels exist and never creates one;
  - `g2-mode` and `g3-mode` from `vars.CASCADE_G2_MODE` and `vars.CASCADE_G3_MODE`, default `warn`.
- **Gate caller** (new `.github/workflows/cascade-gates.yml`): on `pull_request_target` (`opened`, `reopened`, `synchronize`) it calls `cascade-gates.yml@main` (wiring §8.3). Every PR then carries `cascade/freshness` and `cascade/settled`: `n/a` on an ordinary PR, and a gates-only receiver run on a release PR. Both stay warnings (owner decisions 11 and 12). Neither becomes a required check in this change.
- **Dependabot** (`.github/dependabot.yml:3-12`): the `github-actions` update ignores `open-platform-model/.github*`, the same way it already ignores docs-kit. The shared cascade workflows stay at `@main` (owner decision 13) and are never proposed for a SHA or tag pin.
- **`AGENTS.md`** (`:360`): the "Release cascade" bullet names the receiver, the gate caller, the notify job and the repo variables `CASCADE_DRY_RUN`, `CASCADE_NOTIFY`, `CASCADE_G2_MODE` and `CASCADE_G3_MODE`.
- **Already in place, unchanged:**
  - The E2E job applies to cascade PRs. `.github/scripts/e2e-cluster-applies.sh:31-32` and `:61-63` match the branch `deps/cascade` and the label `deps-cascade`, and `e2e-cluster.yml:17-19` re-runs on label changes. The App's push triggers `pull_request` workflows, because it is not `GITHUB_TOKEN` (wiring §11.4 E2 proves this in the sandbox).
  - `pr-title.yml` accepts the bot's `fix(deps)`, `test(fixtures)` and `ci(deps)` titles: the types are listed and the scope is optional.
  - `labels.yml` already declares every label (wiring §10: "unchanged").

Release class: none. This is CI wiring, titled `ci: join the release cascade` (wiring §1). No command, flag, output or shipped file changes. After GA it would still cut no release.

## Depends on / gates

- **`.github` `add-release-cascade-workflows` merged first.** The callers reference `cascade-notify.yml`, `cascade-receive.yml` and `cascade-gates.yml` at `@main`. Until they exist on `.github` `main`, every caller run fails at workflow load. This PR merges only after A has merged (wiring §1: "B1 to B5 ... merge only after A has merged"). A in turn merges only after the sandbox cycle is green (wiring §11).
- **The workspace RELEASING.md amendments of wiring §14** land before or with A. This change relies on them for `CASCADE_NOTIFY`, the fail-closed dry run and the concurrency groups. It does not edit RELEASING.md.
- **Supervisor, before merge:** set the cli repo variable `CASCADE_DRY_RUN=true` (wiring §1, §10; owner decision 22). The receiver would also dry-run with the variable unset, but the explicit value makes the state visible. The schedule must never fire a live run.
- **Phase 0 settings** (verified 2026-10-04 in `owner-selections-verbatim.md`): the cli has the `cascade` Environment (main only) with `CASCADE_APP_PRIVATE_KEY` and `CASCADE_APP_CLIENT_ID`, and the `opm-cascade` App is installed.
- **Already merged:** cli `prepare-release-cascade` (labels, G1, G4), `add-deps-cascade-task` (the task), `add-embedded-operator-e2e-job` (the E2E job on cascade PRs).
- **Not part of this change:**
  - going live (`CASCADE_DRY_RUN=false`), which is Phase 4;
  - making G2 or G3 required, which is Phase 5 (`require-pin-freshness-gate`);
  - any change to the shared workflows or the resolver.

## Capabilities

### New Capabilities

- `release-cascade-wiring`: how the cli joins the release cascade. It covers:
  - the receiver caller's triggers, inputs, permissions and concurrency;
  - the fail-closed dry run;
  - the per-PR gate caller;
  - the shared workflows being called at `main` with no secret passed.

### Modified Capabilities

- `release-workflow`: a new requirement for the `notify-downstream` job (when it runs, the tag it sends, the recovery path, the `CASCADE_NOTIFY` switch). Existing requirements are unchanged.
- `repo-automation`: a new requirement that Dependabot leaves the shared `.github` workflows at `main`. Existing requirements are unchanged.

## Impact

- **Commands and packages:** none. No Go code changes.
- **Files:**
  - new: `.github/workflows/deps-cascade.yml` and `.github/workflows/cascade-gates.yml`;
  - edited: `.github/workflows/release.yml`, `.github/dependabot.yml` and `AGENTS.md`.
- **Secrets:** none passed. The App key is an Environment secret, read only inside the shared reusable workflows' `cascade` jobs (wiring §2.2).
- **Risks:**
  - **Shared key reach.** Any job that runs in the cli's `cascade` Environment can mint a token for all seven App repos (wiring Facts, §11.5). The main-only Environment policy and the cli `main` ruleset are the controls. This change adds no job that declares the Environment itself.
  - **Workflows guard.** Every workflow change on cli `main` (this PR included, and Dependabot `github_actions` bumps) turns a bot-only cascade PR into `recreate`, and one with a human commit into `conflict`, under `WF_GUARD_RULE=strict` (wiring §7.6). A decides the rule from the E4c sandbox result.
  - **Gates on Dependabot PRs.** If a Dependabot `pull_request_target` run cannot post statuses (wiring §11.4 E7), those PRs lack the two contexts. That is harmless while the contexts are warnings, and is a Phase 5 item (wiring §15 item 3).
  - **Duplicate CI on bot pushes.** `ci.yml` runs on a push to any branch (`ci.yml` `push: branches: ['**']`), so each App push to `deps/cascade` runs `ci.yml` as well as `pr.yml`. This happens already for human branches and is not a new cost class.
