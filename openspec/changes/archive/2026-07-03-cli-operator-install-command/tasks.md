# Tasks: cli-operator-install-command

## 1. Field-manager rename (lands first — design decision 8)

- [x] 1.1 Rename `fieldManagerName` from `opm` to `opm-cli` in `internal/kubernetes/labels.go`; sweep tests asserting the old value; run `go test ./internal/kubernetes/...`

## 2. Manifest engine (`internal/operator`, pure — no cluster)

- [x] 2.1 Add `internal/operator/manifest.go`: `go:embed` of the pinned `dist/install.yaml` + `PinnedOperatorVersion` constant, and multi-doc YAML parse to `[]*unstructured.Unstructured` via `k8s.io/apimachinery/pkg/util/yaml`; unit tests parse the real embedded artifact (16 docs, expected kinds)
- [x] 2.2 Add plan functions: install plan (all docs, ascending `pkg/resourceorder` weight), CRDs-only plan (`kind == CustomResourceDefinition`), uninstall plan (all minus CRDs minus Namespace, descending weight); table-driven unit tests including the never-delete-CRDs/Namespace property
- [x] 2.3 Add `task operator:sync VERSION=<tag>` to `Taskfile.yml` (download release asset, replace embed, rewrite pin constant); run it against `v1.0.0-alpha.2` to seed the embedded artifact
- [x] 2.4 Add `internal/operator/fetch.go`: `--version` release-asset fetch over HTTPS (bounded timeout, error naming tag + URL on failure); unit test the URL construction and error shaping with a stub server

## 3. Apply/wait primitives (`internal/kubernetes` + `internal/operator`)

- [x] 3.1 Export a single-resource SSA helper from `internal/kubernetes/apply.go` (promote `applyResource`; `Apply` delegates to it — design decision 2); run existing apply tests unchanged
- [x] 3.2 Add bounded readiness wait in `internal/operator/wait.go`: poll loop (2s interval, `--timeout` bound, context-cancellable) over per-object predicates — CRD `Established=True` (new check) and Deployment rollout via `kubernetes.EvaluateHealth`; unit test predicates against fixture objects, wait loop against a fake that flips ready

## 4. `opm operator install`

- [x] 4.1 Add `internal/operator/install.go`: run a plan through the SSA helper with per-resource status lines, then the readiness wait; wire `--crds-only`, `--version`, `--timeout`
- [x] 4.2 Add `--rbac [--user|--group]`: build `opm-cli-user` ClusterRole (+ ClusterRoleBinding when a subject is given) as unstructured objects appended to the plan; flag-validation error for `--user`/`--group` without `--rbac`; unit test object shapes and flag validation
- [x] 4.3 Add `internal/cmd/operator/` (group + `install.go` cobra wiring, thin per cmd-structure delta) and register `cmdoperator.NewOperatorCmd(&cfg)` in `internal/cmd/root.go`

## 5. `opm operator uninstall`

- [x] 5.1 Add finalizer guard in `internal/operator/uninstall.go`: cluster-wide `moduleinstances` list via the dynamic client (fail closed on list error), refusal naming each `namespace/name` carrying `opmodel.dev/cleanup`; `--remove-finalizers` strips exactly that finalizer via JSON patch and states the orphaning consequence; unit test guard logic against fake dynamic client
- [x] 5.2 Execute the uninstall plan (descending weight, fire-and-report) and add `internal/cmd/operator/uninstall.go` wiring

## 6. Verification and docs

- [x] 6.1 e2e against kind (`task cluster:create`): install → idempotent re-install → `--crds-only` on a fresh cluster → uninstall with an armed finalizer (refusal, then `--remove-finalizers`) → CRDs/Namespace survive; first confirm the pinned operator image is pullable/preloadable in the e2e environment, else assert through CRD `Established` + Deployment created (design risk 1)
- [x] 6.2 Update `README.md` command groups; run `task fmt`, `task lint`, `task test`
