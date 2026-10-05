# instance-init Specification

## Purpose
`opm instance init` turns a published OPM module into a standalone, committable instance package that builds straight away, with no `cue` binary and no hand-written pins.

## Requirements

### Requirement: Init writes a complete standalone instance package

`opm instance init` SHALL write exactly three files into the target directory, forming a standalone CUE module whose package evaluates to a `#ModuleInstance`. Every line derives from the acquired module; nothing is copied from a template the user did not name. Source: 0016:D1:R1/R3.

- `cue.mod/module.cue` declares the package's own module path, a language version, and dependencies: the deployed module pinned to the exact resolved version, `opmodel.dev/core` at the version the module itself declares, and the rest of the dependency closure. The file SHALL be exactly what `opm module tidy` leaves, so a tidy check of it passes. Source: 0016:D9:R1.
- `instance.cue` declares `package instance`, imports core at the module's core major and the module at its major, embeds `core.#ModuleInstance`, sets `metadata.name` and `metadata.namespace`, and sets `#module` to the imported module.
- `values.cue` declares `package instance` and one `values` field, filled as "Values start from the author's intended source" requires.

The package's own module path SHALL default to `instance.local/<instance-name>@v0`. `--module-path` (string, default empty) overrides it with any valid major-qualified module path. The path is never looked up in a registry. Source: 0016:D9:R3.

#### Scenario: Package for a published module

- **WHEN** the user runs `opm instance init cert-manager opmodel.dev/modules/cert_manager --namespace cert-manager` and the newest release of the highest core-compatible major is `2.0.1` in major v2
- **THEN** `cert-manager/cue.mod/module.cue` declares module `instance.local/cert-manager@v0`, pins `opmodel.dev/modules/cert_manager@v2` at `v2.0.1` and `opmodel.dev/core@v2` at the version cert_manager 2.0.1 declares, and lists the rest of the closure
- **AND** `cert-manager/instance.cue` embeds `core.#ModuleInstance` with name `cert-manager`, namespace `cert-manager`, and `#module` set to the imported cert_manager package

#### Scenario: Builds with no step in between

- **WHEN** init has written a package whose values satisfy the module's `#config`
- **THEN** `opm instance build <dir>/instance.cue` and `opm instance vet <dir>/instance.cue` both succeed without any other command in between, and the package loads from CUE's module cache when the registry is unreachable. Source: 0016:D1:R2, 0016:D9:R2

#### Scenario: Module path override

- **WHEN** the user passes `--module-path example.com/deploy/cert-manager@v0`
- **THEN** `cue.mod/module.cue` declares that module path, and no registry request names it

### Requirement: Init arguments mirror module init

The command SHALL take `[instance-name] [module-path]` positionals and the flags `--from <module-path>`, `--version <vN | X.Y.Z>`, `--namespace`/`-n <ns>`, `--dir <dir>` and `--module-path <path>`. With one positional, a value containing `/` or `.` is the module path, and anything else is the instance name. `--from` is another spelling of the module path; naming the module path twice SHALL be refused with exit 2 rather than ranked. The module path SHALL be major-free: a path carrying a major suffix is refused with exit 2 and a hint to select the major with `--version`. The instance name and the namespace SHALL each match the core name type (lowercase letters, digits and `-`, starting and ending with a letter or digit, at most 63 characters), checked before any registry access. `--dir` SHALL default to the instance name. When the instance name, the module path or the namespace is missing and a terminal is attached, the command SHALL prompt for it, in that order; without a terminal the omission SHALL be refused with exit 2 naming the missing input (the positional or `--from` for the module path, `--namespace` for the namespace). A prompted value passes the same checks as the flag or positional it replaces, before any registry access. Source: 0016:D5:R1/R4.

#### Scenario: Positional module path

- **WHEN** the user runs `opm instance init web opmodel.dev/modules/web_app -n demo`
- **THEN** the instance is named `web` in namespace `demo` and written to `./web`

#### Scenario: Module path named twice

