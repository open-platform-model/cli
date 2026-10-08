# Tasks: cite-walkthrough-decisions-by-adr

No design.md: the change replaces one citation in two places and makes no technical choice.

The walkthrough-id check used below, the library's check from its change `record-the-walkthrough-decisions` with the cli's wording ("beta.1 plan walkthrough") added. A line it prints cites a walkthrough decision by its id, by the owner, by the walkthrough or by a walkthrough date. Two lines it prints are not walkthrough citations and stay (task 2.1):

```
git grep -nE "[Oo]wner('s)? (walkthrough )?([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b|walkthrough ([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b|(walkthrough|checklist) items? [a-j][1-5]\b|(beta\.1|beta\.1 plan|kernel[ -]plan) walkthrough|owner decision 2026-10-0[23]|[Oo]wner('s)?,? [a-j][1-5]\b|(walkthrough|kernel[ -]plan),? \(?[a-j][1-5]\b" -- . ':!openspec/changes'
git grep -nE "\b([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b" -- . ':!openspec/changes' | sed -E 's/ADR-013, decisions? [a-j][1-5]((, | and )[a-j][1-5])*//g' | grep -E "\b([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b"
```

## 1. Cite library ADR-013 in the Dependabot configuration

- [x] 1.1 `.github/dependabot.yml`, the comment above the `cuelang.org/go` ignore entry: "(owner decision j4, 2026-10-03; workspace RELEASING.md, section "Pin classes")" becomes "(library ADR-013, decision j4; workspace RELEASING.md, section "Pin classes")", rewrapped so that `library ADR-013, decision j4` stays on one line. Verify: `git diff -U0 origin/main -- .github/dependabot.yml` shows comment lines only.
- [x] 1.2 Verify the claim against the record: the `j4` row of library ADR-013 says "`cuelang.org/go` moves only through a library release", which is what the comment and the requirement cite it for.
- [x] 1.3 `task fmt`, `task vet`, `task lint`, `task openspec:check`, `task docs:bundle:check` and `task test:unit` green, then commit `docs: cite library ADR-013 for walkthrough decision j4`.

## 2. Verify and archive

- [x] 2.1 Archive the change, which syncs the `repo-automation` delta into the main spec. Verify: both commands of the walkthrough-id check print nothing except `openspec/specs/release-workflow/spec.md:156` and `:175`, whose "owner decision 2026-10-02" cites workspace RELEASING.md, section "Owner settings", a release setting with no row in ADR-013 and not a walkthrough id; and `git diff origin/main -- openspec/specs/repo-automation/spec.md` changes only the `Source:` sentence of "Dependabot leaves cuelang.org/go to library releases".
- [x] 2.2 `task openspec:check` green, then commit `docs(openspec): archive cite-walkthrough-decisions-by-adr`.
