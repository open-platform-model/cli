# Tasks: harden-release-workflows

One PR, titled `ci: harden the release workflows`. Workers never touch tags, releases, rulesets, repository settings, Environments or secrets. Out of scope (wave 2): the `.github` pin, `.tasks/cascade/wiring-check.sh`, `deps-cascade.yml`'s publish job.

**Local gate for every section:** `actionlint` (with shellcheck) on every workflow, `shellcheck` on changed scripts, `task cascade:wiring:check`, `task openspec:check`; from section 4 on, `task -x deps:cascade:test` (offline set).

## 1. Release Environment and permissions

- [ ] 1.1 `release.yml`: workflow-level `permissions: {}`; `release-please` gets `environment: release`, `permissions: {}` and `if: github.event_name == 'push' && github.ref == 'refs/heads/main'` (design D1, D2).
- [ ] 1.2 `labels.yml`: workflow-level `permissions: {}`; split into `check` (pull_request, `contents: read`, `issues: read`, dry run) and `sync` (push to main, `contents: read`, `issues: write`).
- [ ] 1.3 `pr-title.yml`: workflow-level `permissions: {}`, job `pull-requests: read`.
- [ ] 1.4 Confirm every workflow has a top-level `permissions` key and every write scope sits on a job (design D2 table).

## 2. No Actions cache in publishing jobs

- [ ] 2.1 `cache: false` on `setup-go` in `release.yml` goreleaser and publish-templates, and in `publish-fixtures.yml`.
- [ ] 2.2 Re-check every publishing job for `actions/cache`, `type=gha` and other caching (`.goreleaser.yml`, docs-kit `publish.yml` at the pinned tag).

## 3. CODEOWNERS

- [ ] 3.1 `.github/CODEOWNERS` with the paths of design D5.

## 4. Docs-bundle check out of the cascade task

- [ ] 4.1 `.github/scripts/docs-pins-check.sh` (`[--warn] [--moved-from REV]`) holding the anonymous GHCR lookup.
- [ ] 4.2 `release-pin-check.sh` section 6 calls it; a non-zero exit with no line is a violation.
- [ ] 4.3 `pr.yml` `lint` job: step "Docs bundles for moved pins".
- [ ] 4.4 `cascade.sh`: drop `docs_check` and its two calls; `test.sh`: drop the S2 docs assertion and scenario S15.
- [ ] 4.5 Exercise the script locally: the unchanged case, a moved case with every bundle published, a moved case with a missing bundle (`--warn` and failing mode), and `task deps:release-check` on the tree.

## 5. Docs and close

- [ ] 5.1 `AGENTS.md`: the release cascade and docs bundle notes.
- [ ] 5.2 Gates, `openspec validate harden-release-workflows --strict`, verify, archive in-branch, PR.
