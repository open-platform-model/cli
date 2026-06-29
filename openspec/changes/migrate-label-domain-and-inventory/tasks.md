Branch: `0002-cli-x1-rename-module-instance` (stacked on X1 `7a9175d` + X2 + X3 `8e6311e`; atomic-PR / bulk-archive model — **not** a fresh branch). X4 is the slice that drives full `task test` green for the first time in the wave. Every rename site carries a `// Was:` breadcrumb (D11/D12). The X1-gap `pkg/loader` public-API accuracy residue and the `modules/`+`releases/` `release.cue` sweep are **out of scope** (see proposal).

## 1. Label domain (`instance-identity-labeling`, D4)

- [x] 1.1 `pkg/core/labels.go`: rename `LabelModuleReleaseName`/`LabelModuleReleaseNamespace`/`LabelModuleReleaseUUID` → `LabelModuleInstance{Name,Namespace,UUID}`; flip the three literal values `module-release.opmodel.dev/{name,namespace,uuid}` → `module-instance.opmodel.dev/{name,namespace,uuid}`; update the doc comments ("release name label" → "instance name label"). Add `// Was: module-release.opmodel.dev/*` breadcrumbs.
- [x] 1.2 Update the two production consumers of the literal: `internal/inventory/crud.go` (the `// 2. Fallback: list Secrets with label …` comment + the GetInventory UUID-label selector; the `FindInventoryByReleaseName` name-label selector) → `module-instance.opmodel.dev/*`. Grep-confirm `grep -rn "module-release.opmodel.dev" pkg internal` returns only `// Was:` breadcrumbs afterward.
- [x] 1.3 Cross-check selector construction in `internal/kubernetes/*` and `pkg/ownership/ownership.go` for any hardcoded `module-release.*` literal (expected: none beyond inventory; confirm).

## 2. Inventory + identity Go symbols (`instance-inventory`)

- [x] 2.1 `internal/inventory/types.go`: rename `ReleaseMetadata` → `InstanceMetadata` (fields `ReleaseName`/`ReleaseNamespace`/`ReleaseID` → `InstanceName`/`InstanceNamespace`/`InstanceID`; JSON tags `name`/`namespace`/`uuid` unchanged — wire-compatible), `ReleaseInventoryRecord` → `InstanceInventoryRecord` (field `ReleaseMetadata` → `InstanceMetadata`), and the `kind: "ModuleRelease"` literal → `"ModuleInstance"`. Breadcrumbs at each.
- [x] 2.2 `internal/inventory/{crud,discover,list,secret}.go`: propagate the type/field renames (`ReleaseInventoryRecord`, `ReleaseMetadata`, `ReleaseName`/`ReleaseID`, `FindInventoryByReleaseName`, `ReleaseSummary`, `ReleaseNotFoundError`) → `Instance*`. Keep exported function names that do not carry the noun (`GetInventory`, `WriteInventory`, `ListInventories`) unchanged.
- [x] 2.3 `internal/kubernetes/{delete,errors,events,status,tree}.go`: rename `ReleaseName`/`ReleaseID`/`ReleaseInfo`/`ReleaseNotFoundError` references → `Instance*`; flip the inventory-discovery selectors to the new label constants.
- [x] 2.4 `internal/workflow/{apply/apply.go,query/{list,status,events}.go}`: propagate `ReleaseInventoryRecord`/`ReleaseMetadata`/`ReleaseName`/`ReleaseID`/`ReleaseSummary` renames; `releaseName`/`releaseID` local identity vars → `instanceName`/`instanceID`.
- [x] 2.5 `pkg/ownership/ownership.go`: rename the `EnsureCLIMutable(createdBy, releaseName, releaseNamespace …)` param names → `instanceName`/`instanceNamespace`; reword the `"release %q in namespace %q is controller-managed"` message → `"instance %q …"`; comment nouns "release" → "instance". (Exported `EnsureCLIMutable`, `CreatedByCLI/Controller` unchanged.)
- [x] 2.6 `pkg/render/execute.go` + `pkg/module/instance.go`: reconcile the residual `ReleaseMetadata` references to `InstanceMetadata` (X1 renamed `pkg/module.Release`→`Instance` but left adjacent `ReleaseMetadata` call-throughs — close them here).

## 3. Logger + output cluster (~122 refs)

- [x] 3.1 `internal/output/log.go`: rename `ReleaseLogger(name string)` → `InstanceLogger(name string)`; update its doc comment ("child logger scoped to a release name" → "instance name"). Breadcrumb.
- [x] 3.2 Rename the `releaseLog` call-var → `instanceLog` at every call site (~16 files): `internal/cmd/instance/{apply,delete,diff,events,status,tree,vet}.go`, `internal/cmd/module/{apply,vet}.go`, `internal/cmdutil/manifest_output.go`, `internal/kubernetes/{apply,delete}.go`, `internal/workflow/{apply/apply.go,query/{events,status}.go,render/{log_output,output_internal}.go}`. Update the `cmdutil`/`InstanceSelectorFlags.LogName()` `"release:<prefix>"` → `"instance:<prefix>"`.

## 4. cmdutil flags + render file-opts (`cmdutil`, D-X4.2/D-X4.4)

