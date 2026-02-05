# Design: CLI Distribution Commands

## Research & Decisions

### Distribution Architecture

**Context**: OPM needs a standard way to distribute and version artifacts (Core Definitions, User Modules). Two primary architecture options were evaluated.

**Explored**: Options analysis comparing Unified CUE Modules vs Split Model approaches.

**Options considered**:

1. **Unified CUE Modules** - Everything is a standard CUE module
   - Pros: Native composition via standard CUE imports, standard tooling (VS Code, `cue` CLI), no custom package manager needed
   - Cons: Strict versioning (major version in import path), UX friction managing `module.cue` dependencies manually

2. **Split Model** - Core as CUE modules, User Modules as custom OPM artifacts
   - Pros: Abstracts `cue.mod` complexity, could map different Core versions dynamically
   - Cons: Requires writing a new package manager, breaks standard CUE imports, limits inter-module composition

**Decision**: Unified CUE Modules

**Rationale**: Aligns with Constitution's Simplicity and Type Safety principles. The ecosystem benefits of standard CUE modules outweigh UX friction. CLI provides "Smart Helpers" (`opm mod get`, `opm mod update`) to mitigate dependency management friction.

---

### Version Tag Strictness

**Context**: Should `opm mod get` support `@latest` for convenience?

**Options considered**:

1. **Allow @latest** - Convenient for development
2. **Require explicit SemVer** - Ensures reproducibility

**Decision**: Require explicit SemVer (no @latest)

**Rationale**: CUE modules require strict versioning. Reproducibility and CUE compatibility are more important than convenience. This matches CUE's behavior.

---

## Technical Approach

### OCI Artifact Format

Modules are packaged as OCI artifacts strictly compatible with CUE module spec:

- Module CUE files
- `cue.mod/` directory
- Metadata annotations

### Registry Authentication

Use standard Docker credential chain via `~/.docker/config.json`. No OPM-specific credential storage.

### Version Resolution

- CUE's MVS (Minimal Version Selection) for dependency resolution
- SemVer 2.0.0 strictly enforced
- No `@latest` support—explicit versions required

### Command Flows

**Publish:**

```text
Validate → Package → Push → Tag
```

**Get:**

```text
Resolve → Pull → Extract → Update module.cue
```

**Update:**

```text
List deps → Query registry → Compare versions → Prompt → Apply
```

**Tidy:**

```text
Analyze imports → Identify unused → Remove from module.cue → Clean cache
```

## Data Model

### OCI Types (`internal/oci`)

```go
type Artifact struct {
    Reference   string            // Full OCI reference (registry/repo:tag)
    Digest      string            // Artifact digest (sha256:...)
    MediaType   string            // application/vnd.opm.module.v1+tar+gzip
    Annotations map[string]string
}

type PublishOptions struct {
    Version string // SemVer tag (required)
    Force   bool   // Overwrite existing tag
}

type FetchOptions struct {
    Reference string // OCI reference with version
    OutputDir string // Directory to extract to (defaults to CUE cache)
}

// OCI Annotations
const (
    AnnotationModuleName    = "dev.opmodel.module.name"
    AnnotationModuleVersion = "dev.opmodel.module.version"
    AnnotationCreated       = "org.opencontainers.image.created"
)
```

### Dependency Types (`internal/cue`)

```go
type Dependency struct {
    ImportPath string // CUE import path
    Version    string // Resolved SemVer version
    Registry   string // Registry URL
}

type UpdateInfo struct {
    Current    string // Currently installed version
    Latest     string // Latest available version
    UpdateType string // "patch", "minor", or "major"
}
```

## Error Handling

| Exit Code | Name | When |
|-----------|------|------|
| 0 | Success | Command completed |
| 1 | General Error | Unspecified error |
| 2 | Validation Error | Module invalid before publish |
| 3 | Connectivity Error | Registry unreachable |
| 4 | Permission Denied | Auth failed |
| 5 | Not Found | Module/version not in registry |

**Error messages include:**

- What went wrong
- Suggested fix (e.g., "Run 'docker login registry.example.com'")
- No stack traces in normal mode

## File Changes

- `cli/internal/cmd/mod/publish.go`
- `cli/internal/cmd/mod/get.go`
- `cli/internal/cmd/mod/update.go`
- `cli/internal/cmd/mod/tidy.go`
- `cli/internal/oci/` - OCI client package
