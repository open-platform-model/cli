## Why

The owner's answers to the beta.1 kernel plan (ids such as `j4`: one lowercase letter and a number) were written down only in a planning log outside every repository. The cli cites one of them, `j4`, in two places as "owner decision j4", a source no reader of the repository can open. Library ADR-013 (`adr/013-kernel-plan-walkthrough-decisions.md` in the library repository) is now the record of those decisions, and it says the cli is to cite it in place of the id.

## What Changes

- `.github/dependabot.yml`: the comment above the `cuelang.org/go` ignore entry cites `library ADR-013, decision j4` in place of "owner decision j4, 2026-10-03". A comment line only; no Dependabot setting changes.
- `openspec/specs/repo-automation/spec.md`: the `Source:` sentence of the requirement "Dependabot leaves cuelang.org/go to library releases" cites `library ADR-013, decision j4`, with the ADR's path in the library repository, in place of "owner decision j4 of the kernel beta.1 plan walkthrough (2026-10-03)". The SHALL sentences and the scenarios are unchanged.
- Nothing else in the live tree cites a walkthrough decision. Archived changes under `openspec/changes/archive/` are history and stay as they are; ADR-013's table resolves the ids they use.
- The citation form is the one ADR-013 sets, `ADR-013, decision <id>`, with `library` in front because the ADR lives in another repository. The cli had no earlier form for citing a library ADR.
- No behaviour changes: no Go file, test, flag, output string or workflow step is touched.

SemVer class: none (a `docs` change, hidden by release-please; it cuts no release, before or after GA).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `repo-automation`: the `Source:` sentence of "Dependabot leaves cuelang.org/go to library releases" names library ADR-013 as the record of decision j4. No obligation changes.

## Impact

- Files: `.github/dependabot.yml` (comment), `openspec/specs/repo-automation/spec.md` (through the delta, at archive).
- `.github/` is covered by `.github/CODEOWNERS`, so the pull request needs a maintainer's review.
- No code, API, dependency or release impact.
