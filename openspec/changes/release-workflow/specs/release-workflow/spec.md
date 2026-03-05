## ADDED Requirements

### Requirement: Release triggers on version tags
The release workflow SHALL trigger when a tag matching `v*` is pushed to the repository.

#### Scenario: Version tag pushed
- **WHEN** a tag like `v0.1.0` is pushed
- **THEN** the release workflow starts

#### Scenario: Non-version tag ignored
- **WHEN** a tag not matching `v*` is pushed
- **THEN** the release workflow does not trigger

### Requirement: Full test suite runs before release
The release workflow SHALL run all test tiers (lint, unit, registry, integration, e2e) in a `test` job before the release job runs.

#### Scenario: Tests pass, release proceeds
- **WHEN** all tests pass in the test job
- **THEN** the release job runs goreleaser

#### Scenario: Tests fail, release is blocked
- **WHEN** any test in the test job fails
- **THEN** the release job does not run and no GitHub Release is created

### Requirement: Goreleaser builds raw binaries for five platforms
The release job SHALL use goreleaser to produce raw binary executables (not tarballs) for: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.

#### Scenario: Binaries produced
- **WHEN** goreleaser runs on a `v*` tag
- **THEN** five binaries are attached to the GitHub Release: `opm-linux-amd64`, `opm-linux-arm64`, `opm-darwin-amd64`, `opm-darwin-arm64`, `opm-windows-amd64.exe`

### Requirement: Checksums file is published
The release job SHALL produce a `checksums.txt` file containing SHA256 digests of all release binaries and attach it to the GitHub Release.

#### Scenario: Checksums attached
- **WHEN** goreleaser completes
- **THEN** `checksums.txt` is present in the GitHub Release assets

### Requirement: Changelog is auto-generated from conventional commits
The release job SHALL generate a changelog from git history since the previous tag, grouping entries by conventional commit type (feat, fix, perf, refactor). Commits with types docs, test, ci, chore SHALL be excluded from the changelog.

#### Scenario: Changelog in release notes
- **WHEN** a GitHub Release is created
- **THEN** the release notes contain grouped sections for Features, Bug Fixes, Performance, and Refactoring based on commit messages since the last tag

### Requirement: Version ldflags are injected at build time
The goreleaser build SHALL inject `Version`, `GitCommit`, and `BuildDate` via ldflags matching the variables in `internal/version/version.go`.

#### Scenario: Version command reflects release tag
- **WHEN** a released binary runs `opm version`
- **THEN** the output shows the tag version, commit SHA, and build date

### Requirement: Full git history is available during release
The release job SHALL perform a full git checkout (`fetch-depth: 0`) to enable accurate changelog generation across all tags.

#### Scenario: Changelog spans multiple releases
- **WHEN** goreleaser generates the changelog
- **THEN** it compares against the previous tag correctly using full git history

### Requirement: Workflow targets self-hosted runner
The release workflow SHALL specify `runs-on: self-hosted` for all jobs.

#### Scenario: Self-hosted runner assignment
- **WHEN** the release workflow triggers
- **THEN** all jobs are assigned to the self-hosted runner pool

### Requirement: Workflow is initially disabled
The release workflow SHALL use only `workflow_dispatch` as an active trigger. Tag push triggers SHALL be present but commented out.

#### Scenario: Manual trigger works
- **WHEN** a user manually dispatches the release workflow
- **THEN** tests run and goreleaser executes

#### Scenario: Tag push does not trigger workflow automatically
- **WHEN** a version tag is pushed before the self-hosted runner is available
- **THEN** the workflow does not run automatically
