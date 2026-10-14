# cli: where do I go?

Router only. Facts and rules are in `AGENTS.md`.

| Task | Read first | Then |
| --- | --- | --- |
| Change how a command picks a platform, `--skip-unprovided`, skew policy, or what `module` and `instance` commands render | `references/render-path.md` | `internal/platform/`, `internal/workflow/render/` |
| Add or change a test fixture module, a template under `templates/`, or a fixture pin | `references/fixtures.md` | `hack/fixtures.sh`, `tests/fixtures/fixtures.go` |
| Pick the release line, type a breaking change, or ask why a release PR is not opening | `references/release-line.md` | `release-please-config.json`, `RELEASING.md` at the workspace root |
| Run or debug `task -x deps:cascade`, the `deps/cascade` PR, notify downstream or the cascade gates | `references/release-cascade.md` | `RELEASING.md` at the workspace root, `.tasks/cascade/` |
| Edit a workflow, the release App key use, permissions, or the cascade SHA pin | `references/cascade-wiring.md` | `.tasks/cascade/wiring-check.yaml` |
| Change help text, a docs page, `docs-kit.cue`, or recover a release with no docs bundle | `references/docs-bundles.md` | `docs/STYLE.md`, `docs/site/` |
| Run the e2e suite against kind, or change `cluster:*` tasks, the `E2E (kind, embedded operator)` check or the operator pin | `references/e2e-cluster.md` | `Taskfile.yml`, `.github/scripts/e2e-cluster-applies.sh` |
| Change `.golangci.yml`, bump golangci-lint, or fix a `Lint` job failure | `references/lint-config.md` | `.github/scripts/lint-config-check.sh` |
| Add or change a command, flag or exit code | `AGENTS.md`, the Layout and Style sections | `internal/cmd/`, `internal/cmdutil/`, `internal/exit/` |
| Start a feature change | `openspec/config.yaml` | `CONSTITUTION.md` |
| Write or fix a record of a decision | `adr/` | `CONSTITUTION.md` |
| Deliver or cite an enhancement | `references/enhancements.md` at the workspace root | `AGENTS.md`, the Repository rules section |
| Look up a registry mapping or a CUE resolution problem | `references/registry.md` at the workspace root | |
| Commit, open a PR or write GitHub text | `references/git-text.md` at the workspace root | |
| Tag, publish or recover a release | `references/releases.md` at the workspace root | `references/release-line.md` |
