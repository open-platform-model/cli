## Requirements

### Requirement: Tree discovers resources via inventory

The `opm mod tree` command SHALL discover deployed resources by looking up the OPM inventory Secret for the release (via `cmdutil.ResolveInventory` / `inventory.DiscoverResourcesFromInventory`), then fetching each tracked resource by GVK + name. It MUST NOT require module source or re-rendering.

#### Scenario: Tree shows deployed resources by release name

- **WHEN** the user runs `opm mod tree --release-name jellyfin -n media`
- **THEN** the command SHALL look up the inventory Secret for release name `jellyfin` in namespace `media` and display all tracked resources

#### Scenario: Tree shows deployed resources by release ID

- **WHEN** the user runs `opm mod tree --release-id abc123-def456 -n media`
- **THEN** the command SHALL look up the inventory Secret for release ID `abc123-def456` in namespace `media` and display all tracked resources

#### Scenario: No resources found

- **WHEN** no inventory Secret (or no tracked resources) is found for the given release selector and namespace
- **THEN** the command SHALL exit with code 5 and display error "no resources found for release <name|id> in namespace <namespace>"

---

### Requirement: Tree groups resources by component

The command SHALL group resources by the `component.opmodel.dev/name` label recorded in the inventory. Resources SHALL be sorted alphabetically by component name. Resources without a component label SHALL be grouped under a special section labeled `(no component)` displayed last.

#### Scenario: Resources grouped by component label

- **WHEN** resources have labels `component.opmodel.dev/name=server`, `component.opmodel.dev/name=database`, and `component.opmodel.dev/name=ingress`
- **THEN** the tree output SHALL show three component groups: `database`, `ingress`, `server` (alphabetically sorted)

#### Scenario: Resources without component label

- **WHEN** some resources lack the `component.opmodel.dev/name` label
- **THEN** those resources SHALL be grouped under `(no component)` section displayed after all named components

#### Scenario: Empty component label treated as missing

- **WHEN** a resource has `component.opmodel.dev/name=""` (empty string)
- **THEN** the resource SHALL be treated as having no component label and grouped under `(no component)`

---

### Requirement: Tree walks Kubernetes ownership chains

The command SHALL walk Kubernetes `ownerReferences` to discover child resources for workload kinds (Deployment, StatefulSet, DaemonSet, Job). Child resources SHALL be displayed as nested nodes in the tree hierarchy.

#### Scenario: Deployment shows ReplicaSets and Pods

- **WHEN** a Deployment resource is discovered
- **THEN** the command SHALL query for ReplicaSets with `ownerReferences[].uid` matching the Deployment UID
- **AND** for each ReplicaSet, SHALL query for Pods with `ownerReferences[].uid` matching the ReplicaSet UID
- **AND** display the hierarchy as Deployment → ReplicaSet → Pod

#### Scenario: StatefulSet shows Pods directly

- **WHEN** a StatefulSet resource is discovered
- **THEN** the command SHALL query for Pods with `ownerReferences[].uid` matching the StatefulSet UID
- **AND** display Pods as direct children (no ReplicaSet layer)

#### Scenario: DaemonSet shows Pods

- **WHEN** a DaemonSet resource is discovered
- **THEN** the command SHALL query for Pods with `ownerReferences[].uid` matching the DaemonSet UID
- **AND** display Pods as direct children

#### Scenario: Job shows Pods

- **WHEN** a Job resource is discovered
- **THEN** the command SHALL query for Pods with `ownerReferences[].uid` matching the Job UID
- **AND** display Pods as direct children

#### Scenario: Passive resources have no children

- **WHEN** a passive resource (ConfigMap, Secret, Service, PVC, etc.) is discovered
- **THEN** the command SHALL NOT attempt to query for child resources
- **AND** the resource SHALL be displayed as a leaf node in the tree

#### Scenario: Ownership walking fails gracefully

