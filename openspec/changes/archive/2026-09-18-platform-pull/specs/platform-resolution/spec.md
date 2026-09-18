## MODIFIED Requirements

### Requirement: Platform source precedence

The CLI SHALL resolve the platform for every render by precedence: `--platform <dir>` (highest, an explicit local platform module directory) > cluster `Platform` CR (cluster-facing commands only) > local default platform module `~/.opm/platform/`. Every source resolves to a platform module directory the kernel acquires; the CR source is generated into one first from the effective registry the operator recorded on the CR's status, or from the CR's spec when no operator has recorded one (see "Acquisition mirrors the operator" and "The cluster arm reads the effective registry"). Every command that renders SHALL report which platform source it resolved, the directory it acquired and, for the CR source, whether the effective registry or the spec was used and the package identity the operator recorded. A `--platform` argument that is a file, or a directory holding no platform module, SHALL fail naming the expected shape (a directory with `cue.mod/module.cue` and a `#Platform` package) and pointing at `opm config init`. The `--provider` flag SHALL NOT exist (superseded by `--platform`, 0006 D21).

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

- **WHEN** `opm instance build` or `opm module build` runs
- **THEN** the CLI SHALL NOT attempt any cluster read for platform resolution
- **AND** the platform SHALL come from `--platform` or the local default only

#### Scenario: A platform file is refused

- **WHEN** `--platform ./platform.cue` names a file
- **THEN** resolution fails before any render, naming the expected module-directory shape and the `opm config init` migration

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

## ADDED Requirements

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
