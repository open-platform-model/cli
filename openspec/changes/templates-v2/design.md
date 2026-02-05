# Design: CLI Templates Commands

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

## File Changes

- `cli/internal/cmd/template.go` - Command group
- `cli/internal/cmd/template_list.go`
- `cli/internal/cmd/template_get.go`
- `cli/internal/cmd/template_show.go`
- `cli/internal/cmd/template_validate.go`
- `cli/internal/cmd/template_publish.go`
- `cli/internal/templates/render.go` - Template rendering
