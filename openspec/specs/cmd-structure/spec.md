## Purpose

Defines the structural conventions for CLI command packages in the OPM codebase. This covers how commands are organized into sub-packages, how configuration is injected, and how the cobra command tree maps to the package layout under `internal/cmd/`.

## Requirements

### Requirement: Commands receive configuration via explicit injection

Global CLI configuration (OPMConfig, resolved registry, verbose flag) SHALL be passed
explicitly to command constructors via a `GlobalConfig` struct rather than read from
package-level variables or accessor functions.

#### Scenario: Sub-command accesses OPMConfig

- **WHEN** a sub-command in `internal/cmd/module/`, `internal/cmd/instance/` or `internal/cmd/config/` needs the loaded OPMConfig
- **THEN** it reads it from the `*GlobalConfig` parameter passed to its constructor, not from a package-level accessor

#### Scenario: No package-level mutable state in sub-packages

- **WHEN** any file in `internal/cmd/module/`, `internal/cmd/instance/` or `internal/cmd/config/` is inspected
- **THEN** it contains no package-level `var` declarations for flags or configuration state

### Requirement: Command packages are organised by command group

The `internal/cmd/` package SHALL be split into sub-packages that mirror the cobra command tree.

#### Scenario: module commands are in their own package

- **WHEN** the `internal/cmd/module/` directory is inspected
- **THEN** it SHALL contain module authoring commands: `init`, `vet`, `build`
- **AND** the `build` subcommand SHALL synthesize a `#ModuleInstance` from a module-package directory and render it through the shared instance-render pipeline
- **AND** the `build` subcommand SHALL reject a single-file argument with an error directing the user to `opm instance build <file>` for instance files

#### Scenario: config commands are in their own package

- **WHEN** the `internal/cmd/config/` directory is inspected
- **THEN** it contains all `config` sub-command implementations (`init`, `vet`)

#### Scenario: instance commands are in their own package

- **WHEN** the `internal/cmd/instance/` directory is inspected
- **THEN** it SHALL contain all `instance` sub-command implementations: `vet`, `build`, `apply`, `diff`, `status`, `tree`, `events`, `delete`, `list`
- **AND** the `instance build` subcommand SHALL accept either an instance `.cue` file or a module-package directory as its positional argument
- **AND** when the argument is a directory the subcommand SHALL delegate to the same module-synthesis path used by `opm module build`

#### Scenario: operator commands are in their own package

- **WHEN** the `internal/cmd/operator/` directory is inspected
- **THEN** it SHALL contain the `operator` sub-command implementations: `install`, `uninstall`
- **AND** the commands SHALL be thin cobra wiring that delegates all behavior to `internal/operator/`

#### Scenario: platform commands are in their own package

- **WHEN** the `internal/cmd/platform/` directory is inspected
- **THEN** it SHALL contain the `platform` sub-command implementations: `check`
- **AND** the commands SHALL be thin cobra wiring that delegates the report to `internal/platform/`

### Requirement: Instance command group registered at root level

The root command SHALL register `opm instance` (alias: `inst`) as a top-level command group via `cmdinstance.NewInstanceCmd(&cfg)`. This SHALL follow the same dependency injection pattern as `mod` and `config` groups. The former `cmdrelease.NewReleaseCmd` registration is removed (no back-compat alias — enhancement 0002 D8).

#### Scenario: Root command registers instance group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdinstance.NewInstanceCmd(&cfg))`
- **AND** it SHALL NOT contain `cmdrelease.NewReleaseCmd`

### Requirement: Operator command group registered at root level

The root command SHALL register `opm operator` as a top-level command group via `cmdoperator.NewOperatorCmd(&cfg)`, following the same `GlobalConfig` dependency-injection pattern as the `module`, `instance`, and `config` groups. The group is noun-first (enhancement 0006 D32): there are no `opm install` or `opm uninstall` verb groups at root level.

#### Scenario: Root command registers operator group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdoperator.NewOperatorCmd(&cfg))`
- **AND** the root command SHALL NOT register `install` or `uninstall` as top-level commands

### Requirement: Platform command group registered at root level

The root command SHALL register `opm platform` as a top-level command group via `cmdplatform.NewPlatformCmd(&cfg)`, following the same `GlobalConfig` dependency-injection pattern as the `module`, `instance`, `config` and `operator` groups. The group is noun-first: there is no `opm check` verb group at root level.

#### Scenario: Root command registers platform group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdplatform.NewPlatformCmd(&cfg))`

#### Scenario: The group lists its subcommands

- **WHEN** `opm platform --help` is run
- **THEN** it SHALL list `check`

### Requirement: Cluster-query commands live only under instance

