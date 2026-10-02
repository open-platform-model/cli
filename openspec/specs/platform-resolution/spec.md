# Capability: platform-resolution

## Purpose

Platform-source resolution by precedence with visible provenance (enhancement 0006 D11/D17/D21/D39): `--platform <dir>`, the cluster `Platform` CR, then a platform generated from the render's own dependency pins; there is no local default platform. Every source resolves to a platform module directory the kernel acquires with `AcquirePlatformFromDir`; the cluster CR is generated into one through the library's platform-module helper, the operator's own ingestion path (0019 D5/D6), from the effective registry the operator recorded on the CR's status (0015 D6).

## Requirements

### Requirement: Legacy Cluster CR Tolerance

Decoding a **cluster CR spec** SHALL tolerate legacy stored shapes permanently: a `filter` key SHALL be ignored without error, and a subscription lacking `version` SHALL decode with an empty version and fail only at platform-module generation, which the CLI SHALL wrap with a hint that the cluster Platform may predate the scalar-version shape and needs its spec re-applied. This tolerance applies to CR reads only — never to platform modules.

#### Scenario: Filter-shaped stored CR still decodes

- **WHEN** the cluster Platform CR carries a legacy `filter` block and no `version`
- **THEN** decode succeeds, and generation fails with the missing-version error plus the legacy-CR hint, before any registry access

### Requirement: Registry-resolved catalog subscription

The CLI SHALL be able to resolve the version a Platform subscription pins by querying the registry for the published versions of a major-suffixed catalog module path, rather than reading a version written by hand. Resolution SHALL be scoped to the major carried by that path, so a build from a different major can never be selected.

Two selection modes exist, and the caller chooses between them explicitly:

- **Release (default)**: the highest published version whose SemVer prerelease part is empty.
- **Prerelease (opt-in)**: the highest published version whose prerelease part begins with a non-numeric identifier.

Development builds SHALL NEVER be selected in either mode. A development build is identified by a prerelease part whose first identifier is numeric, which is the form catalog branch-publish pipelines produce so SemVer ranks those builds below every named prerelease. Versions the registry returns that are not valid SemVer SHALL be ignored rather than treated as candidates.

When no published version satisfies the active mode, resolution SHALL refuse rather than substitute a version from the other mode. The refusal SHALL name the module path queried, the registry consulted, the highest version it did see, and the action that would select it. Resolution SHALL NOT infer maturity, widen the major, or fall back to a hand-pinned literal.

#### Scenario: Stable release selected by default

- **WHEN** the catalog major has published `2.0.0-alpha.3`, `2.0.0`, and `2.1.0`
- **THEN** the default mode SHALL resolve `2.1.0`

#### Scenario: Prerelease-only history refuses under the default

- **WHEN** the catalog major has published only `2.0.0-alpha.1`, `2.0.0-alpha.2`, and `2.0.0-alpha.3`
- **THEN** the default mode SHALL refuse
- **AND** the refusal SHALL name the highest prerelease it saw and the action that selects a prerelease

#### Scenario: Prerelease mode ignores development builds

- **WHEN** the catalog major has published `2.0.0-alpha.3` and a later-pushed `2.0.0-0.dev.1754899200.g9ea5927`
- **THEN** the prerelease mode SHALL resolve `2.0.0-alpha.3`

#### Scenario: Development builds are never the answer

- **WHEN** every published version of the catalog major is a development build
- **THEN** both modes SHALL refuse, and no development build SHALL be offered as a candidate

#### Scenario: Empty history refuses

- **WHEN** the catalog major has no published versions
- **THEN** resolution SHALL refuse naming the module path and the registry

#### Scenario: Registry failure is not a refusal

- **WHEN** the version listing fails in transport
- **THEN** the failure SHALL be reported as a connectivity failure naming the lookup and the registry, distinct from a refusal, because no version was ever judged

### Requirement: Acquisition mirrors the operator

