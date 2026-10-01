## MODIFIED Requirements

### Requirement: Template modules publish on release
The `publish-templates` job SHALL `need` release-please, run only when `releases_created == 'true'` or the run is a manual `workflow_dispatch`, check out `tag_name` (or the `tag` input on a manual run) with `fetch-depth: 0`, build `opm` from it, install cue v0.17.1, log in to GHCR with the repository `GITHUB_TOKEN`, and run `.github/scripts/publish-templates.sh` with `packages: write`. Before it publishes any template, the script SHALL run every gate on every template tree: `opm module tidy --check`, `opm module vet` with the `opm` built from the tag, the `opm module publish --dry-run` gates with already-published as the only acceptable refusal, and changed-implies-bumped against the previous release tag (the newest `v*` tag reachable from the release commit's parent). A template whose directory changed since the previous release tag at a version GHCR already holds SHALL fail the gate, unless its declared version rose since the previous release tag, which is a re-run of this release after a partial publish. If any template fails a gate, the script SHALL publish nothing and fail. Otherwise it SHALL publish only template versions GHCR does not hold yet, so a release with no template version bumped publishes nothing and succeeds. A change to a template's files, its `cue.mod` pins included, SHALL therefore bump that template's identity `Version`, or the release fails before publishing any template.

#### Scenario: No template version bumped
- **WHEN** every template's declared version is already on GHCR and no template changed since the previous release tag
- **THEN** nothing is published and the job succeeds

#### Scenario: Template fails a gate
- **WHEN** a template tree violates a publish gate
- **THEN** the job fails and the release is marked with a failed job

#### Scenario: Bumped template publishes
- **WHEN** a release carries a template whose identity `Version` GHCR does not hold yet
- **THEN** the job publishes that template at that version through `opm module publish`

#### Scenario: A template that does not vet blocks every template publish
- **WHEN** a release carries two bumped templates and one of them fails `opm module vet`
- **THEN** neither template is published
- **AND** the job fails naming the template that failed

#### Scenario: Changed template at a published version fails before any publish
- **WHEN** a template changed since the previous release tag and its declared version, unchanged since that tag, is already on GHCR
- **THEN** no template is published
- **AND** the job fails naming the template and `opm module version set`

#### Scenario: Re-run after a partial publish completes
- **WHEN** the job runs again for the same tag after an earlier attempt pushed one of two templates whose versions this release raised
- **THEN** the pushed template is skipped, the other template is published, and the job succeeds
