## MODIFIED Requirements

### Requirement: Template publish gates run dry-run
The `template-gates` job SHALL check out the pull request with `fetch-depth: 0` (full history and every tag), build `opm` from the pull request, install cue v0.17.1, and run `.github/scripts/publish-templates.sh --dry-run`, so every gate runs over every template tree without pushing. For every template the gates SHALL be: `opm module tidy --check`, `opm module vet` (which renders the template's `debugValues` against a platform generated from its own pins), and the `opm module publish --dry-run` gates. A dry run whose only refusal is that the template's version is already published SHALL pass only when the template's directory is unchanged since the previous release tag, the newest `v*` tag reachable from the parent of the checked-out commit; a template changed since that tag at an already-published version SHALL fail the job, because published versions are immutable. The comparison SHALL use the release tag rather than a merge-base, so a change that reached `main` without a pull request fails every later pull request's job until a bump lands. When no release tag is reachable the job SHALL fail naming `fetch-depth: 0`. Any other refusal or a failed vet SHALL fail the job. The job SHALL report every failing template, not only the first.

#### Scenario: Template gate violation fails the PR
- **WHEN** a template tree violates a publish gate other than already-published
- **THEN** the `template-gates` job exits non-zero naming the refusal

#### Scenario: Already-published template passes
- **WHEN** a template's declared version is already on GHCR, it passes every other gate, and its directory is unchanged since the previous release tag
- **THEN** the `template-gates` job exits zero

#### Scenario: Template that does not vet fails the PR
- **WHEN** a template's `debugValues` do not render, for example a field left as an unresolved disjunction
- **THEN** `opm module vet` prints the render error
- **AND** the `template-gates` job exits non-zero naming the template

#### Scenario: Changed template at an already-published version fails the PR
- **WHEN** a template's directory differs from the previous release tag, its `cue.mod` pins and comments included, and the template's identity `Version` is a version GHCR already holds, including an older version than the one released
- **THEN** the `template-gates` job exits non-zero naming the template and `opm module version set` as the fix

#### Scenario: A change pushed to main without a bump fails later pull requests
- **WHEN** a commit that changed a template without bumping it reached `main` directly, and a pull request that does not touch the template is opened on top of it
- **THEN** the pull request's `template-gates` job exits non-zero naming that template

#### Scenario: A change and its revert are not a change
- **WHEN** a template was changed and the change reverted, both since the previous release tag
- **THEN** the template counts as unchanged and the job does not fail for it
