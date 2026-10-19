# Capability: release-cascade-wiring

## Purpose
How the cli joins the release cascade: after a release, `release.yml` notifies downstream repos through the SHA-pinned `cascade-notify` action; `deps-cascade.yml` receives upstream releases, computes the dependency bump with `task -x deps:cascade` and publishes it as a `deps/cascade` PR; `cascade-gates.yml` reports the freshness and settled statuses on pull requests into `main`; and the wiring check in the required `Lint` job keeps every caller pinned to one `.github` `main` commit, with the App key read only in the `cascade` Environment.

## Requirements

### Requirement: The cascade receiver runs on a dispatch, a daily sweep and a manual run

`.github/workflows/deps-cascade.yml` SHALL run on `repository_dispatch` of type `upstream-released`, on a daily schedule at `17 6 * * *` (UTC), and on `workflow_dispatch` with the boolean inputs `dry_run` and `gates_only`, both defaulting to false. It SHALL declare `permissions: {}` at the workflow level and SHALL have exactly the top-level keys `name`, `on`, `permissions`, `concurrency` and `jobs`. The receiver computes the cascade diff with this repository's `task -x deps:cascade`.

**Source**: workspace RELEASING.md, sections "The receiver", "Two-job split" and "Gates"; Phase 3 wiring contract (version 3.1) §5 and §5.2.

#### Scenario: An upstream release starts the receiver
- **WHEN** library, opm-operator or catalog_opm publishes a release and its notify job dispatches `upstream-released` to the cli
- **THEN** a `Deps cascade` run starts on `main` and runs the shared receiver against this repository

#### Scenario: The daily sweep runs without a dispatch
- **WHEN** no dispatch arrived and the schedule fires
- **THEN** a `Deps cascade` run starts and resolves the newest published upstreams itself

### Requirement: Real receiver runs share one concurrency group without cancelling
Receiver runs on `main` that are not gates-only SHALL share the concurrency group `deps-cascade`, with `cancel-in-progress: false`. Gates-only runs on `main` SHALL use the group `deps-cascade-gates`. Runs from any other ref SHALL use `deps-cascade-<ref>`. A gates-only run or a branch run therefore never replaces a pending real run. Source: workspace RELEASING.md, section "Concurrency".

#### Scenario: A burst of upstream releases
- **WHEN** three dispatches arrive while a real run is active
- **THEN** one run stays active, at most one is pending, and none is cancelled mid-run

#### Scenario: A gates-only run during a pending real run
- **WHEN** a release pull request's head moves while a real run is pending
- **THEN** the gates-only run starts in `deps-cascade-gates` and the pending real run is not replaced

### Requirement: Comments follow the shared cite-lint rule locally and in CI

The cli SHALL run the shared comment rule through `task cite-lint` and a plain call of `.github`'s `cite-lint.yml`, pinned like every `.github` reference. Both SHALL exclude only `tests/fixtures/`.

#### Scenario: A decision number in a comment fails the check

- **WHEN** a Go or CUE comment outside `tests/fixtures/` names a decision number
- **THEN** `task cite-lint` exits non-zero and the CI call fails the pull request

#### Scenario: The call is plain

- **WHEN** `task cascade:wiring:check` runs
- **THEN** it accepts the call only if its job has the keys `name`, `permissions`, `uses` and `with`, with `permissions` exactly `contents: read`

### Requirement: The cascade job calls the shared receiver workflow at the pinned commit with fixed permissions

The job `cascade` of `.github/workflows/deps-cascade.yml` SHALL call the shared `open-platform-model/.github` reusable workflow `cascade-receive.yml` at the repository's pinned `.github` commit, SHALL grant only `contents: read`, `pull-requests: read` and `statuses: write`.

**Source**: workspace RELEASING.md, sections "The receiver", "Two-job split" and "Gates"; Phase 3 wiring contract (version 3.1) §5 and §5.2.

#### Scenario: The cascade job calls the pinned shared receiver
- **WHEN** the `cascade` job of `.github/workflows/deps-cascade.yml` runs
- **THEN** it calls the shared `open-platform-model/.github` reusable workflow `cascade-receive.yml` at the repository's pinned `.github` commit and grants only `contents: read`, `pull-requests: read` and `statuses: write`

