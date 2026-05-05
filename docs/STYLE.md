# Documentation Style Guide — CLI

Inherits all rules from the [workspace STYLE.md](../../STYLE.md). This file adds CLI-specific conventions.

## Audience

**Module and bundle authors who use the OPM CLI.** Many readers are comfortable with Go and Kubernetes tooling but may be new to CUE. Some readers are encountering OPM for the first time.

## Tone

- **Technical but approachable.** Accurate and precise, without assuming deep CUE expertise.
- **Progressive disclosure.** Lead with the most common usage; put advanced flags and edge cases later.
- Use second-person imperatives for instructions: "Run `opm mod build`", "Pass `--dry-run` to preview".
- Avoid jargon without a definition on first use.

## Document Types in This Repo

| Type | Location | Purpose |
|------|----------|---------|
| Design docs | `docs/design/` | Internal design decisions and architecture |
| Comparisons | `docs/comparisons/` | OPM vs. other tools |
| Vision docs | `docs/vision/` | Long-term direction |
| Roadmap | `docs/roadmap.md` | Planned work |
| RFCs | `docs/rfc/` | Proposed changes |

## Command Documentation

- Document commands with: purpose, synopsis, flags, examples, and notes (in that order).
- Synopsis format: `opm <subcommand> [flags]`
- Flag table columns: flag, type, default, description.
- Always show at least one complete runnable example per command.
- Show error output examples when a command has common failure modes.

```bash
# Build a module from the current directory
opm mod build

# Build and write output to a specific path
opm mod build --output ./dist/my-module.tar
```

## Progressive Disclosure

- Put the minimal working example first, before flags and options.
- Use `###` subheadings to separate basic usage from advanced usage.
- Admonitions (`> **Note:**`) are appropriate for gotchas that a first-time user will hit.

## CUE References

- When referencing CUE concepts, link to the glossary on first use per document: [`opm/docs/glossary.md`](../../opm/docs/glossary.md).
- Do not explain CUE internals inline; link to the catalog docs or glossary instead.

## Glossary

Canonical glossary: [`opm/docs/glossary.md`](../../opm/docs/glossary.md).

## What to Omit

- CUE schema definitions (those belong in `catalog/docs/`).
- End-user quickstarts (those belong in `opm/docs/`).
- Kubernetes operator procedures (those belong in `opm-operator/docs/`).
