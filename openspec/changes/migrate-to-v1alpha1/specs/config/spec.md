## MODIFIED Requirements

### Requirement: Consolidated verbose debug output for build pipeline

The build pipeline SHALL emit a single "release built" debug log per release build. Duplicate log lines from internal build stages SHALL be removed.

The following debug log lines SHALL be removed as redundant:

- `"release built successfully"` in `ReleaseBuilder.Build()` and `ReleaseBuilder.BuildFromValue()` — duplicate of pipeline-level "release built"
- `"loading provider"` in `ProviderLoader.LoadProvider()` — redundant with the `"loaded provider"` summary that follows

The `"extracted transformer"` debug log SHALL use the FQN as the `name` field value and SHALL NOT include a separate `fqn` field. Transformer FQNs SHALL use the v1alpha1 format (e.g., `kubernetes#opmodel.dev/providers/kubernetes/transformers@v1#DeploymentTransformer`).

#### Scenario: Single release-built log per build

- **WHEN** `--verbose` flag is specified
- **WHEN** a module is built via the render pipeline
- **THEN** exactly one "release built" debug log SHALL appear (from the pipeline level)
- **THEN** no "release built successfully" log SHALL appear from the release builder

#### Scenario: Transformer log uses FQN as name

- **WHEN** `--verbose` flag is specified
- **WHEN** transformers are extracted from a provider
- **THEN** the "extracted transformer" log SHALL show the v1alpha1 FQN format
- **THEN** the log SHALL NOT include a separate `fqn=` field

## ADDED Requirements

### Requirement: Config default template uses v1 module paths

The `DefaultModuleTemplate` SHALL use `opmodel.dev/config@v1` as the CUE module path. Provider imports SHALL reference `opmodel.dev/providers@v1`.

#### Scenario: Default config module path

- **WHEN** `opm config init` generates a default configuration
- **THEN** the CUE module declaration SHALL be `module: "opmodel.dev/config@v1"`

#### Scenario: Provider import path

- **WHEN** the default config template includes provider imports
- **THEN** the import path SHALL be `opmodel.dev/providers@v1`
