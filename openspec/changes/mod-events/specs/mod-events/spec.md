## ADDED Requirements

### Requirement: Events command discovers release resources and their children

The `opm mod events` command SHALL discover OPM-managed resources using existing label selectors (`DiscoverResources`), then walk ownerReferences downward to find Kubernetes-owned children (ReplicaSets, Pods) of workload resources. The combined set of resource UIDs SHALL be used to filter events.

The ownerReference traversal SHALL cover these workload hierarchies:

| Parent Kind | Children | Grandchildren |
|-------------|----------|---------------|
| Deployment | ReplicaSets | Pods |
| StatefulSet | Pods | - |
| DaemonSet | Pods | - |
| Job | Pods | - |
| CronJob | Jobs | Pods |

Non-workload resources (ConfigMap, Secret, Service, PVC, Ingress, etc.) SHALL NOT be traversed for children — only their own events are included.

#### Scenario: Events include Pod-level events from Deployment children

- **WHEN** the user runs `opm mod events --release-name my-app -n production`
- **AND** the release contains a Deployment `my-app-web` which owns ReplicaSet `my-app-web-abc12` which owns Pod `my-app-web-abc12-x1`
- **THEN** the output SHALL include events for the Deployment, the ReplicaSet, and the Pod

#### Scenario: Events include Pod-level events from StatefulSet children

- **WHEN** the release contains a StatefulSet `my-app-db` which owns Pod `my-app-db-0`
- **THEN** the output SHALL include events for both the StatefulSet and the Pod

#### Scenario: Non-workload resources have no child traversal

- **WHEN** the release contains a ConfigMap `my-app-config`
- **THEN** the output SHALL include events for the ConfigMap itself
- **AND** no ownerReference traversal SHALL be performed for the ConfigMap

#### Scenario: No OPM resources found

- **WHEN** no resources match the release selector
- **AND** `--ignore-not-found` is not set
- **THEN** the command SHALL exit with a non-zero exit code and display an error message

#### Scenario: No OPM resources found with ignore flag

- **WHEN** no resources match the release selector
- **AND** `--ignore-not-found` is set
- **THEN** the command SHALL exit with code 0 and display an informational message

---

### Requirement: Events are fetched in bulk and filtered client-side

The command SHALL fetch all events in the target namespace with a single API call (`CoreV1().Events(namespace).List()`), then filter client-side by matching `event.InvolvedObject.UID` against the collected set of resource UIDs from discovery and child traversal.

#### Scenario: Single API call for event collection

- **WHEN** the release has 6 OPM-managed resources and 15 child resources (Pods, ReplicaSets)
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

- **WHEN** the user runs `opm mod events --release-name my-app -n prod` without `--since`
- **THEN** only events with `lastTimestamp` within the last 1 hour SHALL be displayed

#### Scenario: Custom time window

- **WHEN** the user runs `opm mod events --release-name my-app -n prod --since 30m`
- **THEN** only events with `lastTimestamp` within the last 30 minutes SHALL be displayed

#### Scenario: Day-based time window

- **WHEN** the user runs `opm mod events --release-name my-app -n prod --since 7d`
- **THEN** only events with `lastTimestamp` within the last 7 days SHALL be displayed

#### Scenario: Invalid since value

- **WHEN** the user provides an unparseable `--since` value (e.g., `--since foo`)
- **THEN** the command SHALL exit with an error indicating the value is invalid

---

### Requirement: Events support type filtering via --type

The command SHALL accept a `--type` flag that filters events by their Kubernetes event type. Valid values are `Normal` and `Warning`. When not specified, all event types SHALL be shown.

#### Scenario: Filter to warnings only

- **WHEN** the user runs `opm mod events --release-name my-app -n prod --type Warning`
- **THEN** only events with `type == "Warning"` SHALL be displayed

#### Scenario: Filter to normal events only

- **WHEN** the user runs `opm mod events --release-name my-app -n prod --type Normal`
- **THEN** only events with `type == "Normal"` SHALL be displayed

#### Scenario: Invalid type value

- **WHEN** the user provides an invalid `--type` value (e.g., `--type Error`)
- **THEN** the command SHALL exit with an error indicating valid values are `Normal` and `Warning`

---

### Requirement: Events default output is a table sorted chronologically

The default output format SHALL be a table with columns: LAST SEEN, TYPE, RESOURCE, REASON, MESSAGE. Events SHALL be sorted by `lastTimestamp` ascending (oldest first, newest at bottom).

