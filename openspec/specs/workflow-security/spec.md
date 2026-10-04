# workflow-security Specification

## Purpose
Repository-wide rules for GitHub Actions tokens and caches, so the repository's default token can be read-only and no publishing job trusts what another run wrote: explicit permissions in every workflow, no Actions cache in a publishing job, no write token on a pull-request event, and code owners for the release and CI paths.

## Requirements

### Requirement: Every workflow declares least-privilege permissions

Every file under `.github/workflows/` SHALL declare `permissions` at workflow level, either `{}` or read-only scopes, and every job that needs more SHALL grant exactly the scopes its steps use, so the repository's default `GITHUB_TOKEN` can be read-only without breaking a job. `release.yml` SHALL declare `permissions: {}` at workflow level, and its jobs SHALL grant: release-please none; goreleaser `contents: write` and `packages: write`; publish-templates `contents: read` and `packages: write`; publish-docs `contents: read`, `packages: write` and `id-token: write`; notify-downstream `contents: read`. Source: owner decision 30 (2026-10-04); security pass finding GOV-3.

#### Scenario: The default token turns read-only

- **WHEN** the repository's default workflow permissions are set to read
- **THEN** every workflow runs with the scopes it declares, and no job loses a scope it uses

#### Scenario: A workflow without a permissions key

- **WHEN** a reviewer reads any workflow file
- **THEN** it has a top-level `permissions` key

### Requirement: Publishing jobs restore no Actions cache

A job that holds `packages: write` or `contents: write` and builds or publishes an artifact SHALL NOT restore or save a GitHub Actions cache: every `actions/setup-go` step in it SHALL set `cache: false`, and it SHALL use no `actions/cache` step and no `type=gha` build cache. This covers the goreleaser and publish-templates jobs of `release.yml` and the publish job of `publish-fixtures.yml`. Source: security pass finding CAS-R3.

#### Scenario: A main-ref run wrote a cache entry

- **WHEN** another run on `main` saved a Go cache entry under the key the release job would use
- **THEN** the release job does not restore it and downloads its modules from the module proxy

### Requirement: No write token on a pull-request event

A workflow triggered by `pull_request` SHALL NOT grant a write scope to a job that runs on that event. `labels.yml` SHALL run its pull-request dry run in a `check` job with `contents: read` and `issues: read`, and its sync in a `sync` job, with `issues: write`, that runs only on a push to `main`. Source: security pass supervisor plan, wave 1 item 4.

#### Scenario: A pull request edits the label file

- **WHEN** a pull request changes `.github/labels.yml`
- **THEN** only the `check` job runs, with a token that cannot change a label, and its dry run lists the changes

#### Scenario: The label file changes on main

- **WHEN** a change to `.github/labels.yml` lands on `main`
- **THEN** only the `sync` job runs and applies it

### Requirement: Code owners review the release and CI paths

`.github/CODEOWNERS` SHALL name the maintainers as owners of `/.github/`, `/.tasks/`, `/Taskfile*.yml`, `/release-please-config.json`, `/.release-please-manifest.json`, `/.cascade-frozen`, `/hack/` and `/.goreleaser.yml`, so that, with the `main` ruleset's code-owner review, a change to the workflows, the cascade task, the release configuration, or code a write-token job runs needs a maintainer's approval. Source: owner decision 28 (2026-10-04); security pass finding GOV-1.

#### Scenario: A pull request edits the cascade task

- **WHEN** a pull request changes `.tasks/cascade/cascade.sh`
- **THEN** GitHub requests a review from the code owners

### Requirement: Publishing jobs run only from main and only on a tag

The publish-templates job of `release.yml` and the publish job of `publish-fixtures.yml` SHALL run only when `github.ref` is `refs/heads/main`, also on `workflow_dispatch`, so a dispatch from another branch cannot publish with `packages: write`. publish-templates and goreleaser SHALL check out the release tag as `refs/tags/<tag>`, so a branch name in the `tag` input fails the checkout. These guards stop a mistaken or bot-branch dispatch; a person with write access can still change the workflow on a branch, which code-owner review and the `release` Environment address. Source: security pass review of cli PR 306.

#### Scenario: A manual release run from a branch

- **WHEN** a maintainer dispatches `release.yml` from a branch other than `main` with a draft's tag
- **THEN** publish-templates is skipped, and goreleaser keeps the release a draft because the templates did not publish

#### Scenario: A branch name as the tag input

- **WHEN** `release.yml` is dispatched from `main` with `tag` set to `deps/cascade`
- **THEN** the publish-templates checkout of `refs/tags/deps/cascade` fails and nothing is published