- **WHEN** the user runs `opm instance init web opmodel.dev/modules/web_app --from opmodel.dev/modules/web_app -n demo`
- **THEN** the command exits 2 stating that the module path was named more than once

#### Scenario: Major suffix refused

- **WHEN** the user runs `opm instance init web opmodel.dev/modules/web_app@v1 -n demo`
- **THEN** the command exits 2 without contacting the registry and suggests `--version v1`

#### Scenario: Missing namespace without a terminal

- **WHEN** the command runs with standard input not a terminal and no `--namespace`
- **THEN** it exits 2 naming `--namespace`, and writes nothing

#### Scenario: Missing module path without a terminal

- **WHEN** the user runs `opm instance init web -n demo` with standard input not a terminal
- **THEN** it exits 2 naming the module path positional and `--from`, and writes nothing

#### Scenario: Missing module path on a terminal

- **WHEN** the user runs `opm instance init web -n demo` on a terminal and answers the prompt with `opmodel.dev/modules/web_app@v1`
- **THEN** the command exits 2 without contacting the registry and suggests `--version v1`, as it would for the positional

#### Scenario: Invalid instance name

- **WHEN** the instance name is `Web_App`
- **THEN** the command exits 2 before any registry access, naming the name rule

### Requirement: Init selects the module version like every published-module command

The module version SHALL be resolved exactly as the `published-module-resolution` capability defines: `--version vN` takes the newest release within major N (never a development build), an exact SemVer pins that tag, and no `--version` takes the newest release of the highest major whose core dependency major equals the CLI's core major, skipping any other major. Source: 0016:D5:R2/R3.

#### Scenario: Highest core-compatible major

- **WHEN** the module's highest major depends on a newer core major than the CLI's, and the next major depends on the CLI's core major
- **THEN** init selects the next major's newest release, and the report names the higher major as skipped with its reason

#### Scenario: Pinned version

- **WHEN** the user passes `--version 1.0.3`
- **THEN** the module is pinned at `v1.0.3` in `cue.mod/module.cue`

### Requirement: Values start from the author's intended source

`values.cue` SHALL be filled from the first applicable source in this order, and the report SHALL name the source used:

1. The module's `initValues`, when the module declares it. Its content is rendered even when non-concrete: a defaulted field appears as its default, an undefaulted disjunction appears as the disjunction, and an optional field is omitted. No `debugValues` content SHALL reach the package. Source: 0016:D3:R2, 0016:D4:R3.
2. Otherwise the module's `debugValues`, when the whole value is concrete once defaults are applied. A `debugValues` with any field left without a value (a bare type, an undefaulted disjunction, or `_`) is not concrete and falls to the next rung. The report SHALL warn the user to review the file before deploying. Source: 0016:D2:R1/R2/R3.
3. Otherwise `values: {}`. The report SHALL warn that the values file is empty and point at `opm instance vet <dir>/instance.cue` for the contract the user must now satisfy. Source: 0016:D6:R1/R2.

A concrete non-struct source SHALL be rendered verbatim. When the chosen source renders as an empty struct, the report SHALL still name that source and SHALL also carry the empty-file warning pointing at `opm instance vet <dir>/instance.cue`. `values.cue` SHALL open with a comment naming the source it was filled from. Source: 0016:D6:R3.

#### Scenario: debugValues used and flagged

- **WHEN** the module declares no `initValues` and has concrete `debugValues`
- **THEN** `values.cue` carries the `debugValues` content, and the report names `debugValues` and warns to review it before deploying

#### Scenario: initValues takes precedence

- **WHEN** the module declares both `initValues` and `debugValues` with different content
- **THEN** `values.cue` carries the `initValues` content only, and the report names `initValues`

#### Scenario: Non-concrete initValues

- **WHEN** `initValues` holds a defaulted field, an undefaulted disjunction and an optional field
- **THEN** `values.cue` carries the default value, the disjunction as written, and no line for the optional field

#### Scenario: No usable source

- **WHEN** the module declares no `initValues` and its `debugValues` is not concrete
- **THEN** all three files are written, `values.cue` holds `values: {}`, and the report warns that it is empty and names `opm instance vet`

