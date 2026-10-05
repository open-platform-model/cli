## MODIFIED Requirements

### Requirement: The cascade receiver runs on a dispatch, a daily sweep and a manual run
`.github/workflows/deps-cascade.yml` SHALL run on `repository_dispatch` of type `upstream-released`, on a daily schedule at `17 6 * * *` (UTC), and on `workflow_dispatch` with the boolean inputs `dry_run` and `gates_only`, both defaulting to false. It SHALL declare `permissions: {}` at the workflow level and SHALL have exactly the top-level keys `name`, `on`, `permissions`, `concurrency` and `jobs`. Its job `cascade` SHALL call the shared `open-platform-model/.github` reusable workflow `cascade-receive.yml` at the repository's pinned `.github` commit, SHALL grant only `contents: read`, `pull-requests: read` and `statuses: write`, and SHALL pass `dry-run`, `gates-only` from the `gates_only` input, `g2-mode` and `g3-mode` from the repository variables `CASCADE_G2_MODE` and `CASCADE_G3_MODE` (each defaulting to `warn`), and `setup-go: true`, because the cli's cascade task builds and runs Go. The receiver computes the cascade diff with this repository's `task -x deps:cascade`. Source: workspace RELEASING.md, sections "The receiver", "Two-job split" and "Gates"; Phase 3 wiring contract (version 3.1) §5 and §5.2.

#### Scenario: An upstream release starts the receiver
- **WHEN** library, opm-controller or catalog_opm publishes a release and its notify job dispatches `upstream-released` to the cli
- **THEN** a `Deps cascade` run starts on `main` and runs the shared receiver against this repository

#### Scenario: The daily sweep runs without a dispatch
- **WHEN** no dispatch arrived and the schedule fires
- **THEN** a `Deps cascade` run starts and resolves the newest published upstreams itself
