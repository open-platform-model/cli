## 1. Pin, gate and skills

- [x] 1.1 Change the three pins (`ci.yml`, `pr.yml`, `Taskfile.yml`) to `@fission-ai/openspec@1.14.1`
- [x] 1.2 Drop `--strict` from `openspec:check` and say so in its `desc`, in `AGENTS.md` and in `openspec/config.yaml`
- [x] 1.3 Regenerate the `openspec-*` skills with 1.14.1 and merge the local `openspec-verify-change` edit
- [x] 1.4 `task openspec:check` green and every step of `task check` green except `test:integration` and `test:e2e`, which need the kind-opm-dev cluster and did not run green locally (no Go file is in this diff), then commit `chore(openspec): bump the OpenSpec CLI and skills to 1.14.1`
