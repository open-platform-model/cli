## REMOVED Requirements

### Requirement: Error types package location
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "All error types are defined in `pkg/errors/` and the package exports the named types and sentinels", "Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves".

### Requirement: Fetch and resolution answers are decided by error type
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path", "Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate", "The error-text test reads the source and not its types".

### Requirement: Platform module build failure hints
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A platform module build failure caused by the package shape or a `#registry` key gets a hint that names the fix", "A platform module build that the registry refuses for its credentials gets a login hint and exits 4", "A platform module build that fails on another registry fetch, an unresolved dependency or anything else gets a pin or fix hint", "A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code".

## ADDED Requirements

### Requirement: All error types are defined in `pkg/errors/` and the package exports the named types and sentinels

All error types SHALL be defined in `pkg/errors/`. The package SHALL export `DetailError` (with `NewValidationError` and `Wrap`), `ValidationError`, `GroupedError` and `ErrorLocation` (with `GroupedErrorsFromError`), and the sentinel errors `ErrValidation`, `ErrConnectivity`, `ErrPermission` and `ErrNotFound`.

#### Scenario: Error types importable from pkg/errors
- **WHEN** code imports `github.com/open-platform-model/cli/pkg/errors`
- **THEN** all error types, sentinels, and grouping helpers are accessible

#### Scenario: Import alias convention
- **WHEN** code imports `pkg/errors` alongside stdlib `errors`
- **THEN** the convention `import oerrors "github.com/open-platform-model/cli/pkg/errors"` SHALL be used

### Requirement: Types that no code constructs are removed and every exported type keeps its `.Error()` string and `Unwrap()` behavior across moves

Types that no code in the tree or its consumers constructs (`ConfigError`, `TransformError`, `FieldError`) are removed rather than preserved. Every exported type SHALL keep its `.Error()` string and `Unwrap()` behavior stable across moves within the package.

#### Scenario: ValidationError wrapping is unchanged
- **WHEN** `errors.As` or `errors.Unwrap` is used with a `ValidationError`
- **THEN** the cause error is correctly unwrapped as before

### Requirement: The cli decides what a registry fetch or dependency resolution failure means from the typed error and reads a CUE evaluation error by its path

The cli SHALL decide what a registry fetch failure or a dependency resolution failure means (an exit code, a retry, "absent", a hint) from the typed error the library gives it, and SHALL NOT decide it from the text of the error message.
Where the cause is a CUE evaluation error, the cli SHALL read the error's path, never its text.

**Source**: enhancement 0021 ("A beta line reaches GA only when its exit criteria hold").

#### Scenario: A predecessor package with an unprovided import reads as absent

- **WHEN** the publish compatibility walk loads a published predecessor package and one of its
  imports is provided by no module of its build
- **THEN** the walk reads that predecessor as absent and goes on, as before
- **AND** the answer comes from the library's typed resolution error of kind "import unprovided"

#### Scenario: A registry failure never reads as absent

- **WHEN** the same load fails because the registry refused the connection or answered 503
- **THEN** the walk stops with a connectivity error and exit code 3, with no verdict

### Requirement: Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate

Non-test cli code MAY read error text only in these cases, none of which is a fetch or resolution failure:
- the "module is not tidy" answer of the embedded `cue mod tidy`;
- dropping incompleteness errors from the identity package conformance check;
- dropping CUE's disjunction summary line from grouped validation output;
- splitting the library's duplicate-identities message into a header and its rows.
A test SHALL fail when non-test cli code gains an error-text predicate outside that list.

**Source**: enhancement 0021 ("A beta line reaches GA only when its exit criteria hold").

#### Scenario: A new text match is refused

- **WHEN** non-test cli code tests an error's message with a string predicate outside the listed
  cases
- **THEN** the guard test fails and names the file and line

#### Scenario: A listed text read is allowed

- **WHEN** non-test cli code splits the library's duplicate-identities message into a header and its rows
- **THEN** it MAY read the error text, because that is one of the listed cases and none of the listed cases is a fetch or resolution failure

### Requirement: The error-text test reads the source and not its types

The test SHALL read the source, not its types: it sees a string or pattern predicate, a comparison or a switch over an error's message inside one function, also when the message is wrapped in other calls. It does not see a message made by formatting the error, or a message handed to another function as a string. (Requirement: Non-test cli code reads error text only in four listed cases and a test fails on any other error-text predicate)

**Source**: enhancement 0021 ("A beta line reaches GA only when its exit criteria hold").

#### Scenario: A message made by formatting the error is not seen

- **WHEN** non-test cli code compares a message made by formatting the error
- **THEN** the test does not see it, because it reads the source and not its types

