# opm CLI

The `opm` command-line tool. It authors, renders, publishes and deploys OPM modules and their
instances, and calls the library kernel for all structure. This file fixes one word for each
concept the CLI deals in. The OPM schema terms (`#Module`, `#Component`, `#ModuleInstance` and the
like) are defined here as the CLI uses them; the schema itself is `opmodel.dev/core@v2`.

## Language

### Authored things

**Module**:
An authored CUE package that evaluates to a `#Module`. It is the unit of `opm module build`, `vet`, `apply` and `publish`.
_Avoid_: app, chart, package (alone), service

**Module path**:
The full CUE module path of a module, major suffix included, as `ModulePath` in its identity package and `module:` in its `cue.mod/module.cue` declare it.
_Avoid_: module URL, OCI reference, module name

**Published module path**:
The major-free form of a module path (for example `opmodel.dev/modules/web_app`) that commands such as `opm instance init` take. The version selector, never the path, chooses the major.
_Avoid_: module URL, OCI reference, module name

**Component**:
A named unit inside a module that carries `#Resource` and `#Trait` demands. A skipped resource drops its whole component; a skipped trait leaves the component rendering.
_Avoid_: service, workload, microservice

**Resource**:
A `#Resource` that a component demands of the platform. Like a trait, it is a contract.
_Avoid_: object (that is a Kubernetes object), dependency

**Trait**:
A `#Trait` that a component demands of the platform, such as `backup`. Like a resource, it is a contract.
_Avoid_: feature, plugin, add-on

**Contract**:
A trait or resource definition that a catalog defines, named by a key such as `opmodel.dev/catalogs/opm/traits/backup@v1alpha1`. A component demands contracts; a platform fulfils them.
_Avoid_: interface, API

**Catalog**:
A published CUE module that defines contracts. `opmodel.dev/catalogs/` is the first-party namespace; other domains hold third-party catalogs. `opm catalog` publishes and checks catalogs.
_Avoid_: library, registry, repository

**Template**:
An official starter module published under `opmodel.dev/templates/` (`minimal`, `standard`, `advanced`) that `opm module init` fetches and re-identifies into a new tree.
_Avoid_: scaffold (that is the engine), boilerplate, example

**Identity package**:
The `identity/identity.cue` file of a module, which declares its module path and version. Publish reads the registry coordinates from it.
_Avoid_: manifest, metadata file

### Instances and their record

**ModuleInstance**:
A concrete, deployable `#ModuleInstance`: one module bound to a name, a namespace and values. Written "instance" in command names (`opm instance`).
_Avoid_: release, module release, deployment, install

**Instance package**:
A standalone CUE module whose package evaluates to a `#ModuleInstance`, written by `opm instance init`. `opm instance` commands accept only instance packages.
_Avoid_: release file, values package

**Synthetic instance**:
A `#ModuleInstance` that `opm module` commands build in memory from a module directory or a published module path, using `-f` values or the module's `debugValues`. No `instance.cue` exists for it.
_Avoid_: dry-run instance, temporary release

**Inventory**:
The record of which Kubernetes objects an instance owns, kept in the `status.inventory` block of its `ModuleInstance` custom resource. It lists current ownership only, not history.
_Avoid_: state, release manifest, ledger, inventory Secret (the retired store)

**Legacy inventory Secret**:
The Secret named `opm.<instance name>.<instance id>` that releases up to `v1.0.0-beta.10` read and migrated. Later releases do not read it.
_Avoid_: old state, release Secret

**Stale set**:
The entries of the previous inventory that no entry of the current render names as the same object, compared by group, kind, namespace and name only. `apply` prunes them unless `--no-prune` is set.
_Avoid_: orphans, leftovers, garbage

**Owner**:
The party that renders and applies an instance, recorded in `spec.owner` of its `ModuleInstance`: `cli` or `controller`. The CLI refuses to take over an instance the controller owns.
_Avoid_: manager, controller (alone)

**Adoption**:
Handing an existing object to an instance by annotating it `opmodel.dev/adopt=<instance uuid>`. Without it, an apply refuses to change an object another owner holds.
_Avoid_: takeover, import, claim

### Rendering and platforms

**Kernel**:
The library's render entry point. The CLI makes exactly one `Kernel.Render` call per render and holds no matching or transformer code.
_Avoid_: engine, renderer, builder

**Render-bearing command**:
A command that renders through the kernel: `module build`, `vet`, `apply` and `instance build`, `vet`, `diff`, `apply`. Each follows the platform source precedence.
_Avoid_: build command, deploy command

**Platform**:
A CUE module that subscribes to catalogs and that a render targets, or the cluster `Platform` custom resource named `cluster` that holds the same subscriptions.
_Avoid_: environment, cluster (the cluster is where a platform runs), target

**Platform source**:
Where a render gets its platform, in order: `--platform <dir>`, the cluster `Platform` custom resource, then the deps platform.
_Avoid_: platform provider, platform location

**Deps platform**:
A platform generated from the render's own committed dependency pins, cached under `~/.opm/cache/platforms/`. Skew cannot arise against it.
_Avoid_: default platform, local platform

**Skew policy**:
The setting `skewPolicy` (`warn` or `refuse`) that decides what a catalog version skew in a `--platform` directory does. The cluster `Platform` custom resource's `spec.skewPolicy` wins when it is the source.
_Avoid_: version mismatch mode, compatibility mode

**Unprovided contract**:
A provider-fulfilled contract that no enabled catalog on the platform provides. A render refuses it, or skips it under `--skip-unprovided`.
_Avoid_: missing provider, unresolved trait

### Publishing

**Publish gate**:
One check that `opm module publish` or `opm catalog publish` runs before a push. Every gate that fails adds a refusal; publish stops with exit code 2.
_Avoid_: validation, lint, precheck

**Core-compatible walk**:
The selection of a published module version: the newest release of the highest major whose core dependency major equals the CLI's core major.
_Avoid_: latest, newest version

**Local override**:
A `replaceWith` entry in `cue.mod/local-module.cue` that points a dependency at a local checkout. Publish refuses a module that has one, unless the override check is skipped.
_Avoid_: local replace, dev dependency

### Controller

**Controller pin**:
The pin file in `internal/controller/` that names the controller module version `opm controller install` installs. Only `hack/controller-pin` writes it.
_Avoid_: controller version, controller lock
