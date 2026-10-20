# published-module-resolution Specification

## Purpose
Defines how a command that takes a published OPM module turns a major-free module path and an optional version selector into one exact published version, and how it reports that choice. Every such command resolves the same way. The rules come from enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given"), which fixes them for `opm instance init`; `opm module build` and `opm module apply` follow them by choice, so the CLI has one grammar for deploying a published module.

## Requirements

### Requirement: A published module path is a CUE module path without a major suffix and the version selector chooses the major

A published module path SHALL be a CUE module path without a major suffix: its first element contains a dot, it has at least two elements, and every element uses only lowercase letters, digits, `.`, `_` and `-` (for example `opmodel.dev/modules/web_app`). The major is chosen by the version selector, never by the path.

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: A major-free module path

- **WHEN** a command is given `opmodel.dev/modules/web_app`
- **THEN** it is a published module path without a major suffix, and the major is chosen by the version selector, never by the path

### Requirement: A published module path with an `@` suffix or a URL scheme is refused with exit code 2

A path carrying an `@` suffix SHALL be refused with exit code 2 before any registry access; the message names the path without its suffix and shows the `--version` spelling of the major or version the suffix named. A path with a URL scheme (`oci://`, `https://`) SHALL be refused with exit code 2 stating that modules are named by module path and routed by the registry configuration.

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Major suffix refused

- **WHEN** a command is given `opmodel.dev/modules/web_app@v1`
- **THEN** it exits 2 without contacting the registry, and the message suggests `opmodel.dev/modules/web_app --version v1`

#### Scenario: OCI URL refused

- **WHEN** a command is given `oci://ghcr.io/open-platform-model/opmodel.dev/modules/web_app`
- **THEN** it exits 2 stating that a module is named by its module path

### Requirement: `--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2

`--version` SHALL accept exactly two forms: `vN` (for example `v1`) selects the newest release within major N, and a bare SemVer `X.Y.Z` or `X.Y.Z-<prerelease>` (for example `1.0.4`) pins that exact published tag. Any other value, including `v1.0.4`, SHALL be refused with exit code 2 before any registry access, and the message shows both accepted forms.

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Malformed selector

- **WHEN** the selector is `v1.0.4`
- **THEN** the command exits 2 without contacting the registry and shows the forms `v1` and `1.0.4`

### Requirement: The newest release within a major is the newest stable version, else the newest prerelease that is not a development build

The newest release within a major SHALL be the newest stable version; when the major has no stable version, the newest prerelease that is not a development build (for example `2.0.0-alpha.2` or `2.0.0-rc.1`); a development build (any version whose prerelease carries the identifier `dev`, such as `1.2.0-0.dev.3.gabc1234`) is never selected by a float.

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Major float prefers stable

- **WHEN** major v1 holds `1.0.3`, `1.0.4` and `1.1.0-alpha.1`, and the selector is `v1`
- **THEN** `1.0.4` is selected

#### Scenario: Prerelease-only major

- **WHEN** major v2 holds only `2.0.0-alpha.1`, `2.0.0-alpha.2` and `2.0.1-0.dev.4.gdeadbee`, and the selector is `v2`
- **THEN** `2.0.0-alpha.2` is selected

### Requirement: An exact pin names a published tag, development builds included, and a pin the registry does not hold is refused with exit code 2

An exact pin SHALL name a published tag, development builds included; a pin the registry does not hold SHALL be refused with exit code 2 naming the module path, the missing version and the registry consulted. (Requirement: `--version` accepts exactly the forms `vN` and `X.Y.Z` or `X.Y.Z-<prerelease>` and any other value is refused with exit code 2)

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Exact pin

- **WHEN** the selector is `1.0.3` and the registry holds that tag
- **THEN** `1.0.3` is selected even though `1.0.4` exists

### Requirement: Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against

Without `--version`, the command SHALL list the module's published versions across all majors and walk the majors from highest to lowest. For each major it takes the newest release (as the float defines it) and reads that release's declared dependencies. It SHALL select the first major whose declared `opmodel.dev/core` dependency major equals the core major this CLI build renders against.

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: The first core-compatible major is selected

- **WHEN** `--version` is not given and several majors of the module are published
- **THEN** the command lists the published versions across all majors, walks the majors from highest to lowest and selects the first major whose declared `opmodel.dev/core` dependency major equals the core major this CLI build renders against

### Requirement: A major with no selectable release, no core dependency or a different core major is skipped, and the command exits 2 when every major is skipped

A major SHALL be skipped when it holds no selectable release, declares no `opmodel.dev/core` dependency, or declares a different core major. When every major is skipped, the command SHALL exit 2 naming each major and why it was skipped. (Requirement: Without `--version` the command walks the majors from highest to lowest and selects the first major on the core major the CLI renders against)

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Newer major on an incompatible core is skipped

- **WHEN** the CLI renders against core v2, major v3 of a module depends on core v3, and major v2 depends on core v2
- **THEN** the newest release of v2 is selected, and v3 is reported as skipped because it requires core v3

#### Scenario: Major without a core dependency is skipped

- **WHEN** the highest major of a module declares no `opmodel.dev/core` dependency and the next major depends on the CLI's core major
- **THEN** the next major's newest release is selected, and the higher major is reported as skipped because it declares no core dependency

#### Scenario: Nothing compatible

- **WHEN** no major of the module depends on the CLI's core major
- **THEN** the command exits 2 and lists every major with its skip reason

### Requirement: After resolving, the command reports the choice and the skipped higher majors on standard error

After resolving, the command SHALL report one line naming the module path, the selected major and version, and how it was chosen (pinned, newest in the named major, or highest major on the CLI's core major), followed by one line per higher major it skipped with the reason. The report SHALL go to standard error, so manifests or other data a command writes to standard output stay parseable.

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Report names the selection

- **WHEN** `opm module build opmodel.dev/modules/web_app` resolves to `1.0.4` in major v1 as the highest major on core v2
- **THEN** standard error carries a line naming `opmodel.dev/modules/web_app`, `v1`, `1.0.4` and that it is the highest major on core v2, and standard output carries only manifests

### Requirement: Resolution routes through the CLI's registry configuration and the command exits 3 naming the module path and the registry when the registry cannot be reached

Resolution SHALL route through the CLI's registry configuration (`--registry`, then `OPM_REGISTRY`, then the config file's `registry`). When the registry cannot be reached, the command SHALL exit 3 naming the module path and the registry.

**Source**: enhancement 0016 ("`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").

#### Scenario: Unreachable registry

- **WHEN** the registry serving the module path cannot be reached
- **THEN** the command exits 3 naming the module path and the registry, and renders nothing
