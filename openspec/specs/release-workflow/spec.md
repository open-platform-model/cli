# Capability: release-workflow

## Purpose

`.github/workflows/release.yml` turns merges to main into published releases: release-please maintains the version and changelog and opens the release PR, and once it cuts a tag, a first job gates and publishes the bundled template modules to GHCR, and then goreleaser builds and attaches the cross-platform archives and `checksums.txt` described by `.goreleaser.yml` and publishes the release.

## Requirements

### Requirement: Binaries publish only when a release was created
The `goreleaser` job SHALL `need` release-please and publish-templates and run only when `releases_created == 'true'` or the run is a manual `workflow_dispatch`, with `contents: write` and the repository `GITHUB_TOKEN`; `contents: write` is also what lets it see a draft release, which GitHub hides from read-only tokens. Before building it SHALL read the draft state of the GitHub Release for the tag and SHALL proceed only when the release is a draft. When the release is published it SHALL fail with a message that the release is already published and the next patch release must be cut instead. When no GitHub Release exists for the tag it SHALL fail with a message that names the tag and says to re-run the release-please job of the run that merged the release PR. When the lookup itself fails for any other reason it SHALL fail with that error and SHALL NOT report the release as published. After the draft check and before building, it SHALL fail unless the publish-templates job succeeded, with a message that the release stays a draft, so no release is published without its template modules. It SHALL run in a concurrency group per tag that never cancels a running job. Otherwise it SHALL check out `tag_name`, or the `tag` input on a manual run, with `fetch-depth: 0`, set up Go 1.26.0, and run `goreleaser release --clean`. The manual run is the recovery for a draft release whose tag exists without all of its assets, including one held back by a transient template failure; it SHALL NOT change a published release. Source: 0021:D10:R8.

#### Scenario: Skipped on a non-release push
- **WHEN** release-please reports no release created
- **THEN** the goreleaser job does not run and no assets are attached

#### Scenario: Runs on the release tag
- **WHEN** release-please reports a release created and the publish-templates job succeeded
- **THEN** goreleaser builds from the tagged commit with full git history available

#### Scenario: Manual recovery builds an existing tag
- **WHEN** the workflow is run by hand with `tag` set to an existing release tag whose GitHub Release is still a draft
- **THEN** release-please is skipped, the publish-templates job gates and publishes that tag's templates, and goreleaser builds that tag, attaches its assets to the existing draft, and publishes it

#### Scenario: Manual recovery refuses a published release
- **WHEN** the workflow is run by hand with `tag` set to a release that is already published
- **THEN** the goreleaser job fails before building, names the tag, and says to cut the next patch release, and the published release and its tag are unchanged

#### Scenario: Manual recovery for a tag without a release
- **WHEN** the workflow is run by hand with `tag` set to a tag that exists but has no GitHub Release
- **THEN** the goreleaser job fails before building, names the tag, says to re-run the release-please job, and does not say the release is published

#### Scenario: Template failure keeps the release a draft
- **WHEN** the publish-templates job fails for a release whose GitHub Release is still a draft
- **THEN** the goreleaser job fails after the draft check and before building, attaches no asset, and the release stays a draft

### Requirement: Goreleaser produces per-platform archives, checksums and changelog
Goreleaser SHALL build `opm` for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and windows/amd64 (windows/arm64 excluded), package each as an archive named `opm-<os>-<arch>` bundling `LICENSE`, and attach the archives, `checksums.txt` (SHA256 digests) and `LICENSE` to the GitHub Release. Its changelog SHALL group commits into Features, Bug Fixes, Performance, Refactoring and Other, excluding subjects starting with `docs:`, `test:`, `ci:` or `chore:`. Goreleaser SHALL attach to release-please's draft release (`release.use_existing_draft: true`, matched by a release name equal to the tag) instead of creating a release, SHALL upload every asset while the release is still a draft, replacing an asset a failed earlier run left on that draft, and SHALL publish the release as its last step. The configuration SHALL NOT set `release.draft`, a `release.name_template` that departs from the tag, or `release.replace_existing_draft`. Goreleaser SHALL set the GitHub Release's prerelease flag from the tag (`release.prerelease: auto`): a tag with a SemVer prerelease suffix SHALL be marked Pre-release and SHALL NOT become the repository's Latest release. It SHALL keep the release notes release-please already wrote on the existing GitHub Release (`release.mode: keep-existing`). Source: 0021:D10:R8.

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
- **THEN** goreleaser attaches to release-please's draft for the tag and creates no GitHub Release of its own, the release stays a draft until every asset is attached, and goreleaser then publishes it

