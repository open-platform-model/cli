# cli: where do I go?

Router only. Facts and rules are in `AGENTS.md`.

| Task | Read first | Then |
| --- | --- | --- |
| Add or change a command, a flag (for example `--wait`) or an exit code | `AGENTS.md` | `internal/cmd/`, `internal/cmdutil/`, `internal/exit/` |
| Change how a command picks a platform, `--skip-unprovided`, skew policy, or what `module` and `instance` commands render | `references/render-path.md` | `internal/platform/`, `internal/workflow/render/` |
| Return, wrap or test an error; replace a match on error text with a typed error; use the `oerrors` alias | `AGENTS.md` | `pkg/errors/` |
| Add or change a test fixture module, a template under `templates/`, or a fixture pin | `references/fixtures.md` | `hack/`, `tests/fixtures/` |
| Choose a commit or PR type, or type a pin bump | `references/commit-types.md` | `references/release-line.md` for a break |
| Pick the release line, type a breaking change, fix a broken release, or ask why a release PR is not opening | `references/release-line.md` | `release-please-config.json`, `RELEASING.md` at the workspace root |
| Run or debug `task -x deps:cascade`, the `deps/cascade` PR, notify downstream or the cascade gates | `references/release-cascade.md` | `RELEASING.md` at the workspace root, `.tasks/cascade/` |
| Change a workflow's secrets, permissions, Environment or the cascade SHA pin, or fix a wiring check failure | `references/cascade-wiring.md` | `.tasks/cascade/` |
| Change help text, a docs page, `docs-kit.cue`, or recover a release with no docs bundle | `references/docs-bundles.md` | `docs/STYLE.md`, `docs/site/` |
| Create, fix or reproduce the kind cluster, `cluster:*` tasks or the `E2E (kind, embedded operator)` check | `references/e2e-cluster.md` | `Taskfile.yml`, `.github/scripts/` |
| Change `.golangci.yml`, bump golangci-lint, or fix a `Lint` job failure | `references/lint-config.md` | `.github/scripts/` |
| Start a feature change | `openspec/config.yaml` | `CONSTITUTION.md` |
| Write or fix a record of a decision | `adr/` | `CONSTITUTION.md` |
| Deliver or cite an enhancement | `references/enhancements.md` at the workspace root | `AGENTS.md` |
| Look up a registry mapping or a CUE resolution problem | `references/registry.md` at the workspace root | |
| Commit, open a PR or write GitHub text | `references/git-text.md` at the workspace root | |
| Tag, publish or recover a release | `references/releases.md` at the workspace root | `references/release-line.md` |