A cluster `Platform` CR SHALL be turned into a platform module exactly as the operator does it: the CR's `spec.type` and its registry entries (path, `version`, `enable`) feed the library's platform-module generator, the dependency closure is derived from the pinned modules' published module files through the CLI's configured registry, core is pinned at the library's verified release, and the module path is `opmodel.dev/platforms/cluster@v0`. The registry entries SHALL be the effective registry the operator recorded on `status.registry` when present (each entry's catalog, version and enabled flag, which is the operator's own resolution of subscriptions and active claims), and the `spec.registry` subscriptions (with `enable` defaulting to true) otherwise. Every platform source SHALL then be acquired through the kernel's directory acquisition, so a bad pin, a key-to-import mismatch or an unpublished build fails at acquisition naming the entry or dependency, identically for the flag, CR and local sources. The CLI MUST NOT synthesize a platform value from typed inputs and MUST NOT persist any built platform value.

The generated module SHALL live under the OPM home cache at `cache/platforms/<content-hash>/`, where the hash covers the generated files' bytes, so an unchanged CR maps to the same directory across invocations, generation is idempotent, and two concurrent invocations converge on identical content. The CLI MUST NOT publish the generated module, write it to the cluster, or treat it as user-editable; it is derived state and may be deleted at any time.

#### Scenario: CR generates the operator's module

- **WHEN** the cluster CR subscribes `opmodel.dev/catalogs/opm@v4` at `4.0.1`, its status records that one entry, and the CLI resolves it
- **THEN** the generated `cue.mod/module.cue` pins that catalog, core at the library's verified release and the catalog's transitive dependencies, and `platform.cue` carries one importing `#registry` entry for it, byte-identical to what the operator generates for the same CR and library release

#### Scenario: An active claim's catalog is generated

- **WHEN** the cluster CR's status registry records an entry sourced from a registration for `opmodel.dev/catalogs/k8up@v1` at `1.2.0` beside the subscribed catalogs
- **THEN** the generated `cue.mod` pins that catalog at `1.2.0` and `platform.cue` carries an enabled importing entry for it, and a module demanding a contract the provider fulfils renders on the laptop as it does in the cluster

#### Scenario: Unchanged CR reuses the cache

- **WHEN** two invocations resolve the same cluster CR
- **THEN** both acquire the same `cache/platforms/<hash>/` directory and the second performs no rewrite

#### Scenario: Unpublished pin fails at acquisition

- **WHEN** the CR names a catalog version that is not published
- **THEN** resolution fails naming the catalog path and version, before any instance is rendered

#### Scenario: Same failure surface for every source

- **WHEN** a `--platform` module, a generated CR module or the local default fails to build
- **THEN** the error names the failing dependency or `#registry` entry the same way regardless of source, and the reported provenance still names the source

### Requirement: The cluster arm reads the effective registry

When the cluster Platform's status carries a resolved registry, resolution SHALL generate from it and SHALL report the recorded package identity in the provenance. When it carries none, resolution SHALL generate from the spec and SHALL say so. Resolution SHALL warn, never silently substitute, in two cases: when `status.observedGeneration` is behind `metadata.generation`, naming both generations, because the effective package predates the spec being edited; and when the Platform's `Ready` condition is `False`, naming its reason, because the effective package is the last one the operator accepted. Neither warning changes which package is generated. The write-if-absent seeding of a missing Platform is unchanged: it happens only when no CR exists, and it seeds the spec the render consumed.

#### Scenario: Effective registry wins over the spec

- **WHEN** the CR's spec subscribes one catalog and its status registry records that catalog plus a registration-sourced provider catalog
- **THEN** the generated module carries both entries and the provenance names the effective registry and the package identity

#### Scenario: No recorded registry falls back to the spec

- **WHEN** the CR's status carries no registry
- **THEN** the generated module carries the spec's subscriptions and the provenance says the spec was used because no operator generation is recorded

#### Scenario: A stale status warns

- **WHEN** `metadata.generation` is 5 and `status.observedGeneration` is 4
- **THEN** resolution warns that the effective package describes generation 4 while the spec is at 5, and generates from the effective registry

#### Scenario: A refused Platform renders against the last good package

- **WHEN** the Platform's `Ready` condition is `False` with reason `OverSubscribedContracts`
- **THEN** resolution warns naming that reason, generates from the recorded registry, and the render proceeds against it

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
- The same generator serves the deps fallback of every other render (see "Renders fall back to their own deps").

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

- **WHEN** `opm module apply` or `opm instance build` runs with no `--platform` against a cluster with a readable `Platform` CR
- **THEN** the render SHALL use the platform generated from the cluster `Platform`
- **AND** no platform SHALL be generated from the module's or the instance's deps

