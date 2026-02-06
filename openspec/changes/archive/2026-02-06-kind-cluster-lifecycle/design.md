## Context

The Taskfile currently has build, test, lint, and formatting tasks but no cluster management. The `test:integration` task runs `go test ./tests/integration/... -v` against a nonexistent directory — it assumes a cluster already exists but provides no way to create one. Contributors must manually install kind and run ad-hoc commands to stand up a cluster before they can develop or run integration tests.

The Taskfile uses version 3, defines global vars at the top (`BINARY_NAME`, `BUILD_DIR`, etc.), and groups tasks by namespace prefix (e.g., `test:*`, `build:*`, `lint:*`). New cluster tasks should follow this existing pattern.

## Goals / Non-Goals

**Goals:**
- Provide `cluster:create`, `cluster:delete`, `cluster:status`, and `cluster:recreate` Taskfile tasks
- Pin a reproducible kind cluster configuration in a version-controlled file
- Pin Kubernetes 1.34.0 as the default cluster version for reproducibility
- Fail fast with a clear error if `kind` is not installed
- Make cluster name and Kubernetes version configurable via Taskfile vars with sensible defaults
- Keep cluster lifecycle explicit — never auto-create or auto-destroy

**Non-Goals:**
- Multi-node cluster configurations (single node is sufficient for current needs)
- CI/CD pipeline integration (future concern — this targets local development)
- envtest setup (complementary but separate tooling, different testing strategy)
- Custom CNI, ingress, or registry configuration (add when needed per Principle VII)
- Changes to any Go source code or CLI commands

## Decisions

### 1. Cluster config location: `hack/kind-config.yaml`

Place the kind configuration file at `hack/kind-config.yaml`.

**Why `hack/`**: The `hack/` directory is the Kubernetes ecosystem convention for development scripts and configuration (used by kubernetes/kubernetes, controller-runtime, and many CNCF projects). It signals "developer tooling, not shipped to users." The alternative — `tests/kind-config.yaml` — conflates test fixtures with infrastructure config.

**Why not inline in Taskfile**: A separate YAML file is easier to extend (e.g., adding extra port mappings, mount paths) without cluttering the Taskfile. It also allows `kind create cluster --config` to validate the file independently.

### 2. Task namespace: `cluster:*`

Use `cluster:` as the task prefix rather than `kind:`.

**Rationale**: The abstraction is "manage a local cluster," not "invoke the kind binary." If the project later supports k3d or minikube as alternatives, the `cluster:` namespace remains valid. The kind-specific implementation is an internal detail.

### 3. Configurable vars with defaults

Add two new Taskfile vars:

| Var | Default | Purpose |
|-----|---------|---------|
| `CLUSTER_NAME` | `opm-dev` | Name passed to `kind create/delete cluster --name` |
| `K8S_VERSION` | `1.34.0` | Node image tag for `--image kindest/node:v<version>` |

**Why `opm-dev`**: Distinguishes OPM development clusters from other kind clusters a developer may have. Short, descriptive, unlikely to collide.

**Why pin K8S_VERSION to `1.34.0`**: Ensures all developers and future CI run the same Kubernetes version by default. Kubernetes 1.34 is the current stable release and the `kindest/node:v1.34.0` image is available. Pinning prevents drift between developer environments. Developers can still override with `task cluster:create K8S_VERSION=1.33.0` when testing compatibility.

### 4. Precondition: fail if `kind` not on PATH

Each cluster task will include a Taskfile `preconditions` check:

```yaml
preconditions:
  - sh: command -v kind
    msg: "kind is not installed. Install it: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
```

**Rationale**: Matches the `test:run` task's pattern of using preconditions for required inputs. Gives an actionable error message rather than a cryptic "command not found."

### 5. `cluster:status` uses `kind get clusters` + `kubectl cluster-info`

Rather than parsing docker containers or kind internals, use:
1. `kind get clusters` to check if the named cluster exists
2. `kubectl cluster-info --context kind-<name>` for connection details (only if the cluster exists)

**Rationale**: Both commands are stable, user-facing APIs. This avoids coupling to kind's internal implementation (e.g., container naming conventions).

### 6. `cluster:recreate` composes delete + create

Implemented as:

```yaml
cluster:recreate:
  cmds:
    - task: cluster:delete
    - task: cluster:create
```

**Rationale**: Follows the Taskfile composition pattern already used by `check` (which composes `fmt`, `vet`, `lint`, `test`). No special logic needed — `cluster:delete` is idempotent (kind ignores deleting nonexistent clusters).

### 7. Do not modify `test:integration` or `test:e2e` task behavior

The proposal mentioned updating these tasks to "document the cluster prerequisite." After consideration, documentation belongs in a README or task `summary` field, not in preconditions that would prevent running tests without kind. Integration tests may use envtest (no cluster needed) in the future.

**Decision**: Add a `summary` field to `test:integration` noting the cluster requirement, but do not add preconditions or dependencies on `cluster:create`.

## Risks / Trade-offs

**[kind binary not versioned]** → Tasks depend on whatever version of kind the developer has installed. Different kind versions may produce different cluster behavior.
→ *Mitigation*: Acceptable for local development. Pin kind version in CI when that becomes a goal. Document minimum supported version in the task summary.

**[Docker dependency]** → Kind requires Docker (or Podman). Developers without Docker cannot use these tasks.
→ *Mitigation*: This is an inherent kind constraint, not something we can work around. The precondition check will catch this indirectly (kind itself fails without a container runtime). Out of scope to support alternatives here.

**[Cluster left running]** → Developers may forget to run `cluster:delete`, leaving Docker resources consuming memory.
→ *Mitigation*: Document cleanup in task descriptions. The `opm-dev` naming convention makes it easy to identify stale clusters. A future `clean` task could optionally include cluster teardown, but per Principle VII, not now.

**[Windows portability]** → `command -v kind` and `grep -q` are POSIX shell features that may not work on Windows cmd/PowerShell.
→ *Mitigation*: Taskfile v3 runs commands via `sh` by default, which works on Windows with Git Bash or WSL. Pure Windows cmd is not a supported development environment for this project (Go development on Windows typically uses Git Bash or WSL). Acceptable trade-off given Principle V scope.

**[Pinned K8S_VERSION may go stale]** → As new Kubernetes releases come out, `1.34.0` will age.
→ *Mitigation*: Updating the default is a one-line Taskfile var change. This is preferable to unpinned defaults where different developers silently run different versions.
