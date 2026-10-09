## Why

`QUICKSTART.md` in the repo root cannot be followed at `origin/main` (d5f730c7):

- Its instance steps build, apply, inspect and delete `examples/instances/jellyfin`, `garage` and `mc_java_fleet` (lines 225-335 and 358). `examples/instances/` holds only `podinfo`.
- Line 121 says `cd my-app` after `opm mod init example.com/modules/my_app@v0` created `./my_app/`; lines 146-158 and 190 pass `./my-app` too.
- Line 9 asks for Go 1.25+. `go.mod` says `go 1.26.0`.

The file is also a second quickstart. The site quickstart (`docs/site/start/quickstart.md` in the `opm` repo, which site pages link as `/docs/start/quickstart/`) walks the same path: `opm config init`, `opm module init`, `opm module build`, an instance of a published module, `opm operator install --crds-only`, `opm instance apply`, `status` and `delete`. It was run end to end on 2026-10-03 and carries a note that says so. `docs/STYLE.md` already rules where such a page lives: "End-user quickstarts (those belong in `opm/docs/site/`)".

## What Changes

- `QUICKSTART.md` becomes a short pointer: the site quickstart by the GitHub URL of its source page (design.md, "Which address the pointer carries"), the install page in `docs/site/start/install-the-cli.md`, the build commands in `AGENTS.md`, and `README.md`.
- Nothing else. No file links to `QUICKSTART.md` outside `openspec/changes/archive/` (checked with a case-insensitive search for `quickstart` over the tree), so the file keeps its path and no link changes.

**Not in this change:**

- Example instances. None is added to make the old text true.
- `docs/site/` pages. What the old file said beyond the site quickstart already has a home: build from source in `docs/site/start/install-the-cli.md`, `--skip-unprovided` in `docs/site/diagnostics/unprovided-contracts.md`, the registry variables, the platform precedence and the kind tasks in `AGENTS.md`.
- Any Go code, command, flag or test.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. One repo-local Markdown file changes, so no spec-level behaviour changes. `.openspec.yaml` sets `skip_specs: true`.

## Impact

- **SemVer: PATCH class after GA.** During beta it ships no release by itself: the commit type is `docs`, which release-please hides (`AGENTS.md`, "Documentation And Output Conventions").
- No command, flag, package, output or exit code changes.
- `QUICKSTART.md` is not under `docs/site/`, so the docs bundle and the site do not change.
- A reader who used the old file for the `--skip-unprovided` text or the platform precedence finds them in the files named above; the pointer does not link each of them.
