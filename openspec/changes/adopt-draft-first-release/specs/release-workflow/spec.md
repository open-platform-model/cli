## MODIFIED Requirements

### Requirement: Release runs from pushes to main through release-please
The release workflow SHALL trigger on `push` to `main` and on `workflow_dispatch` with one required `tag` input, and never on a tag push. On a push, the `release-please` job SHALL run first, driven by `release-please-config.json` and `.release-please-manifest.json`: a Go package named `opm` on a `v`-prefixed, `beta` prerelease version line with `CHANGELOG.md` as the changelog. The configuration SHALL set `draft: true` and `force-tag-creation: true`, so release-please creates the git tag at the release commit itself and leaves the GitHub Release as a draft for goreleaser to fill and publish. The job SHALL expose `releases_created`, `tag_name` and `sha` as outputs for the downstream jobs. On a manual run the `release-please` job SHALL be skipped. The configuration SHALL NOT carry a `release-as` value: a move to a new version line (for example `1.0.0-alpha.27` to `1.0.0-beta.1`) SHALL travel as a one-shot `Release-As: <version>` footer in the final commit message of one commit on main, because a configured `release-as` is re-applied on every run.

#### Scenario: Push without a merged release PR
- **WHEN** a commit that is not a release PR merge lands on main
- **THEN** release-please updates or opens the release PR and the verify-release, goreleaser and publish-templates jobs are skipped

#### Scenario: Release PR merged
- **WHEN** the release PR merges to main
- **THEN** release-please creates the tag at the release commit and a draft GitHub Release, and reports `releases_created == 'true'` with the new `tag_name` and the release commit as `sha`

#### Scenario: Line change forced by a footer
- **WHEN** the manifest holds `1.0.0-alpha.27` and a commit whose final footer paragraph carries `Release-As: 1.0.0-beta.1` lands on main
- **THEN** release-please opens or retitles the release PR as `chore(main): release 1.0.0-beta.1`

#### Scenario: Next release stays on the beta line
- **WHEN** the manifest holds `1.0.0-beta.1` and a releasable commit without a `Release-As` footer lands on main
- **THEN** release-please proposes `1.0.0-beta.2`

### Requirement: Binaries publish only when a release was created
The `goreleaser` job SHALL `need` release-please and verify-release, and run only when `releases_created == 'true'` or the run is a manual `workflow_dispatch`, and verify-release succeeded. It SHALL fail before building when verify-release reports the release is not a draft, with a message that the release is already published and the next patch release must be cut instead. Otherwise it SHALL check out the verified tag with `fetch-depth: 0`, set up Go 1.26.0, and run `goreleaser release --clean` with `contents: write` and the repository `GITHUB_TOKEN`. The manual run is the recovery for a draft release whose tag exists without all of its assets; it SHALL NOT change a published release.

#### Scenario: Skipped on a non-release push
- **WHEN** release-please reports no release created
- **THEN** the goreleaser job does not run and no assets are attached

#### Scenario: Runs on the release tag
- **WHEN** release-please reports a release created
- **THEN** goreleaser builds from the tagged commit with full git history available

#### Scenario: Manual recovery builds an existing tag
- **WHEN** the workflow is run by hand with `tag` set to an existing release tag whose GitHub Release is still a draft
- **THEN** release-please is skipped and goreleaser builds that tag, attaches its assets to the existing draft, and publishes it

#### Scenario: Manual recovery refuses a published release
- **WHEN** the workflow is run by hand with `tag` set to a release that is already published
- **THEN** the goreleaser job fails before building, names the tag, and says to cut the next patch release, and the published release and its tag are unchanged

### Requirement: Goreleaser produces per-platform archives, checksums and changelog
Goreleaser SHALL build `opm` for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and windows/amd64 (windows/arm64 excluded), package each as an archive named `opm-<os>-<arch>` bundling `LICENSE`, and attach the archives, `checksums.txt` (SHA256 digests) and `LICENSE` to the GitHub Release. Its changelog SHALL group commits into Features, Bug Fixes, Performance, Refactoring and Other, excluding subjects starting with `docs:`, `test:`, `ci:` or `chore:`. Goreleaser SHALL attach to release-please's draft release (`release.use_existing_draft: true`, matched by a release name equal to the tag) instead of creating a release, SHALL upload every asset while the release is still a draft, replacing an asset a failed earlier run left on that draft, and SHALL publish the release as its last step. Goreleaser SHALL set the GitHub Release's prerelease flag from the tag (`release.prerelease: auto`): a tag with a SemVer prerelease suffix SHALL be marked Pre-release and SHALL NOT become the repository's Latest release. It SHALL keep the release notes release-please already wrote on the existing GitHub Release (`release.mode: keep-existing`).

