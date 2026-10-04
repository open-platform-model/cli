## ADDED Requirements

### Requirement: The cascade receiver runs on a dispatch, a daily sweep and a manual run
`.github/workflows/deps-cascade.yml` SHALL run on `repository_dispatch` of type `upstream-released`, on a daily schedule at `17 6 * * *` (UTC), and on `workflow_dispatch` with the boolean inputs `dry_run` and `gates_only`, both defaulting to false. It SHALL declare `permissions: {}` at the workflow level. It SHALL have one job that calls the shared `open-platform-model/.github` receive workflow at `main`. That job SHALL grant only `contents: read`, `pull-requests: read` and `statuses: write`, and SHALL pass no secrets. It SHALL pass `setup-go: true`, because the cli's cascade task builds and runs Go. It SHALL pass `labels-managed: true`, because `.github/labels.yml` owns the repository's labels, so the receiver checks that the cascade labels exist and never creates one. It SHALL pass `gates-only` from the `gates_only` input, and `g2-mode` and `g3-mode` from the repository variables `CASCADE_G2_MODE` and `CASCADE_G3_MODE`, each defaulting to `warn`. The receiver computes the cascade diff with this repository's `task -x deps:cascade`. Source: workspace RELEASING.md, sections "The receiver", "Labels" and "Gates".

#### Scenario: An upstream release starts the receiver
- **WHEN** library, opm-operator or catalog_opm publishes a release and its notify job dispatches `upstream-released` to the cli
- **THEN** a `Deps cascade` run starts on `main` and runs the shared receiver against this repository

#### Scenario: The daily sweep runs without a dispatch
- **WHEN** no dispatch arrived and the schedule fires
- **THEN** a `Deps cascade` run starts and resolves the newest published upstreams itself

#### Scenario: The receiver never creates a label in the cli
- **WHEN** a live receiver run publishes a cascade pull request
- **THEN** it checks that the cascade labels exist and creates none, and a label missing from `.github/labels.yml` fails the run with a message to declare it there

### Requirement: The receiver is a dry run unless CASCADE_DRY_RUN is exactly false
The caller SHALL pass `dry-run` as true when the `dry_run` input is true, or when the repository variable `CASCADE_DRY_RUN` is anything other than exactly `false`, including unset. In a dry run the receiver computes the diff, title, body and labels and writes them to the job summary, and pushes nothing, opens or edits no pull request, and adds no label or comment. The release-PR gate statuses are still posted. A run from any ref other than `main` is always a dry run. Source: workspace RELEASING.md, sections "The receiver" and "Stop switches".

#### Scenario: Variable unset
- **WHEN** `CASCADE_DRY_RUN` is not set and an upstream dispatch arrives
- **THEN** the run computes and summarises the diff and pushes nothing

#### Scenario: Variable set to true
- **WHEN** `CASCADE_DRY_RUN` is `true`
- **THEN** every receiver run is a dry run

#### Scenario: Manual dry run while live
- **WHEN** `CASCADE_DRY_RUN` is `false` and a human runs the workflow by hand with `dry_run` true
- **THEN** that run computes and summarises the diff and pushes nothing

#### Scenario: Live only at exactly false
- **WHEN** `CASCADE_DRY_RUN` is exactly `false` and a dispatch arrives on `main`
- **THEN** the receiver may push `deps/cascade` and open or update the cascade pull request

### Requirement: Real receiver runs share one concurrency group without cancelling
Receiver runs on `main` that are not gates-only SHALL share the concurrency group `deps-cascade`, with `cancel-in-progress: false`. Gates-only runs on `main` SHALL use the group `deps-cascade-gates`. Runs from any other ref SHALL use `deps-cascade-<ref>`. A gates-only run or a branch run therefore never replaces a pending real run. Source: workspace RELEASING.md, section "Concurrency".

#### Scenario: A burst of upstream releases
- **WHEN** three dispatches arrive while a real run is active
- **THEN** one run stays active, at most one is pending, and none is cancelled mid-run

#### Scenario: A gates-only run during a pending real run
- **WHEN** a release pull request's head moves while a real run is pending
- **THEN** the gates-only run starts in `deps-cascade-gates` and the pending real run is not replaced

### Requirement: Every pull request carries the cascade gate statuses
`.github/workflows/cascade-gates.yml` SHALL run on `pull_request_target` for `opened`, `reopened` and `synchronize`. It SHALL declare `permissions: {}` at the workflow level. It SHALL have one job that calls the shared `open-platform-model/.github` gates workflow at `main` with only `statuses: write` and `actions: write`, passes `g2-mode` and `g3-mode` from `CASCADE_G2_MODE` and `CASCADE_G3_MODE` (default `warn`), passes no secrets, and checks out no pull-request code. The commit statuses `cascade/freshness` (G2) and `cascade/settled` (G3) SHALL then be posted on every pull request opened or synchronized after this change, except a release pull request whose gates-only evaluation fails in `warn` mode before it writes its result: `n/a` success on an ordinary or fork pull request; on a same-repository release pull request, a gates-only receiver run evaluates them. In `warn` mode a problem is posted as a success whose description starts `WARN:`. This change does not make either status a required check. Source: workspace RELEASING.md, section "Gates"; owner decisions 11 and 12 (warn first).

#### Scenario: Ordinary pull request
- **WHEN** a pull request whose head branch does not start with `release-please--` is opened
- **THEN** both `cascade/freshness` and `cascade/settled` are posted as success with `n/a: not a release PR`

#### Scenario: Release pull request with a shipped pin behind
- **WHEN** the release pull request's head moves while a newer library release is published and not yet pinned
- **THEN** a gates-only `Deps cascade` run posts `cascade/freshness` on the new head as success with a `WARN:` description naming the pin

#### Scenario: The gate job checks out no pull request code
- **WHEN** a pull request from a fork changes `Taskfile.yml`
- **THEN** the gate job runs the workflow from `main`, checks out nothing from the pull request, and posts `n/a` statuses

#### Scenario: A release head's code cannot unlock publishing
- **WHEN** a same-repository release pull request's head carries a modified `Taskfile.yml`, and its gates-only receiver run executes that head's `task -x deps:cascade` in the read-only compute job
- **THEN** nothing is pushed and no pull request is opened, because the shared receiver's publish job reads the `dry-run` and `gates-only` inputs and the ref directly and never trusts the compute job's outputs to unlock it

### Requirement: The shared cascade workflows are called at main with no secret passed
Every caller of the shared cascade workflows in this repository SHALL reference `open-platform-model/.github/.github/workflows/<name>.yml@main` and SHALL NOT pass `secrets:` or `secrets: inherit`. No job in this repository SHALL declare `environment: cascade` or read `CASCADE_APP_PRIVATE_KEY`. The App key is read only by the shared workflows' own jobs that declare the `cascade` Environment, which deploys from `main` only. Source: workspace RELEASING.md, sections "Notify after publish" and "Two-job split"; owner decisions 5 and 13.

#### Scenario: Caller files reference main
- **WHEN** `release.yml`, `deps-cascade.yml` and `cascade-gates.yml` are read
- **THEN** each shared-workflow `uses:` ends in `@main` and none of those jobs carries `secrets:` or `environment:`

#### Scenario: Branch run cannot reach the key
- **WHEN** a workflow from a branch other than `main` calls a shared workflow job that declares the `cascade` Environment
- **THEN** the Environment's branch policy refuses the job and no App token is minted
