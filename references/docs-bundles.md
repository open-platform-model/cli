# Docs bundles: build, publish and recover

Covers the cli docs bundle declared in `docs-kit.cue`, the workflows that check and publish it, how
opmodel.dev consumes it, and how to recover a release that has none. The design is in
`docs-kit/DESIGN.md` at the workspace. Prose style is in `docs/STYLE.md`.

## Contents

- What the bundle holds
- Workflows
- What opmodel.dev requires
- Preview and recovery
- Moving the docs pins
- Gates
- Pitfalls
- Check before you finish

## What the bundle holds

`docs-kit.cue` declares one docs-kit bundle, `cli`:

- the command reference, which docs-kit renders from the dump of `hack/docskit-dump` (the cobra command
  tree through `github.com/open-platform-model/docs-kit/cobradump`; with `pins`, the library, core and
  opm-controller versions this build compiles in);
- the authored pages under `docs/site/`;
- in its manifest, the library, core and opm-controller versions the cli pins.

`hack/docskit-dump` is not linked into `opm`. Its test keeps the pins equal to `go.mod`, the library's
the library's schema loader and the controller pin file in `internal/controller/`.

## Workflows

- `.github/workflows/docs.yml` checks the bundle on every pull request (`Docs / check`, which runs the
  dump twice and refuses differing output) and publishes `edge` on every push to `main`.
- `release.yml`'s `publish-docs` publishes a bundle for every release, after `goreleaser` publishes
  it, to `ghcr.io/open-platform-model/docs/cli`.

## What opmodel.dev requires

opmodel.dev anchors a site version on the bundle and pulls the bundles of exactly the versions it
pins. It refuses a pin that has none. So a cli release must not merge until core, the library and
opm-controller have bundles for every version it pins (docs-kit gate G2-pins).

Help text reaches the site only through a release. A revision applies Markdown and comment changes
only.

## Preview and recovery

- Preview: `task docs:bundle` (builds into `out/cli/` with the pinned `opm-docs`; `task
  tools:opm-docs` installs it into `.bin/`) or `opm-docs serve`.
- Check: `task docs:bundle:check` builds and lints into a temporary directory, after `task
  docs:pins:check`.
- A published release with no bundle: `gh workflow run docs.yml --ref main -f mode=release -f
  tag=vX.Y.Z`. Only tags whose tree has `hack/docskit-dump` work; `v1.0.0-beta.5` and earlier never
  get one.
- A page fix for a released bundle: dispatch by hand with `mode=revision` and `-f fix=<40-hex sha>`
  (cli#282).

## Moving the docs pins

`.opm-docs-version` and every `publish.yml@` ref name the same docs-kit release and move in one PR,
after opmodel.dev runs that release. Dependabot ignores both, and the
`github.com/open-platform-model/docs-kit/cobradump` require too (it ignores
`github.com/open-platform-model/*`), so each moves by hand.

## Gates

- `task docs:pins:check` (offline) refuses a `publish.yml@` ref that names another release than
  `.opm-docs-version`.
- The release-pin gate (`task deps:release-check`, run on release PRs) refuses a release whose docs
  pins lack a bundle in `ghcr.io/open-platform-model/docs`. The lookup is
  a script in `.github/scripts/`.
- `pr.yml`'s `Lint` job also runs that script as a warning (`--warn --moved-from <base>`) on every PR
  that moves the library or `PinnedControllerVersion`, the cascade PR included, so a missing bundle
  shows up before the release PR.

## Pitfalls

- Expecting a help text change on the site before a release: it needs one.
- Moving `.opm-docs-version` without every `publish.yml@` ref: the pins check fails.
- Merging a release PR before the upstream bundles exist: opmodel.dev refuses the pin.
- Waiting for Dependabot to move docs-kit pins: it ignores them.
- A dump that differs between two runs: `Docs / check` fails. Remove the nondeterminism.

## Check before you finish

- [ ] `task docs:bundle:check` passes.
- [ ] Help text and flags read as the command reference (what happened, how to fix a failure).
- [ ] A docs pin move changed `.opm-docs-version` and every `publish.yml@` ref together.
- [ ] A new page is ASCII-safe and follows `docs/STYLE.md`.
