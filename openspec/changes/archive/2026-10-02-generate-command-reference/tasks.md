Local gate, named in every commit task below: `task fmt`, `task lint`, `go vet ./...`, `task test:unit`, `task openspec:check` and `task docs:reference:check`. Site verification (section 1): the opmodel.dev `build` and `lint:sources` tasks, run in Docker with `OPM_VERSIONS=v1.0=/src` and `OPM_SRC_CLI` set to this worktree, against the opmodel.dev branch that drops its placeholder CLI Reference page and its RESERVED check.

## 1. Generate the reference

- [x] 1.1 Add `internal/cmdref`: `Generate` (section page and one page per top-level command, entry parts in the fixed order of design.md Decision 1), the `Long` parser and prose formatter (Decision 2), `Compose` and `Sync` with the marker comments and the authored remainder (Decision 3)
- [x] 1.2 Add `hack/cmdref`, which writes the pages or, with `-check`, reports stale pages and exits 1 (Decision 4); it adds cobra's default completion command before rendering
- [x] 1.3 Unit tests: the `Long` parser on tab-indented and unindented sources, the prose formatter, entry parts on a synthetic tree (hidden, deprecated, aliases, inherited and global flags, home-directory defaults), determinism, the page dialect over the real command tree, and `Sync` (check writes nothing, authored text survives, orphans, authored pages untouched)
- [x] 1.4 Add `task docs:reference` and `task docs:reference:check` and run the check from `task check`; generate and commit `docs/site/reference/cli/`; note both tasks and the packages in `AGENTS.md`
- [x] 1.5 Verify on the site build: `lint:sources` and `build` pass, and the built pages under `site/public/v1.0/docs/reference/cli/` show every entry, the flag tables, tagged code blocks and working entry links
- [x] 1.6 Local gate green, then commit `docs(reference): generate the opm command reference from the cobra commands`

## 2. Check the reference in CI

- [x] 2.1 Add the `command-reference` job ("Command Reference (current)") to `.github/workflows/pr.yml` and `.github/workflows/ci.yml`: checkout, setup-go, setup-task, `task docs:reference:check`, actions pinned by SHA as in the other jobs; verify with actionlint
- [x] 2.2 Local gate green, then commit `ci(docs): fail when the command reference is stale`

## 3. Archive

- [x] 3.1 Archive the change on this branch (`openspec archive`), so the archive rides the implementing PR; give the new main spec a Purpose; verify `task openspec:check` passes, then commit `docs(openspec): archive generate-command-reference`

## 4. Review fixes

- [x] 4.1 Join code words separated by one space into one code span, and render a preformatted block whose every line is one bullet or one numbered item as a Markdown list (design.md Decision 2); unit tests for both; regenerate; local gate green, then commit `docs(reference): join adjacent code spans and render help-text lists as lists`
- [x] 4.2 Drop the enhancement citations from the help text of `opm module template` (`0011:D25`, kept as a code comment at the symbol), `opm platform check` (`0015:D5`) and `opm module init` (the bare `D20`), and fix "a instance.cue file" in `opm module apply`; regenerate; local gate green, then commit `fix(cli): drop enhancement citations and a typo from command help`
