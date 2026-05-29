# OPM CLI Constitution

## Purpose

This document is the reader-friendly reference for the principles that shape design, implementation, validation, and change management in the OPM CLI. The CLI is governed by the normative constitutional source in `openspec/config.yaml`.

## Design Principles

| # | Principle | Summary |
| ---- | --------- | ------- |
| **I** | [Type Safety First](#i-type-safety-first) | Invalid CLI input is rejected before execution begins |
| **II** | [Separation of Concerns](#ii-separation-of-concerns) | Commands, workflows, and reusable packages keep clear boundaries |
| **III** | [Composability](#iii-composability) | Commands and packages compose without tight coupling |
| **IV** | [Declarative Intent](#iv-declarative-intent) | CLI behavior and output emphasize outcomes over internals |
| **V** | [Portability by Design](#v-portability-by-design) | The CLI must behave consistently across supported platforms |
| **VI** | [Semantic Versioning](#vi-semantic-versioning) | Releases follow SemVer and commits follow Conventional Commits |
| **VII** | [Simplicity & YAGNI](#vii-simplicity--yagni) | Complexity must be justified; prefer direct, explicit solutions |
| **VIII** | [Small Batch Sizes](#viii-small-batch-sizes-iterative--incremental-delivery) | Changes must stay tiny, incremental, and independently verifiable |

---

### I. Type Safety First

All CLI configuration MUST be validated at load time. Invalid flags, config
files, and module inputs MUST be rejected before any operation begins, never
during execution.

- Validate in order: flags -> config -> module -> execution
- Use CUE for configuration validation where applicable
- Prefer strong Go types over open-ended data structures
- Fail early so users get actionable feedback before side effects occur

```text
flags -> config -> module -> execute
```

---

### II. Separation of Concerns

The CLI MUST maintain clear boundaries between command handling, workflow
orchestration, and reusable library logic.

- Commands handle user interaction, flag parsing, and command wiring
- Internal packages handle workflows, rendering, config loading, and cluster operations
- Shared packages provide reusable, command-agnostic helpers
- Commands orchestrate; they do not implement core business logic

Clear boundaries keep the codebase easier to test, reason about, and change.

---

### III. Composability

Commands and packages MUST compose without tight coupling.

- Command layers should depend on focused internal or package APIs
- Business logic packages should not depend on command packages
- Output formatting should remain separate from data generation
- Shared functionality should be reusable without dragging CLI-specific concerns with it

Composition should come from clear package contracts, not hidden dependencies.

---

### IV. Declarative Intent

CLI behavior and output MUST emphasize what happened, not the internal steps
 used to make it happen.

- Success messages should describe outcomes
- Errors should explain what is wrong and how to fix it
- Internal stack traces and implementation detail should not leak into normal output
- Verbose modes may expose additional execution detail for debugging

This keeps the CLI understandable for users while preserving depth for debugging.

---

### V. Portability by Design

The CLI MUST work across supported platforms without requiring users to adapt
 behavior manually.

- Do not hardcode filesystem paths
- Use cross-platform path handling and home directory resolution
- Do not rely on shell-specific command behavior
- Preserve behavior across Linux, macOS, and Windows

Portability is a product requirement, not a cleanup task.

---

### VI. Semantic Versioning

CLI releases MUST follow SemVer 2.0.0. All commits MUST follow Conventional
Commits: `type(scope): description`.

- MAJOR: breaking command, flag, or behavior changes
- MINOR: new commands or new flags with sensible defaults
- PATCH: bug fixes, refinements, and performance improvements
- Commit messages should be concise, scoped, and end with `Co-Authored-By: Claude <noreply@anthropic.com>`

Versioning communicates compatibility and upgrade risk to users and maintainers.

---

### VII. Simplicity & YAGNI

Start simple. New flags, commands, packages, or abstractions MUST be justified.

- Prefer fewer flags with sensible defaults
- Prefer explicit configuration over magic inference
- Prefer direct solutions over speculative abstractions
- Do not build features that have not been requested or demonstrated as needed

Every new option increases maintenance cost, API surface, and user complexity.

---

### VIII. Small Batch Sizes (Iterative & Incremental Delivery)

All changes MUST be kept tiny. Small, incremental, independently verifiable
steps are required.

- Large requests should be split into smaller sequential tasks
- Tiny changes produce focused, atomic commits
- A single change should ideally address one specific concern
- Validation should stay practical at each step

This principle applies to both implementation and planning. Large bundled
changes hide risk, slow review, and weaken validation.

### Execution Gate

Before beginning any implementation, the scope of the request MUST be evaluated
against the small-batch principle.

If the request is too large, the required response is:

> "🛑 **Scope Warning**: This request is too large for a single safe iteration. I suggest we split it into the following smaller steps: [list 2-3 logical, tiny steps]. Should we start with step 1?"

---

## Quality Gates

Before merge, the expected validation gates are:

1. `task fmt`
2. `task lint`
3. `task test`

---

## How Principles Work Together

These principles reinforce each other:

- Type safety supports clear validation and dependable command behavior
- Separation of concerns keeps workflows composable and packages reusable
- Declarative intent improves user experience and error clarity
- Portability requires explicit behavior and disciplined package boundaries
- Small batch sizes keep validation fast and change quality high

When principles appear to conflict, treat that as a design smell and document
the trade-off explicitly.

## Further Reading

- `openspec/config.yaml` — normative constitutional source
- `AGENTS.md` — repository mechanics, commands, and coding guidance
