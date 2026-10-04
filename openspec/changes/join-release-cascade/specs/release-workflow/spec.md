## ADDED Requirements

### Requirement: Downstream repos are notified after a release is published
`.github/workflows/release.yml` SHALL have a `notify-downstream` job that calls the shared `open-platform-model/.github` notify workflow at `main`, with the release tag as its only input. The tag is `tag_name` from release-please, or the `tag` input on a manual run. The job SHALL `need` release-please and goreleaser. It SHALL run only when goreleaser succeeded, the run's ref is `main`, and the repository variable `CASCADE_NOTIFY` is not `off`. It SHALL run on a release run and on a manual recovery run alike, because release-please is skipped on the manual run. It SHALL NOT depend on the docs bundle job. It SHALL grant only `contents: read`, and SHALL pass no secrets. The dispatch token is minted inside the shared workflow from the repository's `cascade` Environment. The shared workflow, not this repository, chooses the targets: catalog_opm and opm-operator, as release-tool edges. Source: workspace RELEASING.md, sections "Notify after publish" and "Stop switches".

#### Scenario: Release run notifies after publishing
- **WHEN** a push to `main` cuts a release, and publish-templates and goreleaser succeed
- **THEN** the notify job runs after goreleaser with the release's tag, and catalog_opm and opm-operator each receive one `upstream-released` dispatch naming the cli and that tag

#### Scenario: A release left a draft notifies nothing
- **WHEN** publish-templates fails, so goreleaser fails and the release stays a draft
- **THEN** the notify job does not run

#### Scenario: Manual recovery that finishes a draft notifies once
- **WHEN** the workflow is run by hand on `main` with `tag` set to a draft release, and goreleaser publishes it
- **THEN** the notify job runs with that tag

#### Scenario: Manual run on a published release does not notify again
- **WHEN** the workflow is run by hand with `tag` set to a release that is already published
- **THEN** goreleaser fails its draft check and the notify job does not run

#### Scenario: Manual run from another branch does not notify
- **WHEN** the workflow is run by hand from a branch other than `main`
- **THEN** the notify job does not run

#### Scenario: A push without a release does not notify
- **WHEN** a push to `main` cuts no release
- **THEN** goreleaser and the notify job do not run

#### Scenario: Notify switched off
- **WHEN** the repository variable `CASCADE_NOTIFY` is `off` and a release is published
- **THEN** the notify job is skipped and no dispatch is sent

#### Scenario: A failed docs bundle does not suppress the notify
- **WHEN** goreleaser publishes the release and the docs bundle job fails
- **THEN** the notify job still runs
