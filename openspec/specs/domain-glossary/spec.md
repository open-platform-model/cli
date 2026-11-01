# domain-glossary Specification

## Purpose
Keeps a root `GLOSSARY.md` that fixes one word for each domain concept of the CLI, and routes the instruction files to it.

## Requirements

### Requirement: The repo has a root domain glossary

The repo SHALL keep a `GLOSSARY.md` at its root that names the domain terms of the CLI. Each term SHALL have a one or two sentence definition and, where loose synonyms exist, an `_Avoid_` line. The file SHALL hold only terms specific to this repo's domain.

#### Scenario: A schema term is defined as the CLI uses it

- **WHEN** a reader looks up `ModuleInstance` in `GLOSSARY.md`
- **THEN** the entry defines it as the CLI uses it and lists the loose words that blur it under `_Avoid_`

### Requirement: The instruction files route to the glossary

`AGENTS.md` and `CONTEXT.md` SHALL each name `GLOSSARY.md` once, as a file and not as a heading or a term. `CONTEXT.md` SHALL do it as one router row.

#### Scenario: A naming task finds the glossary

- **WHEN** an agent reads `CONTEXT.md` for a task that names a concept or picks a word for a domain term
- **THEN** one row sends it to `GLOSSARY.md`
