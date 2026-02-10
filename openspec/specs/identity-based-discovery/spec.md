## Purpose

Defines how the OPM CLI discovers resources in a Kubernetes cluster using identity-based label selectors (release-id, module-id) in addition to the existing name+namespace selectors. Enables reliable resource discovery even when module names change.

## Requirements

### Requirement: Release-id based resource discovery

The resource discovery system SHALL support finding resources by their `module-release.opmodel.dev/uuid` label as a primary discovery strategy.

#### Scenario: Discover by release-id within namespace

- **WHEN** `DiscoverResources` is called with a release-id UUID and namespace
- **THEN** all resources in the namespace with the matching `module-release.opmodel.dev/uuid` label SHALL be returned
- **AND** cluster-scoped resources with the matching label SHALL also be returned

#### Scenario: No resources match release-id

- **WHEN** `DiscoverResources` is called with a release-id UUID that matches no resources
- **THEN** an empty result set SHALL be returned without error

---

### Requirement: Dual-strategy discovery with union

When both release-id and name+namespace identifiers are available, the discovery system SHALL execute both strategies and return the union of results, deduplicated by resource UID.

#### Scenario: Both strategies find the same resources

- **WHEN** discovery runs with both release-id and name+namespace
- **AND** both selectors match the same set of resources
- **THEN** each resource SHALL appear exactly once in the result

#### Scenario: Release-id finds resources that name+namespace misses

- **WHEN** a resource has a `release-id` label but its `name` label was modified
- **AND** discovery runs with both strategies
- **THEN** the resource SHALL be included in the result (found via release-id)

#### Scenario: Name+namespace finds resources that release-id misses

- **WHEN** a resource was applied before identity labeling was introduced (no release-id label)
- **AND** discovery runs with both strategies
- **THEN** the resource SHALL be included in the result (found via name+namespace fallback)

---

### Requirement: Release-id selector builder

The discovery package SHALL provide a function to build a label selector from a release-id UUID, combining it with the managed-by label.

#### Scenario: Build release-id selector

- **WHEN** `BuildReleaseIDSelector` is called with a UUID string
- **THEN** the returned selector SHALL match on `app.kubernetes.io/managed-by=open-platform-model` AND `module-release.opmodel.dev/uuid=<uuid>`

---

### Requirement: Delete with --release-id flag

The `mod delete` command SHALL accept an optional `--release-id` flag that allows deletion by release identity UUID directly, without requiring `--name`.

#### Scenario: Delete by release-id only

- **WHEN** `opm mod delete --release-id <uuid> -n <namespace>` is run
- **THEN** resources SHALL be discovered using the release-id selector
- **AND** discovered resources SHALL be deleted in reverse weight order

#### Scenario: Delete with both --name and --release-id

- **WHEN** `opm mod delete --name <name> -n <namespace> --release-id <uuid>` is run
- **THEN** dual-strategy discovery SHALL be used (union of both result sets)

#### Scenario: Delete with neither --name nor --release-id

- **WHEN** `opm mod delete -n <namespace>` is run without `--name` or `--release-id`
- **THEN** the command SHALL return a validation error stating that `--name` or `--release-id` is required

---

### Requirement: Status and diff use dual-strategy discovery

The `mod status` and `mod diff` commands SHALL use dual-strategy discovery when release-id is available in the rendered module metadata.

#### Scenario: Status with release-id available

- **WHEN** `mod status` is run and the module has a release identity
- **THEN** discovery SHALL use dual-strategy (release-id + name+namespace)

#### Scenario: Status without release-id

- **WHEN** `mod status` is run and the module has no release identity (older catalog)
- **THEN** discovery SHALL fall back to name+namespace only (current behavior)

---

### Requirement: Identity display in status output

The `mod status` command SHALL display the module-id and release-id when available.

#### Scenario: Status output with identity

- **WHEN** `mod status` is run and identity labels are present on resources
- **THEN** the output SHALL include a "Release ID" field showing the release identity UUID
- **AND** the output SHALL include a "Module ID" field showing the module identity UUID

#### Scenario: Status output without identity

- **WHEN** `mod status` is run and no identity labels are present on resources
- **THEN** the "Release ID" and "Module ID" fields SHALL be omitted from output
