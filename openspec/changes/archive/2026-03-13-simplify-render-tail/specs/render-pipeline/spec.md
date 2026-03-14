## MODIFIED Requirements

### Requirement: Render workflow execution

The CLI MUST share the same execution tail for module-source and release-file rendering.

#### Scenario: Shared render tail execution

- **WHEN** the `Release` or `ReleaseFile` commands are executed
- **THEN** they MUST eventually call `renderPreparedModuleRelease` with a fully resolved `*provider.Provider` and namespace override to process the release and build the workflow `Result`