### Requirement: Version ldflags are injected at build time
The goreleaser build SHALL inject `Version`, `GitCommit`, and `BuildDate` via ldflags matching the variables in `internal/version/version.go`.

#### Scenario: Version command reflects release tag
- **WHEN** a released binary runs `opm version`
- **THEN** the output shows the tag version, commit SHA, and build date

### Requirement: Template modules publish on release
The `publish-templates` job SHALL `need` release-please, run only when `releases_created == 'true'` or the run is a manual `workflow_dispatch`, check out `tag_name` (or the `tag` input on a manual run), build `opm` from it, install the cue version that the tag's `go.mod` requires for `cuelang.org/go`, log in to GHCR with the repository `GITHUB_TOKEN`, and run `.github/scripts/publish-templates.sh` with `packages: write`. It SHALL finish before the goreleaser job builds. Before it publishes any template, the script SHALL run every gate of the pr-workflow requirement "Template publish gates run dry-run" on every template tree, with the `opm` built from the tag:
- the identity, the tree layout, `opm module tidy --check` and `opm module vet`;
- the tree equals its module zip;
- the `opm module publish --dry-run` gates, with already-published as the only acceptable refusal;
- the version order, and for an already-published version the content check: the artifact GHCR holds at that version SHALL contain exactly the tree's files.

If any template fails a gate, or a fetch, listing, verification or comparison fails, the script SHALL publish nothing and fail. Otherwise it SHALL act on each template's gate verdict without reading GHCR again: it SHALL publish every template whose version the gates found unpublished, and skip every template found published and identical. A version that becomes published between the gates and the push SHALL make `opm module publish` refuse and the job fail, never be skipped unchecked. A release with no template changed publishes nothing and succeeds, and a re-run after a partial publish skips what the earlier attempt pushed. A change to a template's files, its `cue.mod` pins included, SHALL therefore bump that template's identity `Version` to a stable version above the highest published one, or the release fails before publishing any template and stays a draft.

#### Scenario: No template version bumped
- **WHEN** every template's declared version is already on GHCR as the highest stable version of its major, and each published artifact holds exactly its template's tree
- **THEN** nothing is published and the job succeeds

#### Scenario: Template fails a gate
- **WHEN** a template tree violates a publish gate
- **THEN** the job fails, no template is published, and the release is marked with a failed job and stays a draft

#### Scenario: Bumped template publishes
- **WHEN** a release carries a template whose identity `Version` is a stable version above every stable version GHCR holds for its major, and every gate passes
- **THEN** the job publishes that template at that version through `opm module publish`

#### Scenario: A template that does not vet blocks every template publish
- **WHEN** a release carries two bumped templates and one of them fails `opm module vet`
- **THEN** neither template is published
- **AND** the job fails naming the template that failed

#### Scenario: Changed template at a published version fails before any publish
- **WHEN** a template's tree differs from the artifact GHCR holds at its declared version
- **THEN** no template is published
- **AND** the job fails naming the template, printing the difference and naming `opm module version set`

#### Scenario: Re-run after a partial publish completes
- **WHEN** the job runs again for the same tag after an earlier attempt pushed one of two templates whose versions this release raised
- **THEN** the pushed template, now identical on GHCR, is skipped, the other template is published, and the job succeeds