#### Scenario: Five archives and checksums attached
- **WHEN** goreleaser completes
- **THEN** the GitHub Release assets include the five `opm-<os>-<arch>` archives, `checksums.txt` and `LICENSE`

#### Scenario: Changelog grouped by type
- **WHEN** goreleaser generates its changelog
- **THEN** entries are grouped by conventional commit type with the excluded types absent

#### Scenario: Beta tag flagged as a pre-release
- **WHEN** goreleaser publishes the assets for `v1.0.0-beta.1`
- **THEN** the GitHub Release is marked Pre-release, is not the Latest release, and still carries release-please's notes

#### Scenario: Release published only with all assets
- **WHEN** release-please has created a draft release and goreleaser runs for its tag
- **THEN** exactly one GitHub Release exists for the tag, it stays a draft until every asset is attached, and goreleaser then publishes it

### Requirement: Template modules publish on release
The `publish-templates` job SHALL `need` release-please and verify-release, run only when `releases_created == 'true'` or the run is a manual `workflow_dispatch`, and verify-release succeeded, check out the verified tag, build `opm` from it, install cue v0.17.1, log in to GHCR with the repository `GITHUB_TOKEN`, and run `.github/scripts/publish-templates.sh` with `packages: write`. It SHALL run whether the release is a draft or already published, since it changes neither the tag nor the GitHub Release. The script SHALL publish only template versions GHCR does not hold yet, so a release with no template version bumped publishes nothing and succeeds. A change to a template's `cue.mod` pins SHALL therefore bump that template's identity `Version` in the same release, or the new pins never reach `opm module init`.

#### Scenario: No template version bumped
- **WHEN** every template's declared version is already on GHCR
- **THEN** nothing is published and the job succeeds

#### Scenario: Template fails a gate
- **WHEN** a template tree violates a publish gate
- **THEN** the job fails and the release is marked with a failed job

#### Scenario: Bumped template publishes
- **WHEN** a release carries a template whose identity `Version` GHCR does not hold yet
- **THEN** the job publishes that template at that version through `opm module publish`

## ADDED Requirements

### Requirement: Release tag verified before any publish step
A `verify-release` job SHALL `need` release-please and run under the same condition as the goreleaser and publish-templates jobs, before both. It SHALL run `.github/scripts/verify-release-tag.sh` with the tag (release-please's `tag_name`, or the `tag` input on a manual run) and the expected commit (release-please's `sha`, or on a manual run the GitHub Release's `target_commitish`). The script SHALL resolve the tag's peeled commit from the remote and SHALL fail when the GitHub Release or the tag does not exist or the tag's commit differs from the expected commit, naming the tag, both commits, and that the tag is never moved so the next release must be cut. On success the job SHALL output the verified tag and whether the release is still a draft.

#### Scenario: Tag at the released commit
- **WHEN** release-please reports a release whose tag points at its `sha`
- **THEN** verify-release succeeds and outputs the tag with `draft == 'true'`

#### Scenario: Stale tag at another commit
- **WHEN** the tag already existed at a different commit and release-please adopted it
- **THEN** verify-release fails naming both commits, and neither goreleaser nor publish-templates runs

#### Scenario: Unknown tag on a manual run
- **WHEN** the workflow is run by hand with a `tag` that has no GitHub Release
- **THEN** verify-release fails and nothing is built or published

### Requirement: The release workflow never mutates a tag
No job, step or script of the release workflow SHALL move, delete or re-create a git tag, delete a GitHub Release, or change the tag or target of an existing GitHub Release. The only tag the workflow creates is the new release tag, created by release-please. A wrong or broken published release SHALL be fixed by releasing the next version.

#### Scenario: Broken published release
- **WHEN** a published cli release turns out wrong or incomplete
- **THEN** the fix is the next release, and the published release's tag and assets stay as they are
