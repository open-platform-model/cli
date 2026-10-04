## Why

The cli's cascade references still pin `open-platform-model/.github` at `2376ffa` (the wiring as first built). Two `.github` changes have merged since, from the release cascade security pass (2026-10-04): `fe0e11b` (PR 11, keep gates-only runs and repo code away from publish) and `7b9ad1b` (PR 12, bound what publish accepts and pin compute's inputs). They reach the cli only when it moves its pin (`.github` README, "Pinning and bumps"), and both change the caller shape:

- `cascade-publish` now takes a required `gates-only` input and refuses to mint unless it is exactly `false`; the receiver's `publish` job's `if:` gains `inputs.gates_only != true`, so a gates-only run, which runs every open release head's task inside `compute`, never starts a `cascade` Environment job.
- The wiring check is now one canonical script in `.github` that every repo copies byte for byte, with its own values in `.tasks/cascade/wiring-check.yaml`. It adds the release App key rule (owner decision 29: every job that reads `RELEASE_APP_PRIVATE_KEY` declares `environment: release`), the no-Actions-cache rule for publishing workflows, and `--pin-on-main`, which confirms through the GitHub API that the pinned SHA is on `.github`'s `main` (refusing a commit that exists only in a fork).

## What Changes

- **Pin.** Every `.github` reference (`cascade-notify` in `release.yml`, `cascade-receive.yml` and `cascade-publish` in `deps-cascade.yml`, `cascade-gates.yml`, the resolver checkout `ref:` in `cascade-task.yml`) moves to `7b9ad1bea132f7a3f053a5db61ac3933b59ee226 # .github main`.
- **Wiring check.** `.tasks/cascade/wiring-check.sh` becomes the byte-identical copy of `.github/scripts/cascade/wiring-check.sh` at that SHA; the new `.tasks/cascade/wiring-check.yaml` holds the cli's values from the README table (`receiver: true`, no `env-allow`, `publish-workflows` `release.yml`, `publish-fixtures.yml`, `docs.yml`, CI `pr.yml`/`lint`, notify `needs`/`if`/`tag` read from `release.yml`, `labels-managed: true`).
- **CI step.** `pr.yml`'s required `lint` job runs `bash .tasks/cascade/wiring-check.sh --pin-on-main` with `GH_TOKEN: ${{ github.token }}` as "Verify the cascade wiring". `task cascade:wiring:check` stays the offline local entry.
- **Receiver.** `deps-cascade.yml`'s `publish` job: `if:` gains `&& inputs.gates_only != true`, and the `Publish` step passes `gates-only: ${{ inputs.gates_only == true }}`.
- **Dependabot.** The `github-actions` entry gets `cooldown: {default-days: 7}`.
- **Docs.** `AGENTS.md`'s release cascade note describes the canonical check and the gates-only switch.

No input `cascade-receive.yml` takes changed for the cli: it passes no `cue-version` (the default `v0.17.1` is one `install-tools.sh` holds a sha256 for). The `.github` mirror of the cli's `pins.sh`, `classes` and publish allow-list was read at cli `5f00930`; `main` since (`5180cad`, PR 306) changed only `cascade.sh` and `test.sh`, dropping the docs check, and the mirror's `pins.sh` prints the same rows as the cli's on `origin/main`.

Release class: none. CI, a copied script, its config and docs; no command, flag, output or shipped file changes. PR title `ci(deps): pin the cascade to .github 7b9ad1b`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `release-cascade-wiring`: the publish job passes `gates-only` and skips gates-only runs; the wiring check is the canonical `.github` copy with a config file, adds the release-key and publish-cache rules, and runs in CI with `--pin-on-main`.

## Impact

- **Files:** `.github/workflows/{release,deps-cascade,cascade-gates,cascade-task,pr}.yml`, `.github/dependabot.yml`, `.tasks/cascade/wiring-check.sh`, new `.tasks/cascade/wiring-check.yaml`, `Taskfile.yml` (task description), `AGENTS.md`.
- **Risks:** the cli is a dry-run receiver (`CASCADE_DRY_RUN` not `false`), so `Publish` stays skipped and the new `cascade-publish` code first meets real GitHub on the first live run (README step 2, canary rule; the supervisor decided all five repos move together while every receiver is dry). The `--pin-on-main` step calls the GitHub API with the job's read-only token; an API failure fails the required `Lint` job.
