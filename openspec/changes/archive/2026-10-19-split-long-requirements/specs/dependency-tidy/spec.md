## REMOVED Requirements

### Requirement: Tidy produces the files cue mod tidy would
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm module tidy` and `opm catalog tidy` rewrite the dependency block of `cue.mod/module.cue` to the exact content `cue mod tidy` produces", "The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other", "`opm module tidy` and `opm catalog tidy` require no `cue` executable and start no child process", "`opm module tidy` and `opm catalog tidy` behave identically and only the wording of messages follows the kind", "Tidy does not read or validate the artifact's OPM identity, schema conformance or declared version".

### Requirement: Tidy targets an explicit module root
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The optional `path` argument of tidy names the module root and tidy searches no parent directory", "Tidy refuses a path that is not a module root with exit 2 before any registry access".

### Requirement: Tidy reports the outcome, not the steps
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module", "Output from the embedded CUE tooling reaches the terminal only through the command's own error message", "Tidy exit codes are 0 tidied, already tidy or check passed, 1 resolution or registry failure and 2 not tidy or not a module root".

## ADDED Requirements

### Requirement: `opm module tidy` and `opm catalog tidy` rewrite the dependency block of `cue.mod/module.cue` to the exact content `cue mod tidy` produces

`opm module tidy [path]` (alias `opm mod tidy`) and `opm catalog tidy [path]` SHALL rewrite the target's `cue.mod/module.cue` dependency block, and its `cue.mod/local-module.cue` when module replaces exist, to the exact content `cue mod tidy` of the CUE version the CLI embeds produces for the same tree and registry.

#### Scenario: Parity with cue mod tidy

- **WHEN** `opm module tidy` has run on a module
- **THEN** `cue mod tidy --check` of the same CUE version, against the same registry, exits 0 on the result

### Requirement: The exact content `cue mod tidy` produces holds every needed dependency at the version minimum version selection picks and no other

The exact content `cue mod tidy` of the CUE version the CLI embeds produces for the same tree and registry SHALL mean every dependency an import in the module needs, at the version minimum version selection picks; a dependency imported for the first time at its latest published version; no dependency that no import needs; and `default: true` markers where CUE writes them.

#### Scenario: Missing dependency is added

- **WHEN** a module imports `opmodel.dev/core@v2` and its `cue.mod/module.cue` has no `deps` entry for it
- **THEN** `opm module tidy` adds `"opmodel.dev/core@v2"` pinned to the newest version the registry serves, and exits 0

#### Scenario: Unused dependency is removed

- **WHEN** `cue.mod/module.cue` pins a module that no file in the module imports, directly or transitively
- **THEN** `opm module tidy` removes that entry and exits 0

### Requirement: `opm module tidy` and `opm catalog tidy` require no `cue` executable and start no child process

`opm module tidy [path]` (alias `opm mod tidy`) and `opm catalog tidy [path]`:  The command SHALL NOT require a `cue` executable on the machine and SHALL NOT start any child process.

#### Scenario: No cue binary needed

- **WHEN** `opm catalog tidy` runs on a machine where no `cue` executable is on `PATH`
- **THEN** the catalog is tidied exactly as it would be with one present

### Requirement: `opm module tidy` and `opm catalog tidy` behave identically and only the wording of messages follows the kind

`opm module tidy [path]` (alias `opm mod tidy`) and `opm catalog tidy [path]`:  The two commands SHALL behave identically; the kind only changes the wording of messages.

#### Scenario: The two commands behave identically

- **WHEN** `opm module tidy` and `opm catalog tidy` run on trees of their kinds
- **THEN** the two commands behave identically, and the kind only changes the wording of messages

### Requirement: Tidy does not read or validate the artifact's OPM identity, schema conformance or declared version

Tidy is a CUE module operation: it SHALL NOT read or validate the artifact's OPM identity, schema conformance or declared version.

#### Scenario: Tidy ignores the OPM identity

- **WHEN** `opm module tidy` runs on a module
- **THEN** it does not read or validate the artifact's OPM identity, schema conformance or declared version, because tidy is a CUE module operation

### Requirement: The optional `path` argument of tidy names the module root and tidy searches no parent directory

`opm module tidy` and `opm catalog tidy`:  The optional `path` argument SHALL name the module root directory and SHALL default to the current directory. The command SHALL tidy that directory only; it SHALL NOT search parent directories for a `cue.mod`.

#### Scenario: Default path

- **WHEN** `opm module tidy` runs with no argument inside a module root
- **THEN** that module is tidied

### Requirement: Tidy refuses a path that is not a module root with exit 2 before any registry access

A path that does not exist, is not a directory, or has no `cue.mod/module.cue` SHALL be refused with exit 2 before any registry access. The refusal names the absolute path it checked; for `opm module tidy` on an existing directory without `cue.mod`, it also points at `opm module init` (no catalog equivalent exists, so `opm catalog tidy` names only the path).

#### Scenario: Not a module root

- **WHEN** `opm module tidy ./sub` names a directory with no `cue.mod/module.cue`, even when a parent directory has one
- **THEN** the command exits 2 naming the absolute path it checked, and no file anywhere is written

### Requirement: On success tidy prints one line saying whether it changed anything and rewrites no file of an already-tidy module

`opm module tidy` and `opm catalog tidy`:  On success the command SHALL print one line saying whether it changed anything: which files it updated (`cue.mod/module.cue`, `cue.mod/local-module.cue`, or both), or that the module was already tidy and nothing was written. An already-tidy module SHALL NOT have any file rewritten or its mtime changed.

#### Scenario: Update reported

- **WHEN** `opm module tidy` adds a dependency
- **THEN** it prints a single success line naming `cue.mod/module.cue` as updated

#### Scenario: Already tidy is a no-op

- **WHEN** `opm module tidy` runs on an already tidy module
- **THEN** it prints that the module is already tidy, exits 0, and no file's bytes or mtime change

### Requirement: Output from the embedded CUE tooling reaches the terminal only through the command's own error message

`opm module tidy` and `opm catalog tidy`:  Output from the embedded CUE tooling SHALL NOT reach the terminal except through the command's own error message.

#### Scenario: Embedded tooling output stays off the terminal

- **WHEN** `opm module tidy` or `opm catalog tidy` succeeds
- **THEN** output from the embedded CUE tooling does not reach the terminal except through the command's own error message

### Requirement: Tidy exit codes are 0 tidied, already tidy or check passed, 1 resolution or registry failure and 2 not tidy or not a module root

`opm module tidy` and `opm catalog tidy`:  Exit codes SHALL be: 0 tidied, already tidy, or check passed; 1 dependency resolution or registry failure; 2 not tidy under `--check`, or the target is not a module root.

#### Scenario: Exit codes by outcome

- **WHEN** `opm module tidy` or `opm catalog tidy` ends
- **THEN** the exit code is 0 when it tidied, found the module already tidy, or the check passed; 1 on a dependency resolution or registry failure; 2 when the module is not tidy under `--check`, or the target is not a module root
