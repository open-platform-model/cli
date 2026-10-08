## Context

`QUICKSTART.md` in the repo root has steps that cannot work at `origin/main` (proposal.md, "Why"). The `opm` repo holds a tested quickstart that the documentation site publishes. This change touches one Markdown file. It has no command syntax, flag, exit code, error message or data flow to design.

## Goals / Non-Goals

**Goals:**

- A reader who opens `QUICKSTART.md` on GitHub reaches a quickstart whose steps work.
- Every link in the file resolves today.

**Non-Goals:**

- A second maintained quickstart in this repo.
- New example instances, new site pages, or a fix for `README.md`.

## Research & Decisions

### Repair the file or retire it

**Context**: Three defects make the file unusable: example instances that are not in the tree, `my-app` against a scaffold named `my_app`, and Go 1.25+ against `go 1.26.0` in `go.mod`.

**Explored**: The old file line by line against the tree at d5f730c7, and `docs/site/start/quickstart.md` at `origin/main` of the `opm` repo, step by step.

**Options considered**:

1. Repair: rewrite the instance steps for `examples/instances/podinfo` and fix the two other defects. The file stays a second quickstart that nobody runs end to end, so it goes stale again.
2. Retire: replace the file with a pointer. One quickstart remains, the one with a dated end-to-end test note (2026-10-03).

**Decision**: Option 2.

**Rationale**: The site quickstart covers the same path: `opm config init`, `opm module init`, `opm module vet` and `build`, `opm instance init`, `opm operator install --crds-only`, `opm instance apply`, `status`, `diff` and `delete`. `docs/STYLE.md` MUST be followed here, and it places end-user quickstarts in `opm/docs/site/`. What the old file said beyond that path already has a home (proposal.md, "Not in this change").

### Which address the pointer carries

**Context**: The pointer is only as good as its link. The first version linked `https://opmodel.dev/docs/start/quickstart/`. A review fetched it twice on 2026-10-08 and got a refused connection.

**Explored**: The `opmodel.dev` repo at `origin/main`. Its `README.md` says the site is published at an interim GitHub Pages address until the Cloudflare deploy, and that this address goes away. It also says every version lives under `/<version>/`, with `/latest/` pointing at the default version. `/docs/start/quickstart/` is the link form of site page sources, not a public URL.

**Options considered**:

1. `https://opmodel.dev/latest/docs/start/quickstart/`: the lasting address, and dead until the site moves to that host.
2. The interim GitHub Pages address: answers today, and breaks when the site moves.
3. The GitHub URL of the page source on `main`: answers today (fetched 2026-10-08) and after the move. The workspace `STYLE.md`, "Cross-Repo Links", asks for this form. GitHub shows the page's front matter as a table, and the page's own `/docs/...` links do not resolve there; the steps and their outputs read as written.

**Decision**: Option 3.

**Rationale**: It is the only address that resolves both now and after the host move, so the file SHALL NOT need a second edit to stay correct. Moving the link to the site address once the site is on its lasting host is a one-line follow-up for the owner to schedule.

## Risks / Trade-offs

- A reader lands on a source page instead of the rendered site. Mitigation: the first sentence says the page is published on the documentation site.
- `main` of the `opm` repo can describe a newer release than the reader installed. Mitigation: the page names the release it was tested with in "Before you begin".
