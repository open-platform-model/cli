## REMOVED Requirements

### Requirement: Inventory contract is exposed as a public package
**Reason**: The shared inventory contract moved to the library's Kubernetes tier, `opm/k8s/inventory`, so both frontends use one definition (0012:D1, 0012:D7). The cli deletes its copy.
**Migration**: Import `github.com/open-platform-model/library/opm/k8s/inventory`; the proposal's migration note maps each removed name. The cli's rule is the `pkg-types` requirement "Core types exported in pkg/".

### Requirement: Public inventory package contains only reusable ownership concerns
**Reason**: The package is deleted. The library package holds only the entry, the stale set and the two digests, and owns no wire shape.
**Migration**: None.

### Requirement: Persisted release record may keep metadata outside the public ownership contract
**Reason**: The record the cli persists is the ModuleInstance CR, specified by `instance-inventory`; there is no public ownership contract in the cli for it to stay outside of.
**Migration**: None.

### Requirement: Storage representation does not define the public contract
**Reason**: The package is deleted. The library `Entry` carries no struct tags, and the cli maps it to the CRD fields explicitly (`instance-inventory`, "Entry wire shape targets the CRD schema").
**Migration**: None.
