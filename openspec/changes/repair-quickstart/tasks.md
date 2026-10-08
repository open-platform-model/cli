# Tasks: repair-quickstart

One section. No Go file changes, so the gates of this section are `task openspec:check` and `task docs:bundle:check`, in place of `task lint` and `task test`.

## 1. Replace QUICKSTART.md with a pointer

- [x] 1.1 Replace the content of `QUICKSTART.md` with a pointer to the site quickstart, the install page, the build commands in `AGENTS.md` and `README.md`. Verify: the file names no `examples/instances/` path, no `my-app` or `my_app`, and no Go version.
- [x] 1.2 Check every link in the new file. Verify: `docs/site/start/install-the-cli.md`, `AGENTS.md` and `README.md` exist in the tree, `AGENTS.md` has the heading "Build And Dev Commands", and `docs/site/start/install-the-cli.md` links the site quickstart as `/docs/start/quickstart/`.
- [x] 1.3 Check links to the file. Verify: a case-insensitive search for `quickstart` over the tree, outside `openspec/changes/`, finds no link to `QUICKSTART.md`.
- [x] 1.4 `task openspec:check` and `task docs:bundle:check` green, then commit `docs: point QUICKSTART.md to the site quickstart`

## 2. Link an address that answers

- [x] 2.1 Replace the `https://opmodel.dev/docs/start/quickstart/` link in `QUICKSTART.md` with the GitHub URL of the page source on `main`, and spell out Open Platform Model on first use. Verify: a fetch of the URL returns the quickstart page, and `git -C <opm checkout> show origin/main:docs/site/start/quickstart.md` prints the file.
- [x] 2.2 Record the choice in `design.md` and correct the "served at" wording in `proposal.md`. Verify: `openspec status --change repair-quickstart` lists no artifact as open.
- [x] 2.3 `task openspec:check` and `task docs:bundle:check` green, then commit `docs: link the quickstart source page from QUICKSTART.md`
