# instance-building — Delta

## MODIFIED Requirements

### Requirement: Schema Line

The builder SHALL load `#ModuleInstance` from `opmodel.dev/core@v2` (resolved from the module's own dependency cache) and inject the module, instance name, namespace, and values via `FillPath`. UUID, labels, and derived metadata fields SHALL be computed by CUE evaluation, not by Go code.

#### Scenario: Builder resolves the v2 line

- **WHEN** a synthetic instance is built for a v2-authored module
- **THEN** it SHALL load `opmodel.dev/core@v2` (not `opmodel.dev/core@v1`)
- **AND** error messages SHALL reference `opmodel.dev/core@v2`
