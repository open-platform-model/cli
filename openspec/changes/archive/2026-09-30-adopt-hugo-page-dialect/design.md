## Context

See proposal.md, "Why". The facts below shape the approach.

- **The pages.** cli owns five leaf pages in four sections: `authoring`, `diagnostics`, `extending` and `reference`. Every one of those sections also holds pages from other repositories, because the site is assembled by section (0018:D8).
- **The dialect.** The contract is `orchestration.md` section 4. The lint is section 4.1: `opm-dialect-lint.sh`, sha256 `dae9717af0c43fc3bdc29a9e730fd171fe7541dab35a9625a35efb1682973c6b`, byte-identical to A's `site/scripts/lint-sources.sh`.
- **Baseline.** On `origin/main` at `93d0bc3` (2026-09-30), the lint over `docs/site` prints five findings and exits 1. Each finding is `<page>:5: sidebar: is Starlight front matter; write weight: N`, one for each of the five pages. This matches the count in `orchestration.md` 4.1 ("cli 5").
- **Dry run.** On 2026-09-30 the recipe below was applied to a copy of that tree (`git archive`). The lint then printed `opm-dialect-lint: OK`. The page-order helper (sha256 `edc62591...`) printed the same output before and after.
- **The RFC.** `docs/rfc/0006-documentation-generation.md` is a Draft RFC. It planned Hugo with Docsy in a separate docs repository. Its line-9 banner, added on 2026-09-24, says the site is built with Astro instead, and that everything about Hugo no longer applies. Only line 9 in the file mentions Astro or Starlight.
- **No code.** No Go code, command, flag, exit code, error message or data flow changes. The design rules in `openspec/config.yaml` about signatures, command syntax, flags, exit codes and example output do not apply to this change.
- **Sources.** `plan-final.md` is the supervisor's plan for the whole set. It is not in this directory. Where this file cites it, the fact it gives is stated here too, and `orchestration.md` is cited instead wherever it carries the same fact.

## Goals / Non-Goals

**Goals:**
- `sh <scratch>/opm-dialect-lint.sh <wt>/docs/site` prints `opm-dialect-lint: OK`.
- The page order is unchanged. The page-order diff between `origin/main` and the branch is empty.
- No cli file describes the retired Astro site as the current one.

**Non-Goals:**
- Building or rendering the site. A renders every page; S workers only lint (`orchestration.md` section 5).
- Committing the lint or the page-order helper to cli. The lint lives in opmodel.dev. The helper is never committed anywhere.
- Changing any sentence of page prose, including planning comments beyond the one stale path.

## Research & Decisions

### Weight values

**Context**: The dialect replaces `sidebar.order` with `weight` (`orchestration.md` section 4). The number could stay the same or be renumbered.

**Explored**: The page-order helper was run over `origin/main` of all six source repos. cli's numbers sit between other repos' numbers in the shared sections:
- authoring: opm 10, catalog_opm 20-21, core 22, catalog_opm 23, **cli 24**;
- diagnostics: library 20-25, **cli 26-27**, opm-operator 40;
- extending: catalog_opm 20-21, **cli 22**;
- reference: catalog_opm 3-5, **cli 6**, opm-operator 7, opm 8.

**Options considered**:
1. Keep each number. The section order across repos is unchanged, and the order diff proves it.
2. Renumber cli's pages from 1 in each section. This looks tidier inside one repo, but it moves cli's pages ahead of other repos' pages in every shared section.

**Decision**: Keep each number. `weight: N` MUST carry the exact `N` that `sidebar.order` had.

**Rationale**: Order is a property of the assembled section, not of one repo. Only option 1 keeps the order diff empty.

### The stale planning-comment path

**Context**: A planning comment in `publish-a-module.md:66` names `opmodel.dev/site/content/docs/reference/cli/index.md`. A renames the nine site-owned section files to `_index.md`, because a section page is `_index.md` (`orchestration.md` section 4, "Files"). `docs/reference/cli/` stays a site-owned section: check 8 in `orchestration.md` section 6 forbids any source page under it.

**Options considered**:
1. Point at `opmodel.dev/site/content/docs/reference/cli/_index.md`: the section file that A keeps.
2. Point at the generated CLI pages. Their location moves later, with the generated-reference work, to `site/.gen/<v>/` (reserved in `orchestration.md` section 6, "Outputs"), so this would go stale again.
3. Drop the path. That loses the brief's pointer to the CLI reference.

**Decision**: Option 1. It is the S1-S6 rule in `plan-final.md` ("Common scope"): a stale `opmodel.dev/site/content/docs/<s>/index.md` path becomes `.../_index.md`.

**Rationale**: The section file is the stable anchor. The generated pages are not.

### The RFC banner

**Context**: Line 9 was true for the Astro site. After A, part of it is wrong: the site is Hugo again. The rest is still wrong: Docsy is not the theme.

**Options considered**:
1. Rewrite the banner only.
2. Delete the banner. The body would then read as current, and its Docsy parts are not.
3. Edit the RFC body. That is a prose rewrite of a historical draft, outside this change.

**Decision**: Option 1. Line 9 becomes exactly this text:

```markdown
> **Superseded in part (2026-09-30):** the site is built in `opmodel.dev` with Hugo and the Hextra theme, vendored as files, not with Docsy through a Hugo module. Everything below about Docsy no longer applies. The `docgen` extraction design still does, and so does the idea of generating definition pages with a Hugo content adapter (`_content.gotmpl`), which is not built yet.
```

