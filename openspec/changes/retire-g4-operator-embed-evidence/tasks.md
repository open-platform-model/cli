# Tasks: retire-g4-operator-embed-evidence

One PR, titled `ci(release): retire the G4 operator-embed evidence check`. Workers never touch tags, releases, repository settings, Environments, variables or secrets; the `main` ruleset edit is the supervisor's.

**Local gate:** `actionlint` on every workflow, `task openspec:check`, `task lint`.

## 1. Retire G4

- [x] 1.1 Delete `.github/workflows/release-evidence.yml` and `.github/scripts/release-evidence.sh`.
- [x] 1.2 Remove the `e2e-verified` entry from `.github/labels.yml`.
- [x] 1.3 `AGENTS.md`: drop the `e2e-verified` clause from the release-pin sentence. Verify: `grep -rn 'e2e-verified\|release-evidence\|G4' --exclude-dir=archive --exclude-dir=changes .` lists nothing outside `openspec/specs/`.
- [x] 1.4 Run the local gate.
- [x] 1.5 Commit `ci(release): retire the G4 operator-embed evidence check`.

## 2. Archive

- [ ] 2.1 Sync the deltas into `openspec/specs/` and archive the change; `task openspec:check` passes and `grep -rn 'e2e-verified' openspec/specs` is empty.
- [ ] 2.2 Commit `chore(openspec): archive retire-g4-operator-embed-evidence`.
