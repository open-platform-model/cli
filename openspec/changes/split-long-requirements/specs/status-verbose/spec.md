## REMOVED Requirements

### Requirement: Verbose mode shows pod details for unhealthy workloads
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`--details` shows a pod detail block for a NotReady workload", "The pod detail block lists each pod of the unhealthy workload with name, phase and detail", "Pods of the unhealthy workload are discovered by its `.spec.selector.matchLabels`".

### Requirement: Verbose pod detail block format
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The pod detail block has a fixed layout", "The pod detail block is colour-coded by ready ratio, phase, detail text and restart count".

## ADDED Requirements

### Requirement: `--details` shows a pod detail block for a NotReady workload

The command SHALL accept a `--details` flag. When `--details` is set and a workload resource (Deployment, StatefulSet, DaemonSet) has a health status of `NotReady`, the output SHALL include a pod detail block for that workload below the resource table.

#### Scenario: Verbose shows pod details for NotReady Deployment

- **WHEN** the user runs `opm instance status my-app -n prod --details`
- **AND** a Deployment `web` has health status `NotReady`
- **AND** the Deployment has 3 pods: one Running/Ready, one in CrashLoopBackOff, one Pending
- **THEN** the output SHALL include a pod detail block for `Deployment/web` showing all 3 pods with their phase and details

#### Scenario: Verbose omits pod details for Ready workloads

- **WHEN** `--details` is set
- **AND** a Deployment has health status `Ready`
- **THEN** no pod detail block SHALL be rendered for that Deployment

#### Scenario: Verbose omits pod details for non-workload resources

- **WHEN** `--details` is set
- **AND** a ConfigMap has health status `Ready`
- **THEN** no pod detail block SHALL be rendered for that ConfigMap

### Requirement: The pod detail block lists each pod of the unhealthy workload with name, phase and detail

The pod detail block SHALL list each pod belonging to the unhealthy workload with:
- **Pod name**: the full pod name
- **Phase**: the pod's phase (Running, Pending, Failed, Succeeded, Unknown) or a container-level waiting reason if more specific (CrashLoopBackOff, ImagePullBackOff, etc.)
- **Detail**: for unhealthy pods, the reason and context (e.g., "OOMKilled, 5 restarts"). For healthy pods, "(ready)".
(Requirement: `--details` shows a pod detail block for a NotReady workload)

#### Scenario: A healthy pod reads (ready)

- **WHEN** the pod detail block lists a healthy pod of the unhealthy workload
- **THEN** the detail of that pod SHALL be "(ready)"

### Requirement: Pods of the unhealthy workload are discovered by its `.spec.selector.matchLabels`

Pods SHALL be discovered by listing pods in the workload's namespace using the workload's `.spec.selector.matchLabels` as a label selector. (The global `--verbose` flag controls log verbosity only and does not affect status output.)
(Requirement: `--details` shows a pod detail block for a NotReady workload)

#### Scenario: Pods are listed by the workload's selector

- **WHEN** the pod detail block of a workload is built
- **THEN** pods SHALL be discovered by listing pods in the workload's namespace using the workload's `.spec.selector.matchLabels` as a label selector

### Requirement: The pod detail block has a fixed layout

The pod detail block SHALL be rendered below the resource table, separated by a blank line. Each block SHALL be headed by the workload's Kind/Name and a ready ratio, followed by indented pod lines.

The format is shown in the scenario `Pod detail block layout`.
(Requirement: `--details` shows a pod detail block for a NotReady workload)

#### Scenario: Pod detail block format

- **WHEN** a Deployment `web` has 3 pods with 1 ready
- **THEN** the pod detail block SHALL begin with `Deployment/web (1/3 ready):` followed by one indented line per pod

#### Scenario: Pod detail block layout

- **WHEN** a pod detail block is rendered
- **THEN** The format SHALL be:

  ```
  <Kind>/<Name> (<ready>/<total> ready):
      <pod-name>    <phase>    <detail>
  ```

### Requirement: The pod detail block is colour-coded by ready ratio, phase, detail text and restart count

The ready ratio SHALL be color-coded: green (all ready), yellow (partial), red (none ready). The phase SHALL be color-coded by severity: green (ready/Succeeded), yellow (transitional: Running not-ready, Pending, ContainerCreating, Unknown), red (error: CrashLoop, Failed, ImagePullBackOff, etc.). The detail text `(ready)` and `(not ready)` SHALL be dim. Restart counts SHALL be color-coded: yellow (1–9), red (10+).
(Requirement: The pod detail block has a fixed layout)

#### Scenario: The ready ratio is red when no pod is ready

- **WHEN** none of the pods of a workload is ready
- **THEN** the ready ratio SHALL be red
