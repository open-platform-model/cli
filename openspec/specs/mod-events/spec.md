## Purpose

Defines the `opm instance events` command that aggregates Kubernetes events from all resources belonging to an OPM instance into a single chronological view, including events from Kubernetes-owned children (Pods, ReplicaSets).

## Requirements

### Requirement: Events command discovers instance resources and their children

The `opm instance events` command SHALL discover OPM-managed resources via `query.ResolveInventory` (inventory-based discovery from the instance's `ModuleInstance` record), then walk ownerReferences downward to find Kubernetes-owned children (ReplicaSets, Pods) of workload resources. The combined set of resource UIDs SHALL be used to filter events. <!-- Was: "discovers release resources" (0002 D9) -->

#### Scenario: Events include Pod-level events from Deployment children

- **WHEN** the user runs `opm instance events my-app -n production`
- **AND** the instance contains a Deployment `my-app-web` which owns ReplicaSet `my-app-web-abc12` which owns Pod `my-app-web-abc12-x1`
- **THEN** the output SHALL include events for the Deployment, the ReplicaSet, and the Pod

#### Scenario: No OPM resources found

- **WHEN** no `ModuleInstance` record exists for the instance in the namespace
- **THEN** the command SHALL exit with code 5 and report the instance as not found

### Requirement: Events are fetched in bulk and filtered client-side

The command SHALL fetch all events in the target namespace with a single API call (`CoreV1().Events(namespace).List()`), then filter client-side by matching `event.InvolvedObject.UID` against the collected set of resource UIDs from discovery and child traversal.

#### Scenario: Single API call for event collection

- **WHEN** the instance has 6 OPM-managed resources and 15 child resources (Pods, ReplicaSets)
- **THEN** the command SHALL make exactly one Events List API call to collect events
- **AND** filtering by UID SHALL be performed in memory

#### Scenario: No matching events

- **WHEN** the namespace has events but none match the collected UIDs
- **THEN** the command SHALL display an empty table (no events found) and exit with code 0

---

### Requirement: Events support time-windowed filtering via --since

The command SHALL accept a `--since` flag that filters events to a time window relative to the current time. The default value SHALL be `1h`.

The flag SHALL accept Go-style duration strings (`30m`, `1h`, `2h30m`) plus a `d` suffix for days (`1d`, `7d`).

#### Scenario: Default time window

- **WHEN** the user runs `opm instance events my-app -n prod` without `--since`
- **THEN** only events with `lastTimestamp` within the last 1 hour SHALL be displayed

#### Scenario: Custom time window

- **WHEN** the user runs `opm instance events my-app -n prod --since 30m`
- **THEN** only events with `lastTimestamp` within the last 30 minutes SHALL be displayed

#### Scenario: Day-based time window

- **WHEN** the user runs `opm instance events my-app -n prod --since 7d`
- **THEN** only events with `lastTimestamp` within the last 7 days SHALL be displayed

#### Scenario: Invalid since value

- **WHEN** the user provides an unparseable `--since` value (e.g., `--since foo`)
- **THEN** the command SHALL exit with an error indicating the value is invalid

---

### Requirement: Events support type filtering via --type

The command SHALL accept a `--type` flag that filters events by their Kubernetes event type. Valid values are `Normal` and `Warning`. When not specified, all event types SHALL be shown.

#### Scenario: Filter to warnings only

- **WHEN** the user runs `opm instance events my-app -n prod --type Warning`
- **THEN** only events with `type == "Warning"` SHALL be displayed

#### Scenario: Filter to normal events only

- **WHEN** the user runs `opm instance events my-app -n prod --type Normal`
- **THEN** only events with `type == "Normal"` SHALL be displayed

#### Scenario: Invalid type value

- **WHEN** the user provides an invalid `--type` value (e.g., `--type Error`)
- **THEN** the command SHALL exit with an error indicating valid values are `Normal` and `Warning`

---

### Requirement: Events default output is a table sorted chronologically

The default output format SHALL be a table with columns: LAST SEEN, TYPE, RESOURCE, REASON, MESSAGE. Events SHALL be sorted by `lastTimestamp` ascending (oldest first, newest at bottom).

#### Scenario: Default table output

- **WHEN** the user runs `opm instance events my-app -n prod` without `-o`
- **THEN** the output SHALL be a formatted table with LAST SEEN, TYPE, RESOURCE, REASON, and MESSAGE columns
- **AND** events SHALL be sorted oldest-first (ascending by lastTimestamp)

#### Scenario: LAST SEEN column uses relative duration

- **WHEN** an event has `lastTimestamp` 5 minutes ago
- **THEN** the LAST SEEN column SHALL display `5m`

#### Scenario: RESOURCE column shows Kind/Name

- **WHEN** an event has `involvedObject.kind == "Pod"` and `involvedObject.name == "my-app-abc12-x1"`
- **THEN** the RESOURCE column SHALL display `Pod/my-app-abc12-x1`

---

### Requirement: Events table output uses color coding

The table output SHALL apply color coding consistent with existing CLI style conventions:

- `Warning` type: yellow (`output.ColorYellow`, ANSI 220)
- `Normal` type: dim/faint (`output.Dim()`)
- Resource names in RESOURCE column: cyan (`output.StyleNoun()`, ANSI 14)

#### Scenario: Warning events displayed in yellow

- **WHEN** an event has `type == "Warning"`
- **THEN** the TYPE column value SHALL be rendered in yellow

#### Scenario: Normal events displayed in dim/faint

- **WHEN** an event has `type == "Normal"`
- **THEN** the TYPE column value SHALL be rendered in dim/faint style via `output.Dim()`

#### Scenario: Resource names displayed in cyan

- **WHEN** the RESOURCE column displays `Pod/my-app-abc12-x1`
- **THEN** the value SHALL be rendered in cyan matching the `styleNoun` convention

---

### Requirement: Events support structured output formats

The command SHALL support `--output`/`-o` with values `table` (default), `json`, and `yaml`.

JSON and YAML output SHALL emit a structured object containing instance metadata and an array of event entries. Each entry SHALL include `lastSeen` (RFC3339), `type`, `kind`, `name`, `reason`, `message`, `count`, and `firstSeen` (RFC3339).

#### Scenario: JSON output

- **WHEN** the user runs `opm instance events my-app -n prod -o json`
- **THEN** the output SHALL be valid JSON with `instanceName` (or `instanceID`), `namespace`, and `events` array fields

#### Scenario: YAML output

- **WHEN** the user runs `opm instance events my-app -n prod -o yaml`
- **THEN** the output SHALL be valid YAML with the same structure as JSON output

#### Scenario: Invalid output format

- **WHEN** the user provides an invalid `-o` value (e.g., `-o xml`)
- **THEN** the command SHALL exit with an error indicating valid formats are `table`, `json`, `yaml`

---

### Requirement: Events support watch mode for real-time streaming

The command SHALL support a `--watch` flag that streams new events in real-time using the Kubernetes Watch API. Watch mode SHALL append new events to the terminal output (streaming style, not clear-and-redraw).

The UID set for filtering SHALL be computed once at startup. Watch mode SHALL exit cleanly on SIGINT/SIGTERM with exit code 0.

#### Scenario: Watch mode streams new events

- **WHEN** the user runs `opm instance events my-app -n prod --watch`
- **AND** a new event occurs for a resource in the instance
- **THEN** the event SHALL be appended to the terminal output

#### Scenario: Watch mode filters by UID set

- **WHEN** watch mode is active
- **AND** an event occurs for a resource not in the instance's UID set
- **THEN** the event SHALL NOT be displayed

#### Scenario: Watch mode respects --type filter

- **WHEN** the user runs `opm instance events my-app -n prod --watch --type Warning`
- **AND** a `Normal` event occurs
- **THEN** the event SHALL NOT be displayed

#### Scenario: Watch mode exits cleanly on interrupt

- **WHEN** the user presses Ctrl+C during watch mode
- **THEN** the command SHALL exit with code 0

---

### Requirement: Events command takes the instance as a positional argument

The command SHALL take the instance as its required positional `<file|name|uuid>` argument, resolved through `cmdutil.ResolveInstanceTarget` exactly as `instance status`, `instance tree` and `instance delete` do: an argument matching the UUID pattern is an instance UUID, a path to an instance file yields the instance name and namespace from that file, and anything else is an instance name. `-n`/`--namespace` overrides the namespace.

#### Scenario: UUID argument

- **WHEN** the user runs `opm instance events 550e8400-e29b-41d4-a716-446655440000 -n prod`
- **THEN** the command SHALL resolve the instance by listing `ModuleInstance` CRs in `prod` and matching `status.instanceUUID`

#### Scenario: Name argument

- **WHEN** the user runs `opm instance events my-app -n prod`
- **THEN** the command SHALL resolve the instance by a direct `ModuleInstance` GET named `my-app` in `prod`

#### Scenario: Missing argument

- **WHEN** the user runs `opm instance events` with no argument
- **THEN** the command SHALL exit with a usage error

### Requirement: Events command uses shared Kubernetes connection flags

The command SHALL use `K8sFlags` for `--kubeconfig`/`--context` with the same resolution precedence as other commands: explicit flag > `OPM_KUBECONFIG` env > `KUBECONFIG` env > `~/.kube/config`.

#### Scenario: Custom kubeconfig

- **WHEN** the user runs `opm instance events my-app -n prod --kubeconfig /path/to/config`
- **THEN** the command SHALL use the specified kubeconfig file

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with a connectivity error code and a clear error message
