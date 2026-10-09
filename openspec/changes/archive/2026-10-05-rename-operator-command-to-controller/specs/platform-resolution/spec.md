## RENAMED Requirements

- FROM: `### Requirement: Only operator install seeds the cluster Platform`
- TO: `### Requirement: Only controller install seeds the cluster Platform`

- FROM: `### Requirement: Operator install renders against the operator module's own pins`
- TO: `### Requirement: Controller install renders against the controller module's own pins`

## REMOVED Requirements

### Requirement: Acquisition mirrors the operator
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## MODIFIED Requirements

### Requirement: The cluster arm reads the effective registry

When the cluster Platform's status carries a resolved registry, resolution SHALL generate from it and SHALL report the recorded package identity in the provenance. When it carries none, resolution SHALL generate from the spec and SHALL say so. Resolution SHALL warn, never silently substitute, in two cases: when `status.observedGeneration` is behind `metadata.generation`, naming both generations, because the effective package predates the spec being edited; and when the Platform's `Ready` condition is `False`, naming its reason, because the effective package is the last one the controller accepted. Neither warning changes which package is generated. The write-if-absent seeding of a missing Platform is unchanged: it happens only when no CR exists, and it seeds the spec the render consumed.

#### Scenario: Effective registry wins over the spec

- **WHEN** the CR's spec subscribes one catalog and its status registry records that catalog plus a registration-sourced provider catalog
- **THEN** the generated module carries both entries and the provenance names the effective registry and the package identity

#### Scenario: No recorded registry falls back to the spec

- **WHEN** the CR's status carries no registry
- **THEN** the generated module carries the spec's subscriptions and the provenance says the spec was used because no controller generation is recorded

#### Scenario: A stale status warns

- **WHEN** `metadata.generation` is 5 and `status.observedGeneration` is 4
- **THEN** resolution warns that the effective package describes generation 4 while the spec is at 5, and generates from the effective registry

#### Scenario: A refused Platform renders against the last good package

- **WHEN** the Platform's `Ready` condition is `False` with reason `OverSubscribedContracts`
- **THEN** resolution warns naming that reason, generates from the recorded registry, and the render proceeds against it

### Requirement: Platform source precedence by command

The CLI SHALL resolve the platform for every render by precedence, and no precedence SHALL include a platform module in the OPM home directory.

- `opm module build` and `opm module vet` resolve `--platform <dir>` (highest) > the module-deps platform (see "Module commands render against the module's deps"), and SHALL NOT read the cluster.
- `opm instance build`, `opm instance vet`, `opm instance diff`, `opm instance apply` and `opm module apply` resolve `--platform <dir>` (highest) > the cluster `Platform` CR named `cluster` > a platform generated from the render's own dependency pins (see "Renders fall back to their own deps").
- `opm platform check` resolves its `[dir]` argument (highest) > `--platform <dir>` > the cluster `Platform` CR, and SHALL refuse when none is available, naming all three.

Every source resolves to a platform module directory the kernel acquires; the CR source is generated into one first from the effective registry the controller recorded on the CR's status, or from the CR's spec when no controller has recorded one (see "Acquisition mirrors the controller" and "The cluster arm reads the effective registry"), and the deps sources are generated into one from committed dependency pins. Every command that renders SHALL report which platform source it resolved, the directory it acquired and, for the CR source, whether the effective registry or the spec was used and the package identity the controller recorded. A `--platform` argument that is a file, or a directory holding no platform module, SHALL fail naming the expected shape (a directory with `cue.mod/module.cue` and a `#Platform` package). The `--provider` flag SHALL NOT exist (superseded by `--platform`, 0006:D21).

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

### Requirement: Only controller install seeds the cluster Platform

`opm controller install` SHALL seed the singleton `cluster` Platform with a plain create (field manager `opm-cli`), treating `AlreadyExists` as success-noop (0006:D22), subscribing to the catalog build it resolved from the registry. The CLI MUST NOT use server-side apply or update for this write and MUST NOT overwrite an existing Platform. A forbidden create SHALL degrade to a warning. The reported provenance SHALL name the catalog coordinate and version the install resolved and SHALL NOT claim any other source.

