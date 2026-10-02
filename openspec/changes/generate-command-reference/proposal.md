## Why

The site's CLI Reference section (`/docs/reference/cli/`) is a placeholder that tells readers to run `opm <command> --help`. Pages in the cli's own `docs/site/` already link there, and nothing documents the commands and flags on the site.

The owner decided on 2026-10-02 (workspace `STYLE.md`, "Site Pages") that every reference fact derivable from cobra is generated in the repository that owns the source, committed under `docs/site/reference/`, and guarded by a check that fails when the committed output is stale. A generated block sits between marker comments, apart from authored text, and a generated entry states only what its source proves. The cli owns the cobra commands, so the generator and its output live here; opmodel.dev stops owning the section and builds it from this repository's `docs/site/` like every other page.

cobra's own `doc.GenMarkdownTree` is not usable as it stands: it writes relative `.md` links, untagged code fences and one file per command, all of which the page dialect forbids.

## What Changes

- **New package `internal/cmdref`** renders the command tree into site pages: the section page `docs/site/reference/cli/_index.md` (the root's description, its usage and the global flags) and one reference page per top-level command (`opm-module.md`, `opm-instance.md`, ...), each holding one entry per command in that subtree. An entry's parts come in a fixed order: summary, usage (with aliases), description, flags, examples, subcommands. Hidden, deprecated and help-topic commands and cobra's `help` command are left out, as `opm --help` leaves them out; cobra's default `completion` command is included, because the CLI shows it.
- **New program `hack/cmdref`** writes the pages, or with `-check` writes nothing and exits 1 naming each stale page.
- **New tasks** `task docs:reference` (regenerate) and `task docs:reference:check` (fail when stale); `task check` runs the check.
- **New CI job** `command-reference` ("Command Reference (current)") in `pr.yml` and `ci.yml` runs `task docs:reference:check`.
- **Committed output**: the generated pages under `docs/site/reference/cli/`.

No command, flag or help text changes. The binary is unchanged.

## Capabilities

### New Capabilities

- `command-reference`: the generated site reference for every `opm` command, its staleness check and where the check runs.

### Modified Capabilities

None. The `pr-workflow` and `ci-workflow` specs enumerate their jobs and already omit the `openspec` job; the new job's requirement lives in `command-reference` instead of widening that drift.

## Impact

- New: `internal/cmdref/`, `hack/cmdref/`, `docs/site/reference/cli/*.md`, two Taskfile tasks, one job in each of `pr.yml` and `ci.yml`.
- `go.mod`: `github.com/spf13/pflag` moves from indirect to direct (same version).
- opmodel.dev removes its placeholder `site/content/docs/reference/cli/_index.md` and its RESERVED check for `/docs/reference/cli/` in its own change; until that lands, a site build that includes this branch reports both pages for `/docs/reference/cli/`.
- SemVer: none. Nothing in the shipped binary changes; the commits are typed `docs` and `ci`, which cut no release. The site builds the cli's pages from the branch head, so no release is needed for the pages to publish.
