# RFC-0006: Documentation Generation from Code & Definitions

| Field       | Value                              |
|-------------|------------------------------------|
| **Status**  | Draft                              |
| **Created** | 2026-02-16                         |
| **Authors** | OPM Contributors                   |

## Summary

Introduce an automated documentation generation pipeline that produces a
public-facing documentation site from CUE definitions in the catalog and CLI
command definitions in the Go codebase. The pipeline lives in a dedicated docs
repository containing a custom Go tool (`docgen`) for high-fidelity CUE schema
extraction via the Go API, Cobra's built-in doc generation for CLI references,
and Hugo with the Docsy theme as the static site generator.

## Motivation

### The Problem

OPM has 93 CUE definition files across 9 modules (core, schemas, resources,
traits, blueprints, policies, providers, examples, schemas\_kubernetes), plus a
CLI with a growing number of commands. Today, all documentation is hand-written
markdown scattered across two repositories. There is no automated reference
documentation, no searchable site, and no way for users to discover available
definition types, their fields, constraints, or defaults without reading raw CUE
source.

```text
┌─────────────────────────────────────────────────────────────────┐
│  Documentation Today                                            │
│                                                                 │
│  catalog/docs/          ~122 hand-written markdown files        │
│  catalog/README.md      High-level overview only                │
│  cli/docs/rfc/          RFCs (design docs, not user-facing)     │
│                                                                 │
│  What's Missing:                                                │
│    - Searchable reference for all definition types              │
│    - Field-level docs with types, constraints, defaults         │
│    - Cross-references (e.g., which Traits apply to which        │
│      Resources)                                                 │
│    - CLI command reference                                      │
│    - Getting started guides integrated with reference           │
│    - Versioned documentation                                    │
│                                                                 │
│  How users learn OPM today:                                     │
│    1. Read raw .cue files                                       │
│    2. Hope the comments are sufficient                          │
│    3. Ask someone                                               │
└─────────────────────────────────────────────────────────────────┘
```

### Why Not Use Existing Tools?

