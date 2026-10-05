## 1. Pin the order and the digest the move must keep

- [ ] 1.1 Add `internal/kubernetes/order_parity_test.go` with `TestWeightTableMatchesRetiredCopy`: literal maps of every `Weight*` constant, every `gvkWeights` entry and every `kindWeights` entry of `pkg/resourceorder/weights.go` at `be1157d5`, plus two fallback rows (`example.com/v1 Foo` weighs 1000; `autoscaling/v2beta2 HorizontalPodAutoscaler` weighs 200). Assert each literal against both `resourceorder.GetWeight` and `object.Weight`, and each constant against both `resourceorder.Weight*` and `object.Weight*`. Both tables are unexported, so the test cannot count entries; a kind the library adds later is caught by the library's own `TestWeightTableGuard`, not here (design KO1)
- [ ] 1.2 In the same file add `TestSortMatchesRetiredCopy`: one shuffled set of unstructured objects covering every weight class, two ConfigMaps and two Deployments in a fixed input order, and one unknown kind; sort copies with `resourceorder.Sort` and `object.Sort`, ascending and descending, and require identical name sequences (this also pins stability on equal weights)
- [ ] 1.3 In `internal/inventory/digest_test.go` add `TestComputeRenderDigest_Golden`: the digest of `renderResources(t)` equals a literal computed at the base of this change (run once, paste the value). The test comment says the value must not move until the inventory adoption changes the digest on purpose (design KO2)
- [ ] 1.4 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `test: pin the weight table and render digest before the Kubernetes tier move`

## 2. Labels from opm/k8s/labels

- [ ] 2.1 Replace every `pkg/core` label constant and `IsOPMManagedBy` use with `github.com/open-platform-model/library/opm/k8s/labels`, imported as `opmlabels` (design KO4), by the table in proposal.md "Migration note": `internal/inventory/{cr,legacy,stale,store}.go` (`cr.go` keeps `LabelInstanceUUID` as an alias of `opmlabels.ModuleInstanceUUID`), `internal/kubernetes/delete.go`, `internal/operator/migration_proof.go`, `internal/platform/cluster.go`, `pkg/inventory/entry.go`
- [ ] 2.2 Move the tests the same way: `internal/cmd/instance/delete_test.go`, `internal/inventory/legacy_test.go`, `internal/kubernetes/delete_test.go`, `pkg/inventory/types_test.go`
- [ ] 2.3 Move the integration programs: `tests/integration/{deploy,inst-list,inst-tree,inventory-apply,inventory-ops,migration}/main.go`; `go vet ./tests/integration/...` compiles all of them
- [ ] 2.4 Confirm `grep -rn 'pkgcore\.Label\|pkgcore\.IsOPMManagedBy' --include=*.go . | grep -v '^./pkg/core/'` finds nothing (evidence for the spec scenario "Label values are unchanged")
- [ ] 2.5 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `refactor: read OPM label keys from the library labels package`

## 3. One export for the render digest and the apply objects

- [ ] 3.1 Change `inventory.ComputeRenderDigest` to take `[]object.Exported`: sort a copy, stable, by group (from `Object.GroupVersionKind()`), kind, namespace and name, then hash each `JSON` in that order. Keep the doc comment's parity note and reword it to say the bytes are the single export's JSON (design KO2)
- [ ] 3.2 In `internal/workflow/render/render.go`, replace the inline `pkgcore.Resource` loop, the digest call and the `ToUnstructured` loop with one `object.Export(object.Resources(out.Compiled))`, the digest over its result and `result.Resources` from each `Exported.Object` in order. An export failure returns `ExitGeneralError` with `converting rendered resources: %w` (design KO3). Drop the `pkgcore` import
- [ ] 3.3 Move `internal/inventory/digest_test.go` to build its input with `object.Export` over the CUE test resources (`object.Resource` in `cueResource`); every existing digest test and the golden from 1.3 stay green unchanged
- [ ] 3.4 Move `tests/integration/render-parity/main.go` to `object.Resources` plus `object.Export` before `ComputeRenderDigest`, and build it (`go vet ./tests/integration/render-parity/`)
- [ ] 3.5 Confirm `grep -n 'MarshalJSON\|ToUnstructured\|pkgcore' internal/workflow/render/render.go` finds nothing (evidence for the spec scenario "The digest and the apply objects come from one export")
- [ ] 3.6 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `refactor(render): feed the render digest and the apply objects from one export`

