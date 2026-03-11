## MODIFIED Requirements

### Requirement: Status discovers resources via ownership inventory

The `opm mod status` command SHALL use the persisted ownership inventory for the release to discover tracked resources. It SHALL perform one targeted GET per tracked entry and MUST NOT require module source or re-rendering to enumerate owned resources.

#### Scenario: Status shows deployed resources via ownership inventory

- **WHEN** the user runs `opm mod status` for a release with a persisted ownership inventory
- **THEN** the command SHALL fetch each tracked resource via targeted GET
- **AND** only resources explicitly tracked in the ownership inventory SHALL appear in the output

### Requirement: Status header does not depend on inventory change history

The status header SHALL NOT require inventory change-history metadata such as source version, raw values, or per-change timestamps. Release metadata shown in the header SHALL come from release-specific state or be omitted until such state exists.

#### Scenario: Status remains functional with ownership-only inventory

- **WHEN** a release has ownership-only inventory and no history-bearing inventory fields
- **THEN** `opm mod status` SHALL still be able to enumerate resources and show their health
