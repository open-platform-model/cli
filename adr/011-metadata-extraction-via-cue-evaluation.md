# ADR-011: Metadata Extraction via CUE Evaluation

## Status

Accepted

## Context

During module loading, scalar metadata fields (Name, DefaultNamespace, FQN, Version, UUID, Labels) must be extracted from the CUE module definition and surfaced as Go values. Two approaches exist for doing this: inspecting the CUE abstract syntax tree (AST) to find field declarations, or evaluating the CUE value and reading the computed result via `LookupPath`.

The distinction matters because some metadata fields are computed rather than literal. The fully qualified name (FQN), for example, is assembled in the v1alpha1 schema as `modulePath/name:semver` — it is not written literally by the module author. AST inspection would find the field but not its computed value; only evaluation produces the final string.

A separate concern applies to the `Config` and `Components` fields. These contain schema definitions with constraints and defaults that downstream consumers need to inspect. Decoding them into Go structs would discard that information.

## Decision

Extract all scalar metadata fields from the fully evaluated `cue.Value` using `LookupPath` and `.String()`, not AST inspection.

AST-based extraction was rejected because it cannot access computed values (such as FQN) and would require reimplementing parts of CUE's evaluation semantics in Go.

FQN is read directly from `metadata.fqn`, where it is pre-computed by the v1alpha1 schema. The loader does not construct the FQN string in Go. ModulePath is read from `metadata.modulePath` and Version from `metadata.version`.

`module.Config` and `module.Components` are kept as non-concrete `cue.Value` references rather than decoded into Go structs. This preserves schema definitions, constraints, and defaults for downstream validation, matching, and generation.

One exception is accepted: `PkgName` is extracted from `build.Instance.PkgName` because it is not available on the evaluated value.

See also ADR-001 for the choice of CUE as the configuration language.

## Consequences

Metadata extraction is simple — `LookupPath` calls instead of AST tree walking, and computed metadata (names built from expressions, schema-assembled FQN) works automatically without any special handling in the loader.

Non-concrete Config and Components preserve full CUE schema semantics for downstream consumers.

The single `PkgName` exception creates a minor asymmetry in the extraction pattern: all other metadata comes from the evaluated value, but `PkgName` comes from a different source.

Depending on evaluated values means metadata extraction cannot happen until CUE evaluation completes. There is no way to quickly peek at metadata without performing full evaluation first.