**Rationale**: The banner is the only line that claims what the site runs on. `plan-final.md` 1.4, item 2 keeps the `_content.gotmpl` content-adapter idea for the generated reference, so the banner must not call it dead.

### Sections and spike

**Context**: `openspec/config.yaml` Principle VIII calls for one commit per section. Section 1 is a spike when design.md carries an unverified assumption.

**Decision**: Two sections, with the commit subjects that `plan-final.md` names in its S4 row:
1. Every `docs/site` edit: `docs(site): adopt the hugo page dialect`.
2. The RFC banner: `docs(rfc): say the site is built with hugo and hextra`.

There is no spike.

**Rationale**:
- The lint gate runs on the whole `docs/site` tree, so all five weight edits MUST land in the same section. Otherwise the section cannot end green.
- The RFC banner is a separate concern with its own commit.
- The dry run leaves no assumption unverified, so no spike is needed.

### `docs/STYLE.md` stays

**Context**: `plan-final.md` asks each S change to update any repo rule that describes the Starlight or Astro format. In cli, two lines in `docs/STYLE.md` touch the subject:
- `docs/STYLE.md:46`: "Admonitions (`> **Note:**`) are appropriate for gotchas".
- `docs/STYLE.md:60`: end-user quickstarts belong in `opm/docs/site/`.

**Decision**: Neither changes.

**Rationale**:
- Line 46 is cli's rule for repo-local docs. The workspace `STYLE.md` keeps `> **Note:**` for those.
- The site page rules are in the workspace `STYLE.md`, "Site Pages" (`orchestration.md` section 3, I1a, done).
- Line 60 names a directory that still exists.
- The S4 row of `plan-final.md` says this file stays. cli's `AGENTS.md` names no site format, and neither does `openspec/config.yaml`.

## Interface

This change adds nothing to `orchestration.md` section 6. It relies on these names:
- **The dialect contract** (section 4) and **the lint** (4.1). A runs the same lint as `task lint:sources`, and as check 2 inside `task build`.
- **Check 3.** Front-matter validation inside Hugo: title, description, and the rules for type.
- **Check 8.** Reserved prefixes: no source page may sit under `docs/reference/cli/`. cli's reference page is `docs/reference/registry-namespaces.md`, which is outside that prefix.
- **Ordering.** `_partials/sidebar.html` and `_partials/opm/section-children.html` order pages by `weight`, then by title.
- **Source mapping.** `_partials/opm/source.html` maps each page to `/src/cli/docs/site/<path>` for its edit link.
- **Source roots.** `OPM_SRC_CLI`, and `OPM_SRC_WORKTREE=site-src`, which resolves to `WS/cli/.claude/worktrees/site-src` (section 5).

## Risks / Trade-offs

- **[The Astro build degrades]** Between this merge and A's merge, Starlight's content schema on opmodel.dev `main` does not read `weight`, so cli's pages lose their sidebar order there. → Accepted: the site is not live. The supervisor tells the owner (`orchestration.md` section 2, "Known consequences").
- **[The banner runs ahead of opmodel.dev]** In the same window, the RFC banner describes the Hugo site while opmodel.dev `main` still builds Astro. → Accepted for the same reason. The banner states the decision, which A delivers.
- **[The stale path names a file that does not exist yet]** A creates `site/content/docs/reference/cli/_index.md`, and A merges after this change. → Accepted: the path is in a planning brief that is never published (A strips planning comments from every output).
- **[`origin/main` moves before this change merges]** A new cli page in the old dialect may land on `main` before the worker starts, or between verify and the squash merge. If it rides this PR unconverted, it breaks A's section 2 build. → Task 1.1 runs the lint on the fresh tree before any edit. After any branch update, the step-7 lines in `tasks.md` write a new baseline from the fresh `origin/main` in a new, empty directory, and re-run every gate before the push.
  - A `sidebar:` finding gets the same edit as the five pages, and the report names the page.
  - Any other finding is a stop-and-report (`orchestration.md` section 7, step 5).
- **[A release PR opens]** release-please releases `docs` commits in cli, so the merge opens or grows a release PR. → Only the owner merges it. The supervisor tells the owner (`orchestration.md` section 9).
- **[A heavy gate gets run]** cli's `task check` and `task test` run the integration and e2e suites against the kind cluster. → The gates for this change are `task openspec:check`, `task vet` and `task test:unit` (`orchestration.md` section 10). The worker MUST NOT run `task check` or `task test`.
- **[A new page in the old dialect after merge]** Once S1-S6 merge, such a page breaks A's build (`orchestration.md` section 11, trap 27). → No in-flight cli change edits `docs/site` today (`plan-final.md` 1.5 lists them). The lint names the file and line. Fix the page, never the lint.
- **[Branch updates]** The history-rewrite hook refuses the other way of updating a branch (trap 34). → Update the branch with `git -C <wt> merge --no-edit origin/main`, re-run the gates as the step-7 lines in `tasks.md` say, then push plainly (`orchestration.md` section 7, step 7).

## Migration Plan

1. Implement on branch `docs/adopt-hugo-page-dialect` in `WS/cli/.claude/worktrees/adopt-hugo-page-dialect`.
2. Verify and report, then wait (`orchestration.md` section 7).
3. On the supervisor's go, archive, push and open the PR (`orchestration.md` section 7, step 7). The PR title is the section 1 subject, `docs(site): adopt the hugo page dialect`.
4. The supervisor squash-merges this change after A section 1 is green, and before A section 2.

**Rollback**: revert the squash commit. This is safe only before A builds the real sources: a revert after that puts `sidebar:` back and fails A's lint.

## Open Questions

None.
