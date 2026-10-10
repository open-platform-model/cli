# Tasks: repin-cascade-6df4000

One PR, titled `ci(deps): pin the cascade to .github 6df4000`. Workers never touch tags, releases, rulesets, repository settings, Environments, variables or secrets.

**Local gate:** `task cascade:wiring:check`, `openspec validate --all --strict` (or `task openspec:check`), `actionlint` on every workflow where installed.

## 1. Pin and canonical wiring check

- [x] 1.1 Compare the sha256 of `.tasks/cascade/pins.sh`, `classes` and `cascade.sh` on `origin/main` with `mirror_sources cli` in `.github` `wiring/lib.sh` at `6df4000`; all three match.
- [x] 1.2 Move all five `.github` references to `6df4000f6cabbf460a5cad6431b6bc9044fa3027 # .github main`; `grep -rn -A1 'open-platform-model/.github' .github/workflows` shows no other SHA.
- [x] 1.3 Prove `.tasks/cascade/wiring-check.sh` is byte-identical to the file at that SHA with `cmp` against the local `.github` object.
- [x] 1.4 Check the README diff between `0f9c6ac` and that SHA: no caller input changed and the cli's wiring-check row is unchanged, so `wiring-check.yaml` and `pr.yml` stay.
- [x] 1.5 Run the local gate.
- [x] 1.6 Commit `ci(deps): pin the cascade to .github 6df4000`.
