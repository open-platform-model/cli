# Commit types and release impact

Covers which Conventional Commits type to use in the cli, what each type releases, and the type of a
pin bump by its class. The workspace `references/tooling.md` and `references/cascade.md` hold the
general rules.

## Contents

- Types and releases
- Pin classes
- Templates
- Pitfalls
- Check before you finish

## Types and releases

Release-please decides the release from the type (`release-please-config.json`).

| Type | Releases | In CHANGELOG |
| --- | --- | --- |
| `feat`, `fix`, `perf`, `revert`, `deps` | yes | yes |
| `refactor` | yes | yes (`hidden: false`) |
| `docs`, `test`, `ci`, `build`, `chore` | no | hidden |

A window with only hidden types opens no release PR. A `release-as` key on such a window does nothing
until a releasable commit lands.

## Pin classes

| Pin | Type |
| --- | --- |
| Shipped pin in `templates/*` | `fix(deps)` |
| `examples/*`, `tests/fixtures/*`, `hack/platform/` | `test(fixtures)` (no release) |
| A cascade PR that mixes classes | One `fix(deps)` commit, only from a `deps-cascade` PR |

## Templates

Never type an edit under `templates/` as `docs`. A template publishes only with a releasing commit,
and the change also bumps the template version with `opm module version set`.

## Pitfalls

- `docs:` on a template edit: the template never publishes.
- A break during beta is a `!` in the PR title: `feat!:`. With a scope, `feat(scope)!:`.
- `feat!(scope):`: the `!` goes after the scope, `feat(scope)!:`, or release-please misses the break.
- Lowering a cascade PR type to make it pass.

## Check before you finish

- [ ] The type matches what changes for the user, and the pin class table.
- [ ] A template edit is not `docs`.
