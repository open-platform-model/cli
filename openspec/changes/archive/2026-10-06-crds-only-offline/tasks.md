Ships inside the controller rename's PR (`refactor!: rename the operator command and package to the controller`),
after the rename's own archive. Implements 0021:D11:R1/R7 (revised 2026-10-06); `enhancement.yaml` claims no decision.

## 1. Share the controller render

- [x] 1.1 Move `moduleRenderOpts` from `internal/cmd/controller/install.go` to `internal/controller` as
  `ModuleRenderOpts`, and make install call it (design C5)
- [x] 1.2 Commit `refactor(controller): render the controller's instance from one set of options`

## 2. Embed the pinned CRDs

- [x] 2.1 Add package `internal/controller/crds`: `//go:embed *.yaml` and `Objects()` returning the manifests in
  file-name order, plus `Manifest(obj)` that serialises one object with the generated header (design C2)
- [x] 2.2 `hack/controller-pin <version>`: resolve, read the controller version, render the version with
  `ModuleRenderOpts`, keep the CRDs, then replace every `*.yaml` in `internal/controller/crds/` and write `pin.go`. A
  failed render writes nothing
- [x] 2.3 `hack/controller-pin --check`: also render the pinned version and report each missing, extra or differing
  manifest with the re-pin command (design C6)
- [x] 2.4 `hack/controller-pin --crds-from <dir>`: render a local module directory and write only the manifests
- [x] 2.5 Tests with an injected renderer: the pin writes the golden pin file and the manifests; a refused version or a
  failed render leaves both unchanged; `--check` reports drifted, missing and extra manifests; `--crds-from` leaves
  `pin.go` alone
- [x] 2.6 Bootstrap the manifests with `--crds-from` from the controller repository's rename branch
  (`modules/opm_controller`), and record the cutover re-pin
- [x] 2.7 Commit `feat(controller): embed the CRDs the pinned controller module renders` (plus
  `fix(controller): keep the pin render's generated platform out of the tree`, found while bootstrapping)
- [x] 2.8 Stage every manifest as a temporary file before removing or replacing any, and write the pin last, so a failed
  write leaves the pin and the manifests unchanged; a test with a read-only directory (review finding); commit
  `fix(controller): stage the pin's manifest writes`

## 3. Install the CRDs offline

- [x] 3.1 `internal/controller`: `PlanCRDsOnly` builds the plan from the embedded CRDs, the pinned target and the
  `--rbac` objects; it waits out terminating objects and runs the apply guard over the CRDs, reading no ModuleInstance
  (design C4). `PlanInstall` loses its CRDs-only option
- [x] 3.2 `internal/cmd/controller/install.go`: `--crds-only` skips the module and catalog resolution and calls
  `PlanCRDsOnly`; `--version` with `--crds-only` is a flag-validation error (design C3); help and the success line say
  the CRDs are the ones pinned with this CLI
- [x] 3.3 Unit tests with the fake client: the CRDs-only install applies exactly the embedded CRDs as `opm-cli` and
  makes no registry call and no ModuleInstance read; a foreign CRD is refused with nothing applied; `--rbac` still
  applies; a full install after it keeps the CRDs; the embedded manifests match the pin (labels, names, the
  ModuleInstance CRD field floor)
- [x] 3.4 Docs: README, QUICKSTART and AGENTS name the offline CRDs-only form and the manifests the pin writes
- [x] 3.5 Commit `feat(controller)!: install the CRDs-only form offline`
- [x] 3.6 `PlanCRDsOnly` refuses (exit 2) a CRD that a newer module version than the pin installed, naming the full
  install of that version; fake-client tests for the refusal and for applying over an older version's CRDs (review
  finding, design C4); commit `fix(controller): refuse to downgrade CRDs a newer module installed`
- [x] 3.7 `PlanCRDsOnly` refuses an `opmodel.dev` CRD without OPM's labels with a message saying an earlier installer
  bootstrapped the cluster, that the CLI neither adopts nor migrates it, and to start fresh (no adoption code); the
  newer-version refusal names both remedies, `--crds-only` from a CLI that pins that version or newer and a full
  install of it (design C4); commit `fix(controller): say why --crds-only refuses CRDs it did not install`
- [x] 3.8 The start-fresh refusal says only "start fresh on a new cluster" (deleting the CRDs would delete every OPM
  object under them); the downgrade comparison applies only to CRDs of the embedded CRD's module
  (`module.opmodel.dev/name`); help, README and QUICKSTART scope "adopts and migrates nothing" to `--crds-only`;
  commit `fix(controller): scope the --crds-only refusals`

## 4. Specs and checks

- [x] 4.1 Deltas: `controller-lifecycle` (CRDs-only replaced, pin and refusing checks modified) and
  `platform-resolution` (controller install render)
- [x] 4.2 `go build ./...`, `go vet ./...`, `task lint`, `go test $(go list ./... | grep -v tests/e2e)`,
  `task openspec:check`
- [x] 4.3 Archive as its own commit `chore(openspec): archive crds-only-offline`
