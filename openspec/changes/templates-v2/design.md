# Design: CLI Templates Commands

## Context

The V1 template system embeds templates in the CLI binary via Go's `//go:embed`:

```go
// cli/internal/templates/embed.go
//go:embed simple/* standard/* advanced/*
var simpleFS, standardFS, advancedFS embed.FS

type TemplateData struct {
    ModuleName string
    ModulePath string
    Version    string
}
```

**Limitations of V1:**

- Templates tied to CLI release cycle
- No custom template distribution
- No template discovery or versioning
- No template manifest or metadata

**Stakeholders:**

- Template authors: Publish and version templates independently
- End users: Discover and use templates from registries
- Platform operators: Curate organization-specific template catalogs

## Goals / Non-Goals

**Goals:**

- Enable OCI-based template distribution (portability, versioning)
- Provide template discovery via CLI (`opm template list`)
- Support multiple reference formats (shorthand, OCI, local file)
- Maintain backward compatibility with V1 placeholders (ModuleName, ModulePath, Version)
- Reuse OCI patterns from distribution-spec (ORAS, authentication, caching)

**Non-Goals:**

- User-defined template parameters beyond fixed placeholders (deferred to V3)
- Interactive template wizards or prompts (deferred)
- Template inheritance or composition (deferred)
- Template marketplace search beyond registry listing (deferred)

## Research & Decisions

### Template Distribution Method

**Context**: Need to distribute templates independently of CLI releases

**Options considered**:

1. OCI artifacts via ORAS - Standard registries work, versioning via tags
2. Git-based templates - Requires git dependency, no standard auth

**Decision**: OCI artifacts via ORAS

**Rationale**: Aligns with module distribution patterns (011-oci-distribution-spec). Standard OCI registries (GHCR, Docker Hub, Harbor) work without custom infrastructure. Versioning via OCI tags.

### Template Manifest Format

**Context**: Need structured metadata for templates

**Options considered**:

1. CUE manifest (template.cue) - Type-safe, consistent with OPM ecosystem
2. YAML/JSON manifest - More familiar, but inconsistent with CUE-first philosophy

**Decision**: CUE manifest (template.cue)

```cue
package template

name:        "standard"
version:     "1.0.0"  
description: "Standard OPM module template"
placeholders: ["ModuleName", "ModulePath", "Version"]
```

**Rationale**: Consistent with OPM ecosystem. Type-safe manifest validation. Self-documenting.

### Reference Resolution

**Context**: Need flexible ways to specify template sources

**Options considered**:

1. OCI-only references - Simple but verbose for common cases
2. Multiple formats with detection - Flexible but more complex

**Decision**: Support three formats with automatic detection

| Input | Resolution |
|-------|------------|
| `standard` | `oci://${REGISTRY}/templates/standard:latest` |
| `oci://reg.io/tpl:v1` | As-is |
| `file://./my-tpl` | Local filesystem |

**Registry precedence**: `--registry` flag > `OPM_REGISTRY` env > `config.registry` > `registry.opmodel.dev`

**Rationale**: Shorthand for convenience, explicit OCI for precision, file:// for development.

### OCI Artifact Structure

**Context**: Need defined media types for template artifacts

**Options considered**:

1. Generic OCI media types - Compatible but indistinguishable from other artifacts
2. Custom media types - Distinguishes templates from modules

**Decision**: Custom media types for template artifacts

| Layer | Media Type |
|-------|------------|
| Config | `application/vnd.opmodel.template.config.v1+json` |
| Content | `application/vnd.opmodel.template.content.v1.tar+gzip` |

**Rationale**: Distinguishes templates from modules. Standard OCI manifest format.

### Template Caching

**Context**: Need to avoid redundant downloads

**Options considered**:

1. Custom cache directory - Full control but unfamiliar to users
2. CUE cache directory structure - Consistent with module caching

**Decision**: Follow CUE cache directory structure

```text
~/.cache/cue/mod/extract/<registry>/<path>/<version>/
```

**Rationale**: Consistent with module caching. Familiar location for users.

### Command Structure

**Context**: Need CLI commands for template operations

**Options considered**:

1. Extend `opm mod` commands - Keeps commands together but conflates concepts
2. New `opm template` group - Clear separation of concerns

**Decision**: New `opm template` command group

