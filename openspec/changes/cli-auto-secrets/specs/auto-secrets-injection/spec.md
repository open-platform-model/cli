## ADDED Requirements

### Requirement: Builder reads _autoSecrets from evaluated ModuleRelease

The builder SHALL read the `_autoSecrets` hidden field from the fully-evaluated `#ModuleRelease` CUE value after component extraction. If the field is absent, bottom, or an empty struct, the builder SHALL skip auto-secrets injection and return the components map unchanged.

#### Scenario: Module with no secrets skips injection

- **WHEN** a module's `#config` contains no `#Secret` fields
- **THEN** `_autoSecrets` SHALL be an empty struct
- **AND** the builder SHALL return the components map unchanged
- **AND** no `opm-secrets` component SHALL exist in the output

#### Scenario: Module with secrets triggers injection

- **WHEN** a module's `#config` contains one or more `#Secret` fields with concrete values
- **THEN** `_autoSecrets` SHALL be a non-empty struct grouped by `$secretName`/`$dataKey`
- **AND** the builder SHALL proceed with component construction

#### Scenario: Old module without _autoSecrets field

- **WHEN** a module is built against an older catalog version that does not define `_autoSecrets` on `#ModuleRelease`
- **THEN** the builder SHALL detect the field as absent
- **AND** the builder SHALL skip injection without error

### Requirement: Builder constructs opm-secrets component via FillPath with #Secrets schema

The builder SHALL load `opmodel.dev/resources/config@v1` from the module's dependency cache and extract the `#Secrets` definition. The builder SHALL construct the `opm-secrets` component by starting from the `#Secrets` schema and using `FillPath` to set `metadata.name` and `spec.secrets` entries. Each entry in `_autoSecrets` SHALL be mapped to `spec.secrets."<secretName>".data`.

#### Scenario: Single secret group produces correct component

- **WHEN** `_autoSecrets` contains one entry `"db-creds"` with keys `"username"` and `"password"`
- **THEN** the constructed component SHALL have `metadata.name` equal to `"opm-secrets"`
- **AND** `spec.secrets."db-creds".data` SHALL contain both `"username"` and `"password"` entries
- **AND** `spec.secrets."db-creds".name` SHALL default to `"db-creds"`

#### Scenario: Multiple secret groups produce correct component

- **WHEN** `_autoSecrets` contains entries for `"db-creds"` and `"api-keys"`
- **THEN** the constructed component SHALL have both groups in `spec.secrets`
- **AND** each group SHALL have its `data` entries populated from `_autoSecrets`

#### Scenario: Component has correct resource FQN for transformer matching

- **WHEN** the `opm-secrets` component is constructed
- **THEN** the component's `Resources` map SHALL contain the key `"opmodel.dev/resources/config/secrets@v1"`
- **AND** the `#SecretTransformer` SHALL be able to match this component via its `requiredResources`

#### Scenario: Component has list-output annotation

- **WHEN** the `opm-secrets` component is constructed
- **THEN** `metadata.annotations` SHALL contain `"transformer.opmodel.dev/list-output": "true"`

### Requirement: Builder injects opm-secrets into the components map

The builder SHALL add the constructed `opm-secrets` component to the components map before returning the `*ModuleRelease`. The injection SHALL occur after user-defined component extraction and before release construction.

#### Scenario: Injected component appears in release components

- **WHEN** a module has `#Secret` fields and the build succeeds
- **THEN** the returned `ModuleRelease.Components` SHALL contain an `"opm-secrets"` entry
- **AND** the component SHALL be alongside user-defined components

### Requirement: Builder rejects user-defined component named opm-secrets

The builder SHALL check for a name collision before injecting the auto-secrets component. If the user-defined components map already contains a key `"opm-secrets"`, the builder SHALL return an error.

#### Scenario: Name collision produces clear error

- **WHEN** a module defines a component named `"opm-secrets"` in its `#components`
- **AND** `_autoSecrets` is non-empty
- **THEN** the builder SHALL return an error containing `"reserved for auto-secret injection"`
- **AND** no `*ModuleRelease` SHALL be returned

#### Scenario: No collision when user has different component names

- **WHEN** a module defines components with names other than `"opm-secrets"`
- **AND** `_autoSecrets` is non-empty
- **THEN** the builder SHALL inject `"opm-secrets"` without error

## MODIFIED Requirements

### Requirement: Builder accepts a loaded module and produces a concrete release

The builder SHALL accept a fully-loaded `*core.Module` and release options (name, namespace, optional values files) and return a concrete `*core.ModuleRelease` with all fields populated. The builder SHALL NOT depend on `Module.Values` being set. When the module's `#config` contains `#Secret` fields, the builder SHALL additionally inject an auto-generated `opm-secrets` component into the release's component map.

#### Scenario: Successful build with default values

- **WHEN** a loaded module is provided and no values files are given
- **AND** `values.cue` exists in the module directory
- **THEN** the builder SHALL return a concrete `*core.ModuleRelease` with metadata, components, and values all set

#### Scenario: Successful build with external values files

- **WHEN** one or more `--values` files are provided
- **THEN** the builder SHALL load and unify those files and use them as the sole values source

#### Scenario: Multiple values files are unified

- **WHEN** more than one values file is provided
- **THEN** the builder SHALL unify all files together before injection, with later files taking precedence over earlier ones

#### Scenario: Build with secrets produces release containing opm-secrets component

- **WHEN** a loaded module with `#Secret` fields in `#config` is provided with concrete secret values
- **THEN** the builder SHALL return a concrete `*core.ModuleRelease`
- **AND** the release's components SHALL include `"opm-secrets"` with the correct `#resources` FQN
