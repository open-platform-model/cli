## MODIFIED Requirements

### Requirement: `opm module build` output format, split files and ordering

The `opm module build` subcommand SHALL accept `--output`/`-o` with exactly the values `yaml` (default) and `json`; any other value SHALL exit with code 1 and the message `invalid output format "<value>" (valid: yaml, json)`. With `--split`, it SHALL write one file per resource into `--out-dir` (default `./manifests`) named `<lowercase-kind>-<name>.<yaml|json>`; the second resource that resolves to the same base name SHALL receive the suffix `-2`, the third `-3`, and so on. Resources SHALL be emitted in a deterministic order, by apply weight (the library's `opm/k8s/object.Weight`), then namespace, then name, so identical input always yields identical output.

#### Scenario: Unsupported output format

- **WHEN** the user runs `opm module build -o toml`
- **THEN** the subcommand SHALL exit with code 1 and print `invalid output format "toml" (valid: yaml, json)`

#### Scenario: Split output names files by kind and name

- **WHEN** the user runs `opm module build --split --out-dir ./manifests` on a module rendering a Deployment `web` and a Service `web`
- **THEN** `./manifests` SHALL contain `deployment-web.yaml` and `service-web.yaml`

#### Scenario: Split output disambiguates colliding names

- **WHEN** two rendered resources share kind and name
- **THEN** the files SHALL be `<kind>-<name>.yaml` and `<kind>-<name>-2.yaml`

#### Scenario: Output order is deterministic

- **WHEN** the same module is built twice
- **THEN** both outputs SHALL list resources in identical order: ascending weight, then namespace, then name
