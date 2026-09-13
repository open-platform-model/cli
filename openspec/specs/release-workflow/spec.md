# Capability: release-workflow

## Purpose

`.github/workflows/release.yml` turns merges to main into published releases: release-please maintains the version and changelog and opens the release PR, and once it cuts a tag, goreleaser builds and attaches the cross-platform archives and `checksums.txt` described by `.goreleaser.yml` while a second job publishes the bundled template modules to GHCR.

## Requirements

### Requirement: Release runs from pushes to main through release-please
The release workflow SHALL trigger on `push` to `main` only (no tag trigger, no `workflow_dispatch`) and SHALL run the `release-please` job first, driven by `release-please-config.json` and `.release-please-manifest.json`: a Go package named `opm` on a `v`-prefixed, `alpha` prerelease version line with `CHANGELOG.md` as the changelog. The job SHALL expose `releases_created` and `tag_name` as outputs for the downstream jobs.

#### Scenario: Push without a merged release PR
- **WHEN** a commit that is not a release PR merge lands on main
- **THEN** release-please updates or opens the release PR and the goreleaser and publish-templates jobs are skipped

#### Scenario: Release PR merged
- **WHEN** the release PR merges to main
- **THEN** release-please creates the tag and the GitHub Release and reports `releases_created == 'true'` with the new `tag_name`

### Requirement: Binaries publish only when a release was created
The `goreleaser` job SHALL `need` release-please, run only when `releases_created == 'true'`, check out `tag_name` with `fetch-depth: 0`, set up Go 1.26.0, and run `goreleaser release --clean` with `contents: write` and the repository `GITHUB_TOKEN`.

#### Scenario: Skipped on a non-release push
- **WHEN** release-please reports no release created
- **THEN** the goreleaser job does not run and no assets are attached

#### Scenario: Runs on the release tag
- **WHEN** release-please reports a release created
- **THEN** goreleaser builds from the tagged commit with full git history available

### Requirement: Goreleaser produces per-platform archives, checksums and changelog
Goreleaser SHALL build `opm` for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and windows/amd64 (windows/arm64 excluded), package each as an archive named `opm-<os>-<arch>` bundling `LICENSE`, and attach the archives, `checksums.txt` (SHA256 digests) and `LICENSE` to the GitHub Release. Its changelog SHALL group commits into Features, Bug Fixes, Performance, Refactoring and Other, excluding subjects starting with `docs:`, `test:`, `ci:` or `chore:`.

#### Scenario: Five archives and checksums attached
- **WHEN** goreleaser completes
- **THEN** the GitHub Release assets include the five `opm-<os>-<arch>` archives, `checksums.txt` and `LICENSE`

#### Scenario: Changelog grouped by type
- **WHEN** goreleaser generates its changelog
- **THEN** entries are grouped by conventional commit type with the excluded types absent

### Requirement: Version ldflags are injected at build time
The goreleaser build SHALL inject `Version`, `GitCommit`, and `BuildDate` via ldflags matching the variables in `internal/version/version.go`.

#### Scenario: Version command reflects release tag
- **WHEN** a released binary runs `opm version`
- **THEN** the output shows the tag version, commit SHA, and build date

### Requirement: Template modules publish on release
The `publish-templates` job SHALL `need` release-please, run only when `releases_created == 'true'`, check out `tag_name`, build `opm` from it, install cue v0.17.1, log in to GHCR with the repository `GITHUB_TOKEN`, and run `.github/scripts/publish-templates.sh` with `packages: write`. The script SHALL publish only template versions GHCR does not hold yet, so a release with no template version bumped publishes nothing and succeeds.

#### Scenario: No template version bumped
- **WHEN** every template's declared version is already on GHCR
- **THEN** nothing is published and the job succeeds

#### Scenario: Template fails a gate
- **WHEN** a template tree violates a publish gate
- **THEN** the job fails and the release is marked with a failed job

### Requirement: Workflow targets GitHub-hosted runner
The release workflow SHALL specify `runs-on: ubuntu-latest` for all jobs.

#### Scenario: GitHub-hosted runner assignment
- **WHEN** the release workflow triggers
- **THEN** all jobs are assigned to the `ubuntu-latest` runner pool
