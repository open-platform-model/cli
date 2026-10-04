# Tasks: harden-release-workflows

One PR, titled `ci: harden the release workflows`. Workers never touch tags, releases, rulesets, repository settings, Environments or secrets. Out of scope (wave 2): the `.github` pin, `.tasks/cascade/wiring-check.sh`, `deps-cascade.yml`'s publish job.

**Local gate for every section:** `actionlint` (with shellcheck) on every workflow, `shellcheck` on changed scripts, `task cascade:wiring:check`, `task openspec:check`; from section 3 on, `task -x deps:cascade:test` (offline set).

## 1. Release Environment and permissions

- [x] 1.1 `release.yml`: workflow-level `permissions: {}`; `release-please` gets `environment: release`, `permissions: {}` and `if: github.event_name == 'push' && github.ref == 'refs/heads/main'` (design D1, D2).
- [x] 1.2 `labels.yml`: workflow-level `permissions: {}`; split into `check` (pull_request, `contents: read`, `issues: read`, dry run) and `sync` (push to main, `contents: read`, `issues: write`).
- [x] 1.3 `pr-title.yml`: workflow-level `permissions: {}`, job `pull-requests: read`.
- [x] 1.4 Confirm every workflow has a top-level `permissions` key and every write scope sits on a job (design D2 table).
- [x] 1.5 No Actions cache in publishing jobs: `cache: false` on `setup-go` in `release.yml` goreleaser and publish-templates, and in `publish-fixtures.yml`.
- [x] 1.6 Re-check every publishing job for `actions/cache`, `type=gha` and other caching (`.goreleaser.yml`, docs-kit `publish.yml` at the pinned tag).
- [x] 1.7 Commit `ci: gate the release App key and grant permissions per job`.

## 2. CODEOWNERS

- [x] 2.1 `.github/CODEOWNERS` with the paths of design D5.
- [x] 2.2 Commit `ci: add code owners for the release and CI paths`.

## 3. Docs-bundle check out of the cascade task

- [x] 3.1 `.github/scripts/docs-pins-check.sh` (`[--warn] [--moved-from REV]`) holding the anonymous GHCR lookup.
- [x] 3.2 `release-pin-check.sh` section 6 calls it; a non-zero exit with no line is a violation.
- [x] 3.3 `pr.yml` `lint` job: step "Docs bundles for moved pins".
- [x] 3.4 `cascade.sh`: drop `docs_check` and its two calls; `test.sh`: drop the S2 docs assertion and scenario S15.
- [x] 3.5 Exercise the script locally: the unchanged case, a moved case with every bundle published, a moved case with a missing bundle (`--warn` and failing mode), and `task deps:release-check` on the tree.
- [x] 3.6 Commit `fix(cascade): check docs bundles in the pull request, not in compute`.

## 4. Docs

- [x] 4.1 `AGENTS.md`: the release cascade and docs bundle notes.
- [x] 4.2 Run the gates: `task lint`, `task test:unit`, `task openspec:check`, `task -x deps:cascade:test` (all set), `actionlint`, `shellcheck`, `task cascade:wiring:check`.
- [x] 4.3 Commit `docs(agents): describe the release Environment, job permissions and docs-pin check`.

## 5. Review fixes (PR 306)

- [x] 5.1 `cascade.sh`: build `opm` from an export of the merge base, not the work tree (design D6); spec and `AGENTS.md` say so.
- [x] 5.2 `test.sh` S2: a `go` shim fails the scenario on any `go` call but `get`, `mod tidy` and the `opm` build from the export.
- [x] 5.3 `docs-pins-check.sh`: bounded GHCR requests, an empty pin read is a warning; the `pr.yml` step gets `timeout-minutes` and `continue-on-error`.
- [x] 5.4 `deps-cascade.yml`: the setup-go comment no longer names `hack/docskit-dump`.
- [x] 5.5 publish-templates and `publish-fixtures.yml` run from `main` only; the release jobs check out `refs/tags/<tag>`.
- [x] 5.6 Re-run the gates.
