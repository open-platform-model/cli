## MODIFIED Requirements

### Requirement: Render workflow execution

The CLI MUST share the same execution tail for release-file rendering via `renderPreparedModuleRelease`.

#### Scenario: Shared render tail execution

- **WHEN** the `FromReleaseFile` entrypoint is executed
- **THEN** it MUST call `renderPreparedModuleRelease` with a fully resolved `*provider.Provider` and namespace override to process the release and build the workflow `Result`

## REMOVED Requirements

### Requirement: FromModule entrypoint

**Reason**: The `FromModule` workflow function and its two internal preparation paths (Path A: pure module source, Path B: module dir with sibling `release.cue`) are removed. These paths duplicated orchestration logic and had inconsistent values-precedence rules. The `FromReleaseFile` entrypoint is the sole render workflow entrypoint.

**Migration**: Use `opm release build -r <release-file>` instead of `opm module build <path>`. Create a `release.cue` file that imports the module.
