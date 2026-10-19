## REMOVED Requirements

### Requirement: Init writes a complete standalone instance package
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Init writes exactly three files forming a standalone CUE module", "The cue.mod/module.cue file declares the package's module path, language version and dependencies", "The module file is exactly what opm module tidy leaves", "The instance.cue file declares the package, the imports and the module instance", "The values.cue file declares one values field", "The package's own module path defaults to a local placeholder and is never looked up".

### Requirement: Init arguments mirror module init
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Init takes the positionals and flags of module init", "With one positional a value with a slash or a dot is the module path", "Naming the module path twice is refused", "The module path is major-free", "The instance name and the namespace match the core name type and --dir defaults to the instance name", "A missing input is prompted on a terminal and refused without one", "A prompted value passes the same checks as the flag it replaces".

### Requirement: Values start from the author's intended source
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "values.cue is filled from the first applicable source and the report names it", "The initValues source", "No debugValues content reaches the package", "The debugValues source", "A debugValues source makes the report warn the user to review the file", "With no usable source values.cue holds an empty struct and the report warns", "A concrete non-struct source is rendered verbatim", "An empty struct source is still named and carries the empty-file warning", "values.cue opens with a comment naming its source".

### Requirement: Init writes all or nothing, and only where a standalone package belongs
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "An existing target directory is refused with exit 2 before any registry access", "A target inside an existing CUE module is refused with exit 2 before any registry access", "The package is written completely or not at all".

### Requirement: Init reports what it wrote and does not validate it
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Init does not evaluate the generated package against the module config", "On success init reports the resolution, the values source, the files and a closing vet line", "The pinned version appears in the report", "Init exit codes are those of module init".

### Requirement: Only a registry that gave no response counts as unreachable
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A registry failure with no HTTP response counts as unreachable", "A registry answer that a dependency is not held is not unreachable", "Reading the module's own module file stays exit 3 for every failure", "The CLI takes the unreachable decision from the library's typed fetch classification".

## ADDED Requirements

### Requirement: Init writes exactly three files forming a standalone CUE module

`opm instance init` SHALL write exactly three files into the target directory, forming a standalone CUE module whose package evaluates to a `#ModuleInstance`. Every line derives from the acquired module; nothing is copied from a template the user did not name. Source: enhancement 0016 ("A new CLI command scaffolds an on-disk instance package from a published module reference").

#### Scenario: Package for a published module

- **WHEN** the user runs `opm instance init cert-manager opmodel.dev/modules/cert_manager --namespace cert-manager` and the newest release of the highest core-compatible major is `2.0.1` in major v2
- **THEN** `cert-manager/cue.mod/module.cue` declares module `instance.local/cert-manager@v0`, pins `opmodel.dev/modules/cert_manager@v2` at `v2.0.1` and `opmodel.dev/core@v2` at the version cert_manager 2.0.1 declares, and lists the rest of the closure
- **AND** `cert-manager/instance.cue` embeds `core.#ModuleInstance` with name `cert-manager`, namespace `cert-manager`, and `#module` set to the imported cert_manager package

#### Scenario: Builds with no step in between

- **WHEN** init has written a package whose values satisfy the module's `#config`
- **THEN** `opm instance build <dir>/instance.cue` and `opm instance vet <dir>/instance.cue` both succeed without any other command in between, and the package loads from CUE's module cache when the registry is unreachable. Source: enhancement 0016 ("A new CLI command scaffolds an on-disk instance package from a published module reference" and "The generated module file pins the module exactly, carries core at the module's major, and names a local placeholder module path")

### Requirement: The cue.mod/module.cue file declares the package's module path, language version and dependencies

`cue.mod/module.cue` SHALL declare the package's own module path, a language version, and dependencies: the deployed module pinned to the exact resolved version, `opmodel.dev/core` at the version the module itself declares, and the rest of the dependency closure. Source: enhancement 0016 ("The generated module file pins the module exactly, carries core at the module's major, and names a local placeholder module path").

