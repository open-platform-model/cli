## MODIFIED Requirements

### Requirement: Inventory entry identity

An `InventoryEntry` SHALL represent a single currently owned Kubernetes resource. Two entries SHALL be considered identity-equal when their Group, Kind, Namespace, Name, and Component fields all match. The Version field SHALL be excluded from identity comparison to prevent false orphans during Kubernetes API version migrations.

#### Scenario: Same resource with different API version

- **WHEN** comparing two entries with identical Group, Kind, Namespace, Name, Component but different Version
- **THEN** the entries SHALL be identity-equal

#### Scenario: Same resource with different component

- **WHEN** comparing two entries with identical Group, Kind, Namespace, Name, Version but different Component
- **THEN** the entries SHALL NOT be identity-equal

### Requirement: Entry construction from rendered resource

The system SHALL construct an `InventoryEntry` from a rendered Kubernetes resource by extracting Group and Kind from the resource's GVK, Version from the GVK's Version field, Namespace and Name from the resource's metadata, and Component from the OPM component label when present.

#### Scenario: Build entry from a namespaced Deployment

- **WHEN** constructing an entry from a resource with GVK `apps/v1/Deployment`, name `my-app`, namespace `production`, component `app`
- **THEN** the entry SHALL have Group=`apps`, Kind=`Deployment`, Namespace=`production`, Name=`my-app`, Version=`v1`, Component=`app`

### Requirement: Inventory represents current ownership only

The public inventory contract SHALL represent the current set of resources owned by a release. It SHALL contain the current `entries` list and MAY include ownership summary fields such as `revision`, `digest`, and `count`.

The public inventory contract MUST NOT require or embed:

- raw values
- source path or source version metadata
- per-change timestamps
- history index
- change map
- remediation counters

#### Scenario: Ownership-only inventory contains current resource refs

- **WHEN** a release currently owns a Deployment, Service, and Ingress
- **THEN** the inventory SHALL contain exactly three entries representing those resources
- **AND** no history entries SHALL be required to determine current ownership

#### Scenario: Inventory exposes summary metadata without history

- **WHEN** an inventory includes `revision`, `digest`, and `count`
- **THEN** those fields SHALL describe the current inventory set only
- **AND** they SHALL NOT imply a retained change history

### Requirement: Kubernetes identity equality

A separate K8s identity comparison SHALL compare entries by Group, Kind, Namespace, and Name only (excluding both Version and Component). This SHALL be used by the component-rename safety check to detect when the same Kubernetes resource appears under a different component name.

#### Scenario: Same K8s resource under different components

- **WHEN** comparing two entries with identical Group, Kind, Namespace, Name but different Component
- **THEN** the entries SHALL be K8s-identity-equal

### Requirement: Deterministic inventory digest

When the system computes an inventory digest, it SHALL do so deterministically from the current owned resource set regardless of input order.

#### Scenario: Same entries in different order produce same digest

- **WHEN** computing the digest for the same ownership set in two different slice orders
- **THEN** the digest SHALL be identical

## REMOVED Requirements

The public inventory contract SHALL NOT require the legacy change-history model (`index`, per-change entries, source metadata, raw values, timestamps) as part of the reusable inventory API.
