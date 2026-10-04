## ADDED Requirements

### Requirement: Dependabot leaves the shared cascade workflows at main
The Dependabot `github-actions` update SHALL ignore every dependency named `open-platform-model/.github*`. The repository's workflows call the shared release-cascade workflows of `open-platform-model/.github` at `main`, and the `.github` ruleset protects that branch. They SHALL NOT be pinned to a tag or a commit by a Dependabot pull request. Other GitHub Actions SHALL keep their weekly updates. Source: workspace RELEASING.md, section "repository_dispatch" ("called at `@main`").

#### Scenario: A newer commit on the shared workflows
- **WHEN** `open-platform-model/.github` `main` moves and Dependabot's weekly `github-actions` run executes
- **THEN** Dependabot opens no pull request for the shared cascade workflow references

#### Scenario: Third-party action update
- **WHEN** a newer release of a SHA-pinned third-party action is published
- **THEN** Dependabot still proposes the bump
