## MODIFIED Requirements

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