#### Scenario: A version published during the run is not skipped
- **WHEN** the gates found a template version unpublished and another run publishes that version before this run pushes it
- **THEN** `opm module publish` refuses the push, the job fails, and no later template in the run is published

#### Scenario: A template whose zip omits a file publishes nothing
- **WHEN** a release carries a bumped template whose tree holds a file its module zip leaves out, such as a symbolic link
- **THEN** no template is published
- **AND** the job fails naming the template and the file

#### Scenario: A prerelease or older template version publishes nothing
- **WHEN** a release carries a template whose version is a prerelease, or is not above the highest stable version GHCR holds for its major while unpublished, or is below it while published
- **THEN** no template is published
- **AND** the job fails naming the template

### Requirement: Workflow targets GitHub-hosted runner
The release workflow SHALL specify `runs-on: ubuntu-latest` for all jobs.

#### Scenario: GitHub-hosted runner assignment
- **WHEN** the release workflow triggers
- **THEN** all jobs are assigned to the `ubuntu-latest` runner pool

### Requirement: The release workflow never mutates a tag
No job, step or script of the release workflow SHALL move, delete or re-create a git tag, delete a GitHub Release, or change the tag or target of an existing GitHub Release. The only tag the workflow creates is the new release tag, created by release-please as the opm-release-please App. A wrong or broken published release SHALL be fixed by releasing the next version. Source: 0021:D10:R1.

#### Scenario: Broken published release
- **WHEN** a published cli release turns out wrong or incomplete
- **THEN** the fix is the next release, and the published release's tag and assets stay as they are

### Requirement: Commit types decide whether a release is cut
`release-please-config.json` SHALL list `feat`, `fix`, `perf`, `revert`, `deps` and `refactor` as visible changelog sections, so a commit of one of those types on `main` opens or updates a release PR. It SHALL list `docs`, `test`, `ci`, `build` and `chore` as hidden sections, so a commit carrying only those types cuts no release. Source: workspace RELEASING.md, section "Pin classes".

#### Scenario: Docs-only commit cuts no release
- **WHEN** only `docs(...)` commits land on `main` after the last release
- **THEN** release-please opens no release PR

#### Scenario: Refactor still releases
- **WHEN** a `refactor(...)` commit lands on `main`
- **THEN** release-please opens or updates the release PR and lists it under Code Refactoring

#### Scenario: Docs commit beside a fix
- **WHEN** a `docs(...)` commit and a `fix(...)` commit land on `main`
- **THEN** the release PR's changelog lists the fix and omits the docs commit

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
A forced version, such as a move to a new version line (for example `1.0.0-alpha.27` to `1.0.0-beta.1`), SHALL be a `release-as` value on the `.` package in `release-please-config.json`, set by a normal pull request. The key SHALL take effect only together with a releasable commit (a visible type: `feat`, `fix`, `perf`, `revert`, `deps` or `refactor`) on `main` since the last release, because release-please opens no release PR for hidden-type commits alone, whatever the key says; so the key pull request SHALL itself carry a releasable type, or land after a releasable commit since the last release. Because release-please re-applies a configured `release-as` on every run, the key SHALL be removed before any later release PR merges. Outside the span from the key pull request to the pull request that removes it, the configuration SHALL carry no `release-as` value. A `Release-As:` footer in a pull request body or a branch commit message SHALL NOT be relied on, because the `BLANK` squash message keeps it off `main`. A `BEGIN_COMMIT_OVERRIDE` block in a pull request body SHALL NOT be used, because release-please would take it as the commit message and apply its footers. Source: workspace RELEASING.md, section "Owner settings" (owner decision 2026-10-02).

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
- **WHEN** a pull request whose body or branch commit message carries `Release-As: 1.1.0-beta.1`, and whose body has no `BEGIN_COMMIT_OVERRIDE` block, merges to main under the `BLANK` squash message
- **THEN** the squash commit on main carries only the pull request title, and release-please proposes the next version from that title alone

