# Proposal: CLI Distribution Commands

## Intent

Implement OCI-based module distribution: `opm mod publish`, `opm mod get`, `opm mod update`, and `opm mod tidy`. These commands enable sharing modules via OCI registries.

## Version Impact

**MINOR** - This change adds new commands without breaking existing functionality. Uses the existing single `config.registry` / `OPM_REGISTRY` configuration.

## Scope

**In scope:**

- `opm mod publish` - Publish module to OCI registry
- `opm mod get` - Download module dependency
- `opm mod update` - Update dependencies to newer versions
- `opm mod tidy` - Remove unused dependencies
- OCI artifact packaging
- Registry authentication via `~/.docker/config.json`

**Out of scope:**

- Template distribution (see cli-templates change)
- Bundle distribution (future)
- Multi-registry routing (see config-registries-v1 change)

## Affected Packages

| Package | Changes |
|---------|---------|
| `internal/cmd/mod/` | New commands: `publish.go`, `get.go`, `update.go`, `tidy.go` |
| `internal/oci/` | New package: OCI client with oras-go |
| `internal/cue/` | Dependency management types |

## Approach

1. Use oras-go for OCI registry interactions
2. Package modules as OCI artifacts compatible with CUE module spec
3. Embed `cuelang.org/go` libraries for standalone binary
4. Leverage CUE's MVS for dependency resolution
5. Follow SemVer for version constraints

## Clarifications

Key design decisions from spec development:

- **Registry Auth**: Strictly rely on `~/.docker/config.json` managed by external tools (e.g., `docker login`). No built-in login command.
- **No @latest**: CUE requires strict versioning. `opm mod get` requires explicit SemVer tags for reproducibility.
- **CUE Embedding**: Embed `cuelang.org/go` libraries for standalone binary without external `cue` CLI dependency.
- **Conflict Resolution**: Use CUE's MVS (Minimal Version Selection); only error when no compatible version exists.
- **Major Updates**: Default to patch/minor only; include major updates only with `--major` flag.
- **Cache Location**: Use CUE cache directory for module storage.
- **Availability**: Best-effort with clear error reporting (no uptime SLA).
