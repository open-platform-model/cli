# published-module-resolution Specification

## Purpose
Defines how a command that takes a published OPM module turns a major-free module path and an optional version selector into one exact published version, and how it reports that choice. Every such command resolves the same way. The rules come from 0016:D5, which fixes them for `opm instance init`; `opm module build` and `opm module apply` follow them by choice, so the CLI has one grammar for deploying a published module.

## Requirements

### Requirement: A published module is named by a major-free module path

A published module path SHALL be a CUE module path without a major suffix: its first element contains a dot, it has at least two elements, and every element uses only lowercase letters, digits, `.`, `_` and `-` (for example `opmodel.dev/modules/web_app`). The major is chosen by the version selector, never by the path. A path carrying an `@` suffix SHALL be refused with exit code 2 before any registry access; the message names the path without its suffix and shows the `--version` spelling of the major or version the suffix named. A path with a URL scheme (`oci://`, `https://`) SHALL be refused with exit code 2 stating that modules are named by module path and routed by the registry configuration. Source: 0016:D5:R1.

#### Scenario: Major suffix refused

- **WHEN** a command is given `opmodel.dev/modules/web_app@v1`
- **THEN** it exits 2 without contacting the registry, and the message suggests `opmodel.dev/modules/web_app --version v1`

#### Scenario: OCI URL refused

- **WHEN** a command is given `oci://ghcr.io/open-platform-model/opmodel.dev/modules/web_app`
- **THEN** it exits 2 stating that a module is named by its module path

### Requirement: The version selector floats within a major or pins a tag

`--version` SHALL accept exactly two forms: `vN` (for example `v1`) selects the newest release within major N, and a bare SemVer `X.Y.Z` or `X.Y.Z-<prerelease>` (for example `1.0.4`) pins that exact published tag. Any other value, including `v1.0.4`, SHALL be refused with exit code 2 before any registry access, and the message shows both accepted forms. The newest release within a major is the newest stable version; when the major has no stable version, the newest prerelease that is not a development build (for example `2.0.0-alpha.2` or `2.0.0-rc.1`); a development build (any version whose prerelease carries the identifier `dev`, such as `1.2.0-0.dev.3.gabc1234`) is never selected by a float. An exact pin SHALL name a published tag, development builds included; a pin the registry does not hold SHALL be refused with exit code 2 naming the module path, the missing version and the registry consulted. Source: 0016:D5:R2.

#### Scenario: Major float prefers stable

- **WHEN** major v1 holds `1.0.3`, `1.0.4` and `1.1.0-alpha.1`, and the selector is `v1`
- **THEN** `1.0.4` is selected

#### Scenario: Prerelease-only major

- **WHEN** major v2 holds only `2.0.0-alpha.1`, `2.0.0-alpha.2` and `2.0.1-0.dev.4.gdeadbee`, and the selector is `v2`
- **THEN** `2.0.0-alpha.2` is selected

#### Scenario: Exact pin

- **WHEN** the selector is `1.0.3` and the registry holds that tag
- **THEN** `1.0.3` is selected even though `1.0.4` exists

#### Scenario: Malformed selector

- **WHEN** the selector is `v1.0.4`
- **THEN** the command exits 2 without contacting the registry and shows the forms `v1` and `1.0.4`

### Requirement: No selector selects the highest core-compatible major

Without `--version`, the command SHALL list the module's published versions across all majors and walk the majors from highest to lowest. For each major it takes the newest release (as the float defines it) and reads that release's declared dependencies. It SHALL select the first major whose declared `opmodel.dev/core` dependency major equals the core major this CLI build renders against. A major is skipped when it holds no selectable release, declares no `opmodel.dev/core` dependency, or declares a different core major. When every major is skipped, the command SHALL exit 2 naming each major and why it was skipped. Source: 0016:D5:R3.

#### Scenario: Newer major on an incompatible core is skipped

- **WHEN** the CLI renders against core v2, major v3 of a module depends on core v3, and major v2 depends on core v2
- **THEN** the newest release of v2 is selected, and v3 is reported as skipped because it requires core v3

#### Scenario: Major without a core dependency is skipped

- **WHEN** the highest major of a module declares no `opmodel.dev/core` dependency and the next major depends on the CLI's core major
- **THEN** the next major's newest release is selected, and the higher major is reported as skipped because it declares no core dependency

#### Scenario: Nothing compatible

- **WHEN** no major of the module depends on the CLI's core major
- **THEN** the command exits 2 and lists every major with its skip reason

### Requirement: Resolution reports the choice and never corrupts command output

After resolving, the command SHALL report one line naming the module path, the selected major and version, and how it was chosen (pinned, newest in the named major, or highest major on the CLI's core major), followed by one line per higher major it skipped with the reason. The report SHALL go to standard error, so manifests or other data a command writes to standard output stay parseable. Resolution SHALL route through the CLI's registry configuration (`--registry`, then `OPM_REGISTRY`, then the config file's `registry`). When the registry cannot be reached, the command SHALL exit 3 naming the module path and the registry. Source: 0016:D5:R3/R7.

#### Scenario: Report names the selection

- **WHEN** `opm module build opmodel.dev/modules/web_app` resolves to `1.0.4` in major v1 as the highest major on core v2
- **THEN** standard error carries a line naming `opmodel.dev/modules/web_app`, `v1`, `1.0.4` and that it is the highest major on core v2, and standard output carries only manifests

#### Scenario: Unreachable registry

- **WHEN** the registry serving the module path cannot be reached
- **THEN** the command exits 3 naming the module path and the registry, and renders nothing
