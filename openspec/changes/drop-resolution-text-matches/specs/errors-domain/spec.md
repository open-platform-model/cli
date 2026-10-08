## ADDED Requirements

### Requirement: Fetch and resolution answers are decided by error type

The cli SHALL decide what a registry fetch failure or a dependency resolution failure means (an
exit code, a retry, "absent", a hint) from the typed error the library gives it, and SHALL NOT
decide it from the text of the error message. Source: 0021:D8:R12.

Where the cause is a CUE evaluation error, the cli SHALL read the error's path, never its text.

Non-test cli code MAY read error text only in these cases, none of which is a fetch or resolution
failure:

- the "module is not tidy" answer of the embedded `cue mod tidy`;
- dropping incompleteness errors from the identity package conformance check;
- dropping CUE's disjunction summary line from grouped validation output;
- splitting the library's duplicate-identities message into a header and its rows.

A test SHALL fail when non-test cli code gains an error-text predicate outside that list.

#### Scenario: A predecessor package with an unprovided import reads as absent

- **WHEN** the publish compatibility walk loads a published predecessor package and one of its
  imports is provided by no module of its build
- **THEN** the walk reads that predecessor as absent and goes on, as before
- **AND** the answer comes from the library's typed resolution error of kind "import unprovided"

#### Scenario: A registry failure never reads as absent

- **WHEN** the same load fails because the registry refused the connection or answered 503
- **THEN** the walk stops with a connectivity error and exit code 3, with no verdict

#### Scenario: A new text match is refused

- **WHEN** non-test cli code tests an error's message with a string predicate outside the listed
  cases
- **THEN** the guard test fails and names the file and line

### Requirement: Platform module build failure hints

When a platform module fails to build, the cli SHALL exit with the validation exit code 2 and
SHALL add one hint chosen from the cause:

- the package is not a single `#Platform` package: the hint names the expected shape;
- a registry fetch failed, or a dependency did not resolve (an import no module provides, an
  ambiguous import, a dependency module file that does not parse): the hint says to pin a
  published build in the platform's `cue.mod/module.cue`;
- a CUE evaluation error at a path under `#registry`: the hint says each `#registry` key must
  equal the module path of the catalog the entry imports, and names `platform.cue`;
- anything else: the hint says to fix the platform module and names its directory and module
  file.

The hint SHALL NOT change the exit code.

#### Scenario: An unpublished catalog pin

- **WHEN** the platform module pins a catalog build the registry does not hold
- **THEN** the build fails with exit code 2 and the hint says to pin a published build

#### Scenario: A refused registry

- **WHEN** the registry refuses the connection while the platform module's imports resolve
- **THEN** the hint says to pin a published build, as before

#### Scenario: A directly imported dependency whose module file does not parse

- **WHEN** the platform module imports a dependency whose `cue.mod/module.cue` does not parse
- **THEN** the hint says to pin a published build, the same hint the defect gets when it is met
  while the module graph is expanded

#### Scenario: An import whose package name does not match

- **WHEN** an import resolves to a directory that holds no file of the imported package name
- **THEN** the hint says to fix the platform module and names its directory and module file

#### Scenario: A registry key that differs from the imported catalog

- **WHEN** a `#registry` entry is keyed at a path other than the module path of its `#catalog`
- **THEN** the message names the entry and the hint says the key must equal the module path of
  the catalog it imports

#### Scenario: A registry entry the shape check refuses

- **WHEN** a `#registry` entry is refused as incomplete before the schema evaluates it
- **THEN** the hint says to fix the platform module and names its directory and module file

### Requirement: Config validation hints read the failing field

When the config file fails schema validation, the cli SHALL pick the removed-field hint
(`providers`, `cacheDir`) or the `skewPolicy` hint from the path of the failing field, and the
generic hint otherwise. A value that merely contains one of those words SHALL NOT pick its hint.
The messages, the hints and the exit code SHALL stay as they are.

#### Scenario: A removed field

- **WHEN** the config file sets `providers`
- **THEN** validation fails and the hint says the `providers` field was removed

#### Scenario: A word in a value is not a field

- **WHEN** the config file is invalid at another field and a value in it contains the word
  `cacheDir`
- **THEN** the hint is the generic one
