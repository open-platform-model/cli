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
| **VIII** | [Mergeable Sections](#viii-mergeable-sections-iterative--incremental-delivery) | Every section ends green and commits; every merge leaves `main` releasable |

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
- Commit messages should be concise and scoped. AI attribution is limited to an optional plain `Co-Authored-By: Claude <noreply@anthropic.com>` trailer — never a `Claude-Session:` trailer, session URL, model-versioned co-author line, or "Generated with …" footer.

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

### VIII. Mergeable Sections (Iterative & Incremental Delivery)

A change is delivered as the sections of its `tasks.md` (`## N. Title` headings with `N.M` checkboxes). Two invariants hold at every section boundary; they replace any size limit on the change itself.

- Every merge leaves `main` releasable: a section MUST end green under the validation gates and MUST close with a commit task naming its Conventional Commit
- Work survives a session boundary: the commit task is the pause point, leaving checked boxes and a clean tree for the next session to resume from
- A change SHOULD cut into at most about five sections; one PR per change with one commit per section is the default
- Section 1 is a spike whenever the design carries an unverified assumption

This principle applies to both planning and implementation. A section that cannot end green on its own hides risk, slows review, and weakens validation.

### Execution Gate

Before beginning any implementation, the request MUST be evaluated against the mergeable-sections principle.

If the request cannot be cut into sections that each end green and leave `main` releasable, or needs more than about five, the required response is:

> "🛑 **Scope Warning**: This request does not cut into a handful of mergeable sections. I suggest we split it into the following changes: [list 2-3 changes, each a few sections that leave main releasable]. Should we start with the first?"

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
- Mergeable sections keep `main` releasable and validation fast

When principles appear to conflict, treat that as a design smell and document
the trade-off explicitly.

## Further Reading

- `openspec/config.yaml` — normative constitutional source
- `AGENTS.md` — repository mechanics, commands, and coding guidance
