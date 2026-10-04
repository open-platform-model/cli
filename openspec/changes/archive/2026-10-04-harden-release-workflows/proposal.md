## Why

The release cascade security pass (2026-10-04) found that the cli's release path trusts more than it needs to. Findings GOV-2, GOV-3, CAS-R2 and CAS-R3 apply here; the owner's decisions 28 to 31 set the response:

- **The release App key has no Environment gate (GOV-2).** `release.yml`'s release-please job reads the organization secret `RELEASE_APP_PRIVATE_KEY`, which any workflow on any branch could read on a push event. Owner decision 29 moves it, unrotated, into a main-only Environment `release` in every repo that reads it. The supervisor has already created that Environment here (custom branch policy `main`); until the owner moves the secret, a job in an Environment without it falls back to the organization secret, so the job can declare the Environment now.
- **Workflows rely on the repository's write default (GOV-3).** `release.yml` grants `contents: write` and `pull-requests: write` to every job at workflow level, and `labels.yml` declares no workflow-level permissions. Owner decision 30 flips the repository default `GITHUB_TOKEN` to read-only once every workflow declares what it needs.
- **Publishing jobs restore the Actions cache (CAS-R3).** `setup-go` caches by default. The goreleaser and publish-templates jobs of `release.yml` and the fixture publish of `publish-fixtures.yml` restore a cache that a main-ref run of other code (the cascade's gates-only and compute runs) can write, and then publish to GHCR with `packages: write`.
- **The cascade's compute runs new dependency code (CAS-R2).** `task deps:cascade` ends with `go run ./hack/docskit-dump pins` on the moved tree, which links the newly pinned library, so the new library's init code runs inside compute before any human has looked at the change.
- **A write token on a pull-request event.** `labels.yml` holds `issues: write` on `pull_request`, where it only needs a dry run.
- **No code owners (GOV-1).** Owner decision 28 requires one approval and code-owner review on `main`; the supervisor edits the ruleset, and the repository needs `.github/CODEOWNERS`.

## What Changes

- **Release Environment.** The `release-please` job, the only job that reads `RELEASE_APP_PRIVATE_KEY`, declares `environment: release` and runs only on a push to `main`.
- **Least-privilege permissions in every workflow.** `release.yml` declares `permissions: {}` at workflow level; each job grants exactly what it uses (release-please: none, since it acts only through the App token; goreleaser: `contents: write`, `packages: write`; publish-templates and publish-docs keep their grants). `labels.yml` and `pr-title.yml` declare `permissions: {}` at workflow level and grant per job. Every other workflow already declares `permissions: {}` or `contents: read` at workflow level.
- **No Actions cache in publishing jobs.** `cache: false` on `actions/setup-go` in `release.yml` (goreleaser, publish-templates) and `publish-fixtures.yml`. `docs.yml` and `publish-docs` call docs-kit's `publish.yml`, which already sets `cache: false`.
- **No write on pull-request events.** `labels.yml` splits into a `check` job on `pull_request` (`issues: read`, dry run) and a `sync` job on a push to `main` (`issues: write`).
- **The docs-bundle check leaves the cascade task.** `task deps:cascade` no longer runs `hack/docskit-dump`. The check moves into the pull request's own CI: a new step in `pr.yml`'s `Lint` job, "Docs bundles for moved pins", runs `.github/scripts/docs-pins-check.sh --warn --moved-from <base sha>` with the job's read-only token. When library or `PinnedOperatorVersion` differs from the base, it prints a warning for each docs bundle the tree pins that is not published, exactly the condition and strength of the old cascade warning (a warning, never a hold). The bundle lookup moves out of `release-pin-check.sh` into that script, and G1 calls it in its failing mode, unchanged in behavior.
- **`.github/CODEOWNERS`.** `/.github/`, `/.tasks/`, `/Taskfile*.yml`, `/release-please-config.json`, `/.release-please-manifest.json`, `/.cascade-frozen`, `/hack/` (its scripts and programs run in `publish-fixtures.yml` and the docs publish, which hold `packages: write`), and `/.goreleaser.yml` (goreleaser runs it with `contents: write` and `packages: write`), owned by the two maintainers.
- **`AGENTS.md`**: the release cascade and docs bundle notes say where the docs-bundle warning now lives, and that `release-please` runs in the `release` Environment.

Unchanged on purpose: the `.github` pin, `.tasks/cascade/wiring-check.sh` and `deps-cascade.yml`'s publish job (wave 2 of the security pass moves those); `.github/dependabot.yml` already covers `github-actions` and ignores `open-platform-model/.github*`.

Release class: none. CI, CODEOWNERS, a cascade task script and docs; no command, flag, output or shipped file changes. PR title `ci: harden the release workflows`.

## Capabilities

### New Capabilities

- `workflow-security`: the repository-wide rules for workflow tokens and caches: explicit workflow-level and job-level permissions in every workflow, no Actions cache in a publishing job, no write token on a pull-request event, and the `CODEOWNERS` file.

### Modified Capabilities

- `release-workflow`: the release-please job runs in the `release` Environment, the only reader of the release App key.
- `deps-cascade`: the cascade task no longer checks docs bundles; its warnings cover `language.version` and new majors.
- `release-gates`: a pull request that moves library or the embedded operator gets a docs-bundle warning from its own CI.

## Impact

- **Files:** edited `.github/workflows/{release,publish-fixtures,labels,pr-title,pr}.yml`, `.github/scripts/release-pin-check.sh`, `.tasks/cascade/cascade.sh`, `.tasks/cascade/test.sh`, `AGENTS.md`; new `.github/CODEOWNERS`, `.github/scripts/docs-pins-check.sh`.
- **Secrets:** no job gains a secret. `RELEASE_APP_PRIVATE_KEY` is still read by one job, now inside `environment: release`.
- **Risks:** a missing job permission breaks that job only after the supervisor flips the repository default to read-only (decision 30); each job's grants were checked against every step (design.md D2). A release run creates a `release` deployment record on each push to `main`. The docs-bundle warning moves from the cascade PR body to the PR's checks, where it is an annotation.