- **WHEN** querying for child resources fails (e.g., RBAC permissions denied)
- **THEN** the command SHALL log a debug message with the error
- **AND** display the parent resource without children
- **AND** continue processing other resources

---

### Requirement: Tree displays health status and replica counts

The command SHALL display health status for each resource using the same evaluation logic as `mod status`. For workload resources (Deployment, StatefulSet, DaemonSet), it SHALL display replica counts in `ready/desired` format. For Pod nodes, the raw Kubernetes phase string SHALL be displayed (matching `mod status` pod display).

#### Scenario: Workload shows replica count

- **WHEN** a Deployment has `status.readyReplicas=3` and `status.replicas=3`
- **THEN** the tree output SHALL display `Deployment/name  Ready  3/3`

#### Scenario: Degraded workload shows current vs desired

- **WHEN** a Deployment has `status.readyReplicas=1` and `status.replicas=3`
- **THEN** the tree output SHALL display `Deployment/name  NotReady  1/3`

#### Scenario: Job shows completion count

- **WHEN** a Job has `status.succeeded=5` and `spec.completions=10`
- **THEN** the tree output SHALL display `Job/name  NotReady  5/10`

#### Scenario: ReplicaSet shows pod count only (no health status)

- **WHEN** a ReplicaSet has `status.replicas=3`
- **THEN** the tree output SHALL display `ReplicaSet/name  3 pods`
- **AND** SHALL NOT display a health status for the ReplicaSet node
  (health is implicit in the parent Deployment's replica ratio)

#### Scenario: Pod shows phase status

- **WHEN** a Pod has `status.phase=Running`
- **THEN** the tree output SHALL display `Pod/name  Running`

#### Scenario: Pod shows detailed container status

- **WHEN** a Pod has `status.containerStatuses[].state.waiting.reason=CrashLoopBackOff`
- **THEN** the tree output SHALL display `Pod/name  CrashLoop`
  (CrashLoopBackOff is shortened to CrashLoop per the project's display convention)

#### Scenario: Passive resource shows Ready

- **WHEN** a ConfigMap, Secret, or Service is discovered
- **THEN** the tree output SHALL display the resource with status `Ready`

#### Scenario: PVC shows phase and storage capacity

- **WHEN** a PersistentVolumeClaim is discovered
- **THEN** the tree output SHALL display the kind abbreviated as `PVC` (e.g. `PVC/data`)
- **AND** SHALL display the PVC lifecycle phase (`Bound`, `Pending`, or `Lost`) as the status
  - `Bound` → green (healthy — volume is attached)
  - `Pending` → yellow (transitional — waiting for provisioning)
  - `Lost` → yellow (warning — backing PersistentVolume is gone)
- **AND** SHALL display the storage capacity after the status (from `status.capacity.storage`,
  falling back to `spec.resources.requests.storage`), e.g. `PVC/data  Bound  10Gi`
- **AND** PVC nodes with no `status.phase` set SHALL display `Ready` (fallback for unprovisioned PVCs)

---

### Requirement: Tree supports depth control

The command SHALL support a `--depth` flag with values 0, 1, or 2 to control tree depth. The default depth SHALL be 2. Invalid depth values SHALL be rejected with an error.

#### Scenario: Depth 0 shows component summary

- **WHEN** the user runs `opm mod tree --release-name app -n ns --depth 0`
- **THEN** the output SHALL display component names with resource counts and aggregate status
- **AND** SHALL NOT display individual resources or Kubernetes-owned children
- **AND** SHALL NOT query the cluster for child resources

#### Scenario: Depth 1 shows components and OPM resources

- **WHEN** the user runs `opm mod tree --release-name app -n ns --depth 1`
- **THEN** the output SHALL display component groups and OPM-managed resources
- **AND** SHALL NOT display Kubernetes-owned children (Pods, ReplicaSets)
- **AND** SHALL NOT query the cluster for child resources

#### Scenario: Depth 2 shows full hierarchy

- **WHEN** the user runs `opm mod tree --release-name app -n ns --depth 2` OR omits `--depth`
- **THEN** the output SHALL display components, OPM-managed resources, and Kubernetes-owned children
- **AND** SHALL query the cluster for ReplicaSets and Pods as needed

#### Scenario: Invalid depth rejected

- **WHEN** the user runs `opm mod tree --release-name app -n ns --depth 5`
- **THEN** the command SHALL exit with code 1 and display error "invalid depth: must be 0, 1, or 2"

---

### Requirement: Tree renders with colored box-drawing characters

The command SHALL render tree structure using Unicode box-drawing characters (├── └── │) with colors applied via lipgloss. Color stripping for non-TTY environments is handled automatically by lipgloss/termenv (which detects `Ascii` color profile on non-TTY stdout).

#### Scenario: Colored tree rendering in TTY

- **WHEN** the command runs in a TTY environment
- **THEN** the output SHALL use Unicode box-drawing characters for tree structure
- **AND** SHALL apply colors: cyan for component names, green for Ready/Bound status, red for NotReady, yellow for Pending/Lost/Unknown, dim gray for tree chrome

#### Scenario: Status column alignment

- **WHEN** a component contains multiple resources (including children at different nesting depths)
- **THEN** the status tokens for all nodes SHALL align to the same horizontal column (two-pass rendering: measure first, then render)
- **AND** there SHALL be a minimum gap of 2 spaces between the end of the resource name and the status token

#### Scenario: Status appears before replica count

- **WHEN** a resource has both a status and a replica count (e.g. Deployment with `Ready  3/3`)
- **THEN** the status token SHALL appear before the replica count on the same line

#### Scenario: Plain rendering in non-TTY

- **WHEN** the command runs in a non-TTY environment (e.g., CI pipeline)
- **THEN** the output SHALL use Unicode box-drawing characters for tree structure
- **AND** SHALL NOT apply ANSI color codes

#### Scenario: Tree chrome uses box-drawing vocabulary

- **WHEN** rendering tree structure
- **THEN** the output SHALL use `├──` for branches with continuation, `└──` for final branches, and `│` for vertical continuation lines

---

### Requirement: Tree supports structured output formats

The command SHALL support `--output`/`-o` with values `table` (default), `json`, and `yaml` for machine-readable output. JSON and YAML output MUST be stable and contain no ANSI color codes.

#### Scenario: Default table output

- **WHEN** the user runs `opm mod tree --release-name app -n ns` without `--output`
- **THEN** the output SHALL be a colored tree with box-drawing characters

#### Scenario: JSON output

- **WHEN** the user runs `opm mod tree --release-name app -n ns -o json`
- **THEN** the output SHALL be valid JSON with structure: `{"release": {...}, "components": [...]}`
- **AND** SHALL contain no ANSI color codes

#### Scenario: YAML output

- **WHEN** the user runs `opm mod tree --release-name app -n ns -o yaml`
- **THEN** the output SHALL be valid YAML with the same structure as JSON
- **AND** SHALL contain no ANSI color codes

#### Scenario: JSON schema includes nested children

- **WHEN** outputting JSON at depth 2
- **THEN** resources with children SHALL have a `children` array with nested child resources
- **AND** each child SHALL recursively contain its own `children` array if applicable

---

### Requirement: Tree accepts release selector flags

The command SHALL accept release selector flags following the same pattern as `mod status` and `mod delete`: exactly one of `--release-name` or `--release-id` MUST be provided, and `--namespace` is required.

#### Scenario: Release name and namespace required

- **WHEN** the user runs `opm mod tree --release-name app -n production`
- **THEN** the command SHALL discover resources using the release name selector

#### Scenario: Release ID selector

- **WHEN** the user runs `opm mod tree --release-id abc123 -n production`
- **THEN** the command SHALL discover resources using the release ID selector

#### Scenario: Both release-name and release-id rejected

- **WHEN** the user provides both `--release-name app` and `--release-id abc123`
- **THEN** the command SHALL exit with code 1 and error "--release-name and --release-id are mutually exclusive"

#### Scenario: Neither release-name nor release-id provided

- **WHEN** the user omits both `--release-name` and `--release-id`
- **THEN** the command SHALL exit with code 1 and error "either --release-name or --release-id is required"

#### Scenario: Missing namespace flag

- **WHEN** the user runs `opm mod tree --release-name app` without `-n`
- **THEN** the command SHALL exit with code 1 and display a usage error indicating `-n` is required

---

### Requirement: Tree accepts Kubernetes connection flags

The command SHALL accept `--kubeconfig` and `--context` flags for cluster connection. Kubeconfig resolution SHALL follow: explicit flag > `KUBECONFIG` env var > default path (`~/.kube/config`).

#### Scenario: Custom kubeconfig

- **WHEN** the user runs `opm mod tree --kubeconfig /custom/config --release-name app -n ns`
- **THEN** the command SHALL use the specified kubeconfig file

#### Scenario: Custom context

- **WHEN** the user runs `opm mod tree --context prod-cluster --release-name app -n ns`
- **THEN** the command SHALL use the specified Kubernetes context

#### Scenario: Default kubeconfig resolution

- **WHEN** the user runs `opm mod tree --release-name app -n ns` without `--kubeconfig`
- **THEN** the command SHALL resolve kubeconfig from `KUBECONFIG` env var if set, otherwise `~/.kube/config`

---

### Requirement: Tree fails fast on connectivity errors

The command SHALL fail immediately with a clear error message if the Kubernetes cluster is unreachable or authentication fails.

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with code 1 and display a connectivity error message

#### Scenario: Authentication failure

- **WHEN** the kubeconfig credentials are invalid or expired
- **THEN** the command SHALL exit with code 1 and display an authentication error message

---

### Requirement: Tree displays release metadata header

The command SHALL display release metadata in the header: release name, module FQN (if available from labels), version, and namespace.

#### Scenario: Header with module FQN and version

- **WHEN** resources have labels `module-release.opmodel.dev/name=jellyfin-media` and `module-release.opmodel.dev/version=1.2.0`
- **THEN** the tree header SHALL display `jellyfin-media (opmodel.dev/community/jellyfin@1.2.0)` or equivalent metadata

#### Scenario: Header without module FQN

- **WHEN** resources lack module FQN metadata
- **THEN** the tree header SHALL display release name and version only

---

### Requirement: Tree sorts resources within components by weight

Within each component group, resources SHALL be sorted by OPM weight (ascending) and then alphabetically by name. This ensures tree output matches apply order. In practice, this ordering is preserved by maintaining the inventory entry order, which is written in weight-sorted order during `opm mod apply`.

#### Scenario: Resources sorted by weight

- **WHEN** a component contains a Deployment (weight 100), a Service (weight 50), and a ConfigMap (weight 15)
- **THEN** the tree SHALL display ConfigMap, then Service, then Deployment (ascending weight order)

#### Scenario: Resources with equal weight sorted by name

- **WHEN** a component contains two ConfigMaps named `config-a` and `config-z` (both weight 15)
- **THEN** the tree SHALL display `config-a` before `config-z` (alphabetical)

---

### Requirement: Tree exit codes match CLI conventions

The command SHALL use exit codes consistently with other CLI commands: 0 for success, 1 for general errors (invalid flags, connectivity failures), 5 for resource not found.

#### Scenario: Successful tree display

- **WHEN** the tree renders successfully
- **THEN** the command SHALL exit with code 0

#### Scenario: Invalid flags

- **WHEN** the user provides invalid or mutually exclusive flags
- **THEN** the command SHALL exit with code 1

#### Scenario: No resources found

- **WHEN** no resources match the selector
- **THEN** the command SHALL exit with code 5

#### Scenario: Cluster connectivity error

- **WHEN** the cluster is unreachable
- **THEN** the command SHALL exit with code 1