The cluster-query commands (`status`, `tree`, `events`, `delete`, `list`) SHALL be implemented in `internal/cmd/instance/` and registered only under the `instance` command group. The `module` group (`internal/cmd/module/`) SHALL NOT carry them or aliases for them: it registers `init`, `template`, `vet`, `build`, `apply`, `publish` and `version` only.

#### Scenario: opm mod status is not a command

- **WHEN** `opm mod status jellyfin` is run
- **THEN** the CLI SHALL fail with an unknown-command error

#### Scenario: opm instance status is the command

- **WHEN** `opm instance status jellyfin` is run
- **THEN** the CLI SHALL execute the status command for the instance `jellyfin`

### Requirement: `opm module build` output format, split files and ordering

The `opm module build` subcommand SHALL accept `--output`/`-o` with exactly the values `yaml` (default) and `json`; any other value SHALL exit with code 1 and the message `invalid output format "<value>" (valid: yaml, json)`. With `--split`, it SHALL write one file per resource into `--out-dir` (default `./manifests`) named `<lowercase-kind>-<name>.<yaml|json>`; the second resource that resolves to the same base name SHALL receive the suffix `-2`, the third `-3`, and so on. Resources SHALL be emitted in a deterministic order, by apply weight (`pkg/resourceorder.GetWeight`), then namespace, then name, so identical input always yields identical output.

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

### Requirement: `opm instance build` accepts only instance packages

