## MODIFIED Requirements

### Requirement: Template publish gates run dry-run
The `template-gates` job SHALL build `opm` from the pull request, install cue v0.17.1, and run `.github/scripts/publish-templates.sh --dry-run`, so every gate runs over every template tree without pushing. For every template the gates SHALL be: `opm module tidy --check`, `opm module vet` (which renders the template's `debugValues` against a platform generated from its own pins), and the `opm module publish --dry-run` gates. A dry run whose only refusal is that the template's version is already published SHALL pass only when the artifact GHCR holds at that version contains exactly the tree's files. The script SHALL:
- fetch the published module zip from GHCR with an anonymous pull token and verify it against its manifest digest;
- build the tree's module zip with `cue mod publish --out`, the same `modzip.CreateFromDir` file selection `opm module publish` zips with;
- compare the two zips file by file, by path and bytes, comments included.

A template whose tree differs from its published artifact SHALL fail the job, because published versions are immutable. The comparison SHALL NOT depend on git history, so a change that reached `main` without a pull request, or under a later release tag, fails every later pull request's job until a bump lands. A failure to fetch, verify, build or compare SHALL fail the job and SHALL NOT count the template as unpublished. Any other refusal or a failed vet SHALL fail the job. The job SHALL report every failing template, not only the first.

#### Scenario: Template gate violation fails the PR
- **WHEN** a template tree violates a publish gate other than already-published
- **THEN** the `template-gates` job exits non-zero naming the refusal

#### Scenario: Already-published template passes
- **WHEN** a template's declared version is already on GHCR, it passes every other gate, and the published artifact holds exactly the tree's files
- **THEN** the `template-gates` job exits zero

#### Scenario: Template that does not vet fails the PR
- **WHEN** a template's `debugValues` do not render, for example a field left as an unresolved disjunction
- **THEN** `opm module vet` prints the render error
- **AND** the `template-gates` job exits non-zero naming the template

#### Scenario: Changed template at an already-published version fails the PR
- **WHEN** a template's tree differs in any file, its `cue.mod` pins and comments included, from the artifact GHCR holds at the template's declared version, including an older version than the one last released
- **THEN** the `template-gates` job exits non-zero, prints the difference, and names the template and `opm module version set` as the fix

#### Scenario: A change pushed to main without a bump fails later pull requests
- **WHEN** a commit that changed a template without bumping it reached `main` directly, possibly with a release cut on top of it, and a pull request that does not touch the template is opened afterwards
- **THEN** the pull request's `template-gates` job exits non-zero naming that template

#### Scenario: A change and its revert are not a change
- **WHEN** a template was changed and the change was reverted, so its tree again equals its published artifact
- **THEN** the template passes and the job does not fail for it

#### Scenario: A fetch error never passes as unpublished
- **WHEN** GHCR holds a template's declared version but the published artifact cannot be fetched, or does not match its manifest digest
- **THEN** the `template-gates` job exits non-zero naming the template
