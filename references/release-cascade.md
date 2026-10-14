# Release cascade: the deps task and its receivers

Covers `task -x deps:cascade`, the `deps/cascade` PR, the receiver workflow, the downstream notify and
the cascade gates. Secrets, permissions and the SHA pin are in `cascade-wiring.md`. The design (release
order, pin classes, gates G1 to G3) is in `RELEASING.md` at the workspace root, sections "The
cascade", "Pinning the cascade code", "Moving the cascade pin" and "Stop switches".

## Contents

- What the task moves
- Running it
- The receiver workflow
- Notify downstream
- Gates
- Pitfalls
- Check before you finish

## What the task moves

`task -x deps:cascade` moves every upstream pin the cli ships or tests against, to the newest
published versions, in the working tree only:

- the library;
- the operator module pin, through `hack/operator-pin`: the newest `opmodel.dev/modules/opm_operator`
  release whose operator `MAJOR.MINOR` is not above the cli's (`hack/operator-pin select` reads each
  candidate, so the lane needs Go and anonymous GHCR read);
- the opm catalog and core in `templates/`, `hack/platform`, `hack/kind-platform.yaml`, `examples/`,
  the podinfo fixture and the test trees.

It exits 0 on a change and 3 on nothing to do. It runs no code from a moved dependency. The only Go
programs it builds are `opm` and `hack/operator-pin`, from an export of the merge base and never the
work tree (in merge mode the work tree may already pin a library an earlier run moved). So the
docs-bundle check lives in the PR's CI, not in the task.

## Running it

- Always run `task -x deps:cascade`, `task -x deps:cascade:title` and `task -x deps:cascade:body`.
  Without `-x`, go-task turns exit 3 into 201.
- `task -x deps:cascade:title` and `:body` describe the result for the rolling `deps/cascade` PR.
- `task -x deps:cascade:test` is the scenario test: the offline set runs in CI's `Lint` job, the full
  set in `cascade-task.yml`.
- All but `:test` need the shared resolver. Check out `open-platform-model/.github` beside this repo,
  or set `CASCADE_RESOLVER`.
- `.cascade-frozen` and `.cascade-hold` steer it.
- Do not run the workspace root `deps:update`, `deps:update:templates` or `deps:pins:*` against the
  cli. They move core past the catalog's pin and ignore those files, until Phase 5 rewires them.

## The receiver workflow

`.github/workflows/deps-cascade.yml` is the receiver. It runs on an upstream's `upstream-released`
dispatch, on a daily sweep at 06:17 UTC, or by hand with `dry_run`.

- Its `cascade` job calls the reusable `cascade-receive.yml` in `open-platform-model/.github`. Its
  `compute` job runs this task (never a local script). Its `gates` job posts the release-PR gates.
  Neither holds a secret.
- Its own `publish` job runs the `cascade-publish` action, which pushes `deps/cascade` and its PR.
- Every run is a dry run unless the repository variable `CASCADE_DRY_RUN` is exactly `false`.
  `publish`'s `if:` and the action's `dry-run` input both read it.
- A gates-only run (sent by `Cascade gates` for a release PR, running that head's task in `compute`)
  never publishes. `publish`'s `if:` reads `inputs.gates_only`, and the action's required
  `gates-only` input fails before the token mint unless it is `false`.

## Notify downstream

After `goreleaser` publishes a release, `release.yml`'s `Notify downstream` runs the `cascade-notify`
action. It dispatches to catalog_opm and opm-operator, which pin the cli as their release tool.
`CASCADE_NOTIFY=off` stops it.

## Gates

`.github/workflows/cascade-gates.yml` posts `cascade/freshness` and `cascade/settled` on PRs into
`main`, as warnings set by `CASCADE_G2_MODE` and `CASCADE_G3_MODE`.

Bump typing for the PR: the workspace `references/cascade.md`. In the cli, a shipped pin in
`templates/*` is `fix(deps)`. Bumps of `examples/*`, `tests/fixtures/*` and `hack/platform/` are
`test(fixtures)`.

## Pitfalls

- Running `task deps:cascade` without `-x`: a nothing-to-do exit shows as 201.
- Running the task with no `.github` checkout and no `CASCADE_RESOLVER`: it cannot resolve.
- Reading a dry run as a published PR: check `CASCADE_DRY_RUN`.
- Using the root `deps:update` on the cli: it ignores `.cascade-frozen`.
- Putting a docs-bundle check into the task: the task runs no moved code.

## Check before you finish

- [ ] The task ran with `-x`, and exit 0 or 3 was read as change or nothing to do.
- [ ] The resolver was available (`.github` beside this repo, or `CASCADE_RESOLVER`).
- [ ] The PR type matches the pin class.
- [ ] The release-pin and docs-pin gates are green before the release PR merges.