- [x] 4.1 `internal/cmdutil/flags.go`: rename `ReleaseSelectorFlags` → `InstanceSelectorFlags` (fields `ReleaseName`/`ReleaseID` → `InstanceName`/`InstanceID`); flip flag names `--release-name`/`--release-id` → `--instance-name`/`--instance-id` (help text + the two `Validate()` error messages + `LogName()`); rename `ReleaseFileFlags` → `InstanceFileFlags`; rename `ResolveReleaseIdentifier` → `ResolveInstanceIdentifier`. Breadcrumbs. **No back-compat alias** for the old flags (D-X4.2).
- [x] 4.2 `internal/cmdutil/instance_arg.go` + `instance_target.go`: flip the X3-left `*ReleaseSelectorFlags` cross-type references → `*InstanceSelectorFlags` (the `ToSelectorFlags` return type, the `ResolvedInstanceTarget.Selector` field), and the `ResolveReleaseIdentifier` call → `ResolveInstanceIdentifier`. Remove the now-resolved "renamed in the X4 slice" notes.
- [x] 4.3 `internal/workflow/render/{render,types,module}.go` + `pkg/render`: rename `FromReleaseFile` → `FromInstanceFile`, `ReleaseFilePath` → `InstanceFilePath`, `ReleaseFileOpts` → `InstanceFileOpts`, and their call sites in `internal/cmd/instance/{apply,build,diff,vet}.go`. Breadcrumbs (D-X4.4: accept future merge-touch if single-build/0006 resumes).
- [x] 4.4 `internal/cmd/module/{build,apply}.go`: reword the `--name` help string "Override synthetic release name" → "Override synthetic instance name" (rides the mod-* noun rename; flag name `--name` unchanged).

## 5. On-disk moves + fixtures → full `task test` green

- [x] 5.1 `git mv examples/releases/ examples/instances/`; within it `git mv {garage,jellyfin,mc_java_fleet}/release.cue → instance.cue`. Update any path references in `README.md`/`QUICKSTART.md` (the example *paths* X3 deliberately left).
- [x] 5.2 `git mv tests/integration/rel-list → inst-list`, `tests/integration/rel-tree → inst-tree`; update the `"ModuleRelease"` literal + `release.cue` testdata inside (`tests/integration/rel-tree/testdata/release.cue` → `inst-tree/testdata/instance.cue`), and any Taskfile/CI references to the `rel-*` integration dirs.
- [x] 5.3 `git mv tests/e2e/testdata/vet-errors/release/ → instance/` and its `release.cue` → `instance.cue`; flip the `kind: "ModuleRelease"` literal inside to `"ModuleInstance"` (or confirm the e2e asserts the new unknown-kind path).
- [x] 5.4 Flip remaining `kind: "ModuleRelease"` fixture literals to `"ModuleInstance"`: `internal/workflow/apply/{apply.go,apply_test.go}`, `internal/inventory/crud_test.go`, `internal/workflow/query/inventory_test.go`, `internal/cmd/module/vet_test.go`, `tests/integration/{deploy,inventory-ops,inventory-apply}/main.go`. Include the X2-flagged stale fixture in `internal/workflow/render/render_test.go`.
- [x] 5.5 Verify `pkg/loader/instance_kind.go`'s `DetectInstanceKind` default-arm message and the `kindModuleInstance` const are consistent (a stray `"ModuleRelease"` fixture should now route to `unknown instance kind`).
- [x] 5.6 Update `internal/workflow/query/inventory_test.go` label literal `module-release.opmodel.dev/name` → `module-instance.opmodel.dev/name` (shared file with X3 — X4 owns the label keys; X3 owned the verbs).

## 6. Validation gate (held from archive — one-PR bulk-archive model)

- [x] 6.1 `grep -rn "Release\b\|release-\|module-release" pkg internal tests examples --include=*.go --include=*.cue` and confirm every survivor is a `// Was:` breadcrumb, the unrelated `--name` flag, or a legitimately non-renamed token (`createdBy` provenance, `lastTransitionTime`). No live `Release`-named identifier for the renamed concept remains.
- [x] 6.2 `go build ./...` + `go vet ./...` green.
- [x] 6.3 `task lint` — 0 new issues vs the X3 baseline.
- [~] 6.4 **`task test` green** — unit (`go test ./internal/... ./pkg/...`, 24 pkgs) ✅ and e2e (`go test ./tests/e2e/...`) ✅ pass locally. `task test:integration` requires a live kind cluster (not available in this env); the integration programs compile (`go vet ./...`) and their functional fixes are in (JSON tag `release`→`instance` key checks in `inst-tree`, `opm instance build` assertion in `mod_build_test`). Run integration against a cluster before the PR merges.
- [x] 6.5 `openspec validate --strict migrate-label-domain-and-inventory` passes.
- [ ] 6.6 `openspec-verify-change` across X1–X4; then open the single atomic `cli` PR and bulk-archive (`openspec-bulk-archive-change`) — expect `--skip-specs` + a manual `release-*`→`instance-*` main-spec folder hygiene pass (D-X3.4 malformed-header hazard). Record the X-wave `config.yaml` history event in enhancement 0002 at archive time.