#### Scenario: A local catalog checkout is rendered

- **WHEN** the module's `cue.mod/local-module.cue` replaces `opmodel.dev/catalogs/opm@v4` with `../catalog_opm/src` and `opm module build` runs with no `--platform`
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

### Requirement: Platform source precedence by command

The CLI SHALL resolve the platform for every render by precedence, and no precedence SHALL include a platform module in the OPM home directory.

- `opm module build` and `opm module vet` resolve `--platform <dir>` (highest) > the module-deps platform (see "Module commands render against the module's deps"), and SHALL NOT read the cluster.
- `opm instance build`, `opm instance vet`, `opm instance diff`, `opm instance apply` and `opm module apply` resolve `--platform <dir>` (highest) > the cluster `Platform` CR named `cluster` > a platform generated from the render's own dependency pins (see "Renders fall back to their own deps").
- `opm platform check` resolves its `[dir]` argument (highest) > `--platform <dir>` > the cluster `Platform` CR, and SHALL refuse when none is available, naming all three.

Every source resolves to a platform module directory the kernel acquires; the CR source is generated into one first from the effective registry the operator recorded on the CR's status, or from the CR's spec when no operator has recorded one (see "Acquisition mirrors the operator" and "The cluster arm reads the effective registry"), and the deps sources are generated into one from committed dependency pins. Every command that renders SHALL report which platform source it resolved, the directory it acquired and, for the CR source, whether the effective registry or the spec was used and the package identity the operator recorded. A `--platform` argument that is a file, or a directory holding no platform module, SHALL fail naming the expected shape (a directory with `cue.mod/module.cue` and a `#Platform` package). The `--provider` flag SHALL NOT exist (superseded by `--platform`, 0006:D21).

#### Scenario: Flag wins

- **WHEN** `opm instance apply --platform ./my-platform/` runs against a cluster that has a `Platform` CR
- **THEN** the render SHALL use the platform module at `./my-platform/`
- **AND** the output SHALL report the platform source as the flag-provided directory

#### Scenario: Cluster CR used when no flag

- **WHEN** `opm instance apply` or `opm instance build` runs with no `--platform` and a kubeconfig context whose cluster has a readable `Platform` CR
- **THEN** the render SHALL use a platform module generated from the cluster CR's effective registry, or from its spec when the status carries none
- **AND** the output SHALL report the platform source as the cluster CR and which of the two it generated from

#### Scenario: Absent Platform falls back to the deps

- **WHEN** `opm instance apply` runs with no `--platform` and the cluster `Platform` CR is absent or unreadable (RBAC denied)
- **THEN** the render SHALL use a platform generated from the instance package's own dependency pins
- **AND** a warning SHALL state that the cluster Platform was not used and why
- **AND** the apply SHALL NOT create a `Platform`

#### Scenario: Module commands never read the cluster

- **WHEN** `opm module build` or `opm module vet` runs
- **THEN** the CLI SHALL NOT attempt any cluster read for platform resolution
- **AND** the platform SHALL come from `--platform` or the module-deps platform only

#### Scenario: A platform file is refused

- **WHEN** `--platform ./platform.cue` names a file
- **THEN** resolution fails before any render, naming the expected module-directory shape

#### Scenario: platform check with no source refuses

- **WHEN** `opm platform check` runs with no argument, no `--platform` and no readable cluster `Platform`
- **THEN** the command SHALL fail naming the argument, the flag and the cluster as the sources it looked for
- **AND** it SHALL NOT read any directory under the OPM home

### Requirement: Only operator install seeds the cluster Platform

`opm operator install` SHALL seed the singleton `cluster` Platform with a plain create (field manager `opm-cli`), treating `AlreadyExists` as success-noop (0006:D22), subscribing to the catalog build it resolved from the registry. The CLI MUST NOT use server-side apply or update for this write and MUST NOT overwrite an existing Platform. A forbidden create SHALL degrade to a warning. The reported provenance SHALL name the catalog coordinate and version the install resolved and SHALL NOT claim any other source.

No render-bearing command SHALL create, update or seed a `Platform`: `opm instance apply` and `opm module apply` SHALL leave a cluster without a `Platform` without one.

#### Scenario: Install seeds an absent Platform

