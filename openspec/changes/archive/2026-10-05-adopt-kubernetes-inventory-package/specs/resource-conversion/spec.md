## ADDED Requirements

### Requirement: One export feeds the apply objects and the shared render digest

The CLI SHALL convert a render's compiled objects with the library's `opm/k8s/object`: it SHALL wrap them with `object.Resources` and export them with exactly one `object.Export` call per render. The objects the CLI applies SHALL be the exported objects, in render order, and the render digest SHALL be the library's `opm/k8s/inventory.RenderDigest` of the same exported set, so no compiled object is exported from CUE twice and the CLI keeps no render digest algorithm of its own (0012:D6). An export failure SHALL exit with the general error code and name the failing resource.

#### Scenario: The digest and the apply objects come from one export

- **WHEN** `opm instance apply` or `opm module apply` renders a module
- **THEN** the render digest and the objects passed to apply both come from one `object.Export` over the render's compiled objects

#### Scenario: The render digest is the library's

- **WHEN** the CLI digests an exported object set
- **THEN** the digest equals `inventory.RenderDigest` of that set

## REMOVED Requirements

### Requirement: Rendered objects convert through the library's single export
**Reason**: It required the render digest to keep the CLI's earlier algorithm and value. The CLI now records the library's shared render digest, which changes the stored value once (0012:D6, 0012:D7:R4).
**Migration**: The single-export rule continues as the ADDED requirement "One export feeds the apply objects and the shared render digest". The stored `status.lastAppliedRenderDigest` changes once; see the change's migration note.
