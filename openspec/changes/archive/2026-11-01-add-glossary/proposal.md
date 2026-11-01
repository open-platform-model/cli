## Why

Agents and the owner use several words for one concept in this repo: instance, release and module release; platform and cluster; orphan and stale. A root `GLOSSARY.md` fixes one word for each concept, so design and refactoring work, and the skills that read the file, use the repo's own terms.

## What Changes

- Add `GLOSSARY.md` at the repo root with the domain terms of the CLI, each with a definition and the loose words to avoid.
- Route to it from `AGENTS.md` (one clause in the intro) and `CONTEXT.md` (one router row).

Release class: none. Documentation only; `docs` is hidden from the changelog in this repo.

## Capabilities

### New Capabilities

- `domain-glossary`: the root glossary file and its routing.

### Modified Capabilities

None.

## Enhancement

None.

## Impact

- **Files:** `GLOSSARY.md`, `AGENTS.md`, `CONTEXT.md`.
- **Risks:** a term goes stale when a spec or a doc renames a concept. The glossary then misleads agents until someone edits it.
