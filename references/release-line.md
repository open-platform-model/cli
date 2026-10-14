# Release line: beta, breaking changes and release PRs

Covers the cli's `1.0.0-beta.N` line, how to change it, how a break is typed during beta, and the
gates a release PR passes. The cascade that moves pins is in `release-cascade.md`. Tag and recovery
rules are in the workspace `references/releases.md`; the design is in `RELEASING.md` at the workspace
root.

## Contents

- The line
- Changing the line
- Beta promise and breaking changes
- Beta skew rule
- Release PR gates
- Pitfalls
- Check before you finish

## The line

The cli releases on the `1.0.0-beta.N` line (`prerelease-type: beta` in `release-please-config.json`).
The cli embeds the library on the core v2 line. The cluster Platform CR surface is scalar
subscriptions (`{enable?, version!}`, registry keys carry the catalog's major suffix), and module
identity is read verbatim from core-v2 metadata (`metadata.modulePath` is the complete registry
address).

release-please hides `chore`, `test`, `ci`, `build` and `docs`. It releases `feat`, `fix`, `perf`,
`revert`, `deps` and `refactor`.

## Changing the line

A line change is a `release-as` key in `release-please-config.json`, landed by a normal PR. Remove it
before any later release PR merges, because release-please re-applies it on every run while it stays.

The key takes effect only once a releasable (visible-type) commit has landed since the last release.
A hidden-type key PR alone opens no release PR. So the key PR is itself of a releasable type, or it
lands after a releasable commit since the last release.

A `Release-As:` footer never reaches `main` under the `BLANK` squash message (`RELEASING.md`, section
"Owner settings").

## Beta promise and breaking changes

From its first beta the cli, with `opmodel.dev/core@v2`, the library and opm-operator, is on the path
to GA.

- A break is allowed in beta only as a `!` in the PR title (`feat!:`). That title is the squash
  commit title and the CHANGELOG entry.
- Until the owner merge settings land, merge by squash only, and give a one-commit PR's commit subject
  the same `!`.
- The migration note goes in the PR body, and in the user docs when users need it to upgrade.
- Whoever merges the release PR edits its `CHANGELOG.md` by hand to carry the note, as the last step
  before merging. Redo the edit if anything landed on `main` since: release-please rebuilds the
  release PR and drops it.
- A break advances the `-beta.N` counter. It never moves the module path to a new major.
- Stable lines (`opmodel.dev/catalogs/opm@v4` and the module fleets) keep normal SemVer: a break is a
  new major. A core beta break that would force a catalogs/opm major needs owner sign-off.
- GA drops the suffix: `prerelease: false` plus a visible carrier commit per package, in dependency
  order.

## Beta skew rule

An opm-operator `feat!` that the released cli cannot drive merges only after the cli release that can
drive it. No minor or major hop during beta: never a `release-as` of `1.1.0-beta.1` or beyond. The
operator ceiling gate compares MAJOR.MINOR only and refuses nothing inside the `1.0` line.

## Release PR gates

- Release PRs must pass the release-pin gate: `task deps:release-check`, run in CI's `Lint` job on
  release-please branches. Its step 5 runs `go run ./hack/operator-pin --check`, so the pinned
  operator module must be published and `PinnedOperatorVersion` must be the operator release it
  states.
- The required check `E2E (kind, embedded operator)` runs the cluster-backed e2e suite on every
  release PR (`e2e-cluster.md`).
- The release-pin gate also refuses a release whose docs pins lack a bundle (`docs-bundles.md`).
- See `RELEASING.md`, section "Gates".

A broken release is fixed by the next release, with a Go `retract` when the broken version must not be
selected. A published GitHub Release is never deleted. A draft release is finished with the release
workflow's manual run. Only the owner deletes a duplicate or tagless draft (runbook in
`.github/workflows/release.yml`).

## Pitfalls

- Leaving the `release-as` key after the release: the next release PR re-applies it.
- Landing the `release-as` key as a hidden type with nothing releasable before it: no release PR
  opens.
- Typing a break `feat:` or `fix:` without `!`: it ships as a normal change.
- Writing `Release-As:` in a commit footer: it does not reach `main`.
- Editing the release PR's CHANGELOG and then merging other work: the edit is dropped. Redo it.

## Check before you finish

- [ ] A `release-as` key, if added, is on a releasable commit and has a removal planned.
- [ ] A break has `!` in the PR title and a migration note in the PR body.
- [ ] The release PR's CHANGELOG carries the note after the last rebuild.
- [ ] No `release-as` moves the line to `1.1.0-beta.1` or beyond.