```text
opm template list                    # List from registry
opm template show <ref>              # Show metadata
opm template get <ref> [--dir]       # Download for editing
opm template validate                # Validate local template
opm template publish <oci-ref>       # Publish to registry
```

**Rationale**: Separates template operations from module operations. Clear verb structure.

### Rendering Integration

**Context**: Need to integrate with module initialization

**Options considered**:

1. New `opm template init` command - Clear but duplicates `opm mod init`
2. Modify `opm mod init --template` - Seamless upgrade path

**Decision**: Modify `opm mod init --template` to use new template system

```bash
opm mod init my-app --template standard        # Shorthand
opm mod init my-app --template oci://reg/tpl   # Explicit OCI
opm mod init my-app --template file://./local  # Local dev
```

**Rationale**: Transparent upgrade path. Same command, enhanced backend.

**Ownership boundaries (Principle II):**

- Template author: Creates template.cue and .tmpl files
- CLI: Fetches, validates, caches, renders templates
- End user: Selects template and provides module name

## Technical Approach

### Template Structure

```text
my-template/
├── template.cue          # Manifest
├── module.cue.tmpl       # Template files
├── values.cue.tmpl
└── cue.mod/
    └── module.cue.tmpl
```

### Placeholder Substitution

Standard placeholders using Go `text/template`:

- `{{.ModuleName}}` - Module name
- `{{.ModulePath}}` - Module path
- `{{.Version}}` - Initial version

### Template Reference Resolution

| Format | Resolution |
|--------|------------|
| `standard` | `oci://${REGISTRY}/templates/standard:latest` |
| `oci://...` | As-is |
| `file://...` | Local filesystem |

### Official Templates

Published to `registry.opmodel.dev/templates/`:

- `simple` - Single-file for learning
- `standard` - Separated files for teams
- `advanced` - Multi-package for enterprise

## Command Syntax

```text
opm template list [--registry <url>] [-o json|table]
opm template show <ref> [--registry <url>]
opm template get <ref> [--dir <path>] [--force]
opm template validate [<path>]
opm template publish <oci-ref>
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--registry` | string | `registry.opmodel.dev` | OCI registry URL for shorthand resolution |
| `--dir` | string | `./<template-name>` | Download directory for `template get` |
| `--force` | bool | `false` | Overwrite existing directory |
| `-o, --output` | string | `table` | Output format (`table`, `json`) |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage error (invalid flags, missing arguments) |
| 2 | Validation error (invalid template manifest, missing .tmpl files) |
| 3 | Connectivity error (registry unreachable) |
| 5 | Not found (template doesn't exist) |

## Example Output

### Success: template list

```text
$ opm template list
NAME       VERSION  DESCRIPTION
simple     1.0.0    Single-file module for learning
standard   1.0.0    Separated files for teams
advanced   1.0.0    Multi-package for enterprise
```

### Success: template show

```text
$ opm template show standard
Name:        standard
Version:     1.0.0
Description: Standard OPM module template

Placeholders:
  - ModuleName
  - ModulePath
  - Version

Files:
  ├── template.cue
  ├── module.cue.tmpl
  ├── values.cue.tmpl
  └── cue.mod/
      └── module.cue.tmpl
```

### Error: template not found

```text
$ opm template show unknown
Error: template not found: unknown
Hint: Run 'opm template list' to see available templates
```

### Error: registry unreachable

```text
$ opm template list --registry unreachable.io
Error: failed to connect to registry: unreachable.io
Hint: Check your network connection and registry URL
```

## Risks / Trade-offs

**[Risk] Registry availability** → Mitigation: Local caching, fallback to embedded templates during transition.

**[Risk] OCI media type acceptance** → Mitigation: Use standard OCI patterns, test against major registries.

**[Risk] Breaking change for CI using `--template embedded`** → Mitigation: Embedded templates remain available during deprecation period.

**[Trade-off] No template parameters** → Keeps V2 scope focused. Parameters can be added as V3 feature without architectural changes.

**[Trade-off] `:latest` allowed for templates** → Unlike modules (which require explicit versions), templates allow `:latest` for convenience. Template authors manage versioning.

## File Changes

- `cli/internal/cmd/template.go` - Command group
- `cli/internal/cmd/template_list.go`
- `cli/internal/cmd/template_get.go`
- `cli/internal/cmd/template_show.go`
- `cli/internal/cmd/template_validate.go`
- `cli/internal/cmd/template_publish.go`
- `cli/internal/templates/render.go` - Template rendering