#### Scenario: The module file declares the closure
- **WHEN** `opm instance init` writes `cue.mod/module.cue`
- **THEN** it declares the package's own module path, a language version, and dependencies: the deployed module pinned to the exact resolved version, `opmodel.dev/core` at the version the module itself declares, and the rest of the dependency closure

### Requirement: The module file is exactly what opm module tidy leaves

The `cue.mod/module.cue` file SHALL be exactly what `opm module tidy` leaves, so a tidy check of it passes. Source: enhancement 0016 ("The generated module file pins the module exactly, carries core at the module's major, and names a local placeholder module path").

#### Scenario: A tidy check of the module file passes
- **WHEN** `opm instance init` has written `cue.mod/module.cue`
- **THEN** the file SHALL be exactly what `opm module tidy` leaves, so a tidy check of it passes

### Requirement: The instance.cue file declares the package, the imports and the module instance

`instance.cue` SHALL declare `package instance`, imports core at the module's core major and the module at its major, embeds `core.#ModuleInstance`, sets `metadata.name` and `metadata.namespace`, and sets `#module` to the imported module.

#### Scenario: The instance file
- **WHEN** `opm instance init` writes `instance.cue`
- **THEN** it declares `package instance`, imports core at the module's core major and the module at its major, embeds `core.#ModuleInstance`, sets `metadata.name` and `metadata.namespace`, and sets `#module` to the imported module

### Requirement: The values.cue file declares one values field

`values.cue` SHALL declare `package instance` and one `values` field, filled as "values.cue is filled from the first applicable source and the report names it" requires.

#### Scenario: The values file
- **WHEN** `opm instance init` writes `values.cue`
- **THEN** it declares `package instance` and one `values` field, filled as "values.cue is filled from the first applicable source and the report names it" requires

### Requirement: The package's own module path defaults to a local placeholder and is never looked up

The package's own module path SHALL default to `instance.local/<instance-name>@v0`. `--module-path` (string, default empty) overrides it with any valid major-qualified module path. The path is never looked up in a registry. Source: enhancement 0016 ("The generated module file pins the module exactly, carries core at the module's major, and names a local placeholder module path").

#### Scenario: Module path override

- **WHEN** the user passes `--module-path example.com/deploy/cert-manager@v0`
- **THEN** `cue.mod/module.cue` declares that module path, and no registry request names it

### Requirement: Init takes the positionals and flags of module init

The command SHALL take `[instance-name] [module-path]` positionals and the flags `--from <module-path>`, `--version <vN | X.Y.Z>`, `--namespace`/`-n <ns>`, `--dir <dir>` and `--module-path <path>`. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: Positionals and flags
- **WHEN** `opm instance init` is run
- **THEN** the command SHALL take `[instance-name] [module-path]` positionals and the flags `--from <module-path>`, `--version <vN | X.Y.Z>`, `--namespace`/`-n <ns>`, `--dir <dir>` and `--module-path <path>`

### Requirement: With one positional a value with a slash or a dot is the module path

With one positional, a value containing `/` or `.` SHALL be the module path, and anything else SHALL be the instance name. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: Positional module path

- **WHEN** the user runs `opm instance init web opmodel.dev/modules/web_app -n demo`
- **THEN** the instance is named `web` in namespace `demo` and written to `./web`

### Requirement: Naming the module path twice is refused

`--from` is another spelling of the module path; naming the module path twice SHALL be refused with exit 2 rather than ranked. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: Module path named twice

- **WHEN** the user runs `opm instance init web opmodel.dev/modules/web_app --from opmodel.dev/modules/web_app -n demo`
- **THEN** the command exits 2 stating that the module path was named more than once

### Requirement: The module path is major-free

The module path SHALL be major-free: a path carrying a major suffix is refused with exit 2 and a hint to select the major with `--version`. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: Major suffix refused

- **WHEN** the user runs `opm instance init web opmodel.dev/modules/web_app@v1 -n demo`
- **THEN** the command exits 2 without contacting the registry and suggests `--version v1`

