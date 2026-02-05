# Delta for CLI Distribution

## User Stories

### Story 1: Publishing a Module (P1)

A Module Author wants to publish their module to an OCI registry so that others can consume it. They need a simple command that validates the module and pushes it to their registry of choice.

**Why this priority**: Fundamental action for sharing content. Without publishing, there is no distribution.

**Independent Test**: Run a local registry (e.g., `zot`), publish a module, verify the artifact exists.

---

### Story 2: Consuming a Module (P2)

A Module Author wants to use a published module (e.g., `registry.example.com/simple-blog@v1`) in their own project. They need to download the dependency and have `module.cue` updated automatically.

**Why this priority**: Primary way users build upon the platform. Manual editing of `module.cue` is error-prone.

**Independent Test**: Publish a "provider" module, then in a separate "consumer" project, run `opm mod get` and verify the dependency is usable.

---

### Story 3: Updating Dependencies (P3)

A Module Author wants to check if newer versions of their dependencies are available and upgrade them easily.

**Why this priority**: Keeps ecosystem healthy. Reduces friction of strict versioning by automating the upgrade path.

**Independent Test**: Publish `v1.0.0` and `v1.1.0` of a module, consume `v1.0.0`, then run `opm mod update` to see the prompt.

---

### Story 4: Platform Composition (P2)

A Platform Operator wants to consume generic upstream modules and extend them with organizational specifics (e.g., mandatory Policy, logging sidecar, default labels) without forking upstream code.

**Why this priority**: Enables "Platform as Product" model. Validates that distribution preserves CUE's unification.

**Independent Test**: Create module that imports upstream, adds Policy, run `opm mod build`. Verify output contains combined configuration.

---

## ADDED Requirements

### Requirement: Publish Command

The CLI MUST provide `opm mod publish` to publish modules to OCI registries.

#### Scenario: Publish to registry
- GIVEN a valid module
- WHEN the user runs `opm mod publish registry.example.com/my-module --version v1.0.0`
- THEN the module is validated (vet), packed, and pushed to the registry

#### Scenario: Validation before publish
- GIVEN a module with validation errors
- WHEN the user runs `opm mod publish`
- THEN the process fails with exit code 2 and clear error messages

#### Scenario: Overwrite existing version
- GIVEN a version already exists in the registry
- WHEN the user runs `opm mod publish --version v1.0.0 --force`
- THEN the existing version is overwritten

---

### Requirement: Get Command

The CLI MUST provide `opm mod get` to download module dependencies.

#### Scenario: Add dependency
- GIVEN an initialized project
- WHEN the user runs `opm mod get registry.example.com/dep@v1.2.3`
- THEN the module is downloaded to CUE cache and `module.cue` deps updated

#### Scenario: Reject @latest
- GIVEN a user attempts to use @latest
- WHEN the user runs `opm mod get registry.example.com/dep@latest`
- THEN the CLI rejects with error requiring explicit SemVer version

#### Scenario: Transitive dependencies
- GIVEN a dependency with its own dependencies
- WHEN the user runs `opm mod get`
- THEN transitive dependencies are resolved and fetched

---

### Requirement: Update Command

The CLI MUST provide `opm mod update` to update dependencies.

#### Scenario: Check for updates
- GIVEN a project with dependencies
- WHEN the user runs `opm mod update --check`
- THEN available updates are displayed and CLI exits with code 1 if updates exist

#### Scenario: Patch/minor only by default
- GIVEN a dependency at v1.0.0 with v1.1.0 and v2.0.0 available
- WHEN the user runs `opm mod update`
- THEN only v1.1.0 is offered (not v2.0.0)

#### Scenario: Include major updates
- GIVEN a dependency with major update available
- WHEN the user runs `opm mod update --major`
- THEN major version updates are included in the prompt

---

### Requirement: Tidy Command

The CLI MUST provide `opm mod tidy` to remove unused dependencies.

#### Scenario: Remove unused
- GIVEN a project with unused dependencies in module.cue
- WHEN the user runs `opm mod tidy`
- THEN unused dependencies are removed from module.cue and local cache

---

### Requirement: Registry Authentication

