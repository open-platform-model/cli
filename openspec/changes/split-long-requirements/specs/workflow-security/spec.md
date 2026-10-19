## REMOVED Requirements

### Requirement: Every workflow declares least-privilege permissions
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Every workflow file declares `permissions` at workflow level and each job grants exactly the scopes its steps use", "`release.yml` declares `permissions: {}` and grants each job the scopes it lists".

### Requirement: Publishing jobs run only from main and only on a tag
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The publish-templates job and the publish job of `publish-fixtures.yml` run only from main", "publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`".

## ADDED Requirements

### Requirement: Every workflow file declares `permissions` at workflow level and each job grants exactly the scopes its steps use

Every file under `.github/workflows/` SHALL declare `permissions` at workflow level, either `{}` or read-only scopes, and every job that needs more SHALL grant exactly the scopes its steps use, so the repository's default `GITHUB_TOKEN` can be read-only without breaking a job.

**Source**: owner decision 30 (2026-10-04); security pass finding GOV-3.

#### Scenario: The default token turns read-only

- **WHEN** the repository's default workflow permissions are set to read
- **THEN** every workflow runs with the scopes it declares, and no job loses a scope it uses

#### Scenario: A workflow without a permissions key

- **WHEN** a reviewer reads any workflow file
- **THEN** it has a top-level `permissions` key

### Requirement: `release.yml` declares `permissions: {}` and grants each job the scopes it lists

`release.yml` SHALL declare `permissions: {}` at workflow level, and its jobs SHALL grant: release-please none; goreleaser `contents: write` and `packages: write`; publish-templates `contents: read` and `packages: write`; publish-docs `contents: read`, `packages: write` and `id-token: write`; notify-downstream `contents: read`.

**Source**: owner decision 30 (2026-10-04); security pass finding GOV-3.

#### Scenario: A release job holds only its listed scopes

- **WHEN** a reviewer reads the `permissions` of `release.yml`
- **THEN** the workflow level SHALL be `permissions: {}` and its jobs SHALL grant: release-please none; goreleaser `contents: write` and `packages: write`; publish-templates `contents: read` and `packages: write`; publish-docs `contents: read`, `packages: write` and `id-token: write`; notify-downstream `contents: read`

### Requirement: The publish-templates job and the publish job of `publish-fixtures.yml` run only from main

The publish-templates job of `release.yml` and the publish job of `publish-fixtures.yml` SHALL run only when `github.ref` is `refs/heads/main`, also on `workflow_dispatch`, so a dispatch from another branch cannot publish with `packages: write`.

**Source**: security pass review of cli PR 306.

#### Scenario: A manual release run from a branch

- **WHEN** a maintainer dispatches `release.yml` from a branch other than `main` with a draft's tag
- **THEN** publish-templates is skipped, and goreleaser keeps the release a draft because the templates did not publish

### Requirement: publish-templates and goreleaser check out the release tag as `refs/tags/<tag>`

publish-templates and goreleaser SHALL check out the release tag as `refs/tags/<tag>`, so a branch name in the `tag` input fails the checkout. These guards stop a mistaken or bot-branch dispatch; a person with write access can still change the workflow on a branch, which code-owner review and the `release` Environment address.
(Requirement: The publish-templates job and the publish job of `publish-fixtures.yml` run only from main)

**Source**: security pass review of cli PR 306.

#### Scenario: A branch name as the tag input

- **WHEN** `release.yml` is dispatched from `main` with `tag` set to `deps/cascade`
- **THEN** the publish-templates checkout of `refs/tags/deps/cascade` fails and nothing is published
