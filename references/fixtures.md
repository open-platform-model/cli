# Fixtures and templates: publish, seed and bump

Covers the test fixture modules on `testing.opmodel.dev`, the PR jobs that seed them, how a fixture
bump moves, and the rule for the official templates under `templates/`. It does not cover the registry
mapping itself; that is the workspace `references/registry.md`.

## Contents

- Where fixtures live
- How CI resolves them
- Bumping a fixture
- Templates
- Pitfalls
- Check before you finish

## Where fixtures live

`tests/fixtures/modules/*` declare `testing.opmodel.dev/modules/cli/<name>@v0`. Each carries an
`identity/` package as the single source of its path and version. Merge to `main` publishes them to
GHCR through `.github/workflows/publish-fixtures.yml`, which runs `opm module publish`
(`hack/fixtures.sh publish`): the same pipeline and gates the official templates go through.

Never give a fixture an `opmodel.dev/*` path. CUE routes by longest prefix, so a fixture there drags
core and the catalogs onto whatever registry serves the fixture.

Go programs read a fixture coordinate through `tests/fixtures/fixtures.go`, never a literal.
`hack/fixtures.sh` and `tests/fixtures/fixtures.go` are byte-identical copies of opm-operator's. Edit
both, then run the workspace root `task fixtures:lint`.

## How CI resolves them

PR CI never waits for GHCR.

- The `fixtures` job in `pr.yml` runs `hack/fixtures.sh check`: every gate, plus a rule that a fixture
  changed in the PR carries a version GHCR does not hold yet. It then runs `hack/fixtures.sh seed`
  into a job-local registry and runs render parity against it with the mixed mapping
  (`testing.opmodel.dev` local, everything else GHCR).
- The `unit` and `e2e` jobs in `pr.yml` and the `unit` job in `ci.yml` run the same `seed` into their
  own job-local registry and resolve `testing.opmodel.dev` from it under the same mixed mapping. The
  examples and the e2e testdata pin the tree's fixture version, and GHCR holds that version only
  after `publish-fixtures.yml` runs.
- `task test:fixtures` reproduces the PR job locally.

## Bumping a fixture

A bump is `opm module version set` on the fixture, plus the `cue.mod` pins in
`tests/e2e/testdata/operator-owned` and `examples/`. `task -x deps:cascade` does both, once per PR,
when it moves the fixture's catalog or core.

The consumers' core and catalog pins follow the fixture's. CUE keeps a dependency a consumer already
lists, so the `fixtures` job checks it with `hack/fixtures.sh consumers`. `FIX=1` against a seeded
registry writes the fix.

## Templates

`templates/` holds the official template module trees (`minimal`, `standard`, `advanced`). They are
real CUE modules at `opmodel.dev/templates/<name>`, published by release CI through
`opm module publish`. `task -x deps:cascade` moves their catalog and core pins and advances each
changed template's version once per PR.

Rule: any change to a template's files, its `cue.mod` pins and comments included, needs
`opm module version set` on that template in the same change, direct pushes included. The version
must be stable and above the template's highest published one. Published versions are immutable.

`.github/scripts/publish-templates.sh` enforces it. It fails the `template-gates` PR job and the
release run, before any publish (so the release stays a draft), when a template's tree:

- differs from the artifact GHCR holds at its declared version;
- holds a file its module zip omits (a symlink, for one);
- declares a prerelease or an older version.

A brand-new template needs one manual step. After its first publish the owner makes its GHCR package
(`opmodel.dev/templates/<name>`) public in the package settings, because `opm module init` and the
gate both read it anonymously. Until then every release fails for that template.

Never type a `templates/` edit as `docs`. Pins in `templates/*` are shipped, so a bump is
`fix(deps)`. Bumps of `examples/*`, `tests/fixtures/*` and `hack/platform/` are `test(fixtures)`.

## Pitfalls

- A fixture under `opmodel.dev/*`: move it to `testing.opmodel.dev/*`.
- Editing a template comment without a version bump: the release fails before publish.
- Editing only one copy of `hack/fixtures.sh` or `fixtures.go`: `task fixtures:lint` fails.
- A fixture bump without its consumers' pins: run `hack/fixtures.sh consumers`.
- A new template left private on GHCR: the owner must make the package public.

## Check before you finish

- [ ] The fixture or template version moved with `opm module version set`.
- [ ] `task test:fixtures` passes for a fixture change.
- [ ] The workspace `task fixtures:lint` passes after an edit to a shared copy.
- [ ] The commit type matches the pin class.