- **WHEN** `opm operator install` completes against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` SHALL be created subscribing to the resolved catalog build

#### Scenario: Concurrent create tolerated

- **WHEN** the create returns `AlreadyExists`
- **THEN** the CLI SHALL treat it as success and SHALL NOT modify the existing Platform

#### Scenario: RBAC-denied create degrades

- **WHEN** the create is forbidden
- **THEN** the CLI SHALL warn and the install SHALL still complete

#### Scenario: Install reports its own provenance

- **WHEN** `opm operator install` seeds the Platform from a registry-resolved catalog version
- **THEN** the reported provenance SHALL name the catalog module path and the resolved version

#### Scenario: Apply never seeds a Platform

- **WHEN** `opm instance apply` succeeds against a cluster with no `Platform` CR
- **THEN** no `Platform` SHALL exist in the cluster afterwards
- **AND** the output SHALL NOT mention seeding

### Requirement: Renders fall back to their own deps

When `opm instance build`, `opm instance vet`, `opm instance diff`, `opm instance apply` or `opm module apply` runs without `--platform` and resolution does not use a cluster `Platform`, the CLI SHALL generate a platform from the render's own committed dependency pins with the generator of "Module commands render against the module's deps", under the same rules for catalog entries, the core floor, the closure, carried replacements, the cache and the provenance line.

- For an instance command, the pins SHALL be read from the instance package's own `cue.mod/module.cue`, and the replacements from the instance package's `cue.mod/local-module.cue`.
- For `opm module apply`, the pins SHALL be read from the module's acquired source, exactly as `opm module build` reads them.
- The skew policy SHALL NOT be applied to this source.
- An instance package whose `cue.mod/module.cue` lists no `opmodel.dev/catalogs/*` dependency SHALL render against an empty registry, and a component SHALL be refused as unmatched with a hint naming `cue mod tidy` in the instance package and `--platform <dir>`.

#### Scenario: Instance build with no cluster renders against its pins

- **WHEN** `opm instance build ./hello` runs with no `--platform`, no kubeconfig context, and the package's `cue.mod/module.cue` pins `opmodel.dev/catalogs/opm@v4` at `v4.4.0`
- **THEN** the render SHALL use a generated platform whose `#registry` has one enabled entry for `opmodel.dev/catalogs/opm@v4` at `4.4.0`
- **AND** the provenance line SHALL name the instance package's deps as the source

#### Scenario: module apply on a cluster without a Platform

- **WHEN** `opm module apply ./mymodule` runs against a cluster with no `Platform` CR
- **THEN** the render SHALL use the platform generated from the module's deps, as `opm module build` would
- **AND** a warning SHALL state that the cluster Platform was not used and why

### Requirement: Instance build and vet probe the cluster without depending on it

`opm instance build` and `opm instance vet` SHALL try the cluster `Platform` before the deps, and SHALL NOT fail because of the cluster.

- With `--offline`, the CLI SHALL NOT resolve a kubeconfig or contact any API server, and SHALL resolve `--platform`, else the deps.
- With no resolvable kubeconfig context, the CLI SHALL resolve the deps and SHALL NOT warn.
- When the `Platform` CR is absent (including a cluster without the `Platform` resource type) or the read is forbidden, the CLI SHALL warn that the cluster Platform was not used and why, and resolve the deps.
- When the API server does not answer within a bounded wait (at most 10 seconds for the whole lookup), or the read fails for any other reason, the CLI SHALL warn naming the error and resolve the deps.
- `opm instance diff`, `opm instance apply` and `opm module apply` keep failing on an unreachable cluster: they need the cluster to do their work.

#### Scenario: offline never contacts the cluster

- **WHEN** `opm instance build ./hello --offline` runs with a kubeconfig context pointing at an unreachable server
- **THEN** the render SHALL complete against the deps with no delay and no cluster warning

#### Scenario: Unreachable cluster degrades to the deps

- **WHEN** `opm instance vet ./hello` runs and the kubeconfig context's server does not answer
- **THEN** within the bounded wait the command SHALL warn that the cluster could not be reached and render against the deps
- **AND** the exit code SHALL be that of the render, not a connection error

#### Scenario: Kubeconfig flags select the cluster

- **WHEN** `opm instance build ./hello --context staging` runs
- **THEN** the cluster `Platform` SHALL be read from the `staging` context
