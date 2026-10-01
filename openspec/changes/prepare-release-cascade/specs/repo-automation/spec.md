## Purpose

Repository automation files that other tools act on: the label sync, which must keep the labels release-please, Dependabot and the release cascade rely on, and Dependabot, which must leave OPM-owned Go modules to the release cascade.

## ADDED Requirements

### Requirement: The label sync keeps bot-managed and cascade labels
`.github/labels.yml` SHALL list every label that a tool applies to this repository's pull requests, because the label sync deletes any repository label the file does not list. It SHALL list the release-please labels `autorelease: pending` and `autorelease: tagged`, the Dependabot labels `dependencies`, `go` and `github_actions`, and the release-cascade labels `deps-cascade`, `deps-cascade:conflict`, `deps-cascade:hold`, `deps-cascade:breaking`, `need-human-review` and `e2e-verified`. Bot-managed labels SHALL keep the color and description they carry in the repository today, so a sync changes nothing about them. Source: workspace RELEASING.md, section "Cascade files".

#### Scenario: Sync after a labels.yml change deletes no bot label
- **WHEN** a change to `.github/labels.yml` merges and the label sync runs with deletion enabled
- **THEN** `autorelease: pending`, `autorelease: tagged`, `dependencies`, `go` and `github_actions` still exist with unchanged color and description

#### Scenario: Cascade labels exist before the cascade runs
- **WHEN** the label sync has run on `main`
- **THEN** each of the six release-cascade labels exists in the repository

#### Scenario: Pull-request dry run shows no deletion
- **WHEN** a pull request edits `.github/labels.yml`
- **THEN** the label sync's dry run lists no deletion of a label any tool applies

### Requirement: Dependabot leaves OPM Go modules to the release cascade
The Dependabot `gomod` update SHALL ignore every dependency named `github.com/open-platform-model/*`, so library pins move only through the release cascade or by hand as `fix(deps)`. Other Go modules and GitHub Actions SHALL keep their weekly updates. Source: workspace RELEASING.md, section "Cascade files".

#### Scenario: New library release
- **WHEN** a new `github.com/open-platform-model/library` version is tagged and Dependabot's weekly `gomod` run executes
- **THEN** Dependabot opens no pull request for library

#### Scenario: Third-party module update
- **WHEN** a newer `golang.org/x/term` is published
- **THEN** Dependabot still proposes the bump
