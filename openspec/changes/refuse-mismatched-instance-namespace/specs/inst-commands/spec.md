## ADDED Requirements

### Requirement: instance render commands refuse a namespace override that disagrees with the instance file

The instance file owns its namespace: `metadata.namespace` is part of the instance's identity. When `opm instance apply`, `build`, `diff` or `vet` resolves its namespace from the `-n`/`--namespace` flag or the `OPM_NAMESPACE` environment variable, and that value differs from the acquired instance's `metadata.namespace`, the command SHALL refuse with exit code 2 before resolving the platform, rendering, diffing or applying anything. The refusal SHALL name the source of the override (`--namespace` or `OPM_NAMESPACE`), the override value, the instance's `metadata.namespace` value and the instance argument, and SHALL tell the user to edit `metadata.namespace` in the instance file to deploy the instance elsewhere. An override equal to `metadata.namespace` SHALL be accepted. A namespace that comes from `kubernetes.namespace` in `~/.opm/config.cue` or from the built-in default is not an override and SHALL NOT be compared; the instance renders in its file's namespace. The `-n` help text of these four commands SHALL state that the value must equal the instance file's `metadata.namespace`.

The module commands (`opm module build`, `apply`, `vet`) are unaffected: they synthesize the instance in the override namespace, so the override and `metadata.namespace` agree.

#### Scenario: Flag equal to the file's namespace

- **WHEN** `opm instance apply ./jellyfin -n media` is run and the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL render and apply the instance in `media` as without the flag

#### Scenario: Flag differs from the file's namespace

- **WHEN** `opm instance apply ./jellyfin -n staging` is run and the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL exit with code 2 and apply nothing
- **AND** the error SHALL name `--namespace`, `staging`, `media` and `./jellyfin`, and tell the user to edit `metadata.namespace` in the instance file

#### Scenario: Environment variable differs from the file's namespace

- **WHEN** `OPM_NAMESPACE=staging opm instance build ./jellyfin` is run without `-n` and the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL exit with code 2 and print no manifests
- **AND** the error SHALL name `OPM_NAMESPACE`, `staging` and `media`

#### Scenario: No override

- **WHEN** `opm instance diff ./jellyfin` is run with neither `-n` nor `OPM_NAMESPACE`, and `~/.opm/config.cue` sets `kubernetes.namespace: "staging"`
- **AND** the instance declares `metadata.namespace: "media"`
- **THEN** the command SHALL NOT refuse, and SHALL diff the instance in `media`

#### Scenario: Module path still honours the override

- **WHEN** `opm module apply ./my-module -n staging` is run
- **THEN** the synthesized instance's `metadata.namespace` SHALL be `staging` and the command SHALL NOT refuse

#### Scenario: Help text states the constraint

- **WHEN** `opm instance apply --help`, `opm instance build --help`, `opm instance diff --help` or `opm instance vet --help` is run
- **THEN** the `-n, --namespace` line SHALL say the value must equal the instance file's `metadata.namespace`