### Requirement: The cascade job passes the dry-run, gates-only, gate mode and setup-go inputs

The job `cascade` of `.github/workflows/deps-cascade.yml` SHALL pass `dry-run`, `gates-only` from the `gates_only` input, `g2-mode` and `g3-mode` from the repository variables `CASCADE_G2_MODE` and `CASCADE_G3_MODE` (each defaulting to `warn`), and `setup-go: true`, because the cli's cascade task builds and runs Go.

**Source**: workspace RELEASING.md, sections "The receiver", "Two-job split" and "Gates"; Phase 3 wiring contract (version 3.1) §5 and §5.2.

#### Scenario: The gate modes default to warn
- **WHEN** the repository variables `CASCADE_G2_MODE` and `CASCADE_G3_MODE` are not set
- **THEN** the `cascade` job passes `g2-mode` and `g3-mode` as `warn`

### Requirement: The publish job needs cascade, runs in the cascade Environment and grants only two read permissions

`deps-cascade.yml` SHALL have a second job, `publish`, that needs `cascade`, runs on `ubuntu-latest`, declares `environment: cascade`, has a 15-minute timeout, grants only `contents: read` and `pull-requests: read`.

**Source**: workspace RELEASING.md, sections "Two-job split" and "Labels"; Phase 3 wiring contract (version 3.1) §5, §5.2 and §6.4; `open-platform-model/.github` README at `7b9ad1b`, "Receiver caller" (security pass 2026-10-04).

#### Scenario: A live run publishes from the caller's own job
- **WHEN** `CASCADE_DRY_RUN` is exactly `false`, a run on `main` computes a change, and its compute job succeeds
- **THEN** the `publish` job starts in the `cascade` Environment and the pinned `cascade-publish` action verifies the plan, mints the App token and updates `deps/cascade` and its pull request

### Requirement: The publish job has exactly one step, the shared cascade-publish action with fixed inputs

`deps-cascade.yml` SHALL have a second job, `publish`, that has exactly one step: the shared `cascade-publish` action at the repository's pinned `.github` commit, with the inputs `dry-run` (the same expression the `cascade` job passes), `gates-only: ${{ inputs.gates_only == true }}`, `labels-managed: true`, `client-id: ${{ vars.CASCADE_APP_CLIENT_ID }}` and `private-key: ${{ secrets.CASCADE_APP_PRIVATE_KEY }}`.

**Source**: workspace RELEASING.md, sections "Two-job split" and "Labels"; Phase 3 wiring contract (version 3.1) §5, §5.2 and §6.4; `open-platform-model/.github` README at `7b9ad1b`, "Receiver caller" (security pass 2026-10-04).

#### Scenario: The publish job runs the shared action as its one step
- **WHEN** the `publish` job of `deps-cascade.yml` runs
- **THEN** its one step is the shared `cascade-publish` action at the repository's pinned `.github` commit, with the inputs `dry-run`, `gates-only`, `labels-managed: true`, `client-id` and `private-key`

### Requirement: The publish job runs only on the receiver caller's if: byte for byte

The `if:` of the `publish` job SHALL be the receiver caller's `if:` in `.github` `references/cascade-workflows.md` byte for byte, which reads `inputs.dry_run`, `inputs.gates_only`, `vars.CASCADE_DRY_RUN` and `github.ref` itself and uses the `cascade` job's outputs only as an extra filter, so a gates-only run never starts a `cascade` Environment job.

**Source**: workspace RELEASING.md, sections "Two-job split" and "Labels"; Phase 3 wiring contract (version 3.1) §5, §5.2 and §6.4; `open-platform-model/.github` README at `7b9ad1b`, "Receiver caller" (security pass 2026-10-04).

#### Scenario: A gates-only run never publishes
- **WHEN** `CASCADE_DRY_RUN` is exactly `false` and `Cascade gates` dispatches a run with `gates_only` true, whose compute runs an open release head's task and forges `action: push` and `compute-ok: true`
- **THEN** the `publish` job does not start, because its `if:` reads `inputs.gates_only` itself, and the `cascade-publish` action would fail before minting because its `gates-only` input is not `false`