#### Scenario: Default table output

- **WHEN** the user runs `opm mod events --release-name my-app -n prod` without `-o`
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

- `Warning` type: yellow (`ColorYellow`, ANSI 220)
- `Normal` type: dim gray (`colorDimGray`, ANSI 240)
- Resource names in RESOURCE column: cyan (`ColorCyan`, ANSI 14)

#### Scenario: Warning events displayed in yellow

- **WHEN** an event has `type == "Warning"`
- **THEN** the TYPE column value SHALL be rendered in yellow

#### Scenario: Normal events displayed in dim gray

- **WHEN** an event has `type == "Normal"`
- **THEN** the TYPE column value SHALL be rendered in dim gray

#### Scenario: Resource names displayed in cyan

- **WHEN** the RESOURCE column displays `Pod/my-app-abc12-x1`
- **THEN** the value SHALL be rendered in cyan matching the `styleNoun` convention

---

### Requirement: Events support structured output formats

The command SHALL support `--output`/`-o` with values `table` (default), `json`, and `yaml`.

JSON and YAML output SHALL emit a structured object containing release metadata and an array of event entries. Each entry SHALL include `lastSeen` (RFC3339), `type`, `kind`, `name`, `reason`, `message`, `count`, and `firstSeen` (RFC3339).

#### Scenario: JSON output

- **WHEN** the user runs `opm mod events --release-name my-app -n prod -o json`
- **THEN** the output SHALL be valid JSON with `releaseName`, `namespace`, and `events` array fields

#### Scenario: YAML output

- **WHEN** the user runs `opm mod events --release-name my-app -n prod -o yaml`
- **THEN** the output SHALL be valid YAML with the same structure as JSON output

#### Scenario: Invalid output format

- **WHEN** the user provides an invalid `-o` value (e.g., `-o xml`)
- **THEN** the command SHALL exit with an error indicating valid formats are `table`, `json`, `yaml`

---

### Requirement: Events support watch mode for real-time streaming

The command SHALL support a `--watch` flag that streams new events in real-time using the Kubernetes Watch API. Watch mode SHALL append new events to the terminal output (streaming style, not clear-and-redraw).

The UID set for filtering SHALL be computed once at startup. Watch mode SHALL exit cleanly on SIGINT/SIGTERM with exit code 0.

#### Scenario: Watch mode streams new events

- **WHEN** the user runs `opm mod events --release-name my-app -n prod --watch`
- **AND** a new event occurs for a resource in the release
- **THEN** the event SHALL be appended to the terminal output

#### Scenario: Watch mode filters by UID set

- **WHEN** watch mode is active
- **AND** an event occurs for a resource not in the release's UID set
- **THEN** the event SHALL NOT be displayed

#### Scenario: Watch mode respects --type filter

- **WHEN** the user runs `opm mod events --release-name my-app -n prod --watch --type Warning`
- **AND** a `Normal` event occurs
- **THEN** the event SHALL NOT be displayed

#### Scenario: Watch mode exits cleanly on interrupt

- **WHEN** the user presses Ctrl+C during watch mode
- **THEN** the command SHALL exit with code 0

---

### Requirement: Events command uses shared release selector flags

The command SHALL use `ReleaseSelectorFlags` for `--release-name`/`--release-id`/`-n` with the same mutual exclusivity validation as `mod status` and `mod delete`. Exactly one of `--release-name` or `--release-id` MUST be provided.

#### Scenario: Both selectors provided

- **WHEN** the user provides both `--release-name` and `--release-id`
- **THEN** the command SHALL exit with error: `"--release-name and --release-id are mutually exclusive"`

#### Scenario: Neither selector provided

- **WHEN** the user provides neither `--release-name` nor `--release-id`
- **THEN** the command SHALL exit with error: `"either --release-name or --release-id is required"`

---

### Requirement: Events command uses shared Kubernetes connection flags

The command SHALL use `K8sFlags` for `--kubeconfig`/`--context` with the same resolution precedence as other commands: explicit flag > `OPM_KUBECONFIG` env > `KUBECONFIG` env > `~/.kube/config`.

#### Scenario: Custom kubeconfig

- **WHEN** the user runs `opm mod events --kubeconfig /path/to/config --release-name my-app -n prod`
- **THEN** the command SHALL use the specified kubeconfig file

#### Scenario: Cluster unreachable

- **WHEN** the cluster specified by kubeconfig/context is not reachable
- **THEN** the command SHALL exit with a connectivity error code and a clear error message
