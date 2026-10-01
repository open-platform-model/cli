## MODIFIED Requirements

### Requirement: Template publish gates run dry-run
The `template-gates` job SHALL check out with `fetch-depth: 0`, build `opm` from the pull request, install cue v0.17.1, and run `.github/scripts/publish-templates.sh --dry-run` with `BASE_REF` set to `origin/<base branch>`, so every gate runs over every template tree without pushing. For every template the gates SHALL be: `opm module tidy --check`, `opm module vet` (which renders the template's `debugValues` against a platform generated from its own pins), and the `opm module publish --dry-run` gates. A dry run whose only refusal is that the template's version is already published SHALL pass only when the template's directory is unchanged since the merge-base of `BASE_REF` and the pull request head; a changed template at an already-published version SHALL fail the job, because published versions are immutable. Any other refusal or a failed vet SHALL fail the job. The job SHALL report every failing template, not only the first.

#### Scenario: Template gate violation fails the PR
- **WHEN** a template tree violates a publish gate other than already-published
- **THEN** the `template-gates` job exits non-zero naming the refusal

#### Scenario: Already-published template passes
- **WHEN** a template's declared version is already on GHCR, it passes every other gate, and its directory is unchanged since the merge-base
- **THEN** the `template-gates` job exits zero

#### Scenario: Template that does not vet fails the PR
- **WHEN** a template's `debugValues` do not render, for example a field left as an unresolved disjunction
- **THEN** `opm module vet` prints the render error
- **AND** the `template-gates` job exits non-zero naming the template

#### Scenario: Changed template at an already-published version fails the PR
- **WHEN** a pull request changes any file under a template's directory, its `cue.mod` pins included, and leaves the template's identity `Version` at a version GHCR already holds
- **THEN** the `template-gates` job exits non-zero naming the template and `opm module version set` as the fix

#### Scenario: Branch that only lags main is not changed
- **WHEN** a pull request's branch forked before a template bump landed on main and does not itself touch the template
- **THEN** the template counts as unchanged and the job does not fail for it
