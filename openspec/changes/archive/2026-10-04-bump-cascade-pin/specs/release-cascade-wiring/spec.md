## MODIFIED Requirements

### Requirement: The receiver publishes only through its own cascade Environment job
`deps-cascade.yml` SHALL have a second job, `publish`, that needs `cascade`, runs on `ubuntu-latest`, declares `environment: cascade`, has a 15-minute timeout, grants only `contents: read` and `pull-requests: read`, and has exactly one step: the shared `cascade-publish` action at the repository's pinned `.github` commit, with the inputs `dry-run` (the same expression the `cascade` job passes), `gates-only: ${{ inputs.gates_only == true }}`, `labels-managed: true`, `client-id: ${{ vars.CASCADE_APP_CLIENT_ID }}` and `private-key: ${{ secrets.CASCADE_APP_PRIVATE_KEY }}`. Its `if:` SHALL be the `.github` README's receiver `if:` byte for byte, which reads `inputs.dry_run`, `inputs.gates_only`, `vars.CASCADE_DRY_RUN` and `github.ref` itself and uses the `cascade` job's outputs only as an extra filter, so a gates-only run never starts a `cascade` Environment job. Because `.github/labels.yml` owns the repository's labels, publish SHALL check that the cascade labels exist and SHALL never create one. Source: workspace RELEASING.md, sections "Two-job split" and "Labels"; Phase 3 wiring contract (version 3.1) §5, §5.2 and §6.4; `open-platform-model/.github` README at `7b9ad1b`, "Receiver caller" (security pass 2026-10-04).

#### Scenario: A live run publishes from the caller's own job
- **WHEN** `CASCADE_DRY_RUN` is exactly `false`, a run on `main` computes a change, and its compute job succeeds
- **THEN** the `publish` job starts in the `cascade` Environment and the pinned `cascade-publish` action verifies the plan, mints the App token and updates `deps/cascade` and its pull request

#### Scenario: The receiver never creates a label in the cli
- **WHEN** a live receiver run publishes a cascade pull request
- **THEN** it checks that the cascade labels exist and creates none, and a label missing from `.github/labels.yml` fails the run with a message to declare it there

#### Scenario: A gates-only run never publishes
- **WHEN** `CASCADE_DRY_RUN` is exactly `false` and `Cascade gates` dispatches a run with `gates_only` true, whose compute runs an open release head's task and forges `action: push` and `compute-ok: true`
- **THEN** the `publish` job does not start, because its `if:` reads `inputs.gates_only` itself, and the `cascade-publish` action would fail before minting because its `gates-only` input is not `false`

### Requirement: Every cascade reference is pinned to one .github main commit and checked in CI
Every reference this repository makes to the cascade code SHALL name the same full 40-character SHA of a commit on `open-platform-model/.github` `main`, followed by the comment ` # .github main`, never a branch or a tag: the `cascade-notify` action in `release.yml`, the `cascade-receive.yml` workflow and the `cascade-publish` action in `deps-cascade.yml`, the `cascade-gates.yml` workflow in `cascade-gates.yml`, and the `ref:` of the `open-platform-model/.github` resolver checkout in `cascade-task.yml`. A `.github` change SHALL reach this repository only through a pull request titled `ci(deps): pin the cascade to .github <first 7 of the SHA>` that moves every reference together, and `.github/dependabot.yml` SHALL ignore `open-platform-model/.github*` for `github-actions`.

`.tasks/cascade/wiring-check.sh` SHALL be a byte-identical copy of `.github/scripts/cascade/wiring-check.sh` at the pinned SHA, moved only with the pin, and `.tasks/cascade/wiring-check.yaml` SHALL hold the cli's values from the `.github` README table: `pin-comment: .github main`, `receiver: true`, an empty `env-allow`, `publish-workflows` `release.yml`, `publish-fixtures.yml` and `docs.yml`, `ci` `pr.yml`/`lint`, the `notify` `needs`, `if:` and `tag` of `release.yml`'s `notify-downstream` job, and `publish.labels-managed: true`. The check SHALL assert the pin, the cascade key rule, the release key rule (every job that reads `RELEASE_APP_PRIVATE_KEY` declares exactly `environment: release`, and no other job declares it), the exact shapes of the two key-holding jobs (including `runs-on: ubuntu-latest`), the receiver and gates caller shapes, the empty allow-list of `release.yml` workflow `env` keys, and no Actions cache in a publishing workflow. `pr.yml`'s required `lint` job SHALL run it on every pull request as the step "Verify the cascade wiring" with exactly `env: {GH_TOKEN: ${{ github.token }}}` and `run: bash .tasks/cascade/wiring-check.sh --pin-on-main`, which also confirms through the GitHub API that the SHA is on `.github`'s `main`. `task cascade:wiring:check` SHALL run the same check offline, and the aggregate `task check` SHALL include it. Source: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `7b9ad1b`, "The wiring check".

#### Scenario: All five references carry one SHA
- **WHEN** `task cascade:wiring:check` runs on the repository's workflows
- **THEN** it prints `cascade wiring: ok, .github <SHA> (.github main)` with the one SHA all five references carry

#### Scenario: One reference moves alone
- **WHEN** a pull request changes the SHA of only `cascade-gates.yml`, or sets the resolver checkout back to `ref: main`
- **THEN** the wiring check fails in the required `Lint` job and the pull request cannot merge

#### Scenario: A workflow env key in release.yml
- **WHEN** a pull request adds any workflow-level `env` key to `release.yml`, `BASH_ENV` or `NODE_OPTIONS` included
- **THEN** the wiring check fails, because the cli allows none

#### Scenario: A pin that exists only in a fork
- **WHEN** a pull request moves every reference to a SHA that GitHub resolves under `open-platform-model/.github` but that is not on its `main`
- **THEN** the "Verify the cascade wiring" step fails after the shapes match, because the compare status is neither `identical` nor `ahead`

#### Scenario: The copy is the canonical script
- **WHEN** a reviewer runs `gh api "repos/open-platform-model/.github/contents/.github/scripts/cascade/wiring-check.sh?ref=<SHA>" -H 'Accept: application/vnd.github.raw' | cmp - .tasks/cascade/wiring-check.sh`
- **THEN** `cmp` reports no difference

#### Scenario: A publishing workflow restores a cache
- **WHEN** a pull request drops `cache: false` from an `actions/setup-go` step in `release.yml` or `publish-fixtures.yml`
- **THEN** the wiring check fails and names the workflow