No render-bearing command SHALL create, update or seed a `Platform`: `opm instance apply` and `opm module apply` SHALL leave a cluster without a `Platform` without one.

#### Scenario: Install seeds an absent Platform

- **WHEN** `opm controller install` completes against a cluster with no `Platform` CR
- **THEN** a `Platform` named `cluster` SHALL be created subscribing to the resolved catalog build

#### Scenario: Concurrent create tolerated

- **WHEN** the create returns `AlreadyExists`
- **THEN** the CLI SHALL treat it as success and SHALL NOT modify the existing Platform

#### Scenario: RBAC-denied create degrades

- **WHEN** the create is forbidden
- **THEN** the CLI SHALL warn and the install SHALL still complete

#### Scenario: Install reports its own provenance

- **WHEN** `opm controller install` seeds the Platform from a registry-resolved catalog version
- **THEN** the reported provenance SHALL name the catalog module path and the resolved version

#### Scenario: Apply never seeds a Platform

- **WHEN** `opm instance apply` succeeds against a cluster with no `Platform` CR
- **THEN** no `Platform` SHALL exist in the cluster afterwards
- **AND** the output SHALL NOT mention seeding

### Requirement: Controller install renders against the controller module's own pins

`opm controller install`, the CRDs-only form included, SHALL render the controller module against a platform generated from the module's own committed dependency pins, and SHALL NOT read the cluster `Platform` or accept a `--platform` directory for that render. Every other command keeps its precedence under "Platform source precedence by command". The same module version and values SHALL render the same objects whether or not the cluster holds a Platform and whatever it subscribes to, and a Platform that is missing, not Ready or unreadable SHALL NOT refuse the render. The provenance line SHALL name the module's deps as the source. This narrows 0006:D11's precedence for the controller's own instance only, so repairing the controller never depends on the Platform it serves. Source: 0021:D11:R10.

#### Scenario: Seeded Platform on another catalog release

- **WHEN** the cluster Platform subscribes to a catalog release other than the one the controller module pins, and install re-runs with the same module version and values
- **THEN** the render is the one the module's own pins produce, and no live object changes

#### Scenario: Stalled Platform does not block repair

- **WHEN** the cluster Platform is `Stalled` and install runs
- **THEN** the render succeeds from the module's pins and install proceeds to its checks

## ADDED Requirements

### Requirement: Acquisition mirrors the controller

A cluster `Platform` CR SHALL be turned into a platform module exactly as the controller does it: the CR's `spec.type` and its registry entries (path, `version`, `enable`) feed the library's platform-module generator, the dependency closure is derived from the pinned modules' published module files through the CLI's configured registry, core is pinned at the library's verified release, and the module path is `opmodel.dev/platforms/cluster@v0`. The registry entries SHALL be the effective registry the controller recorded on `status.registry` when present (each entry's catalog, version and enabled flag, which is the controller's own resolution of subscriptions and active claims), and the `spec.registry` subscriptions (with `enable` defaulting to true) otherwise. Every platform source SHALL then be acquired through the kernel's directory acquisition, so a bad pin, a key-to-import mismatch or an unpublished build fails at acquisition naming the entry or dependency, identically for the flag, CR and local sources. The CLI MUST NOT synthesize a platform value from typed inputs and MUST NOT persist any built platform value.

The generated module SHALL live under the OPM home cache at `cache/platforms/<content-hash>/`, where the hash covers the generated files' bytes, so an unchanged CR maps to the same directory across invocations, generation is idempotent, and two concurrent invocations converge on identical content. The CLI MUST NOT publish the generated module, write it to the cluster, or treat it as user-editable; it is derived state and may be deleted at any time.

#### Scenario: CR generates the controller's module

- **WHEN** the cluster CR subscribes `opmodel.dev/catalogs/opm@v4` at `4.0.1`, its status records that one entry, and the CLI resolves it
- **THEN** the generated `cue.mod/module.cue` pins that catalog, core at the library's verified release and the catalog's transitive dependencies, and `platform.cue` carries one importing `#registry` entry for it, byte-identical to what the controller generates for the same CR and library release

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
