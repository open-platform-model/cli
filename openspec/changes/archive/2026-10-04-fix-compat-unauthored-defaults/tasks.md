## 1. Compare only authored defaults

- [x] 1.1 Add the failing cases to `TestCheck` in `internal/compat/compat_test.go`: the catalog role case (`subjects!: [...#S] & [_, ...]` to `subjects?:` with `#S` a disjunction of structs), the `[...string] & [_, ...]` probe under `!` to `?`, a plain `[...string]` made optional, and `[...string]` narrowed to `[...string] & [_, ...]` (want `domain narrowed` only); verify they fail on the current `checkDefaults` with `default changed`
- [x] 1.2 Add the negative cases that must keep reporting: `*["a"] | [...string]` to `*["b"] | [...string]` (`default changed`), to `[...string]` (`default removed`), `*[] | [...string]` to `*["a"] | [...string]` (`default changed`), and a default reached through a definition reference; verify they pass after the fix (the removed case reported `default changed` instead of `default removed` before it)
- [x] 1.3 Add `authoredDefault` and `implicitListDefault` in `internal/compat/compat.go`, call `authoredDefault` from `checkDefaults` on both sides, and update the `checkDefaults` doc comment; verify `go test ./internal/compat/...` passes with every pre-existing case unchanged
- [x] 1.4 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `fix(compat): compare only authored defaults`

## 2. Review fixes

- [x] 2.1 Compare authored defaults inside an open list's fixed elements (`openListDefaults`, `walkDefaults`), with test cases for a removed top-level element default, a removed and a changed nested element default, an unchanged one under a marker change, and an added one; verify the three reporting cases fail without the element walk
- [x] 2.2 Replace the simplified catalog case with catalog_opm's literal `#RoleSchema` shape, both directions of the subjects marker
- [x] 2.3 Add the fixed-element rule and its scenario to the Compatibility Comparison requirement, in the delta spec and the synced main spec
- [x] 2.4 File the remaining marker sensitivity of written defaults as cli issue #301 and state it in proposal.md (Why)
