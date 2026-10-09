# Tasks: rename-operator-command-to-controller

Ships as its own PR after `retire-legacy-manifest-migration` has merged, and only after the controller release, its module 0.1.0 and its docs bundle are published, since the pin and the e2e job need them.

**Local gate:** `task fmt`, `task lint`, unit tests (`go test ./internal/... ./pkg/... ./hack/...`), `go vet -tags e2e ./tests/e2e/`, every `tests/integration/*/main.go` builds, `task cascade:wiring:check`, `CASCADE_TEST_SET=offline task -x deps:cascade:test`, `task docs:bundle:check`, `task openspec:check`.

## 1. Rename the code, tasks, CI and docs

- [x] 1.1 `git mv` `internal/operator` → `internal/controller` (`operatortest` → `controllertest`, `opversion*.go` → `version*.go`), `internal/cmd/operator` → `internal/cmd/controller` (`controller.go`, `controller_test.go`), `hack/operator-pin` → `hack/controller-pin`, `hack/kind-operator-rbac.yaml` → `hack/kind-controller-rbac.yaml`, `tests/e2e/{operator_test,instance_operator_owned_test}.go` → `{controller_test,instance_controller_owned_test}.go`, `tests/e2e/testdata/operator-owned` → `controller-owned`, RFC 0007 file name.
- [x] 1.2 Go: every product identifier and string (`names.go`: `opm-controller`, `opm-system`, `opm-controller-manager`, `opmodel.dev/modules/opm_controller`, `DeploysController`; `version.go`: sub-package `controller`; `inventory.OwnerController`, `ModeControllerOwned`, `DisplayOwner("")` = `controller`; `gates.go` reads `status.controllerVersion` only; `Use: "controller"`, no alias; every hint and error string). Fix stutters and articles by hand.
- [x] 1.3 `pin.go` and the `controller-pin` golden carry the new names; the controller version value waits for `task controller:pin VERSION=0.1.0` at cutover.
- [x] 1.4 Non-Go: Taskfile cluster tasks and `controller:pin`, `hack/kind-controller-rbac.yaml`, `hack/kind-platform.yaml`, `hack/opm-config.cue`, `hack/fixtures.sh`, `.tasks/cascade/{cascade,pins,test}.sh` and testdata, `docs-kit.cue`, `.github/workflows/{pr,e2e-cluster,release,docs,deps-cascade}.yml` (keep the job name `E2E (kind, embedded operator)`), `.github/scripts/{e2e-cluster-applies,release-pin-check,docs-pins-check}.sh` (one OLD_PIN fallback), `opencode.json`, README, QUICKSTART, AGENTS, CONSTITUTION, `openspec/config.yaml`, `docs/`, `adr/008`, `examples/README.md`.
- [x] 1.5 Fixtures: podinfo comment, `opm module version set 0.1.13`, re-pin `examples/` and `tests/e2e/testdata/controller-owned`.
- [x] 1.6 Local gate green (registry-backed tests that need podinfo `0.1.13` on GHCR are recorded as cutover-pending), then commit `refactor!: rename the operator command and package to controller`.

## 2. Specs and archive

- [x] 2.1 Deltas: `controller-lifecycle` ADDED-all with a Purpose, `operator-lifecycle` REMOVED-all (`retire_capabilities: true`), and the wording deltas listed in the proposal; `openspec validate rename-operator-command-to-controller --strict` passes.
- [x] 2.2 Verify the implementation against the change (opsx verify), archive it, edit the Purpose lines of the surviving main specs that name the operator, confirm `git grep -i operator openspec/specs` lists only the kept check name, `task openspec:check` passes, then commit `chore(openspec): archive rename-operator-command-to-controller`.
