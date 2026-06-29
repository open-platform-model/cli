# ADR-010: Optional Instance Files

## Status

Accepted

## Context

An `instance.cue` file defines how a module is deployed: instance name, namespace, values overrides, and provider bindings. During development, authors iterate on their module and want to quickly build and apply without maintaining a separate instance file.

Requiring `instance.cue` for every `opm mod build` or `opm mod apply` adds friction to the inner development loop, especially for prototyping and testing.

Modules already define `debugValues` for development use, and `values.cue` provides defaults — both can serve as value sources without an instance file.

## Decision

Allow `opm mod build` and `opm mod apply` to work without an `instance.cue` file by synthesizing a `ModuleInstance` in memory.

Requiring `instance.cue` for all operations was rejected because it slows down the development workflow for no benefit during iteration.

When no `instance.cue` exists: use `debugValues` from the module as the value source (or explicit `--values` files if provided). When `instance.cue` exists but no `--values` is given: use `debugValues` (existing behavior). When `--values` is provided: use only those files regardless of whether `instance.cue` exists.

Synthesized instances derive the instance namespace from `metadata.defaultNamespace` in the module (overridable with `-n`), and the instance name from `metadata.name` (overridable with `--name`).

The full render pipeline executes identically for synthesized and file-backed instances — there is no separate code path.

Produce a clear error when no `instance.cue`, no `debugValues`, and no `--values` are available.

## Consequences

**Positive:** Faster inner development loop — authors can build and apply a module with zero ceremony beyond the module itself.

**Positive:** Identical render pipeline for both paths means no behavioral divergence between development and production workflows.

**Positive:** Clear error message guides users when no value source is available.

**Negative:** `debugValues` becomes load-bearing for the development workflow — it must be concrete enough to produce a valid instance.

**Trade-off:** Synthesis is transparent to the user, which is convenient but may obscure the distinction between a development build and a properly configured instance.
