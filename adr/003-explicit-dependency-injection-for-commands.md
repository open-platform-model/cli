# ADR-003: Explicit Dependency Injection for Commands

## Status

Accepted

## Context

The CLI uses cobra for command registration and dispatch. Commands need access to shared runtime state: configuration, Kubernetes client settings, logging preferences, and registry URLs. Two common patterns exist for providing this state: package-level global variables or singletons, and explicit injection through constructor parameters.

As the command set grows, the relationship between commands and their dependencies becomes harder to follow if state is sourced from globals. New contributors need to be able to trace what a command depends on without reading across multiple packages. The physical layout of command packages also affects discoverability — the directory structure should reflect the user-facing CLI hierarchy so that finding the implementation of `opm release status` is straightforward.

Cluster-query operations (status, tree, events, delete, list) were historically grouped under the module authoring commands, but they operate on releases rather than on module source. This mismatch made the command tree harder to explain and placed operational commands in the wrong conceptual group.

## Decision

Pass all CLI configuration to command constructors explicitly via a `GlobalConfig` struct (see ADR-004), rather than reading from package-level variables or accessor functions. Package-level globals were rejected because they make dependencies invisible, complicate testing (global state must be reset between tests), and create implicit coupling between packages.

Organize `internal/cmd/` into sub-packages that mirror the cobra command tree: `internal/cmd/mod/` for module authoring commands, `internal/cmd/config/` for config sub-commands, and `internal/cmd/release/` for release sub-commands.

Register the release command group at root level (`opm release`, alias `rel`) as a first-class command group, not nested under module commands. Cluster-query commands (status, tree, events, delete, list) are migrated from the mod group to the release group. Deprecation aliases are kept in the mod package to preserve existing user workflows during the transition.

## Consequences

Commands become pure functions of their inputs, making them straightforward to test without manipulating global state. Dependencies are visible in constructor signatures, so coupling between a command and its runtime requirements is explicit rather than implicit.

The package layout mirrors the user-facing CLI hierarchy, so `internal/cmd/release/` corresponds directly to `opm release`. This reduces the navigational overhead for contributors adding or modifying commands.

Deprecation aliases provide a migration path rather than breaking existing user workflows when commands move groups.

Every command constructor must accept and thread the `GlobalConfig` struct, which adds a small amount of boilerplate. Promoting release to a first-class command group changes the CLI surface area, but this better reflects that releases — not just modules — are the primary operational unit.