The `opm instance build` subcommand SHALL take one positional argument naming an instance: a `.cue` file (the instance is the package in the file's directory) or a directory (the instance is the package in that directory). The directory may be a standalone CUE module or a package inside another CUE module, such as an instance directory within a module tree. The subcommand SHALL acquire that package through the library kernel as an instance (`AcquireInstanceFromDir`, with any `-f`/`--values` files layered as values sources) and render it; it SHALL NOT synthesize an instance from a module. What the package is decides the outcome, not whether the argument is a file or a directory: when the package is a module (`kind: "Module"`), the subcommand SHALL exit 2 naming the path and pointing to `opm module build <path>`.

#### Scenario: Argument is an instance file

- **WHEN** the user runs `opm instance build ./web_app_instance.cue` and the file's package embeds `#ModuleInstance`
- **THEN** the subcommand SHALL acquire the file's package through the library kernel and render it

#### Scenario: Argument is an instance package directory

- **WHEN** the user runs `opm instance build ./hello` and `./hello` holds a package embedding `#ModuleInstance`, with or without its own `cue.mod`
- **THEN** the subcommand SHALL acquire that package through the library kernel and render it

#### Scenario: Argument is a module directory

- **WHEN** the user runs `opm instance build ./my-module` and the directory's package is a module
- **THEN** the subcommand SHALL exit 2 without rendering, and the message SHALL point to `opm module build ./my-module`

#### Scenario: Argument does not exist

- **WHEN** the positional argument cannot be `os.Stat`'ed
- **THEN** the subcommand SHALL return a clear error naming the missing path

### Requirement: `opm module build` (alias `opm mod build`) accepts a module directory or a published module

The `module` command group SHALL register a `build` subcommand with an optional positional argument defaulting to `"."`, naming either a local module package directory or a published module path (see the `published-module-resolution` capability). The argument SHALL be classified before any registry access:

1. `.`, a path starting with `./` or `../`, or an absolute path is a local directory.
2. Otherwise, an argument whose first element contains a dot is a published module path.
3. Anything else is a local directory.

An argument classified as a published module path that also names an existing local directory SHALL be refused with exit 2 as ambiguous, suggesting `./<argument>` for the directory. A new `--version` flag (string, default empty) selects the published version; passing it with a local directory SHALL exit 2 stating that `--version` applies only to a published module. A published module SHALL be acquired from the registry and synthesized and rendered exactly as a local module is, with values from `-f`/`--values` or the module's `debugValues`; nothing SHALL be written to disk except CUE's own module cache.

#### Scenario: Default to current directory

- **WHEN** the user runs `opm module build` with no positional argument from inside a module package directory
- **THEN** the subcommand SHALL synthesize and render that directory

#### Scenario: Explicit module directory

- **WHEN** the user runs `opm module build ./my-module`
- **THEN** the subcommand SHALL synthesize and render that directory

#### Scenario: File argument rejected

- **WHEN** the user runs `opm module build ./my-module/module.cue`
- **THEN** the subcommand SHALL return an error stating that module build expects a directory and pointing the user to `opm instance build <file>` for instance files

#### Scenario: Published module without a version

- **WHEN** the user runs `opm module build opmodel.dev/modules/web_app`
- **THEN** the subcommand SHALL resolve the newest release of the highest core-compatible major, report it on standard error, and write the rendered manifests to standard output

#### Scenario: Published module with a pinned version

- **WHEN** the user runs `opm module build opmodel.dev/modules/web_app --version 1.0.3 -f values.cue`
- **THEN** the subcommand SHALL render version `1.0.3` with the values from `values.cue`

#### Scenario: Version flag with a local directory

- **WHEN** the user runs `opm module build ./my-module --version v1`
- **THEN** the subcommand SHALL exit 2 stating that `--version` applies only to a published module

#### Scenario: Dotted argument that is also a local directory

- **WHEN** the user runs `opm module build web.app` and `./web.app` is an existing directory
- **THEN** the subcommand SHALL exit 2 as ambiguous and suggest `opm module build ./web.app`

### Requirement: `opm module apply` (alias `opm mod apply`) accepts a module directory or a published module

The `module` command group SHALL register an `apply` subcommand with an optional positional argument defaulting to `"."`, classified exactly as `opm module build` classifies its argument, and the same `--version` flag. The subcommand SHALL synthesize a `#ModuleInstance` from the module (reusing the `module-synthetic-instance` capability), render the result through the same pipeline as `opm instance apply`, and apply the produced resources to a Kubernetes cluster with full inventory, prune, dry-run, and ownership semantics. Version resolution SHALL complete before any cluster contact. An apply from a published module SHALL record the module's declared path and version in the instance's spec as any apply does, and SHALL NOT carry the local-render provenance annotation, because its module bytes come from the registry. When the values come from the module's `debugValues` (no `-f`/`--values`), from a local directory or a published module alike, the subcommand SHALL print one warning on standard error before applying: the values are the module's test values, they should be reviewed before a real deployment, and `opm instance init` writes an editable instance package. `opm module build` SHALL NOT print this warning, because it changes nothing.

#### Scenario: Default to current directory

- **WHEN** the user runs `opm module apply` with no positional argument from inside a module package directory
- **THEN** the subcommand SHALL synthesize and apply that directory

#### Scenario: Apply warns when it deploys debugValues

- **WHEN** the user runs `opm module apply opmodel.dev/modules/web_app -n demo` with no `-f`
- **THEN** standard error SHALL carry one warning that the module's `debugValues` are being applied, to review them before a real deployment, naming `opm instance init`
- **AND** the warning SHALL appear before any resource is applied

#### Scenario: No warning with values files

- **WHEN** the user runs `opm module apply ./my-module -f values.cue`
- **THEN** no `debugValues` warning SHALL be printed

#### Scenario: Explicit module directory

- **WHEN** the user runs `opm module apply ./my-module`
- **THEN** the subcommand SHALL synthesize and apply that directory

#### Scenario: File argument rejected

- **WHEN** the user runs `opm module apply ./my-module/module.cue`
- **THEN** the subcommand SHALL return an error stating that `module apply` expects a directory
- **AND** SHALL point the user to `opm instance apply <file>` for instance files

#### Scenario: Alias `mod apply` resolves to `module apply`

- **WHEN** the user runs `opm mod apply ./my-module`
- **THEN** the CLI SHALL execute the same subcommand as `opm module apply ./my-module`

#### Scenario: Apply a published module

- **WHEN** the user runs `opm module apply opmodel.dev/modules/web_app --version v1 --name hello -n demo`
- **THEN** the subcommand SHALL apply the newest v1 release as instance `hello` in namespace `demo`
- **AND** the instance's spec SHALL name `opmodel.dev/modules/web_app@v1` and the resolved version
- **AND** the instance SHALL NOT carry `module-instance.opmodel.dev/source: local`

#### Scenario: Resolution failure exits before cluster contact

- **WHEN** the user runs `opm module apply opmodel.dev/modules/web_app --version 9.9.9` and that tag is not published
- **THEN** the subcommand SHALL exit 2 without issuing any apiserver request

### Requirement: `--name` flag for synthetic-instance builds

The `opm module build` and `opm module apply` subcommands SHALL accept a `--name <string>` flag that overrides the synthetic `metadata.name`, for a local directory and a published module alike. Defaults are described in the `module-synthetic-instance` capability spec. `opm instance build` SHALL NOT accept `--name`: an instance names itself.

#### Scenario: Flag overrides the default name

- **WHEN** the user passes `--name foo` to `opm module build`
- **THEN** the synthetic `metadata.name` SHALL be `foo`

#### Scenario: Flag rejected by instance build

- **WHEN** the user runs `opm instance build ./real-instance.cue --name foo`
- **THEN** the CLI SHALL exit 1 with an unknown-flag usage error and render nothing

#### Scenario: Flag participates in synthetic instance identity for `module apply`

- **WHEN** the user runs `opm module apply ./foo --name custom`
- **THEN** the synthetic `metadata.name` SHALL be `custom`
- **AND** the resolved instance UUID SHALL be derived from `custom` (not the default `<module>-debug`)
- **AND** running the same command again with a different `--name` value SHALL produce a distinct instance identity and a separate inventory record
