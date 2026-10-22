## 1. The hint line

- [ ] 1.1 On the fake cluster, with a throwaway test that is not committed, confirm that the guard refuses the four CRDs for a manifest install and for an install by the earlier module, and that a Get of the earlier Deployment works with the install's client
- [ ] 1.2 Add `EarlierInstallHint` in `internal/controller`, the wrapping of a `GuardError` and the warning after a passed plan in `internal/cmd/controller/install.go`, for the full form and `--crds-only`; the error keeps its exit code 2 and its `GuardError` in the chain
- [ ] 1.3 Table-driven tests on the fake cluster: hint on a refusal and warning on a passed plan (manifest install, module install, CRDs already annotated), no Deployment, denied read, a failure that is not a guard refusal, the exit code in every row
- [ ] 1.4 `task fmt`, `task lint` and `task test` green, then commit `fix(controller): point an install refused over the earlier controller to its docs page`

## 2. The docs page

- [ ] 2.1 Check the object names against the earlier release manifests (tags `v1.0.0-alpha.18` to `v1.0.0-beta.8` and the operator module render) and write `docs/site/diagnostics/install-over-the-earlier-controller.md` after the model of `legacy-inventory-secret.md`
- [ ] 2.2 Check the page: front matter, links, `docs-kit` check as the repo runs it, and the hint text names the page title exactly
- [ ] 2.3 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `docs(diagnostics): add the page for an install over the earlier controller`