### Requirement: A breaking change is marked by a bang in the pull request title
A breaking change SHALL be marked by `!` in the pull request title (`feat!:`, `fix(deps)!:`), which is the squash commit title once `squash_merge_commit_title` is `PR_TITLE`. Until that owner setting lands, a one-commit pull request squashes under its commit subject, so that subject SHALL carry the same `!`. A `BREAKING CHANGE:` footer in a pull request body or a branch commit message SHALL NOT be relied on, because the `BLANK` squash message keeps it off `main`; the migration note goes in the pull request body, and in the user docs when users need it to upgrade, and the release pull request's `CHANGELOG.md` SHALL be edited by hand to carry it as the last step before that release pull request merges, and the edit SHALL be redone if anything landed on `main` since, because release-please rebuilds the release pull request on every push to `main` and drops a hand edit. A `BEGIN_COMMIT_OVERRIDE` block in a pull request body SHALL NOT be used, because release-please would take it as the commit message. Source: workspace RELEASING.md, section "Owner settings" (owner decision 2026-10-02).

#### Scenario: Breaking change marked in the title
- **WHEN** a pull request titled `feat!: ...` squash-merges to main while the manifest holds `1.0.0-beta.3`
- **THEN** release-please lists it under breaking changes and proposes `1.0.0-beta.4`, advancing the `-beta.N` counter

#### Scenario: A breaking footer alone marks nothing
- **WHEN** a pull request titled `feat: ...` whose body carries a `BREAKING CHANGE:` footer and no `BEGIN_COMMIT_OVERRIDE` block squash-merges to main under the `BLANK` squash message
- **THEN** the squash commit carries only the title, and release-please files it as an ordinary feature

### Requirement: Downstream repos are notified after a release is published
`.github/workflows/release.yml` SHALL have a `notify-downstream` job, its last, that `needs` release-please and goreleaser. It SHALL run only when goreleaser succeeded, the run's ref is `main`, and the repository variable `CASCADE_NOTIFY` is not `off`. It SHALL run on a release run and on a manual recovery run alike, because release-please is skipped on the manual run. It SHALL NOT depend on the docs bundle job. It is a caller-owned job: it SHALL run on `ubuntu-latest`, declare `environment: cascade`, have a 20-minute timeout, grant only `contents: read`, and have exactly one step, the shared `open-platform-model/.github` `cascade-notify` action at the repository's pinned `.github` commit, with the inputs `tag` (`tag_name` from release-please, or the `tag` input on a manual run), `client-id: ${{ vars.CASCADE_APP_CLIENT_ID }}` and `private-key: ${{ secrets.CASCADE_APP_PRIVATE_KEY }}`. The action mints the dispatch token and chooses the targets: catalog_opm and opm-operator, as release-tool edges. Source: workspace RELEASING.md, sections "Notify after publish" and "Stop switches"; Phase 3 wiring contract (version 3.1) §4.5 and §4.6.

#### Scenario: Release run notifies after publishing
- **WHEN** a push to `main` cuts a release, and publish-templates and goreleaser succeed
- **THEN** the notify job runs after goreleaser in the `cascade` Environment with the release's tag, and catalog_opm and opm-operator each receive one `upstream-released` dispatch naming the cli and that tag

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
- **THEN** the notify job is skipped, and if that run published a draft, the downstream receivers' daily sweep picks the release up

#### Scenario: A push without a release does not notify
- **WHEN** a push to `main` cuts no release
- **THEN** goreleaser and the notify job do not run

#### Scenario: Notify switched off
- **WHEN** the repository variable `CASCADE_NOTIFY` is `off` and a release is published
- **THEN** the notify job is skipped and no dispatch is sent

#### Scenario: A failed docs bundle does not suppress the notify
- **WHEN** goreleaser publishes the release and the docs bundle job fails
- **THEN** the notify job still runs
