## REMOVED Requirements

### Requirement: Release runs from pushes to main through release-please

**Reason**: Its last sentence required a version-line change to travel as a one-shot `Release-As:` footer and forbade a `release-as` key. Under the `BLANK` squash message (workspace RELEASING.md, section "Owner settings") a footer never reaches `main`, and the key is the mechanism. The requirement is re-added below without that sentence, and the footer scenario is replaced by the new requirement "A version line moves only through a release-as key".

**Migration**: The workflow is unchanged. A contributor who forced a version with a `Release-As:` footer now sets `release-as` in `release-please-config.json` in a normal PR and removes it in the next PR once that release is cut.

## ADDED Requirements

### Requirement: Release-please runs on pushes to main
The release workflow SHALL trigger on `push` to `main` and on `workflow_dispatch` with one required `tag` input, and never on a tag push. On a push, the `release-please` job SHALL run first, authenticated as the opm-release-please App with no `GITHUB_TOKEN` fallback, driven by `release-please-config.json` and `.release-please-manifest.json`: a Go package named `opm` on a `v`-prefixed, `beta` prerelease version line with `CHANGELOG.md` as the changelog. The configuration SHALL set `draft: true` and `force-tag-creation: true`, so release-please creates the git tag at the release commit itself and leaves the GitHub Release as a draft for goreleaser to fill and publish. The job SHALL expose `releases_created` and `tag_name` as outputs for the downstream jobs. On a manual run the `release-please` job SHALL be skipped. Source: 0021:D10:R8.

#### Scenario: Release-please on a push without a merged release PR
- **WHEN** a commit that is not a release PR merge lands on main
- **THEN** release-please updates or opens the release PR and the goreleaser and publish-templates jobs are skipped

#### Scenario: Release-please on a merged release PR
- **WHEN** the release PR merges to main
- **THEN** release-please creates the tag at the release commit and a draft GitHub Release, and reports `releases_created == 'true'` with the new `tag_name`

#### Scenario: Releases without a release-as key stay on the beta line
- **WHEN** the manifest holds `1.0.0-beta.1`, `release-please-config.json` carries no `release-as` key, and a releasable commit lands on main
- **THEN** release-please proposes `1.0.0-beta.2`

### Requirement: A version line moves only through a release-as key
A forced version, such as a move to a new version line (for example `1.0.0-alpha.27` to `1.0.0-beta.1`), SHALL be a `release-as` value on the `.` package in `release-please-config.json`, set by a normal pull request. The key SHALL take effect only together with a releasable commit (a visible type: `feat`, `fix`, `perf`, `revert`, `deps` or `refactor`) on `main` since the last release, because release-please opens no release PR for hidden-type commits alone, whatever the key says; so the key pull request SHALL itself carry a releasable type, or land after a releasable commit since the last release. Because release-please re-applies a configured `release-as` on every run, the key SHALL be removed before any later release PR merges. Outside the span from the key pull request to the pull request that removes it, the configuration SHALL carry no `release-as` value. A `Release-As:` footer in a pull request body or a branch commit message SHALL NOT be relied on, because the `BLANK` squash message keeps it off `main`. Source: workspace RELEASING.md, section "Owner settings" (owner decision 2026-10-02).

#### Scenario: A release-as key forces the line change
- **WHEN** the manifest holds `1.0.0-alpha.27`, a pull request that sets `release-as` to `1.0.0-beta.1` in `release-please-config.json` merges to main, and it or another commit since the last release has a releasable type
- **THEN** release-please opens or retitles the release PR as `chore(main): release 1.0.0-beta.1`

#### Scenario: A key pull request of a hidden type alone opens no release PR
- **WHEN** the only commit on main since the last release is a `chore(...)`, `build(...)` or `ci(...)` pull request that sets `release-as`
- **THEN** release-please opens no release PR, and the forced version waits for the next releasable commit

#### Scenario: The key is removed once its release is cut
- **WHEN** the release PR for the `release-as` version has merged and that version is tagged
- **THEN** a pull request removes the `release-as` key before any later release PR merges, and later releasable commits advance the line normally

#### Scenario: A footer forces nothing
- **WHEN** a pull request whose body or branch commit message carries `Release-As: 1.1.0-beta.1` merges to main under the `BLANK` squash message
- **THEN** the squash commit on main carries only the pull request title, and release-please proposes the next version from that title alone

### Requirement: A breaking change is marked by a bang in the pull request title
A breaking change SHALL be marked by `!` in the pull request title (`feat!:`, `fix(deps)!:`), which is the squash commit title once `squash_merge_commit_title` is `PR_TITLE`. Until that owner setting lands, a one-commit pull request squashes under its commit subject, so that subject SHALL carry the same `!`. A `BREAKING CHANGE:` footer in a pull request body or a branch commit message SHALL NOT be relied on, because the `BLANK` squash message keeps it off `main`; the migration note goes in the pull request body, and in the user docs when users need it to upgrade, and the release pull request's `CHANGELOG.md` SHALL be edited by hand to carry it before that release pull request merges. Source: workspace RELEASING.md, section "Owner settings" (owner decision 2026-10-02).

#### Scenario: Breaking change marked in the title
- **WHEN** a pull request titled `feat!: ...` squash-merges to main while the manifest holds `1.0.0-beta.3`
- **THEN** release-please lists it under breaking changes and proposes `1.0.0-beta.4`, advancing the `-beta.N` counter

#### Scenario: A breaking footer alone marks nothing
- **WHEN** a pull request titled `feat: ...` whose body carries a `BREAKING CHANGE:` footer squash-merges to main under the `BLANK` squash message
- **THEN** the squash commit carries only the title, and release-please files it as an ordinary feature
