## MODIFIED Requirements

### Requirement: mod vet command flags and syntax

The `opm mod vet` command SHALL accept an optional module path argument that defaults to the current directory, and SHALL expose `-f`/`--values` (repeatable), `-n`/`--namespace`, `--name` and `--platform`. All four affect the verdict: `-f` selects the values, `-n` and `--name` set the synthesized instance's namespace and name for the render, and `--platform` renders against that platform module instead of the module-deps platform.

`--instance-name` SHALL stay accepted as a deprecated alias of `--name` with the same effect. It SHALL NOT appear in the command's help, and its use SHALL print one line on standard error that names `--name`. Passing both `--name` and `--instance-name` SHALL be a usage error, and nothing SHALL be validated or rendered.

```text
opm mod vet [path] [flags]

Arguments:
  path    Path to module directory (default: .)

Flags:
  -f, --values strings        Additional values files (can be repeated)
  -n, --namespace string      Namespace of the synthesized instance
      --name string           Name of the synthesized instance (default: <module name>-debug)
      --platform string       Render against this platform module instead of the module's deps
  -h, --help                  Help for vet
```

#### Scenario: Default flags match expected behavior

- **WHEN** `opm mod vet` is run without any flags
- **THEN** path SHALL default to `"."`
- **AND** the render SHALL use the module-deps platform

#### Scenario: Name flag sets the synthesized instance name

- **WHEN** `opm module vet ./my-module --name web` is run
- **THEN** the synthesized instance SHALL be named `web`
- **AND** nothing about a deprecated flag SHALL be printed

#### Scenario: Deprecated alias keeps working

- **WHEN** `opm module vet ./my-module --instance-name web` is run
- **THEN** the synthesized instance SHALL be named `web`
- **AND** standard error SHALL carry one line saying that `--instance-name` is deprecated and naming `--name`

#### Scenario: Both spellings together are refused

- **WHEN** `opm module vet ./my-module --name a --instance-name b` is run
- **THEN** the command SHALL fail with a usage error before the module is loaded
