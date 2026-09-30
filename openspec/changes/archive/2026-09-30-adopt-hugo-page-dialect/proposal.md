## Why

opmodel.dev is moving from Astro, Starlight and the Black theme to Hugo and Hextra v0.13.0. That move is change A, `port-site-to-hugo-hextra`, in opmodel.dev. The owner decided that the source pages are rewritten to the Hugo page dialect now, with no compatibility layer (`orchestration.md` section 4). A's source lint runs before every site build and rejects the old dialect.

On `origin/main` (2026-09-30), that lint reports five findings in cli's `docs/site`. All five pages set their order with Starlight's `sidebar:` / `order:` front matter. Until these pages use `weight`, A cannot build the real sources.

`docs/rfc/0006-documentation-generation.md` line 9 is also wrong once A lands. Its banner says the site is built with Astro, Starlight and Black, not Hugo, and that "everything below about Hugo ... no longer applies".

## What Changes

- **Weights.** Five pages change `sidebar:` plus `  order: N` into `weight: N`, keeping the same number:
  - `docs/site/authoring/publish-a-module.md`: 24
  - `docs/site/diagnostics/publish-refusals.md`: 26
  - `docs/site/diagnostics/unprovided-contracts.md`: 27
  - `docs/site/extending/publish-a-catalog.md`: 22
  - `docs/site/reference/registry-namespaces.md`: 6
- **Stale path.** One planning comment names a page by its old path. In `docs/site/authoring/publish-a-module.md:66`, `opmodel.dev/site/content/docs/reference/cli/index.md` becomes `opmodel.dev/site/content/docs/reference/cli/_index.md`. A renames that site-owned section file.
- **RFC banner.** The banner in `docs/rfc/0006-documentation-generation.md:9` now says the site is built in `opmodel.dev` with Hugo and a vendored Hextra theme. It also says which parts of the RFC still apply.
- **No other edits.** The lint found nothing else in cli's `docs/site`: no `.mdx` file, no `index.md`, no Starlight aside, no import, no component tag, no shortcode, no image, no bad link and no untagged code fence. So this change renames no file, adds no alert and touches no link or fence.

**Not in this change:**
- Prose edits and new pages.
- A source-repo CI lint. That is a 0018:D13 follow-up.
- `docs/STYLE.md:46`. It is cli's rule for `> **Note:**` in repo-local docs. The site page rules live in the workspace `STYLE.md`.
- How the prose in `registry-namespaces.md` cites enhancement decisions. It writes the id and the decision with a space, not in the colon form.
- Any Go code, command or test.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change edits `docs/site` pages and one RFC banner only, so no spec-level behaviour changes. `.openspec.yaml` sets `skip_specs: true`.

## Impact

- **SemVer: PATCH class.** No command, flag, package, output or exit code changes. No command or package is affected. release-please releases `docs` commits in cli (see `AGENTS.md`, "Documentation And Output Conventions"), so the merge opens or grows a release PR. Only the owner merges that PR.
- **Touches.**
  - `docs/site/**`: five pages, plus any cli page in the old dialect that reaches `origin/main` before this change merges (design.md, "Risks").
  - `docs/rfc/0006-documentation-generation.md`.
  - This change's own directory: ticked tasks, and the archive.
- **Depends on.**
  - Starts when: this change is planned on `main`, with its `orchestration.md` copy.
  - Merges when: verify is green, after A section 1 is green, and before A section 2. A's fixture build is the evidence that the dialect renders.
- **Downstream.**
  - A reads cli's pages from the supervisor's `site-src` worktree once S1-S6 have merged, and renders them in its section 2.
  - Between this merge and A's merge, the Astro build on opmodel.dev `main` ignores `weight`, so cli's pages lose their sidebar order there. The site is not live, so this is accepted (`orchestration.md` section 2, "Known consequences").
- **Related 0018 decisions.**
  - 0018:D7: a page declares a title, a one-line description and a type, and nothing else is required.
  - 0018:D7:R1: the four types.
  - `weight` is also the order key of `#Page` in 0018's contracts.

  The change sets no delivery claim (`orchestration.md` section 1).
- **Complexity (Principle VII).** None is added. Front-matter lines are removed and one line is rewritten.
