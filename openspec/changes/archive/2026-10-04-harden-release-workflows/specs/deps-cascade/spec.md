## REMOVED Requirements

### Requirement: The cascade task leaves non-cascade files alone and reports risks as warnings

**Reason**: The docs-bundle warning ran `go run ./hack/docskit-dump pins` on the moved tree, which executes the newly pinned library's code inside the cascade's compute step (security pass finding CAS-R2). The check moves to the pull request's own CI (release-gates, "Pull requests that move library or the operator warn about missing docs bundles"); the rest of the requirement is restated without it.

**Migration**: None for users. The docs-bundle warning for a cascade pull request appears on its `Lint` check instead of in its body.

## ADDED Requirements

### Requirement: The cascade task leaves non-cascade files alone and runs no dependency code

`task deps:cascade` SHALL NOT edit any of these:

- `.cascade-frozen` or `.cascade-hold`;
- `release-please-config.json`, `.release-please-manifest.json` or `CHANGELOG.md`;
- anything under `.github/`;
- `.opm-docs-version` or `docs-kit.cue`;
- any `language.version`;
- `hack/fixtures.sh` or `tests/fixtures/fixtures.go`.

It SHALL NOT publish, seed a real registry or push. It SHALL NOT build or run a program that links a moved Go dependency: the only Go program it runs is the `opm` it builds, before any pin moves, from the merge base with `CASCADE_BASE` (default `origin/main`), never from the work tree, which in merge mode may already carry an earlier run's library move.

It SHALL append a warning to the cascade warnings file, without failing, in each of these cases:

- a CUE upstream that differs from the merge base declares a `language.version` newer than the `cue` version `.github/workflows/pr.yml` installs;
- a new major of a pinned upstream is available.

#### Scenario: Release and settings files stay untouched by a full run

- **WHEN** a run moves every pin
- **THEN** the changed paths include nothing under `.github/`, no release-please file, no `CHANGELOG.md`, no `.cascade-*` file and no `.opm-docs-version`

#### Scenario: A library move runs no library code

- **WHEN** a run moves library to a newer version
- **THEN** the task runs `go get` and `go mod tidy` for it and does not run `hack/docskit-dump` or any other program built from the moved tree

#### Scenario: Merge mode builds opm from the merge base

- **WHEN** a run on a `deps/cascade` branch that already pins a moved library, with `main` merged in, needs a version advance
- **THEN** the task builds `opm` from the merge base's tree, which pins `main`'s library, and runs `go` for nothing else but `get`, `mod tidy` and that build

#### Scenario: A newer language version is a warning

- **WHEN** a moved catalog declares a `language.version` newer than the `cue` that `pr.yml` installs
- **THEN** the pin still moves, and the warnings file names the module, its version and the two language versions