### Requirement: The receiver checks that the cascade labels exist and never creates one

Because `.github/labels.yml` owns the repository's labels, publish SHALL check that the cascade labels exist and SHALL never create one.

**Source**: workspace RELEASING.md, sections "Two-job split" and "Labels"; Phase 3 wiring contract (version 3.1) §5, §5.2 and §6.4; `open-platform-model/.github` README at `7b9ad1b`, "Receiver caller" (security pass 2026-10-04).

#### Scenario: The receiver never creates a label in the cli
- **WHEN** a live receiver run publishes a cascade pull request
- **THEN** it checks that the cascade labels exist and creates none, and a label missing from `.github/labels.yml` fails the run with a message to declare it there

### Requirement: The caller passes dry-run so that the receiver is live only when CASCADE_DRY_RUN is exactly false

The caller SHALL pass `dry-run` as `${{ inputs.dry_run == true || vars.CASCADE_DRY_RUN != 'false' }}` to both the reusable `cascade` job and the `cascade-publish` action, so the run is a dry run when the `dry_run` input is true or when the repository variable `CASCADE_DRY_RUN` is anything other than exactly `false`, including unset. A run from any ref other than `main` is always a dry run.

**Source**: workspace RELEASING.md, sections "The receiver" and "Stop switches"; Phase 3 wiring contract (version 3.1) §5 and §9.1.

#### Scenario: Variable unset
- **WHEN** `CASCADE_DRY_RUN` is not set and an upstream dispatch arrives
- **THEN** the run computes and summarises the diff, the `publish` job is skipped, and nothing is pushed

#### Scenario: Variable set to true
- **WHEN** `CASCADE_DRY_RUN` is `true`
- **THEN** every receiver run is a dry run

#### Scenario: Manual dry run while live
- **WHEN** `CASCADE_DRY_RUN` is `false` and a human runs the workflow by hand with `dry_run` true
- **THEN** that run computes and summarises the diff and pushes nothing

#### Scenario: Live only at exactly false
- **WHEN** `CASCADE_DRY_RUN` is exactly `false` and a dispatch arrives on `main`
- **THEN** the receiver may push `deps/cascade` and open or update the cascade pull request

### Requirement: The dry-run switch is enforced in three places and a dry run pushes nothing

The `dry-run` switch SHALL be enforced in three places that read `inputs` and `vars`: the reusable workflow's `dry-run` input, the `publish` job's `if:`, and the `cascade-publish` action's `dry-run` input, which mints no token unless the value is exactly `false`. In a dry run the receiver computes the diff, title, body and labels and writes them to the job summary, and pushes nothing, opens or edits no pull request, and adds no label or comment. The release-PR gate statuses are still posted.

**Source**: workspace RELEASING.md, sections "The receiver" and "Stop switches"; Phase 3 wiring contract (version 3.1) §5 and §9.1.

#### Scenario: Repository code cannot unlock publishing in a dry run
- **WHEN** `CASCADE_DRY_RUN` is not exactly `false`, and the compute job runs repository code (on a release pull request's gates-only run, that head's `task -x deps:cascade`) that forges the job's `action`, `dry-run` and `compute-ok` outputs
- **THEN** the `publish` job does not start, because its `if:` reads `vars.CASCADE_DRY_RUN` itself, and the `cascade-publish` action would refuse to mint because its `dry-run` input is not `false`

### Requirement: The cascade gates workflow runs on pull_request_target with no workflow permissions

`.github/workflows/cascade-gates.yml` SHALL run on `pull_request_target` for `opened`, `reopened` and `synchronize`. It SHALL declare `permissions: {}` at the workflow level.

**Source**: workspace RELEASING.md, section "Gates"; owner decisions 11 and 12 (warn first); Phase 3 wiring contract (version 3.1) §8.3.

#### Scenario: The gates workflow runs on pull request events
- **WHEN** a pull request is opened, reopened or synchronized
- **THEN** `.github/workflows/cascade-gates.yml` runs on `pull_request_target` and declares `permissions: {}` at the workflow level

### Requirement: The cascade gates job calls the shared gates workflow and checks out no pull request code