#### Scenario: Partly concrete debugValues

- **WHEN** the module declares no `initValues` and its `debugValues` is `{image: "nginx:1.27", replicas: int}`
- **THEN** `values.cue` holds `values: {}`, and the report names the source as empty and warns that the file is empty

#### Scenario: Empty debugValues

- **WHEN** the module declares no `initValues` and its `debugValues` is `{}`
- **THEN** `values.cue` holds `values: {}`, and the report names `debugValues`, warns to review it, and also warns that the file is empty, naming `opm instance vet`

### Requirement: Init writes all or nothing, and only where a standalone package belongs

The command SHALL refuse with exit 2, before any registry access, when the target directory already exists (whether empty or holding a module or instance package), or when the target directory would sit inside an existing CUE module (a parent directory holds `cue.mod/module.cue`). The package SHALL be written completely or not at all: on any failure after writing starts, nothing is left at the target path or beside it, so a retry into the same directory is never refused because of init's own leftovers. Source: 0016:D5:R5/R6, 0016:D10:R1.

#### Scenario: Existing directory refused

- **WHEN** `./web` already exists
- **THEN** `opm instance init web opmodel.dev/modules/web_app -n demo` exits 2 naming `./web`, and changes nothing

#### Scenario: Inside a CUE module refused

- **WHEN** the current directory is inside a module tree whose root holds `cue.mod/module.cue`
- **THEN** init exits 2 naming the enclosing module root, and suggests a target directory outside it

#### Scenario: Failure leaves nothing behind

- **WHEN** dependency resolution fails after the files were staged
- **THEN** no directory exists at the target path, no staging directory remains beside it, and rerunning the same command after the cause is fixed succeeds

### Requirement: Init reports what it wrote and does not validate it

Init SHALL NOT evaluate the generated package against the module's `#config`. On success it SHALL report: the resolution (module path, selected major and version, and how it was chosen, with every skipped higher major and its reason); the values source and any warning; the files written; and a closing line naming `opm instance vet <dir>/instance.cue`. The pinned version SHALL appear in the report; pins never float. Exit codes SHALL be those of `opm module init`: 0 written, 2 refused, 3 registry unreachable at any stage, including while the dependency closure is resolved; an unexpected internal failure exits 1. Source: 0016:D5:R3/R7, 0016:D9:R4, 0016:D8.

#### Scenario: Success report

- **WHEN** init succeeds from `debugValues`
- **THEN** the output names the resolved version, `debugValues` with a review warning, the three files, and ends with `opm instance vet <dir>/instance.cue`, and the exit code is 0

#### Scenario: Registry unreachable

- **WHEN** the registry serving the module path cannot be reached
- **THEN** init exits 3 naming the module path and the registry, and writes nothing

#### Scenario: Registry unreachable while resolving dependencies

- **WHEN** the module was acquired, and the registry cannot be reached while the staged package's dependency closure is resolved
- **THEN** init exits 3 naming the registry, and nothing is left at the target path or beside it

#### Scenario: Values that violate the contract still initialize

- **WHEN** the module's `debugValues` does not satisfy its `#config`
- **THEN** init writes the package and exits 0; the violation surfaces at the user's first `opm instance vet`

### Requirement: Only a registry that gave no response counts as unreachable

While `opm instance init` acquires the module archive and while it resolves the staged package's dependency closure, a registry failure SHALL count as unreachable (exit 3) when no HTTP response was received: a refused connection, a DNS or TLS failure, or a timeout. A registry answer that a dependency of the staged package is not held SHALL NOT count as unreachable: init exits 1 naming the failure. Reading the module's own module file, between resolution and acquire, stays exit 3 for every failure. Source: 0016:D5:R7. The CLI SHALL take the unreachable decision from the library's typed fetch classification and SHALL NOT match the registry error's message text itself. Source: 0021:D8:R12.

#### Scenario: A dependency the registry does not hold

- **WHEN** the registry answers that a dependency of the staged package is not held
- **THEN** init exits 1, not 3, and nothing is left at the target path or beside it
