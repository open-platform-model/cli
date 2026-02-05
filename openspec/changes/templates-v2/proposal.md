# Proposal: CLI Templates Commands

## Intent

Implement template management commands: `opm template list`, `opm template get`, `opm template show`, `opm template validate`, and `opm template publish`. Enable module scaffolding from OCI-distributed templates.

## Why

The V1 template system embeds three templates (simple, standard, advanced) directly in the CLI binary. This limits template distribution to CLI releases, prevents organizations from publishing custom templates, and requires CLI updates for any template changes. V2 enables OCI-based template distribution, allowing templates to be versioned, shared, and discovered independently of the CLI.

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

**SemVer**: MINOR (additive features, V1 embedded templates remain functional during transition)

## Affected Packages

- **CLI code**: New `internal/template/` package (client, manifest, reference, renderer, cache)
- **CLI commands**: New `internal/cmd/template/` command group
- **Dependencies**: oras.land/oras-go/v2 (already in use for OCI distribution)
- **Infrastructure**: Official templates published to `registry.opmodel.dev/templates/`
- **Documentation**: Template authoring guide, quickstart updates
- **Migration**: V1 templates remain embedded during transition; deprecation notice for future release
