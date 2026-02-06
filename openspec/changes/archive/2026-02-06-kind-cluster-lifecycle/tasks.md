## 1. Kind cluster configuration

- [x] 1.1 Create `hack/` directory
- [x] 1.2 Create `hack/kind-config.yaml` with a single-node cluster definition (`apiVersion: kind.x-k8s.io/v1alpha4`, `kind: Cluster`, one node with role `control-plane`)

## 2. Taskfile variables

- [x] 2.1 Add `CLUSTER_NAME: opm-dev` to the `vars:` section of `Taskfile.yml`
- [x] 2.2 Add `K8S_VERSION: "1.34.0"` to the `vars:` section of `Taskfile.yml`

## 3. Cluster tasks

- [x] 3.1 Add `cluster:create` task with precondition checking `command -v kind`, running `kind create cluster --name {{.CLUSTER_NAME}} --config hack/kind-config.yaml --image kindest/node:v{{.K8S_VERSION}}`
- [x] 3.2 Add `cluster:delete` task with precondition checking `command -v kind`, running `kind delete cluster --name {{.CLUSTER_NAME}}`
- [x] 3.3 Add `cluster:status` task with precondition checking `command -v kind`, using `kind get clusters | grep -q {{.CLUSTER_NAME}}` to check existence, then `kubectl cluster-info --context kind-{{.CLUSTER_NAME}}` if found, or printing a "not running" message if not
- [x] 3.4 Add `cluster:recreate` task that composes `cluster:delete` then `cluster:create` via Taskfile `task:` directives in `cmds`

## 4. Test task documentation

- [x] 4.1 Add a `summary` field to the existing `test:integration` task explaining that a running cluster is required and referencing `task cluster:create` — do not change the `cmds` list

## 5. Validation

- [x] 5.1 Verify `task --list` shows all four new `cluster:*` tasks with descriptions
- [x] 5.2 Run `task cluster:create` and confirm a kind cluster named `opm-dev` is created with Kubernetes v1.34.0
- [x] 5.3 Run `task cluster:status` and confirm it reports the running cluster with connection details
- [x] 5.4 Run `task cluster:delete` and confirm the cluster is destroyed
- [x] 5.5 Run `task cluster:delete` again and confirm it completes without error (idempotency)
- [x] 5.6 Run `task cluster:recreate` and confirm it provisions a fresh cluster
- [x] 5.7 Verify precondition error message appears when `kind` is not on PATH (e.g., temporarily rename the binary)
- [x] 5.8 Run `task test:integration --summary` and confirm it mentions the cluster prerequisite
