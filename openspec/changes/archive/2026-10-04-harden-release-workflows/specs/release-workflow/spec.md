## ADDED Requirements

### Requirement: Only the release-please job reads the release App key, inside the release Environment

The `release-please` job of `release.yml` SHALL be the only job in the repository that reads `secrets.RELEASE_APP_PRIVATE_KEY`, and SHALL read it only as the `private-key` input of `actions/create-github-app-token`. That job SHALL declare `environment: release`, the Environment whose deployment branch policy allows `main` only, and SHALL run only when the event is a `push` and the ref is `refs/heads/main`. It SHALL grant the `GITHUB_TOKEN` no permission (`permissions: {}`), because it acts only through the App token. No other job SHALL declare the `release` Environment. Source: owner decision 29 (2026-10-04); security pass finding GOV-2.

#### Scenario: A push to main mints the token inside the Environment

- **WHEN** a commit lands on `main`
- **THEN** the release-please job runs as a deployment to the `release` Environment and mints the App token there

#### Scenario: A manual run does not reach the key

- **WHEN** the workflow is run by hand with a `tag`
- **THEN** the release-please job is skipped and no job reads the release App key
