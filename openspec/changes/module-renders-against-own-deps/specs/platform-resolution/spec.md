## MODIFIED Requirements

### Requirement: Platform source precedence

The CLI SHALL resolve the platform for every render by precedence. `opm module build` and `opm module vet` resolve `--platform <dir>` (highest) > the module-deps platform (see "Module commands render against the module's deps") and SHALL read neither the cluster nor the local default platform module. Every other render resolves `--platform <dir>` (highest, an explicit local platform module directory) > cluster `Platform` CR (cluster-facing commands only) > local default platform module `~/.opm/platform/`. Every source resolves to a platform module directory the kernel acquires; the CR source is generated into one first from the effective registry the operator recorded on the CR's status, or from the CR's spec when no operator has recorded one (see "Acquisition mirrors the operator" and "The cluster arm reads the effective registry"), and the module-deps source is generated into one from the module's committed dependency pins. Every command that renders SHALL report which platform source it resolved, the directory it acquired and, for the CR source, whether the effective registry or the spec was used and the package identity the operator recorded. A `--platform` argument that is a file, or a directory holding no platform module, SHALL fail naming the expected shape (a directory with `cue.mod/module.cue` and a `#Platform` package) and pointing at `opm config init`. The `--provider` flag SHALL NOT exist (superseded by `--platform`, 0006 D21).

#### Scenario: Flag wins

- **WHEN** `opm instance apply --platform ./my-platform/` runs against a cluster that has a `Platform` CR
- **THEN** the render SHALL use the platform module at `./my-platform/`
- **AND** the output SHALL report the platform source as the flag-provided directory

#### Scenario: Cluster CR used when no flag

- **WHEN** `opm instance apply` runs with no `--platform` against a cluster with a readable `Platform` CR
- **THEN** the render SHALL use a platform module generated from the cluster CR's effective registry, or from its spec when the status carries none
- **AND** the output SHALL report the platform source as the cluster CR and which of the two it generated from

#### Scenario: Fallback to local default warns

- **WHEN** `opm instance apply` runs with no `--platform` and the cluster `Platform` CR is absent or unreadable (RBAC denied)
- **THEN** the render SHALL use the local default platform module `~/.opm/platform/`
- **AND** a warning SHALL state that the cluster Platform was not used and why

#### Scenario: Offline commands never read the cluster

- **WHEN** `opm instance build`, `opm instance vet`, `opm module build` or `opm module vet` runs
- **THEN** the CLI SHALL NOT attempt any cluster read for platform resolution
- **AND** the platform of `opm instance build` and `opm instance vet` SHALL come from `--platform` or the local default only
- **AND** the platform of `opm module build` and `opm module vet` SHALL come from `--platform` or the module-deps platform only

#### Scenario: A platform file is refused

- **WHEN** `--platform ./platform.cue` names a file
- **THEN** resolution fails before any render, naming the expected module-directory shape and the `opm config init` migration

## ADDED Requirements

### Requirement: Module commands render against the module's deps

When `opm module build` or `opm module vet` runs without `--platform`, the CLI SHALL generate a platform module from the module's committed `cue.mod/module.cue` and render against it. Source: cli issue 229.

- The generated platform SHALL carry one enabled `#registry` entry per dependency whose major-qualified path starts with `opmodel.dev/catalogs/`, at the version the module pins, and SHALL pin core at the higher of the module's pin and the core release the kernel was verified against, the floor the module generated from the cluster CR also carries.
- Its dependency list SHALL be the minimum-version-selection closure of those pins over the published module files, so a catalog's own dependencies that the module's tidied list omits are present.
- Each `replaceWith` in the module's `cue.mod/local-module.cue` whose path the generated platform pins SHALL be carried into the generated platform's `cue.mod/local-module.cue`, a relative directory resolved against the module root. The closure SHALL read a directory-replaced module's requirements from that directory's `cue.mod/module.cue` instead of the registry.
- The deps SHALL be read from the module's acquired source, so a module acquired from a directory and a published module acquired from the registry follow one rule. A published module has no local module context, so nothing is carried for it.
- The generated module SHALL be cached under `cache/platforms/` beside the config file, named by the hash of its content, exactly as the module generated from the cluster CR is.
- The provenance line SHALL name the source as the module's deps and list each catalog path and version. The skew policy SHALL NOT be named or applied: every pinned path is at least the version the module requires.
- A module with no `opmodel.dev/catalogs/*` dependency SHALL render against a generated platform with an empty registry: a module without components passes with zero objects, and a component is refused as unmatched with a hint naming `opm module tidy` and `--platform <dir>`.
- A pin that is not published SHALL fail at generation naming the module path and version.
- When a render against this source is refused for unresolved demands, the output SHALL add a hint that a provider-fulfilled contract comes from a platform, naming `--platform <dir>`.
- `opm module apply` and every `opm instance` command SHALL NOT use this source.

