## MODIFIED Requirements

### Requirement: Every cascade reference is pinned to one .github main commit and checked in CI
Every reference this repository makes to the cascade code SHALL name the same full 40-character SHA of a commit on `open-platform-model/.github` `main`, followed by the comment ` # .github main`, never a branch or a tag: the `cascade-notify` action in `release.yml`, the `cascade-receive.yml` workflow and the `cascade-publish` action in `deps-cascade.yml`, the `cascade-gates.yml` workflow in `cascade-gates.yml`, and the `ref:` of the `open-platform-model/.github` resolver checkout in `cascade-task.yml`. A `.github` change SHALL reach this repository only through a pull request titled `ci(deps): pin the cascade to .github <first 7 of the SHA>` that moves every reference together, and `.github/dependabot.yml` SHALL ignore `open-platform-model/.github*` for `github-actions`.

`.tasks/cascade/wiring-check.sh` SHALL be a byte-identical copy of `.github/scripts/cascade/wiring-check.sh` at the pinned SHA, moved only with the pin, and `.tasks/cascade/wiring-check.yaml` SHALL hold the cli's values from the `.github` README table: `pin-comment: .github main`, `receiver: true`, an empty `env-allow`, `publish-workflows` `release.yml`, `publish-fixtures.yml` and `docs.yml`, `ci` `pr.yml`/`lint`, the `notify` `needs`, `if:` and `tag` of `release.yml`'s `notify-downstream` job, and `publish.labels-managed: true`. The check SHALL assert the pin, the cascade key rule, the release key rule (every job that reads `RELEASE_APP_PRIVATE_KEY` declares exactly `environment: release`, and no other job declares it), the exact shapes of the two key-holding jobs (including `runs-on: ubuntu-latest`), the receiver and gates caller shapes, the empty allow-list of `release.yml` workflow `env` keys, and no Actions cache in a publishing workflow. `pr.yml`'s required `lint` job SHALL run it on every pull request as the step "Verify the cascade wiring" with exactly `env: {GH_TOKEN: ${{ github.token }}}` and `run: bash .tasks/cascade/wiring-check.sh --pin-on-main`, which also confirms through the GitHub API that the SHA is on `.github`'s `main` and that the running copy has exactly the bytes of the canonical file at that SHA. Nothing SHALL reach that step from around it: `pr.yml`'s workflow `env` and the `lint` job's `env` SHALL name only `CUE_*`, `OPM_*`, `REGISTRY` or `IMAGE_NAME` variables, the job SHALL have no `container` or `services`, and every step before the wiring step SHALL be an action from another repository at a full SHA with only `id`, `name`, `uses` and `with`. Every checkout of `.github` (the `cascade-task.yml` resolver checkout) SHALL be `actions/checkout` at a full SHA with exactly `repository`, `ref`, `path` and `persist-credentials: false`, and no `run:` step SHALL name `open-platform-model/.github`. `task cascade:wiring:check` SHALL run the same check offline, and the aggregate `task check` SHALL include it. Source: owner decision 24 and the supervisor's extension to the reusable workflows and the resolver checkout; owner decision 29; workspace RELEASING.md, sections "Pinning the cascade code", "Two-job split" and "Moving the cascade pin"; Phase 3 wiring contract (version 3.1) §2.4 and §10.1 items 2, 3, 6 and 7; `open-platform-model/.github` README at `0f9c6ac`, "The wiring check".

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

#### Scenario: A copy that drifted from the pinned file
- **WHEN** a pull request edits `.tasks/cascade/wiring-check.sh`, or moves the pin without replacing the copy
- **THEN** the "Verify the cascade wiring" step fails after the compare check with a `differs from` message

#### Scenario: A run step before the wiring step
- **WHEN** a pull request adds a `run:` step to `pr.yml`'s `lint` job above "Verify the cascade wiring", or a `BASH_ENV` or `PATH` key to the workflow's or the job's `env`
- **THEN** the wiring check fails, because such a step or variable could change what the check runs