`.github/workflows/cascade-gates.yml` SHALL have one job that calls the shared `open-platform-model/.github` reusable workflow `cascade-gates.yml` at the repository's pinned `.github` commit with only `statuses: write` and `actions: write`, passes `g2-mode` and `g3-mode` from `CASCADE_G2_MODE` and `CASCADE_G3_MODE` (default `warn`), passes no secrets, and checks out no pull-request code.

**Source**: workspace RELEASING.md, section "Gates"; owner decisions 11 and 12 (warn first); Phase 3 wiring contract (version 3.1) §8.3.

#### Scenario: The gate job checks out no pull request code
- **WHEN** a pull request from a fork into `main` changes `Taskfile.yml`
- **THEN** the gate job runs the workflow file from `main`, checks out nothing from the pull request, and posts `n/a` statuses

### Requirement: Gate statuses are posted on every pull request into main except in two cases

The commit statuses `cascade/freshness` (G2) and `cascade/settled` (G3) SHALL then be posted on every pull request into `main` opened or synchronized after this change. Two cases are excepted: a release pull request whose gates-only evaluation fails before it writes its result in `warn` mode, and a Dependabot pull request whose `pull_request_target` token cannot post statuses (shown possible only in a private sandbox repository). This change does not make either status a required check.

**Source**: workspace RELEASING.md, section "Gates"; owner decisions 11 and 12 (warn first); Phase 3 wiring contract (version 3.1) §8.3.

#### Scenario: A Dependabot pull request whose token cannot post statuses
- **WHEN** a Dependabot pull request's `pull_request_target` token cannot post statuses
- **THEN** that pull request is one of the two excepted cases

### Requirement: An ordinary or fork pull request gets n/a gate statuses

The commit statuses `cascade/freshness` (G2) and `cascade/settled` (G3) SHALL then be posted on every pull request into `main` opened or synchronized after this change: `n/a` success on an ordinary or fork pull request. (Requirement: Gate statuses are posted on every pull request into main except in two cases)

**Source**: workspace RELEASING.md, section "Gates"; owner decisions 11 and 12 (warn first); Phase 3 wiring contract (version 3.1) §8.3.

#### Scenario: Ordinary pull request
- **WHEN** a pull request into `main` whose head branch does not start with `release-please--` is opened
- **THEN** both `cascade/freshness` and `cascade/settled` are posted as success with `n/a: not a release PR`

### Requirement: A same-repository release pull request gets its gate statuses from a gates-only receiver run

The commit statuses `cascade/freshness` (G2) and `cascade/settled` (G3) SHALL then be posted on every pull request into `main` opened or synchronized after this change: on a same-repository release pull request, a gates-only receiver run evaluates them. In `warn` mode a problem is posted as a success whose description starts `WARN:`. (Requirement: Gate statuses are posted on every pull request into main except in two cases)

**Source**: workspace RELEASING.md, section "Gates"; owner decisions 11 and 12 (warn first); Phase 3 wiring contract (version 3.1) §8.3.

#### Scenario: Release pull request with a shipped pin behind
- **WHEN** the release pull request's head moves while a newer library release is published and not yet pinned
- **THEN** a gates-only `Deps cascade` run posts `cascade/freshness` on the new head as success with a `WARN:` description naming the pin

### Requirement: Only the caller-owned cascade jobs read the App key with a fixed job shape

The key SHALL be read only in the caller-owned `notify-downstream` job of `release.yml` or `publish` job of `deps-cascade.yml`, which MUST declare `environment: cascade` and MUST pass `secrets.CASCADE_APP_PRIVATE_KEY` only as the `private-key` input of the SHA-pinned cascade action; that job MUST have no checkout or `run:` of its own, and no `env:`, `container:` or `services:` (the action checks out the repo but never runs it); a reusable call MUST NOT pass `secrets:` or `secrets: inherit`.

**Source**: workspace RELEASING.md, sections "Notify after publish" and "Two-job split"; owner decision 5; Phase 3 wiring contract (version 3.1) §2.2 and §10.1 item 9.

#### Scenario: A pull request adds a step to a key-holding job
- **WHEN** a pull request adds a `run:` step, an `env:` key or a `container:` to `notify-downstream` or `publish`
- **THEN** the "Verify the cascade wiring" step of the required `Lint` job fails and names the job

