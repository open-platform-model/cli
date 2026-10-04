## Context

The supervisor plan of the release cascade security pass (2026-10-04) sets the scope: wave 1 hardens each product repo's own workflows; wave 2 moves the `.github` pin and the canonical wiring check. This change is the cli's wave 1. Owner decisions 28 (code-owner review on `main`), 29 (the release App key moves into a main-only `release` Environment, unrotated) and 30 (repository default token read-only, applied by the supervisor after this merges) bind it.

## Decisions

### D1. Only the release-please job enters the `release` Environment

`RELEASE_APP_PRIVATE_KEY` is read in exactly one place: the `Mint the release App token` step of `release.yml`'s `release-please` job. That job, and no other, declares `environment: release`. It already runs only on `push`, and `release.yml` triggers on a push to `main` only; the job condition also checks `github.ref == 'refs/heads/main'`, so a later trigger change cannot reach the key from another ref even if the Environment's branch policy were loosened. A manual `workflow_dispatch` skips the job, as before.

The Environment is created by the supervisor; until the owner moves the secret into it, GitHub resolves `secrets.RELEASE_APP_PRIVATE_KEY` from the organization secret, so merging this before the move changes nothing at run time but the deployment record.

### D2. Per-job grants, checked step by step

The repository default token turns read-only after this merges, so every job must name what it uses. Workflow level is `permissions: {}` where any job writes, `contents: read` where every job only reads.

| Workflow / job | Grants | Why |
| --- | --- | --- |
| `release.yml` / `release-please` | none | it calls GitHub only through the App token (`create-github-app-token`, then `release-please-action` with `token:`); no checkout |
| `release.yml` / `goreleaser` | `contents: write`, `packages: write` | draft check (`gh release view` sees drafts only with write), asset upload and publish, GHCR image push |
| `release.yml` / `publish-templates` | `contents: read`, `packages: write` | checkout, GHCR push of template modules |
| `release.yml` / `publish-docs` | `contents: read`, `packages: write`, `id-token: write` | docs-kit `publish.yml` (signing) |
| `release.yml` / `notify-downstream` | `contents: read` | fixed by the wiring check |
| `labels.yml` / `check` (pull_request) | `contents: read`, `issues: read` | checkout, dry-run label listing |
| `labels.yml` / `sync` (push to main) | `contents: read`, `issues: write` | label create, update, delete |
| `pr-title.yml` / `validate` | `pull-requests: read` | the action reads the PR title |

The other workflows (`cascade-gates`, `cascade-task`, `ci`, `deps-cascade`, `docs`, `e2e-cluster`, `pr`, `publish-fixtures`, `release-evidence`) already declare workflow-level `permissions: {}` or `contents: read` and job grants for every write; they are unchanged here except for the cache and docs-pin steps.

`release.yml` keeps no workflow-level `env` (the wiring check refuses one); workflow-level `permissions` is not checked by it.

### D3. No Actions cache where a job publishes

`actions/setup-go` restores and saves a cache keyed on `go.sum` by default. The cache is scoped by ref, and `main`'s scope is written by every main-ref run, including the cascade's compute and gates-only runs, which execute release-head and repository code (CAS-R3). A publishing job that restores it builds release artifacts from inputs another run wrote. `cache: false` goes on every `setup-go` in a job holding `packages: write` or `contents: write`: `release.yml` goreleaser and publish-templates, `publish-fixtures.yml`. goreleaser's buildx uses no `type=gha` cache (`.goreleaser.yml` sets none), and docs-kit's `publish.yml` already sets `cache: false`. Read-only jobs keep the cache.

### D4. The docs-bundle check runs in the pull request, not in compute

`task deps:cascade` ran `go run ./hack/docskit-dump pins` on the moved tree to warn about docs bundles that were not published yet. `hack/docskit-dump` imports the library's `opm/schema`, so the run executed the newly pinned library's package init in compute, the step that holds the run's cache scope and whose output publish pushes.

The check moves to `pr.yml`'s required `Lint` job, which runs on the cascade pull request (and any other) with a read-only token and the merged tree. It keeps the old strength and trigger:

- **When:** library's `go.mod` version or `PinnedOperatorVersion` differs between the pull request's base commit and the merge tree. The base is `github.event.pull_request.base.sha`, fetched at depth 1. The cascade compared against the merge base; the pull request's base commit gives the same answer for the pull request's own diff.
- **What:** each pin `hack/docskit-dump pins` prints is looked up anonymously at `ghcr.io/open-platform-model/docs/<project>:<pin>`, the lookup G1 uses.
- **Strength:** a warning (`::warning::` and the job summary), never a failure, also when the program does not build or a lookup fails. The release PR's G1 still fails on a missing bundle.

One script, `.github/scripts/docs-pins-check.sh`, holds the lookup. G1 (`release-pin-check.sh`) calls it in its failing mode and turns each printed line into a G1 violation; a non-zero exit with no line is itself a violation, so a crash never reads as a pass. The PR step calls it with `--warn --moved-from <sha>`.

The cascade PR body no longer lists docs-bundle warnings; the warning is on the PR's checks instead.

### D5. CODEOWNERS covers the code that runs with a write token

The plan's list (`/.github/`, `/.tasks/`, `/Taskfile*.yml`, the release-please files, `/.cascade-frozen`), each present here, plus:

- `/hack/`: `hack/fixtures.sh` runs in `publish-fixtures.yml` with `packages: write`, and `hack/docskit-dump` runs in the docs publish with `packages: write` and `id-token: write`.
- `/.goreleaser.yml`: goreleaser executes it (hooks included) with `contents: write` and `packages: write`.

`/Dockerfile` (no `RUN` step) and `/.opm-docs-version` (selects a released binary of the org's own tool) are left out.

### D6. opm is built from the merge base, not the work tree

The task builds `opm` for version advances. It used to build it from the work tree before any edit. In merge mode the work tree is `deps/cascade` with `main` merged in, so it already pins whatever library an earlier bot run moved, and that library's code ran in compute before anyone reviewed it. The task now exports the merge base (`git archive "$M"`; in merge mode that is `origin/main`) to `$STATE/opm-src`, builds `opm` there with `-buildvcs=false`, and removes the export. S2 puts a `go` shim on `PATH` and fails on any `go` call other than `get`, `mod tidy` and that build.

## Risks / Trade-offs

- A grant missing from D2 breaks its job only after the default flips. Mitigation: the table was built from every step; the supervisor flips the default after the merge and watches the next runs.
- Uncached publishing jobs take longer (a cold Go module download per release job).
- The `release` Environment shows a deployment for every push to `main`.
- The docs-bundle warning is less visible as a check annotation than as a line in the cascade PR body; G1 remains the blocking gate on the release PR.