The CLI MUST use credentials from `~/.docker/config.json` for registry access.

#### Scenario: Authenticated push
- GIVEN valid registry credentials configured via docker login
- WHEN the user runs `opm mod publish`
- THEN authentication is handled automatically

#### Scenario: Missing credentials
- GIVEN no credentials for a private registry
- WHEN the user runs `opm mod get`
- THEN CLI exits with code 4 and hint to run docker login

---

### Requirement: CUE Module Compatibility

The CLI MUST produce OCI artifacts compatible with CUE module specification.

#### Scenario: CUE CLI compatibility
- GIVEN a module published with `opm mod publish`
- WHEN a user with the module as dependency runs `cue eval`
- THEN the standard CUE CLI can evaluate the module

---

### Requirement: Platform Composition

The distribution system MUST preserve CUE's unification for platform composition.

#### Scenario: Extend upstream module
- GIVEN a local module importing an upstream OCI module
- WHEN the user defines a component that unifies with imported definitions
- THEN `opm mod build` generates the combined configuration

#### Scenario: Mandatory policy enforcement
- GIVEN an upstream module with mandatory Policy
- WHEN consumers try to override the policy
- THEN the policy cannot be bypassed (CUE unification enforces it)

#### Scenario: Reproducible builds
- GIVEN a dependency on a specific version
- WHEN `opm mod build` is run
- THEN the exact version from module.cue is used

---

## Edge Cases

- **Network Failure**: If registry is unreachable, CLI fails gracefully with exit code 3 and descriptive error (no stack trace).
- **Version Conflict**: If two dependencies require incompatible versions (Diamond Dependency), use CUE's MVS to pick compatible version; only error when no compatible version exists.
- **Corrupt Cache**: `opm mod get --force` re-downloads even if cached.
- **Auth Failure**: Exit code 4 with hint to run `docker login` or `oras login`.
- **Not Found**: Exit code 5 when module/version doesn't exist in registry.

---

## Exit Codes Contract

| Code | Name | Description |
|------|------|-------------|
| 0 | Success | Command completed successfully |
| 1 | General Error | Unspecified error / updates available (--check mode) |
| 2 | Validation Error | Module validation failed before publish |
| 3 | Connectivity Error | Cannot reach OCI registry |
| 4 | Permission Denied | Authentication failed or insufficient permissions |
| 5 | Not Found | Module/artifact not found in registry |

**Usage by Command:**

| Command | Possible Exit Codes |
|---------|---------------------|
| `opm mod publish` | 0, 1, 2, 3, 4 |
| `opm mod get` | 0, 1, 3, 4, 5 |
| `opm mod update` | 0, 1, 3, 4 |
| `opm mod tidy` | 0, 1 |

---

## Key Entities

- **CUE Module**: A directory containing `cue.mod/module.cue` and CUE source files.
- **OCI Artifact**: The packaged representation of a CUE module stored in a registry.
- **Dependency Map**: The `deps` field in `module.cue` mapping import paths to registry versions.

---

## Success Criteria

- **SC-001**: User can publish a module and consume it in another project without manual JSON editing in `module.cue` 100% of the time.
- **SC-002**: `opm mod get` updates `module.cue` deps within 2 seconds for cached registry response.
- **SC-003**: OPM module format remains 100% compatible with standard `cue` CLI commands.
- **SC-004**: `opm mod update` correctly identifies available SemVer updates (patch, minor, major).
- **SC-005**: Registry operations have clear, actionable error reporting.

---

## Command Reference

### `opm mod publish`

```text
opm mod publish <oci-ref> --version <semver> [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--version` | `-v` | required | SemVer version tag (e.g., v1.2.3) |
| `--force` | `-f` | false | Overwrite existing version |

### `opm mod get`

```text
opm mod get <oci-ref>@<version> [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | | false | Re-download even if cached |

### `opm mod update`

```text
opm mod update [dependency] [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--check` | false | Check only, exit 1 if updates available (CI mode) |
| `--major` | false | Include major version updates |

### `opm mod tidy`

```text
opm mod tidy
```

No flags. Removes unused dependencies from `module.cue` and cleans cache.
