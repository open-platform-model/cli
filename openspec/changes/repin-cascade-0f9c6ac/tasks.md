# Tasks: repin-cascade-0f9c6ac

One PR, titled `ci(deps): pin the cascade to .github 0f9c6ac`. Workers never touch tags, releases, rulesets, repository settings, Environments, variables or secrets.

**Local gate:** `actionlint` on every workflow, `bash .tasks/cascade/wiring-check.sh --pin-on-main`, `CASCADE_TEST_SET=offline task -x deps:cascade:test`, `task openspec:check`, `task lint`.

## 1. Pin and canonical wiring check

- [x] 1.1 Compare the sha256 of `.tasks/cascade/pins.sh`, `classes` and `cascade.sh` on `origin/main` with `mirror_sources cli` in `.github` `wiring/lib.sh` at `0f9c6ac`; all three match.
- [x] 1.2 Move all five `.github` references to `0f9c6ac2c9b752a79f4874f637ef9955bcf00c13 # .github main`; `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows no other SHA.
- [x] 1.3 Prove `.tasks/cascade/wiring-check.sh` is byte-identical to the file at that SHA with `cmp` against the local `.github` object and the `gh api` raw content.
- [x] 1.4 Check the README at that SHA: no caller input changed and the cli's wiring-check row is unchanged, so `wiring-check.yaml` and `pr.yml` stay.
- [x] 1.5 Commit `ci(deps): pin the cascade to .github 0f9c6ac`.

## 2. Verification

- [x] 2.1 Run the gates.