## 4. Order by the library weight table

- [ ] 4.1 `internal/kubernetes/sort.go`: `SortObjects(objs, dir object.Direction)` calls `object.Sort` (design KO5). Move `apply.go`, `delete.go`, `tree.go` and `delete_test.go` to `object.Ascending` and `object.Descending`
- [ ] 4.2 `internal/inventory/stale.go`: prune order through `object.Sort(..., object.Descending)`
- [ ] 4.3 `internal/output/manifest.go`: the display sort reads the weight from `object.Weight`; its namespace and name keys stay
- [ ] 4.4 Confirm `grep -rn 'resourceorder' --include=*.go . | grep -v '^./pkg/resourceorder/\|order_parity_test.go'` finds nothing (evidence for the spec requirement "Object order comes from the library weight table"), and that the section-1 parity tests and the existing apply, delete, tree and manifest order tests pass unchanged
- [ ] 4.5 `task fmt`, `task lint`, `task test:unit` and `task openspec:check` green, then commit `refactor(kubernetes): order objects by the library weight table`

## 5. Delete pkg/core and pkg/resourceorder

- [ ] 5.1 Delete `pkg/core/` and `pkg/resourceorder/` with their tests
- [ ] 5.2 In `internal/kubernetes/order_parity_test.go`, drop the `resourceorder` side of both tests: the literal table now asserts `object.Weight` and the `object.Weight*` constants alone, and the sort test compares `object.Sort` against the literal expected name sequences it produced in section 1. Reword the test comments to say the literals are the table the cli applied by before the move
- [ ] 5.3 `.golangci.yml`: enable `depguard` with one rule over all files that denies `github.com/open-platform-model/cli/pkg/core` (message: use `github.com/open-platform-model/library/opm/k8s/object` or `.../opm/k8s/labels`) and `github.com/open-platform-model/cli/pkg/resourceorder` (message: use `.../opm/k8s/object`), no allow list (design KO6). `task lint` stays green. Then prove the rule fires: in a scratch copy of the worktree outside the repo, add a stub `pkg/core` package and one import of it, run golangci-lint there and see the depguard finding; never commit the stub
- [ ] 5.4 `AGENTS.md` project layout: one line saying the Kubernetes object wrapper, the label vocabulary and the weight table are the library's `opm/k8s/object` and `opm/k8s/labels`, with no local copy (a depguard rule refuses the old `pkg/core` and `pkg/resourceorder` paths)
- [ ] 5.5 Confirm `grep -rn 'pkg/core\|pkg/resourceorder\|resourceorder\.\|pkgcore' --exclude-dir=.git --exclude-dir=archive .` finds only the depguard rule, the AGENTS.md line, this change's own files, `docs/rfc/0007-*.md` (a historical RFC, left as written) and the main specs this change's deltas replace at archive
- [ ] 5.6 Cluster suites under the kind lock: `flock <lock> task test:integration` and `flock <lock> go test ./tests/e2e/... -v -timeout 25m`, where `<lock>` is the lock file that serializes use of the kind-opm-dev cluster; then `task test:fixtures` (render-parity against the local registry; start it from the workspace root with `task registry:start` if needed). A SKIP or an unavailable registry counts as a failed task; if the shared local registry lacks an artifact, do not publish to it, record which one and leave the run to PR CI
- [ ] 5.7 `task fmt`, `task lint`, `task test` and `task openspec:check` green, then commit `feat!: delete pkg/core and pkg/resourceorder in favour of the library Kubernetes tier` with a body naming the replacement packages and the footer `BREAKING CHANGE: pkg/core and pkg/resourceorder are removed; importers use github.com/open-platform-model/library/opm/k8s/object and opm/k8s/labels (see the migration note in the proposal)`