#### Scenario: Catalog pins come from the module

- **WHEN** `opm module build` runs with no `--platform` on a module whose `cue.mod/module.cue` pins `opmodel.dev/catalogs/opm@v4` at `v4.4.0` and `opmodel.dev/core@v2` at `v2.0.0-alpha.10`
- **AND** the kernel was verified against core `v2.0.0-alpha.10`
- **THEN** the render SHALL use a generated platform whose `#registry` has one enabled entry for `opmodel.dev/catalogs/opm@v4` at `4.4.0` and whose dependency list pins core at `v2.0.0-alpha.10`
- **AND** the output SHALL name the source as the module's deps and list `opmodel.dev/catalogs/opm@v4 v4.4.0`
- **AND** no version-skew warning SHALL be printed

#### Scenario: A catalog's own dependencies are closed over

- **WHEN** the module pins `opmodel.dev/catalogs/opm@v4` but not `cue.dev/x/k8s.io@v0`, which that catalog's own `cue.mod/module.cue` requires
- **THEN** the generated platform's dependency list SHALL pin `cue.dev/x/k8s.io@v0` at the version the catalog requires
- **AND** the render SHALL succeed

#### Scenario: No local default platform is needed

- **WHEN** `opm module build` runs with no `--platform` and no `~/.opm/platform/` exists
- **THEN** the render SHALL succeed against the generated platform
- **AND** no cluster read SHALL be attempted

#### Scenario: The flag overrides the deps

- **WHEN** `opm module build --platform ./pulled/` runs
- **THEN** the render SHALL use the platform module at `./pulled/`
- **AND** no platform SHALL be generated from the module's deps
- **AND** the configured skew policy SHALL apply as for any `--platform` directory

#### Scenario: Deploying commands keep the platform

- **WHEN** `opm module apply` or `opm instance build` runs with no `--platform`
- **THEN** the render SHALL resolve its platform exactly as before this change and SHALL NOT generate one from the module's deps

#### Scenario: A local catalog checkout is rendered

- **WHEN** the module's `cue.mod/local-module.cue` replaces `opmodel.dev/catalogs/opm@v4` with `../catalog_opm/opm` and `opm module build` runs with no `--platform`
- **THEN** the generated platform SHALL serve `opmodel.dev/catalogs/opm@v4` from the checkout's absolute directory
- **AND** the rendered objects SHALL reflect the checkout's transformer bytes
- **AND** the closure SHALL read the checkout's `cue.mod/module.cue`, so the render succeeds even when the pinned catalog version is not published

#### Scenario: A published module renders against its own deps

- **WHEN** `opm module build` renders a module acquired from the registry, with no `--platform`
- **THEN** the generated platform SHALL be derived from the `cue.mod/module.cue` committed in the published artifact
- **AND** no replacement SHALL be carried

#### Scenario: A module without a catalog dependency renders against an empty registry

- **WHEN** `opm module build` runs with no `--platform` on a module whose `cue.mod/module.cue` lists no `opmodel.dev/catalogs/*` dependency
- **THEN** a module without components SHALL render zero objects and succeed
- **AND** a module with a component SHALL be refused as unmatched with exit code 2, and the output SHALL add a hint naming `opm module tidy` and `--platform <dir>`

#### Scenario: Core never falls below the kernel's verified release

- **WHEN** the module pins `opmodel.dev/core@v2` at a release older than the one the kernel was verified against
- **THEN** the generated platform SHALL pin core at the kernel's verified release
- **AND** no version-skew warning SHALL be printed

#### Scenario: A provider-fulfilled demand names the platform flag

- **WHEN** a component attaches a trait whose contract is provider-fulfilled and required, and `opm module build` runs with no `--platform`
- **THEN** the render SHALL be refused with the kernel's unresolved-demand verdict and exit code 2
- **AND** the output SHALL add a hint that provider-fulfilled contracts come from a platform and name `--platform <dir>`

#### Scenario: Unchanged deps reuse the cache

- **WHEN** `opm module build` runs twice on a module whose pins and replacements did not change
- **THEN** both runs SHALL render against the same generated directory under `cache/platforms/`, and the second run SHALL NOT rewrite it