### Requirement: A platform module build failure caused by the package shape or a `#registry` key gets a hint that names the fix

When a platform module fails to build, the cli SHALL add one hint chosen from the cause:
- the package is not a single `#Platform` package: the hint names the expected shape;
- a CUE evaluation error at a path under `#registry`: the hint says each `#registry` key must equal the module path of the catalog the entry imports, and names `platform.cue`;

#### Scenario: The package is not a single `#Platform` package

- **WHEN** the platform module's package is not a single `#Platform` package
- **THEN** the hint names the expected shape

#### Scenario: A registry key that differs from the imported catalog

- **WHEN** a `#registry` entry is keyed at a path other than the module path of its `#catalog`
- **THEN** the message names the entry and the hint says the key must equal the module path of
  the catalog it imports

### Requirement: A platform module build that the registry refuses for its credentials gets a login hint and exits 4

When a platform module fails to build, the cli SHALL add one hint chosen from the cause:
- the registry refused the credentials (a 401 answer from the registry or from its token endpoint, or a 403 answer that reaches the cli as a refusal): the hint says to log in to the registry and names the command `opm registry login`, followed by the registry host when every dependency the platform's `cue.mod/module.cue` declares routes to one host through the configured registry mapping;

#### Scenario: A refused credential

- **WHEN** the registry answers 401 while the platform module's imports resolve, and every
  dependency the platform declares routes to one registry host
- **THEN** the build fails with exit code 4, the hint is
  `Log in to the registry, then retry:  opm registry login <host>` with that host, and the
  hint does not say to pin a published build

#### Scenario: A refused credential under a prefix mapping

- **WHEN** the registry refuses the credentials, the configured registry mapping routes a
  module path prefix to one host and everything else to another, and every dependency the
  platform declares is under that prefix
- **THEN** the hint names `opm registry login` with the host of the prefix

#### Scenario: A refused credential with dependencies on several registry hosts

- **WHEN** the registry refuses the credentials and the dependencies the platform declares
  route to more than one host
- **THEN** the hint names `opm registry login` without a host

#### Scenario: A token endpoint that refuses the credentials

- **WHEN** the registry hands out bearer tokens and its token endpoint answers 401 while the
  platform module's imports resolve
- **THEN** the build fails with exit code 4 and the hint is the login hint with the registry
  host, as for a 401 answer from the registry itself

### Requirement: A platform module build that fails on another registry fetch, an unresolved dependency or anything else gets a pin or fix hint

When a platform module fails to build, the cli SHALL add one hint chosen from the cause:
- any other registry fetch failed, or a dependency did not resolve (an import no module provides, an ambiguous import, a dependency module file that does not parse): the hint says to pin a published build in the platform's `cue.mod/module.cue`;
- anything else: the hint says to fix the platform module and names its directory and module file.

#### Scenario: An unpublished catalog pin

- **WHEN** the platform module pins a catalog build the registry does not hold
- **THEN** the build fails with exit code 2 and the hint says to pin a published build

#### Scenario: A refused registry

- **WHEN** the registry refuses the connection while the platform module's imports resolve
- **THEN** the hint says to pin a published build, as before

#### Scenario: A refusal the cli receives as not found

- **WHEN** the registry answers 403 to the tag lookup, or its token endpoint answers 403
- **THEN** the build fails with exit code 2 and the hint says to pin a published build, because
  the registry client reports both as a module that is not found

#### Scenario: A directly imported dependency whose module file does not parse

- **WHEN** the platform module imports a dependency whose `cue.mod/module.cue` does not parse
- **THEN** the hint says to pin a published build, the same hint the defect gets when it is met
  while the module graph is expanded

#### Scenario: An import whose package name does not match

- **WHEN** an import resolves to a directory that holds no file of the imported package name
- **THEN** the hint says to fix the platform module and names its directory and module file

#### Scenario: A registry entry the shape check refuses

- **WHEN** a `#registry` entry is refused as incomplete before the schema evaluates it
- **THEN** the hint says to fix the platform module and names its directory and module file

### Requirement: A platform module build failure hint is read from the type of the error and only a refused credential changes the exit code

When a platform module fails to build, the cause SHALL be read from the type of the error, never from its text. A refused credential SHALL exit with the permission exit code 4. Every other cause SHALL exit with the validation exit code 2, and its hint SHALL NOT change the exit code.

#### Scenario: A refused credential exits 4 and every other cause exits 2

- **WHEN** a platform module fails to build because the registry refused the credentials
- **THEN** the build exits with the permission exit code 4

- **WHEN** a platform module fails to build for any other cause
- **THEN** the build exits with the validation exit code 2 and its hint does not change the exit code
