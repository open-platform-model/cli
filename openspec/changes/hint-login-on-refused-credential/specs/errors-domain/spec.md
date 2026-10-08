## MODIFIED Requirements

### Requirement: Platform module build failure hints

When a platform module fails to build, the cli SHALL add one hint chosen from the cause:

- the package is not a single `#Platform` package: the hint names the expected shape;
- the registry refused the credentials (a 401 answer, or a 403 answer that reaches the cli as
  a refusal): the hint says to log in to the registry and names the command
  `opm registry login`, followed by the registry host when the configured registry mapping
  holds exactly one host;
- any other registry fetch failed, or a dependency did not resolve (an import no module
  provides, an ambiguous import, a dependency module file that does not parse): the hint says
  to pin a published build in the platform's `cue.mod/module.cue`;
- a CUE evaluation error at a path under `#registry`: the hint says each `#registry` key must
  equal the module path of the catalog the entry imports, and names `platform.cue`;
- anything else: the hint says to fix the platform module and names its directory and module
  file.

The cause SHALL be read from the type of the error, never from its text. A refused credential
SHALL exit with the permission exit code 4. Every other cause SHALL exit with the validation
exit code 2, and its hint SHALL NOT change the exit code.

#### Scenario: An unpublished catalog pin

- **WHEN** the platform module pins a catalog build the registry does not hold
- **THEN** the build fails with exit code 2 and the hint says to pin a published build

#### Scenario: A refused registry

- **WHEN** the registry refuses the connection while the platform module's imports resolve
- **THEN** the hint says to pin a published build, as before

#### Scenario: A refused credential

- **WHEN** the registry answers 401 while the platform module's imports resolve, and the
  configured registry mapping holds one host
- **THEN** the build fails with exit code 4, the hint is
  `Log in to the registry, then retry:  opm registry login <host>` with that host, and the
  hint does not say to pin a published build

#### Scenario: A refused credential with several registry hosts

- **WHEN** the registry refuses the credentials and the configured registry mapping holds
  more than one host
- **THEN** the hint names `opm registry login` without a host

#### Scenario: A refusal the cli receives as not found

- **WHEN** the registry answers 403 to the tag lookup, or its token endpoint answers 403
- **THEN** the build fails with exit code 2 and the hint says to pin a published build, until
  the library reads that answer as a refusal

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
