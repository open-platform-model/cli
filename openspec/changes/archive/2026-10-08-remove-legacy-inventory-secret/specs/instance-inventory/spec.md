## REMOVED Requirements

### Requirement: Entry wire shape targets the CRD schema

**Reason**: The requirement also bound the reader of a legacy inventory Secret, which is removed. "Entry wire shape follows the CRD schema" replaces it without that reader.
**Migration**: None for the record. For an instance recorded only in a legacy Secret, apply it once with opm v1.0.0-beta.10, then upgrade.

## ADDED Requirements

### Requirement: Entry wire shape follows the CRD schema

Conversion between the library's inventory entry and the CR's `status.inventory.entries[]` SHALL be performed by explicit mapping functions in the CLI that produce and consume the CRD's field names (`group`, `kind`, `namespace`, `name`, `v`, `component`). The library entry carries no struct tags, so no Go struct tag SHALL decide the wire shape. The mapping SHALL round-trip losslessly.

#### Scenario: Version serializes as `v`

- **WHEN** an entry with Version `v1` is written to the CR
- **THEN** the entry object in `status.inventory.entries[]` SHALL carry the key `v` with value `v1`

#### Scenario: Round-trip preserves the entry set

- **WHEN** an entry list is written to a CR and read back
- **THEN** the resulting entries SHALL equal the originals
