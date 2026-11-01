## Context

The repo has `AGENTS.md`, `CONTEXT.md`, `CONSTITUTION.md`, ADRs and specs, but no single list of its domain words. The workspace docs form allows `GLOSSARY.md` as a root file name.

## Goals / Non-Goals

**Goals:** one root file in the glossary form (context heading, `## Language`, bold terms, `_Avoid_` lines); every term backed by a file in this repo; routing from the two instruction files.

**Non-Goals:** no change to code, specs of other capabilities, or the schema terms themselves.

## Decisions

- One glossary at the repo root, not a map: the CLI is a single context.
- Only terms specific to the CLI domain. General programming words are left out.
- Schema terms (`#Module`, `#Component`, `#ModuleInstance`) are defined as the CLI uses them, with the loose words that blur them under `_Avoid_`.
- `AGENTS.md` names the file in its intro sentence about other root documents; `CONTEXT.md` gets one row. Both name the file only, never a term or a heading.

## Risks / Trade-offs

A glossary can drift from the specs. Editing it is part of any change that renames a concept.