### Requirement: No other job declares the cascade Environment, release.yml sets no workflow-level env key and the Environment deploys from main only

No other job in the repository SHALL declare `environment: cascade`, and `release.yml` SHALL set no workflow-level `env` key, because workflow-level env reaches the notify action's steps. The `cascade` Environment deploys from `main` only. (Requirement: Only the caller-owned cascade jobs read the App key with a fixed job shape)

**Source**: workspace RELEASING.md, sections "Notify after publish" and "Two-job split"; owner decision 5; Phase 3 wiring contract (version 3.1) §2.2 and §10.1 item 9.

#### Scenario: A branch run cannot reach the key
- **WHEN** a workflow run from a branch other than `main` reaches a job that declares the `cascade` Environment
- **THEN** the Environment's branch policy refuses the job before any step runs and no App token is minted

#### Scenario: Only the key-holding jobs declare the cascade Environment
- **WHEN** a job of the repository declares `environment: cascade`
- **THEN** that job is the `notify-downstream` job of `release.yml` or the `publish` job of `deps-cascade.yml`

### Requirement: The release and receiver workflows name the same full SHA of a .github main commit

Every reference this repository makes to the cascade code SHALL name the same full 40-character SHA of a commit on `open-platform-model/.github` `main`, followed by the comment ` # .github main`, never a branch or a tag: the `cascade-notify` action in `release.yml`, the `cascade-receive.yml` workflow and the `cascade-publish` action in `deps-cascade.yml`.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: All five references carry one SHA
- **WHEN** `task cascade:wiring:check` runs on the repository's workflows
- **THEN** it prints `cascade wiring: ok, .github <SHA> (.github main)` with the one SHA all five references carry

### Requirement: The gates workflow and the resolver checkout name the same full SHA of a .github main commit

Every reference this repository makes to the cascade code SHALL name the same full 40-character SHA of a commit on `open-platform-model/.github` `main`, followed by the comment ` # .github main`, never a branch or a tag: the `cascade-gates.yml` workflow in `cascade-gates.yml`, and the `ref:` of the `open-platform-model/.github` resolver checkout in `cascade-task.yml`.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: One reference moves alone
- **WHEN** a pull request changes the SHA of only `cascade-gates.yml`, or sets the resolver checkout back to `ref: main`
- **THEN** the wiring check fails in the required `Lint` job and the pull request cannot merge

### Requirement: A .github change reaches the cli only through a pin pull request

A `.github` change SHALL reach this repository only through a pull request titled `ci(deps): pin the cascade to .github <first 7 of the SHA>` that moves every reference together, and `.github/dependabot.yml` SHALL ignore `open-platform-model/.github*` for `github-actions`.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: A .github change arrives as one pin pull request
- **WHEN** a `.github` change reaches this repository
- **THEN** it arrives only through a pull request titled `ci(deps): pin the cascade to .github <first 7 of the SHA>` that moves every reference together

### Requirement: The wiring-check script is a byte-identical copy of the pinned file and runs offline

`.tasks/cascade/wiring-check.sh` SHALL be a byte-identical copy of `.github/scripts/cascade/wiring-check.sh` at the pinned SHA, moved only with the pin. `task cascade:wiring:check` SHALL run the same check offline, and the aggregate `task check` SHALL include it.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: The copy is the canonical script
- **WHEN** a reviewer runs `gh api "repos/open-platform-model/.github/contents/.github/scripts/cascade/wiring-check.sh?ref=<SHA>" -H 'Accept: application/vnd.github.raw' | cmp - .tasks/cascade/wiring-check.sh`
- **THEN** `cmp` reports no difference

#### Scenario: The task runs the check offline
- **WHEN** `task cascade:wiring:check` runs
- **THEN** it runs the same check offline, and the aggregate `task check` includes it

### Requirement: The wiring-check values file holds the cli's values