### Requirement: The instance name and the namespace match the core name type and --dir defaults to the instance name

The instance name and the namespace SHALL each match the core name type (lowercase letters, digits and `-`, starting and ending with a letter or digit, at most 63 characters), checked before any registry access. `--dir` SHALL default to the instance name. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: Invalid instance name

- **WHEN** the instance name is `Web_App`
- **THEN** the command exits 2 before any registry access, naming the name rule

### Requirement: A missing input is prompted on a terminal and refused without one

When the instance name, the module path or the namespace is missing and a terminal is attached, the command SHALL prompt for it, in that order; without a terminal the omission SHALL be refused with exit 2 naming the missing input (the positional or `--from` for the module path, `--namespace` for the namespace). Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: Missing namespace without a terminal

- **WHEN** the command runs with standard input not a terminal and no `--namespace`
- **THEN** it exits 2 naming `--namespace`, and writes nothing

#### Scenario: Missing module path without a terminal

- **WHEN** the user runs `opm instance init web -n demo` with standard input not a terminal
- **THEN** it exits 2 naming the module path positional and `--from`, and writes nothing

#### Scenario: Missing module path on a terminal

- **WHEN** the user runs `opm instance init web -n demo` on a terminal and answers the prompt with `opmodel.dev/modules/web_app@v1`
- **THEN** the command exits 2 without contacting the registry and suggests `--version v1`, as it would for the positional

### Requirement: A prompted value passes the same checks as the flag it replaces

A prompted value SHALL pass the same checks as the flag or positional it replaces, before any registry access. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: A prompted value is checked
- **WHEN** a value is prompted
- **THEN** it SHALL pass the same checks as the flag or positional it replaces, before any registry access

### Requirement: values.cue is filled from the first applicable source and the report names it

`values.cue` SHALL be filled from the first applicable source in this order, and the report SHALL name the source used.

#### Scenario: The report names the source
- **WHEN** `opm instance init` fills `values.cue`
- **THEN** it SHALL be filled from the first applicable source in this order, and the report SHALL name the source used

### Requirement: The initValues source

`values.cue` SHALL be filled from the first applicable source in this order: The module's `initValues`, when the module declares it. Its content is rendered even when non-concrete: a defaulted field appears as its default, an undefaulted disjunction appears as the disjunction, and an optional field is omitted. Source: enhancement 0016 ("The field is `initValues?: _`, open and optional, with no schema-side `#config` assertion").

#### Scenario: Non-concrete initValues

- **WHEN** `initValues` holds a defaulted field, an undefaulted disjunction and an optional field
- **THEN** `values.cue` carries the default value, the disjunction as written, and no line for the optional field

### Requirement: No debugValues content reaches the package

`values.cue` SHALL be filled from the first applicable source in this order: The module's `initValues`, when the module declares it. No `debugValues` content SHALL reach the package. Source: enhancement 0016 ("`#Module` gains an optional field carrying the author-intended init template, taking precedence over `debugValues`" and "The field is `initValues?: _`, open and optional, with no schema-side `#config` assertion").

#### Scenario: initValues takes precedence

- **WHEN** the module declares both `initValues` and `debugValues` with different content
- **THEN** `values.cue` carries the `initValues` content only, and the report names `initValues`

### Requirement: The debugValues source

`values.cue` SHALL be filled from the first applicable source in this order: in every other case than the one of requirement "The initValues source", the module's `debugValues`, when the whole value is concrete once defaults are applied. A `debugValues` with any field left without a value (a bare type, an undefaulted disjunction, or `_`) is not concrete and falls to the next rung. Source: enhancement 0016 ("`debugValues` is the default template source for the generated `values.cue`").

#### Scenario: Partly concrete debugValues

- **WHEN** the module declares no `initValues` and its `debugValues` is `{image: "nginx:1.27", replicas: int}`
- **THEN** `values.cue` holds `values: {}`, and the report names the source as empty and warns that the file is empty

