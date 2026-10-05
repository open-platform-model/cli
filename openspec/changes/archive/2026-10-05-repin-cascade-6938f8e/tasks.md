# Tasks: repin-cascade-6938f8e

One PR, titled `ci(deps): pin the cascade to .github 6938f8e`. Workers never touch tags, releases, rulesets, repository settings, Environments, variables or secrets.

**Local gate:** `actionlint` on every workflow, `bash .tasks/cascade/wiring-check.sh --pin-on-main`, `task -x deps:cascade:test` (offline set), `task openspec:check`, `task lint`.

## 1. Pin and canonical wiring check

- [x] 1.1 Move all five `.github` references to `6938f8e0247e019cb0c2db13fff5b7b558a6b67d # .github main`; `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows no other SHA.
- [x] 1.2 Copy `.github/scripts/cascade/wiring-check.sh` at that SHA to `.tasks/cascade/wiring-check.sh` and prove it with `cmp` against the local `.github` object and the `gh api` raw content.
- [x] 1.3 Check `pr.yml`'s `lint` job against the README at that SHA (workflow and job `env`, no `container`/`services`, only SHA-pinned actions before the wiring step) and `cascade-task.yml`'s checkout; confirm `wiring-check.yaml` needs no new key and no caller input changed.
- [x] 1.4 Commit `ci(deps): pin the cascade to .github 6938f8e`.

## 2. Docs and verification

- [x] 2.1 `AGENTS.md` release cascade note and the wiring step comment in `pr.yml`: `--pin-on-main` also compares the copy with the file at the SHA.
- [x] 2.2 Run the gates.
- [x] 2.3 Commit `docs(agents): say the wiring step compares the copy`.
