# Package Types (pkg/)

## Purpose

Defines the exported `pkg/` package structure external tools can import, and which shared types it does not declare because they live in the library: the Kubernetes object wrapper, the OPM label vocabulary, the kind-class weight table, and the inventory entry, stale set and digests.

## Requirements

### Requirement: Core types exported in pkg/
Shared domain types the CLI owns SHALL be exported under `pkg/` for reuse by external tools. The package structure SHALL be:
- `pkg/errors/` — CLI error types, sentinels and grouped CUE error helpers
- `pkg/loader/` — instance-file loading and local-replacement provenance readers

The Kubernetes object wrapper, the OPM label vocabulary and the kind-class weight table SHALL NOT be declared in `pkg/`: they are the library's `opm/k8s/object` (`Resource`, `Export`, `Weight`, `Sort`) and `opm/k8s/labels`, and the CLI SHALL use those packages wherever it wraps a rendered object, reads or writes an OPM label, or orders objects. There SHALL be no `pkg/core/` and no `pkg/resourceorder/` package (0012:D1, 0012:D5:R2).

The inventory entry, its identity comparison, the stale set, the inventory digest and the render digest SHALL NOT be declared in `pkg/`: they are the library's `opm/k8s/inventory` (`Entry`, `NewEntry`, `SameObject`, `StaleSet`, `Digest`, `RenderDigest`), and the CLI SHALL use that package wherever it builds or compares an inventory entry, computes a stale set or computes either digest. There SHALL be no `pkg/inventory/` package (0012:D1, 0012:D7).

Module and instance metadata types SHALL NOT be declared in `pkg/`: they are the library's `opm/schema.ModuleMetadata` and `opm/schema.InstanceMetadata` (aliased in `opm/module`), carried by every acquired artifact, and the CLI SHALL use those types wherever it holds decoded metadata. Where the CLI holds an acquired `*module.Module`, it SHALL read the module's identity fields (`name`, `modulePath`, `version`) from the module's decoded `Metadata` and SHALL NOT look them up in the module's `Package` value.

The CLI SHALL read the other kernel artifact fields the library exposes through the library's accessors, and SHALL NOT look them up in `Package` with schema paths:
- an instance's merged values through `Instance.Values()`;
- the metadata of the module an instance embeds through `Instance.ModuleMetadata()`, where a nil result is a module with no metadata and the CLI carries it as zero metadata;
- a module's `debugValues` through `Module.DebugValues()`, and its `#config` through `Module.ConfigSchema()`.

There SHALL be no `pkg/bundle/` package — bundle support is not implemented (0002:D15 removed the bundle path; 0002:D15 supersedes 0002:D7).

#### Scenario: External tool imports pkg/core
- **WHEN** an external Go module imports `github.com/open-platform-model/cli/pkg/core`
- **THEN** compilation fails — the package does not exist; the resource wrapper is the library's `opm/k8s/object.Resource` and the label constants are in `opm/k8s/labels`

#### Scenario: External tool imports pkg/resourceorder
- **WHEN** an external Go module imports `github.com/open-platform-model/cli/pkg/resourceorder`
- **THEN** compilation fails — the package does not exist; the weight table is the library's `opm/k8s/object.Weight`, with `Sort` and `Direction` beside it

#### Scenario: External tool imports pkg/inventory
- **WHEN** an external Go module imports `github.com/open-platform-model/cli/pkg/inventory`
- **THEN** compilation fails — the package does not exist; the inventory entry, stale set and digests are the library's `opm/k8s/inventory`

#### Scenario: External tool imports pkg/module
- **WHEN** code attempts to import `github.com/open-platform-model/cli/pkg/module`
- **THEN** compilation fails — the package does not exist; the metadata types are the library's `ModuleMetadata` and `InstanceMetadata`

#### Scenario: pkg/bundle does not exist
- **WHEN** code attempts to import `github.com/open-platform-model/cli/pkg/bundle`
- **THEN** compilation fails — the package does not exist

#### Scenario: Scaffold reads module identity from decoded metadata
- **WHEN** `opm module init` checks that a scaffolded or cloned tree derives its new identity, or reads the version a tree states to create its identity package
- **THEN** it reads the identity fields (`modulePath`, `version`) from the acquired module's `Metadata`, not from its `Package` value
- **AND** a clone source whose metadata does not derive the new identity still refuses with the existing "does not derive metadata" refusal

#### Scenario: Render reads the instance's values and module metadata through accessors
- **WHEN** a module or instance render builds its result
- **THEN** the result's values come from `Instance.Values()` and its module metadata from `Instance.ModuleMetadata()`
- **AND** the result's module name, module path and version equal the rendered module's own metadata

#### Scenario: Instance without module metadata
- **WHEN** `Instance.ModuleMetadata()` returns nil for a rendered instance
- **THEN** the render result carries zero module metadata, as it did when the CLI decoded the subtree itself

#### Scenario: debugValues read through the module
- **WHEN** `opm module build`, `opm module apply` or `opm module vet` runs without `-f`, or `opm instance init` walks its values ladder
- **THEN** the module's `debugValues` come from `Module.DebugValues()`
- **AND** a module without `debugValues` gives the same error, values source and exit code as before

### Requirement: No Component Go type
There SHALL be no `Component` struct type in `pkg/`. Component information for display purposes SHALL be derived from the `MatchPlan` result or CUE value iteration.

#### Scenario: No component package exists
- **WHEN** code attempts to import `pkg/component`
- **THEN** compilation fails — the package does not exist
