# Proposal: CLI Templates Commands

## Intent

Implement template management commands: `opm template list`, `opm template get`, `opm template show`, `opm template validate`, and `opm template publish`. Enable module scaffolding from OCI-distributed templates.

## Scope

**In scope:**

- `opm template list` - List available templates
- `opm template get` - Download template for editing
- `opm template show` - Show template details
- `opm template validate` - Validate template structure
- `opm template publish` - Publish template to registry
- Template placeholder substitution
- Official templates: simple, standard, advanced

**Out of scope:**

- Interactive template wizards
- Custom parameters beyond standard placeholders

## Approach

1. Templates distributed as OCI artifacts to `registry.opmodel.dev/templates/`
2. Use Go `text/template` for placeholder substitution
3. Support shorthand names resolving to official templates
4. Template manifest in `template.cue`
