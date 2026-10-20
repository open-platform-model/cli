# Capability: status-verbose

## Purpose

`opm instance status --details` explains an unhealthy workload instead of merely flagging it. For each Deployment, StatefulSet, or DaemonSet that evaluates to NotReady, it lists the workload's pods with their phase or container waiting reason, the last termination reason, and the restart count, as an indented block beneath the table or as a `verbose` object on the resource in JSON and YAML output.

## Requirements

### Requirement: Verbose pod details show container waiting reasons

When a pod has containers in a waiting state, the verbose output SHALL display the container's waiting reason instead of the pod phase. Common waiting reasons include CrashLoopBackOff, ImagePullBackOff, ErrImagePull, CreateContainerConfigError, and ContainerCreating.

The waiting reason SHALL be extracted from `status.containerStatuses[].state.waiting.reason`.

#### Scenario: Pod with CrashLoopBackOff container

- **WHEN** a pod has a container with `state.waiting.reason: CrashLoopBackOff`
- **THEN** the pod's phase SHALL be displayed as `CrashLoop` in the verbose output

#### Scenario: Pod with ImagePullBackOff container

- **WHEN** a pod has a container with `state.waiting.reason: ImagePullBackOff`
- **THEN** the pod's phase SHALL be displayed as `ImagePullBackOff` in the verbose output

### Requirement: Verbose pod details show termination reasons

When a pod has containers that were terminated, the verbose output SHALL include the termination reason in the detail text. Common termination reasons include OOMKilled and Error.

The termination reason SHALL be extracted from `status.containerStatuses[].state.terminated.reason` or `status.containerStatuses[].lastState.terminated.reason`.

#### Scenario: Pod terminated by OOMKilled

- **WHEN** a pod has a container with `lastState.terminated.reason: OOMKilled`
- **THEN** the detail text SHALL include "OOMKilled"

Note: Memory limit enrichment (e.g. "OOMKilled (512Mi limit)") is deferred — the termination reason is surfaced as-is from `lastState.terminated.reason`.

#### Scenario: Pod terminated with generic error

- **WHEN** a pod has a container with `lastState.terminated.reason: Error`
- **THEN** the detail text SHALL include "Error"

### Requirement: Verbose pod details show restart counts

When a pod has containers with restarts, the verbose output SHALL include the total restart count across all containers in the detail text.

The restart count SHALL be extracted from `status.containerStatuses[].restartCount`.

#### Scenario: Pod with multiple restarts

- **WHEN** a pod has containers with a combined `restartCount` of 5
- **THEN** the detail text SHALL include "5 restarts"

#### Scenario: Pod with zero restarts

- **WHEN** a pod has containers with `restartCount` of 0
- **THEN** the detail text SHALL NOT include restart information

### Requirement: Verbose mode works with structured output

When `--details` is combined with `-o json` or `-o yaml`, the pod details SHALL be included in the structured output under a `verbose` field on each resource, containing a `pods` array with name, phase, ready, reason, and restarts fields.

#### Scenario: Verbose JSON output includes pods

- **WHEN** the user runs `opm instance status my-app -n prod --details -o json`
- **AND** a Deployment is NotReady with 2 pods
- **THEN** the JSON output for that resource SHALL include a `verbose` object with a `pods` array containing 2 entries with `name`, `phase`, `ready`, `reason`, and `restarts` fields

#### Scenario: Verbose JSON omits pods for healthy resources

- **WHEN** a resource is Ready
- **THEN** the JSON output for that resource SHALL NOT include a `verbose` field

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