`.tasks/cascade/wiring-check.yaml` SHALL hold the cli's values from the table in `.github` `references/cascade-wiring-check.md`: `pin-comment: .github main`, `receiver: true`, an empty `env-allow`, `publish-workflows` `release.yml`, `publish-fixtures.yml` and `docs.yml`, `ci` `pr.yml`/`lint`, the `notify` `needs`, `if:` and `tag` of `release.yml`'s `notify-downstream` job, and `publish.labels-managed: true`.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: The values file holds the cli's values
- **WHEN** the check reads `.tasks/cascade/wiring-check.yaml`
- **THEN** it holds the cli's values from the table in `.github` `references/cascade-wiring-check.md`: `pin-comment: .github main`, `receiver: true`, an empty `env-allow`, `publish-workflows` `release.yml`, `publish-fixtures.yml` and `docs.yml`, `ci` `pr.yml`/`lint`, the `notify` `needs`, `if:` and `tag` of `release.yml`'s `notify-downstream` job, and `publish.labels-managed: true`

### Requirement: The wiring check asserts the pin, the key rules, the job shapes and the caller shapes

The check of `.tasks/cascade/wiring-check.sh` SHALL assert the pin, the cascade key rule, the release key rule (every job that reads `RELEASE_APP_PRIVATE_KEY` declares exactly `environment: release`, and no other job declares it), the exact shapes of the two key-holding jobs (including `runs-on: ubuntu-latest`), the receiver and gates caller shapes, the empty allow-list of `release.yml` workflow `env` keys, and no Actions cache in a publishing workflow.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: A workflow env key in release.yml
- **WHEN** a pull request adds any workflow-level `env` key to `release.yml`, `BASH_ENV` or `NODE_OPTIONS` included
- **THEN** the wiring check fails, because the cli allows none

#### Scenario: A publishing workflow restores a cache
- **WHEN** a pull request drops `cache: false` from an `actions/setup-go` step in `release.yml` or `publish-fixtures.yml`
- **THEN** the wiring check fails and names the workflow

### Requirement: The required lint job runs the wiring check on every pull request

`pr.yml`'s required `lint` job SHALL run `.tasks/cascade/wiring-check.sh` on every pull request as the step "Verify the cascade wiring" with exactly `env: {GH_TOKEN: ${{ github.token }}}` and `run: bash .tasks/cascade/wiring-check.sh --pin-on-main`, which also confirms through the GitHub API that the SHA is on `.github`'s `main` and that the running copy has exactly the bytes of the canonical file at that SHA.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: A pin that exists only in a fork
- **WHEN** a pull request moves every reference to a SHA that GitHub resolves under `open-platform-model/.github` but that is not on its `main`
- **THEN** the "Verify the cascade wiring" step fails after the shapes match, because the compare status is neither `identical` nor `ahead`

#### Scenario: A copy that drifted from the pinned file
- **WHEN** a pull request edits `.tasks/cascade/wiring-check.sh`, or moves the pin without replacing the copy
- **THEN** the "Verify the cascade wiring" step fails after the compare check with a `differs from` message

### Requirement: Nothing reaches the wiring step from around it

Nothing SHALL reach the step "Verify the cascade wiring" from around it: `pr.yml`'s workflow `env` and the `lint` job's `env` SHALL name only `CUE_*`, `OPM_*`, `REGISTRY` or `IMAGE_NAME` variables, the job SHALL have no `container` or `services`, and every step before the wiring step SHALL be an action from another repository at a full SHA with only `id`, `name`, `uses` and `with`.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: A run step before the wiring step
- **WHEN** a pull request adds a `run:` step to `pr.yml`'s `lint` job above "Verify the cascade wiring", or a `BASH_ENV` or `PATH` key to the workflow's or the job's `env`
- **THEN** the wiring check fails, because such a step or variable could change what the check runs

### Requirement: Every checkout of .github is pinned to a full SHA with fixed keys

Every checkout of `.github` (the `cascade-task.yml` resolver checkout) SHALL be `actions/checkout` at a full SHA with exactly `repository`, `ref`, `path` and `persist-credentials: false`, and no `run:` step SHALL name `open-platform-model/.github`.

**Source**: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `6df4000`, "The wiring check".

#### Scenario: The resolver checkout carries only fixed keys
- **WHEN** the `cascade-task.yml` resolver checks out `.github`
- **THEN** the checkout is `actions/checkout` at a full SHA with exactly `repository`, `ref`, `path` and `persist-credentials: false`, and no `run:` step names `open-platform-model/.github`
