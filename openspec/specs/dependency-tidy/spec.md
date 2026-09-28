# Capability: dependency-tidy

## Purpose

`opm module tidy` and `opm catalog tidy` keep an OPM artifact's CUE module dependencies resolved, pinned and minimal without the separate `cue` binary. They produce the same `cue.mod` files `cue mod tidy` would, and their `--check` form works as a CI gate.

## Requirements

### Requirement: Tidy produces the files cue mod tidy would

`opm module tidy [path]` (alias `opm mod tidy`) and `opm catalog tidy [path]` SHALL rewrite the target's `cue.mod/module.cue` dependency block, and its `cue.mod/local-module.cue` when module replaces exist, to the exact content `cue mod tidy` of the CUE version the CLI embeds produces for the same tree and registry. That means every dependency an import in the module needs, at the version minimum version selection picks; a dependency imported for the first time at its latest published version; no dependency that no import needs; and `default: true` markers where CUE writes them. The command SHALL NOT require a `cue` executable on the machine and SHALL NOT start any child process.

The two commands SHALL behave identically; the kind only changes the wording of messages. Tidy is a CUE module operation: it SHALL NOT read or validate the artifact's OPM identity, schema conformance or declared version.

#### Scenario: Missing dependency is added

- **WHEN** a module imports `opmodel.dev/core@v2` and its `cue.mod/module.cue` has no `deps` entry for it
- **THEN** `opm module tidy` adds `"opmodel.dev/core@v2"` pinned to the newest version the registry serves, and exits 0

#### Scenario: Unused dependency is removed

- **WHEN** `cue.mod/module.cue` pins a module that no file in the module imports, directly or transitively
- **THEN** `opm module tidy` removes that entry and exits 0

#### Scenario: Parity with cue mod tidy

- **WHEN** `opm module tidy` has run on a module
- **THEN** `cue mod tidy --check` of the same CUE version, against the same registry, exits 0 on the result

#### Scenario: No cue binary needed

- **WHEN** `opm catalog tidy` runs on a machine where no `cue` executable is on `PATH`
- **THEN** the catalog is tidied exactly as it would be with one present

### Requirement: Tidy targets an explicit module root

The optional `path` argument SHALL name the module root directory and SHALL default to the current directory. The command SHALL tidy that directory only; it SHALL NOT search parent directories for a `cue.mod`. A path that does not exist, is not a directory, or has no `cue.mod/module.cue` SHALL be refused with exit 2 before any registry access. The refusal names the absolute path it checked; for `opm module tidy` on an existing directory without `cue.mod`, it also points at `opm module init` (no catalog equivalent exists, so `opm catalog tidy` names only the path).

#### Scenario: Default path

- **WHEN** `opm module tidy` runs with no argument inside a module root
- **THEN** that module is tidied

#### Scenario: Not a module root

- **WHEN** `opm module tidy ./sub` names a directory with no `cue.mod/module.cue`, even when a parent directory has one
- **THEN** the command exits 2 naming the absolute path it checked, and no file anywhere is written

### Requirement: Check mode fails without writing

With `--check` (boolean, default false) the command SHALL resolve dependencies the same way and SHALL write nothing. It exits 0 when the module is already tidy. It exits 2 when tidying would change `cue.mod/module.cue` or `cue.mod/local-module.cue`, and the message names the reason CUE gives (for example a missing dependency and the package that needs it) and the command that fixes it (`opm module tidy` or `opm catalog tidy`).

#### Scenario: Untidy module fails the check

- **WHEN** `opm module tidy --check` runs on a module whose imports need a dependency its `cue.mod/module.cue` does not pin
- **THEN** the command exits 2, names the missing dependency, suggests `opm module tidy`, and leaves every file byte-identical and with an unchanged mtime

#### Scenario: Tidy module passes the check

- **WHEN** `opm catalog tidy --check` runs on a catalog that `opm catalog tidy` has just tidied
- **THEN** the command exits 0 and writes nothing

### Requirement: Tidy resolves through the CLI's registry

Tidy SHALL resolve modules through the registry mapping the CLI resolves for every other command (`--registry` flag, then `OPM_REGISTRY`, then the config file's `registry`). When none of those is set, it SHALL fall back to the process's `CUE_REGISTRY` and then CUE's default, as the `cue` tool would. The registry resolved for one tidy SHALL NOT leak into anything the same process does afterwards.

#### Scenario: Flag routes resolution

- **WHEN** `opm --registry 'testing.opmodel.dev=localhost:5000+insecure,registry.cue.works' module tidy` runs and the process environment sets a different `CUE_REGISTRY`
- **THEN** modules under `testing.opmodel.dev` are fetched from `localhost:5000`

#### Scenario: Unreachable registry

- **WHEN** a dependency must be fetched and the registry that serves it cannot be reached
- **THEN** the command exits 1 with the resolver's error text, and no file is written

### Requirement: Tidy reports the outcome, not the steps

On success the command SHALL print one line saying whether it changed anything: which files it updated (`cue.mod/module.cue`, `cue.mod/local-module.cue`, or both), or that the module was already tidy and nothing was written. An already-tidy module SHALL NOT have any file rewritten or its mtime changed. Output from the embedded CUE tooling SHALL NOT reach the terminal except through the command's own error message.

Exit codes: 0 tidied, already tidy, or check passed; 1 dependency resolution or registry failure; 2 not tidy under `--check`, or the target is not a module root.

#### Scenario: Update reported

- **WHEN** `opm module tidy` adds a dependency
- **THEN** it prints a single success line naming `cue.mod/module.cue` as updated

#### Scenario: Already tidy is a no-op

- **WHEN** `opm module tidy` runs on an already tidy module
- **THEN** it prints that the module is already tidy, exits 0, and no file's bytes or mtime change