### Requirement: A debugValues source makes the report warn the user to review the file

`values.cue` SHALL be filled from the first applicable source in this order: in every other case than the one of requirement "The initValues source", the module's `debugValues`, when the whole value is concrete once defaults are applied. The report SHALL warn the user to review the file before deploying. Source: enhancement 0016 ("`debugValues` is the default template source for the generated `values.cue`").

#### Scenario: debugValues used and flagged

- **WHEN** the module declares no `initValues` and has concrete `debugValues`
- **THEN** `values.cue` carries the `debugValues` content, and the report names `debugValues` and warns to review it before deploying

### Requirement: With no usable source values.cue holds an empty struct and the report warns

`values.cue` SHALL be filled from the first applicable source in this order: in every other case than the one of requirement "The initValues source" and the one of requirement "The debugValues source", `values: {}`. The report SHALL warn that the values file is empty and point at `opm instance vet <dir>/instance.cue` for the contract the user must now satisfy. Source: enhancement 0016 ("With no usable template source, init writes `values: {}` and warns").

#### Scenario: No usable source

- **WHEN** the module declares no `initValues` and its `debugValues` is not concrete
- **THEN** all three files are written, `values.cue` holds `values: {}`, and the report warns that it is empty and names `opm instance vet`

### Requirement: A concrete non-struct source is rendered verbatim

A concrete non-struct source SHALL be rendered verbatim. Source: enhancement 0016 ("With no usable template source, init writes `values: {}` and warns").

#### Scenario: Non-struct source verbatim
- **WHEN** the chosen source is a concrete non-struct value
- **THEN** it SHALL be rendered verbatim

### Requirement: An empty struct source is still named and carries the empty-file warning

When the chosen source renders as an empty struct, the report SHALL still name that source and SHALL also carry the empty-file warning pointing at `opm instance vet <dir>/instance.cue`. Source: enhancement 0016 ("With no usable template source, init writes `values: {}` and warns").

#### Scenario: Empty debugValues

- **WHEN** the module declares no `initValues` and its `debugValues` is `{}`
- **THEN** `values.cue` holds `values: {}`, and the report names `debugValues`, warns to review it, and also warns that the file is empty, naming `opm instance vet`

### Requirement: values.cue opens with a comment naming its source

`values.cue` SHALL open with a comment naming the source it was filled from. Source: enhancement 0016 ("With no usable template source, init writes `values: {}` and warns").

#### Scenario: The comment names the source
- **WHEN** `values.cue` is written
- **THEN** it SHALL open with a comment naming the source it was filled from

### Requirement: An existing target directory is refused with exit 2 before any registry access

The command SHALL refuse with exit 2, before any registry access, when the target directory already exists (whether empty or holding a module or instance package). Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given" and "Init refuses a target directory inside an existing CUE module").

#### Scenario: Existing directory refused

- **WHEN** `./web` already exists
- **THEN** `opm instance init web opmodel.dev/modules/web_app -n demo` exits 2 naming `./web`, and changes nothing

### Requirement: A target inside an existing CUE module is refused with exit 2 before any registry access

The command SHALL refuse with exit 2, before any registry access, when the target directory would sit inside an existing CUE module (a parent directory holds `cue.mod/module.cue`). Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given" and "Init refuses a target directory inside an existing CUE module").

#### Scenario: Inside a CUE module refused

- **WHEN** the current directory is inside a module tree whose root holds `cue.mod/module.cue`
- **THEN** init exits 2 naming the enclosing module root, and suggests a target directory outside it

### Requirement: The package is written completely or not at all

The package SHALL be written completely or not at all: on any failure after writing starts, nothing is left at the target path or beside it, so a retry into the same directory is never refused because of init's own leftovers. Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given" and "Init refuses a target directory inside an existing CUE module").

#### Scenario: Failure leaves nothing behind

- **WHEN** dependency resolution fails after the files were staged
- **THEN** no directory exists at the target path, no staging directory remains beside it, and rerunning the same command after the cause is fixed succeeds

