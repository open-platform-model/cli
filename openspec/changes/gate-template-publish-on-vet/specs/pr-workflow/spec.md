## MODIFIED Requirements

### Requirement: Template publish gates run dry-run
The `template-gates` job SHALL build `opm` from the pull request, install the cue version that the cli's `go.mod` requires for `cuelang.org/go`, and run `.github/scripts/publish-templates.sh --dry-run`, so every gate runs over every template tree without pushing. For every template the gates SHALL be, in order:
- the identity: its `Version` SHALL be a stable SemVer `X.Y.Z`, and its `ModulePath` SHALL be `opmodel.dev/templates/<directory>@v<major of Version>`;
- the tree layout: it SHALL hold no symbolic link, no special file and no nested `cue.mod`;
- `opm module tidy --check`;
- `opm module vet`, which renders the template's `debugValues` against a platform generated from its own pins;
- the module zip that `cue mod publish --out` builds from the tree, the same `modzip.CreateFromDir` file selection `opm module publish` zips with, SHALL hold exactly the tree's files, so the published module is the one vet rendered;
- the `opm module publish --dry-run` gates, which SHALL either pass or refuse only because the template's version is already published;
- the version order: an unpublished version SHALL be above the highest stable version GHCR holds for the template's major, and an already-published version SHALL be that highest version, because `opm module init` resolves it.

An already-published version SHALL pass only when the artifact GHCR holds at that version contains exactly the tree's files. The script SHALL fetch the published module zip from GHCR with an anonymous pull token, verify it against its manifest digest, and compare it with the tree's module zip file by file, by path and bytes, comments included.

A template whose tree differs from its published artifact SHALL fail the job, because published versions are immutable. The comparison SHALL NOT depend on git history, so a change that reached `main` without a pull request, or under a later release tag, fails every later pull request's job until a bump lands. A failure to fetch, list, verify, build or compare SHALL fail the job and SHALL NOT count the template as unpublished. Any other refusal or a failed vet SHALL fail the job. The job SHALL report every failing template, not only the first, including when a template's identity cannot be evaluated.

#### Scenario: Template gate violation fails the PR
- **WHEN** a template tree violates a publish gate other than already-published
- **THEN** the `template-gates` job exits non-zero naming the refusal

#### Scenario: Already-published template passes
- **WHEN** a template's declared version is already on GHCR and is the highest stable version there for its major, it passes every other gate, and the published artifact holds exactly the tree's files
- **THEN** the `template-gates` job exits zero

#### Scenario: Template that does not vet fails the PR
- **WHEN** a template's `debugValues` do not render, for example a field left as an unresolved disjunction
- **THEN** `opm module vet` prints the render error
- **AND** the `template-gates` job exits non-zero naming the template

#### Scenario: Changed template at an already-published version fails the PR
- **WHEN** a template's tree differs in any file, its `cue.mod` pins and comments included, from the artifact GHCR holds at the template's declared version
- **THEN** the `template-gates` job exits non-zero, prints the difference, and names the template and `opm module version set` as the fix

#### Scenario: A change pushed to main without a bump fails later pull requests
- **WHEN** a commit that changed a template without bumping it reached `main` directly, possibly with a release cut on top of it, and a pull request that does not touch the template is opened afterwards
- **THEN** the pull request's `template-gates` job exits non-zero naming that template

#### Scenario: A change and its revert are not a change
- **WHEN** a template was changed and the change was reverted, so its tree again equals its published artifact
- **THEN** the template passes and the job does not fail for it

#### Scenario: A fetch error never passes as unpublished
- **WHEN** GHCR holds a template's declared version but the published artifact or the template's version list cannot be fetched, or the artifact does not match its manifest digest
- **THEN** the `template-gates` job exits non-zero naming the template

#### Scenario: A file the module zip omits fails the PR
- **WHEN** a template tree holds a symbolic link, a special file, a nested `cue.mod`, or any other file its module zip leaves out, whether or not its version is already published
- **THEN** the `template-gates` job exits non-zero naming the template and the file

#### Scenario: A version init would not resolve fails the PR
- **WHEN** a template declares a prerelease version, an unpublished version not above the highest published stable version of its major, or a published version below that highest one
- **THEN** the `template-gates` job exits non-zero naming the template and the highest published version

#### Scenario: A module path that does not match its directory fails the PR
- **WHEN** the identity `ModulePath` of the template in `templates/<directory>/` is not `opmodel.dev/templates/<directory>@v<major>`
- **THEN** the `template-gates` job exits non-zero naming the expected path

#### Scenario: A broken identity does not hide other failures
- **WHEN** one template's identity package cannot be evaluated and another template fails a gate
- **THEN** the `template-gates` job exits non-zero naming both templates