**CUE has no documentation generator.** This is a known ecosystem gap
([cue-lang/cue#2794](https://github.com/cue-lang/cue/discussions/2794)). The
community workaround is exporting to OpenAPI and using OpenAPI doc tools, but
this loses CUE-specific semantics that are critical for OPM:

| CUE Feature                  | OpenAPI Equivalent         | Information Lost            |
|------------------------------|----------------------------|-----------------------------|
| Disjunctions `"a" \| "b"`   | `enum: [a, b]`             | Nested disjunctions lost    |
| Bounds `>=0 & <=65535`       | `minimum/maximum`          | Mostly preserved            |
| Optional `?` vs Required `!` | `required` array           | Preserved                   |
| Definitions `#Foo`           | `$ref` / `$defs`           | Structure lost              |
| `close()` semantics          | `additionalProperties`     | Partially preserved         |
| Comprehensions               | Not representable          | Completely lost             |
| `appliesTo` references       | Not representable          | Completely lost             |
| Default values               | `default`                  | Preserved                   |
| Nested structs with docs     | Flattened `$ref` chains    | Context lost                |

For a project that uses CUE as its core language, flattening to OpenAPI strips
too much information. A custom extractor that preserves CUE semantics is the
right approach.

### Goals

1. **Automated reference docs** — Every CUE definition type generates a
   reference page with fields, types, constraints, defaults, and doc comments.
2. **CLI reference** — Every `opm` command generates a reference page with
   synopsis, flags, examples, and cross-links.
3. **Cross-references** — Traits link to their applicable Resources. Blueprints
   link to their composed Resources and Traits. Components show available
   options.
4. **Searchable** — Full-text search across all documentation.
5. **Versioned** — Documentation can be versioned alongside catalog releases.
6. **Composable** — Hand-written guides, tutorials, and conceptual docs coexist
   with generated reference docs.
7. **CI-friendly** — The entire site can be built in CI from source with no
   manual steps.

## Design

### Repository Structure

The documentation pipeline lives in a dedicated repository. The Go docgen tool
is co-located with the Hugo site source because its only consumer is the site
build — it is not a user-facing CLI tool and has no reason to exist in the OPM
CLI repository.

```text
opmodel-docs/
├── cmd/docgen/            # Go tool: CUE -> JSON, cobra -> markdown
│   └── main.go
├── internal/
│   ├── cuedoc/            # CUE schema extraction logic
│   └── cobradoc/          # Cobra doc generation with Hugo front matter
├── site/                  # Hugo site source
│   ├── hugo.toml
│   ├── content/
│   │   ├── getting-started/
│   │   ├── guides/
│   │   ├── reference/
│   │   │   ├── definitions/
│   │   │   │   └── _content.gotmpl    # Content adapter: JSON -> pages
│   │   │   └── cli/                   # Generated cobra markdown
│   │   └── rfcs/
│   ├── data/
│   │   └── schema/                    # Generated JSON from docgen
│   ├── layouts/
│   │   └── shortcodes/
│   │       ├── def-fields.html        # Definition fields table
│   │       ├── def-ref.html           # Cross-reference links
│   │       └── cue-source.html        # CUE source view
│   ├── static/
│   └── themes/                        # Docsy via Hugo module
├── go.mod
├── go.sum
├── Taskfile.yml
└── README.md
```

### Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Documentation Pipeline                       │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐   │
│  │  CUE Catalog │    │  Go CLI      │    │  Hand-written    │   │
│  │  (external)  │    │  (external)  │    │  Guides/Concepts │   │
│  │  .cue files  │    │  cobra cmds  │    │  (in this repo)  │   │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────────┘   │
│         │                   │                   │               │
│         ▼                   ▼                   │               │
│  ┌──────────────────────────────────┐           │               │
│  │         cmd/docgen               │           │               │
│  │                                  │           │               │
│  │  cuedoc:    CUE API -> JSON      │           │               │
│  │  cobradoc:  cobra/doc -> .md     │           │               │
│  └──────┬──────────────┬────────────┘           │               │
│         │              │                        │               │
│         ▼              ▼                        ▼               │
│  ┌────────────┐  ┌────────────┐  ┌──────────────────────┐       │
│  │ site/data/ │  │ site/      │  │ site/content/        │       │
│  │ schema/    │  │ content/   │  │ getting-started/     │       │
│  │ *.json     │  │ reference/ │  │ guides/              │       │
│  │            │  │ cli/*.md   │  │                      │       │
│  └─────┬──────┘  └─────┬──────┘  └──────────┬───────────┘       │
│        │               │                    │                   │
│        ▼               ▼                    ▼                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Hugo + Docsy                         │    │
│  │                                                         │    │
│  │  _content.gotmpl    Content adapters generate pages     │    │
│  │                     from JSON schema data               │    │
│  │  layouts/shortcodes Custom shortcodes for type          │    │
│  │                     rendering, cross-references         │    │
│  │  Docsy theme        Search, versioning, navigation      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                     │
│                           ▼                                     │
│                    Static Site (HTML)                           │
│                    Hosted at opmodel.dev/docs                   │
└─────────────────────────────────────────────────────────────────┘
```

### Component 1: `docgen` — CUE Schema Extractor

A Go tool that uses `cuelang.org/go/cue` to walk CUE definitions and produce
structured JSON for Hugo consumption. This is the core of the pipeline and the
only custom component. It lives at `cmd/docgen/` in the docs repository.

#### CUE Go API Surface Used

| API                          | Purpose                                                                     |
|------------------------------|-----------------------------------------------------------------------------|
| `load.Instances()`           | Load CUE modules from directory                                             |
| `Value.Fields()`             | Iterate struct fields (with `cue.Optional(true)`, `cue.Definitions(true)`)  |
| `Value.Doc()`                | Extract `//` doc comments as `[]*ast.CommentGroup`                          |
| `Value.Default()`            | Get default values                                                          |
| `Value.IncompleteKind()`     | Get type kind (string, int, struct, list, etc.)                             |
| `Value.Expr()`               | Get constraint expressions (disjunctions, bounds)                           |
| `Selector.IsDefinition()`    | Identify `#Foo` definitions                                                 |
| `Selector.ConstraintType()`  | Detect optional (`?`) vs required (`!`)                                     |
| `Value.LookupPath()`         | Navigate to nested fields (e.g., `appliesTo`)                               |

#### Output Schema

The tool produces one JSON file per CUE module containing all definitions:

```json
{
  "module": "opmodel.dev/core@v0",
  "version": "v0.1.21",
  "definitions": [
    {
      "name": "Resource",
      "kind": "Resource",
      "fqn": "opmodel.dev/core/v0#Resource",
      "description": "Defines a resource of deployment within the system...",
      "fields": [
        {
          "name": "apiVersion",
          "path": "metadata.apiVersion",
          "type": "string",
          "constraint": "#APIVersionType",
          "required": true,
          "description": "Example: \"resources.opmodel.dev/workload@v0\"",
          "default": null
        },
        {
          "name": "name",
          "path": "metadata.name",
          "type": "string",
          "constraint": "#NameType (RFC 1123 DNS label, max 63 chars)",
          "required": true,
          "description": "Example: \"container\"",
          "default": null
        },
        {
          "name": "description",
          "path": "metadata.description",
          "type": "string",
          "constraint": null,
          "required": false,
          "description": "Human-readable description of the definition",
          "default": null
        }
      ],
      "spec_description": "MUST be an OpenAPIv3 compatible schema...",
      "related": []
    }
  ]
}
```

For Traits, the `related` field includes `appliesTo` references:

```json
{
  "name": "ScalingTrait",
  "related": [
    {
      "type": "appliesTo",
      "target": "opmodel.dev/resources/workload@v0#Container",
      "label": "ContainerResource"
    }
  ]
}
```

#### Module Processing Order

The tool processes modules in dependency order (matching the catalog's
`versions.yml`):

```text
core -> schemas -> schemas_kubernetes -> resources -> policies
     -> traits -> blueprints -> providers -> examples
```

Cross-references are resolved after all modules are loaded.

#### CLI Interface

```bash
# Generate all schema JSON to a directory
docgen schema --catalog-dir ../catalog --output ./site/data/schema/

# Generate CLI reference markdown (imports opm's root command)
docgen cli --output ./site/content/reference/cli/

# Generate everything
docgen all --catalog-dir ../catalog --output ./site/
```

### Component 2: Cobra CLI Doc Generation

Uses `github.com/spf13/cobra/doc.GenMarkdownTreeCustom()` with a Hugo front
matter prepender — the same approach Hugo itself uses for its own CLI reference.

```go
filePrepender := func(filename string) string {
    name := filepath.Base(filename)
    base := strings.TrimSuffix(name, path.Ext(name))
    return fmt.Sprintf("---\ntitle: \"%s\"\nslug: %s\n---\n",
        strings.ReplaceAll(base, "_", " "), base)
}

linkHandler := func(name string) string {
    base := strings.TrimSuffix(name, path.Ext(name))
    return "/reference/cli/" + strings.ToLower(base) + "/"
}

doc.GenMarkdownTreeCustom(rootCmd, outputDir, filePrepender, linkHandler)
```

Each command produces a markdown file with:

- Synopsis and description
- Usage pattern
- Flags table (local and inherited)
- Examples
- Cross-links to parent/child commands

The docs repo imports the CLI's root command package as a Go dependency. This
means CLI doc generation stays in sync with the actual command tree — no manual
maintenance.

### Component 3: Hugo Site with Docsy

#### Why Docsy

| Criterion                | Docsy                                    |
|--------------------------|------------------------------------------|
| **Proven at scale**      | kubernetes.io, grpc.io, etcd.io, knative |
| **Reference docs**       | Built for API/schema reference layouts   |
| **Search**               | Algolia, Lunr.js, Google CSE             |
| **Versioning**           | Built-in multi-version support           |
| **Shortcodes**           | Tabs, feature-state, alerts, diagrams    |
| **OpenAPI**              | `swaggerui` shortcode (for optional use) |
| **Maintenance**          | Backed by Google, active community       |

#### Content Adapters for Schema Pages

Hugo's `_content.gotmpl` (v0.126.0+) generates reference pages from the JSON
data produced by `docgen`:

```text
site/content/
└── reference/
    ├── _index.md
    ├── definitions/
    │   ├── _content.gotmpl     <- Generates pages from data/schema/*.json
    │   └── _index.md
    └── cli/
        ├── _index.md
        └── opm_mod_apply.md    <- Generated by cobra/doc
```

The content adapter reads `data/schema/*.json` and creates a page per
definition:

```go-html-template
{{ range $file := (resources.Match "data/schema/*.json") }}
  {{ $data := $file | transform.Unmarshal }}
  {{ range $data.definitions }}
    {{ $content := dict "mediaType" "text/markdown" "value" .description }}
    {{ $params := dict
      "definition_kind" .kind
      "fqn" .fqn
      "fields" .fields
      "related" .related
      "module" $data.module
      "module_version" $data.version
    }}
    {{ $page := dict
      "content" $content
      "kind" "page"
      "params" $params
      "path" (printf "%s/%s" $data.module .name)
      "title" .name
    }}
    {{ $.AddPage $page }}
  {{ end }}
{{ end }}
```

#### Custom Shortcodes

**Definition fields table** (`layouts/shortcodes/def-fields.html`):

Renders the fields from page params into a styled table with type badges,
required/optional indicators, constraint details, and default values.

**Cross-reference links** (`layouts/shortcodes/def-ref.html`):

Renders links to related definitions (e.g., Traits that apply to a Resource)
with hover previews.

**CUE source view** (`layouts/shortcodes/cue-source.html`):

Optionally shows the original CUE source alongside the generated reference,
linking back to the catalog repository.

#### Site Structure

```text
opmodel.dev/docs/
├── Getting Started/
│   ├── Installation
│   ├── Your First Module
│   └── Core Concepts
├── Guides/
│   ├── Module Authoring
│   ├── Platform Operations
│   └── Blueprint Patterns
├── Reference/
│   ├── Definitions/              [x] Generated from CUE
│   │   ├── Core/
│   │   │   ├── Module
│   │   │   ├── ModuleRelease
│   │   │   ├── Component
│   │   │   ├── Resource
│   │   │   ├── Trait
│   │   │   ├── Blueprint
│   │   │   ├── Policy
│   │   │   └── ...
│   │   ├── Resources/
│   │   │   ├── ContainerResource
│   │   │   ├── VolumesResource
│   │   │   ├── ConfigMapResource
│   │   │   ├── SecretResource
│   │   │   └── ...
│   │   ├── Traits/
│   │   │   ├── ScalingTrait
│   │   │   ├── HealthCheckTrait
│   │   │   ├── SizingTrait
│   │   │   ├── ExposeTrait
│   │   │   ├── HTTPRouteTrait
│   │   │   └── ...
│   │   ├── Blueprints/
│   │   │   ├── StatelessWorkload
│   │   │   ├── StatefulWorkload
│   │   │   ├── DaemonWorkload
│   │   │   └── ...
│   │   └── Providers/
│   │       └── Kubernetes/
│   │           ├── DeploymentTransformer
│   │           ├── ServiceTransformer
│   │           └── ...
│   └── CLI/                      [x] Generated from cobra
│       ├── opm
│       ├── opm mod apply
│       ├── opm mod build
│       ├── opm mod init
│       ├── opm mod status
│       ├── opm mod delete
│       ├── opm mod template
│       └── opm config init
└── RFCs/
```

### Build Pipeline

```text
┌────────────────────────────────────────────────────────────┐
│  Taskfile.yml                                              │
│                                                            │
│  task generate:                                            │
│    1. docgen schema                                        │
│       --catalog-dir ../catalog                             │
│       --output ./site/data/schema/                         │
│    2. docgen cli                                           │
│       --output ./site/content/reference/cli/               │
│                                                            │
│  task serve:                                               │
│    hugo server --source ./site                             │
│                                                            │
│  task build:                                               │
│    1. task generate                                        │
│    2. hugo --minify --source ./site --destination ./public │
│                                                            │
│  task deploy:                                              │
│    1. task build                                           │
│    2. Deploy ./public (GitHub Pages / Cloudflare Pages)    │
└────────────────────────────────────────────────────────────┘
```

The pipeline runs in CI on:

- Push to catalog `main` branch (CUE definitions changed — trigger via webhook
  or scheduled rebuild)
- Push to docs repo `main` branch (content, templates, or docgen tool changed)

CLI doc generation stays in sync automatically because the docs repo imports the
CLI's root command package as a Go dependency. Updating the CLI dependency
version triggers a rebuild.

## Alternatives Considered

### A: CUE -> OpenAPI -> Redoc/Swagger

Export CUE as OpenAPI 3.0 using `cue export --out openapi`, then render with
Redoc or Swagger UI.

**Rejected because:**

- Loses CUE-specific semantics (comprehensions, `close()`, `appliesTo`)
- OpenAPI is designed for HTTP APIs, not schema definition systems
- Produces a single monolithic API doc rather than per-definition reference pages
- No cross-referencing between definition types

### B: CUE -> JSON Schema -> json-schema-for-humans

Export CUE as JSON Schema, render with `coveooss/json-schema-for-humans`.

**Rejected because:**

- Same semantic loss as OpenAPI path
- JSON Schema doc tools produce flat, unstyled output
- No integration with a broader documentation site
- Cannot represent the OPM type hierarchy

### C: hof (hofstadter-io/hof) Templating

Use `hof gen` with Go templates to generate docs directly from CUE definitions.

**Considered but deferred because:**

- Adds a dependency on the `hof` tool and its CUE conventions
- The CUE Go API is sufficient and more directly controlled
- Could revisit if the custom Go tool becomes unwieldy

### D: Docusaurus Instead of Hugo

Docusaurus has the best JSON Schema plugin ecosystem
(`docusaurus-json-schema-plugin`, `docusaurus-openapi-docs`).

**Rejected because:**

- OPM is a Go project — Hugo keeps the toolchain Go-native
- Docusaurus requires Node.js/React for the entire site
- Hugo's content adapters provide equivalent page generation capability
- Docsy (Hugo theme) is proven at Kubernetes scale, which is OPM's peer group

### E: MkDocs Material

Python-based SSG with `mkdocs-schema-reader` plugin.

**Rejected because:**

- Introduces Python into a Go toolchain
- Schema reader plugin is low-activity (5 stars)
- No content adapter equivalent for programmatic page generation

### F: Docgen Tool in the OPM CLI

Embed the doc generation as an `opm docgen` subcommand in the CLI repository.

**Rejected because:**

- The docgen tool is a build-time utility for the site, not a user-facing
  command
- Adding it to the CLI couples the CLI's release cycle with documentation
  builds
- The tool's dependencies (Hugo front matter generation, site-specific JSON
  schemas) are site concerns, not CLI concerns
- Keeping it in the docs repo means one `task build` produces the entire site
  with no cross-repo coordination at build time

## Open Questions

### Q1: Catalog Data Fetching

The docs repo needs access to the catalog's CUE source. Options:

| Option                           | Pros                                | Cons                                |
|----------------------------------|-------------------------------------|-------------------------------------|
| Git submodule                    | Always in sync, local path access   | Submodule management overhead       |
| Clone in CI                      | Simple, no local coupling           | Slower builds, version pinning      |
| Hugo module mount                | Hugo-native, version-controlled     | Only for Hugo content, not Go tool  |
| OCI registry (published modules) | Uses the real distribution path     | Requires registry access, may lag   |

Git submodule is likely the pragmatic choice for development; CI can clone at a
pinned tag for reproducible builds.

### Q2: Versioning Strategy

Should docs be versioned per-catalog-release (e.g., docs for `v0.1.21` vs
`v0.2.0`)? Docsy supports this but it adds build complexity. For an early-stage
project, a single "latest" version with a changelog may suffice initially.

### Q3: Spec Field Rendering

The `#spec` field in OPM definitions uses dynamic field names derived from
`metadata.name` via `KebabToPascal`. The docgen tool needs to resolve these
computed field names and render the actual spec schema from concrete definition
implementations (e.g., `#ContainerResource`, not just `#Resource`).

### Q4: Constraint Rendering Depth

How deeply should constraints be rendered? Options range from simple type
annotations (`string`, `int`) to full CUE expression rendering
(`int & >=0 & <=65535`). The initial implementation should aim for full
constraint rendering since that is the primary value over OpenAPI-based tools.

## Implementation Plan

### Phase 1: Core Pipeline (MVP)

1. **Create docs repository** with Go module and Hugo site scaffold.
2. **Build `docgen schema`** — Go tool that walks CUE definitions in the
   catalog's `core/` module and outputs JSON.
3. **Build `docgen cli`** — Cobra doc generation with Hugo front matter.
4. **Scaffold Hugo site** — Docsy theme, basic layout, content adapter for
   schema JSON.
5. **Local build** — `task build` produces a working site from source.

### Phase 2: Full Coverage

1. **Extend to all modules** — Process all 9 CUE modules with cross-references
   resolved.
2. **Custom shortcodes** — Type rendering, cross-reference links, CUE source
   view.
3. **Hand-written content** — Getting started guide, core concepts, module
   authoring guide.

### Phase 3: CI & Publishing

1. **CI pipeline** — Automated builds on push to catalog or docs repo.
2. **Hosting setup** — Deploy to GitHub Pages or Cloudflare Pages.
3. **Versioning** — Add multi-version support if and when needed.

## References

- [CUE Go API — Walking Schemas](https://cuelang.org/docs/howto/walk-schemas-using-go-api/)
- [Hugo Content Adapters](https://gohugo.io/content-management/content-adapters/)
- [Docsy Theme](https://www.docsy.dev/)
- [cobra/doc package](https://pkg.go.dev/github.com/spf13/cobra/doc)
- [cue-lang/cue#2794 — Doc generation discussion](https://github.com/cue-lang/cue/discussions/2794)
- [Kubernetes docs (Docsy example)](https://kubernetes.io/)
- [Hugo's own CLI doc generation](https://github.com/gohugoio/hugo/blob/master/commands/gen.go)
- [gomarkdoc — Go markdown doc generator](https://github.com/princjef/gomarkdoc)