### Requirement: Init does not evaluate the generated package against the module config

Init SHALL NOT evaluate the generated package against the module's `#config`. Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given", "The generated module file pins the module exactly, carries core at the module's major, and names a local placeholder module path" and "Init does not validate the generated package; it tells the user how to").

#### Scenario: Values that violate the contract still initialize

- **WHEN** the module's `debugValues` does not satisfy its `#config`
- **THEN** init writes the package and exits 0; the violation surfaces at the user's first `opm instance vet`

### Requirement: On success init reports the resolution, the values source, the files and a closing vet line

On success `opm instance init` SHALL report: the resolution (module path, selected major and version, and how it was chosen, with every skipped higher major and its reason); the values source and any warning; the files written; and a closing line naming `opm instance vet <dir>/instance.cue`. Source: enhancement 0016 ("Init does not validate the generated package; it tells the user how to").

#### Scenario: Success report

- **WHEN** init succeeds from `debugValues`
- **THEN** the output names the resolved version, `debugValues` with a review warning, the three files, and ends with `opm instance vet <dir>/instance.cue`, and the exit code is 0

### Requirement: The pinned version appears in the report

The pinned version SHALL appear in the report; pins never float. Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given", "The generated module file pins the module exactly, carries core at the module's major, and names a local placeholder module path" and "Init does not validate the generated package; it tells the user how to").

#### Scenario: The pinned version is reported
- **WHEN** `opm instance init` succeeds
- **THEN** the pinned version SHALL appear in the report; pins never float

### Requirement: Init exit codes are those of module init

Exit codes SHALL be those of `opm module init`: 0 written, 2 refused, 3 registry unreachable at any stage, including while the dependency closure is resolved; an unexpected internal failure exits 1. Source: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Registry unreachable

- **WHEN** the registry serving the module path cannot be reached
- **THEN** init exits 3 naming the module path and the registry, and writes nothing

#### Scenario: Registry unreachable while resolving dependencies

- **WHEN** the module was acquired, and the registry cannot be reached while the staged package's dependency closure is resolved
- **THEN** init exits 3 naming the registry, and nothing is left at the target path or beside it

### Requirement: A registry failure with no HTTP response counts as unreachable

While `opm instance init` acquires the module archive and while it resolves the staged package's dependency closure, a registry failure SHALL count as unreachable (exit 3) when no HTTP response was received: a refused connection, a DNS or TLS failure, or a timeout. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: No response is unreachable
- **WHEN** `opm instance init` acquires the module archive or resolves the staged package's dependency closure and a registry failure carries no HTTP response
- **THEN** the failure SHALL count as unreachable (exit 3): a refused connection, a DNS or TLS failure, or a timeout

### Requirement: A registry answer that a dependency is not held is not unreachable

A registry answer that a dependency of the staged package is not held SHALL NOT count as unreachable: init exits 1 naming the failure. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: A dependency the registry does not hold

- **WHEN** the registry answers that a dependency of the staged package is not held
- **THEN** init exits 1, not 3, and nothing is left at the target path or beside it

### Requirement: Reading the module's own module file stays exit 3 for every failure

Reading the module's own module file, between resolution and acquire, SHALL stay exit 3 for every failure. Source: enhancement 0016 (a published module floats to the highest core-compatible major).

#### Scenario: Reading the module file
- **WHEN** the module's own module file is read between resolution and acquire and the read fails
- **THEN** it SHALL stay exit 3 for every failure

### Requirement: The CLI takes the unreachable decision from the library's typed fetch classification

The CLI SHALL take the unreachable decision from the library's typed fetch classification and SHALL NOT match the registry error's message text itself. Source: enhancement 0021 ("A beta line reaches GA only when its exit criteria hold").

#### Scenario: No message matching
- **WHEN** the CLI decides that a registry is unreachable
- **THEN** it SHALL take the decision from the library's typed fetch classification and SHALL NOT match the registry error's message text itself
