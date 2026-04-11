# ADR-007: AST-Based Overlay and Recursive Values Validation

## Status

Accepted

## Context

During module loading (see ADR-006), consumer-provided values must be merged with the module's schema as a CUE overlay before evaluation. An earlier approach used `fmt.Sprintf` string formatting to generate the overlay CUE file. That approach was fragile: escaping and quoting edge cases were easy to get wrong, and the output could not be validated at compile time, making it possible to produce syntactically invalid CUE silently.

Values validation needs to check every field the consumer provides against the module's schema, detecting unknown fields, type mismatches, and constraint violations. When multiple `--values` files are provided, individual files may be intentionally incomplete — for example, a base config in one file and environment-specific overrides in another — so validating files individually would produce false errors on fields that are only present in other files.

Validation errors need to be actionable. Users need to know which file, line, and column caused the problem, and error paths should be rooted in the values namespace rather than the internal schema namespace, because users reason about their values files rather than the module's internal structure.

## Decision

Construct CUE overlay files using typed AST construction (`cuelang.org/go/cue/ast`) rather than string formatting. The AST approach produces byte-identical CUE output compared to the previous `fmt.Sprintf` approach while making the structure explicit in Go types that the compiler can check. String-based overlay generation was rejected because it is fragile and cannot be validated at compile time.

Unify multiple `--values` files before validation, not individually. Individual files may be intentionally incomplete, and unifying first ensures that all fields are present before any constraint is checked.

Validate unified values recursively by walking every field in the merged values struct and checking each against the corresponding schema node, using custom closedness checking. This gives precise, per-field error reporting that the CUE evaluator's built-in error messages do not always surface with sufficient granularity.

Root all validation error paths at `values` — for example, `values.media."test-key"` rather than `#config.media."test-key"`. Users think in terms of their values files, not the module's internal schema namespace.

Include file:line:col position from source in every validation error, resolved via `cue.Value.Expr()`. This enables IDE-style error navigation and lets users jump directly to the offending line.

Gate on `IsConcrete()` per component immediately after `core.ExtractComponents()` returns. Non-concrete components produce an error naming the component. This check catches missing required values before any further pipeline work is attempted.

## Consequences

**Positive:** AST-based overlay construction is type-safe at the Go level — invalid CUE structure is caught at compile time rather than at runtime when the generated string is evaluated.

**Positive:** Values-rooted error paths match the user's mental model, reducing confusion when an error refers to a path in the values file rather than an internal schema path.

**Positive:** Position tracking (file:line:col) in every validation error enables IDE-style error navigation and faster iteration for users fixing configuration problems.

**Positive:** Unified validation handles multi-file value composition correctly, so intentionally split values files do not produce false validation errors.

**Negative:** AST construction is more verbose Go code than a format string; adding new overlay structures requires constructing explicit AST nodes rather than adjusting a template string.

**Negative:** Recursive validation with custom closedness checking is complex to maintain; contributors modifying the validation path must understand both CUE's evaluation model and the custom walk logic.

**Trade-off:** Per-component concreteness checks catch problems early but require all components to be concrete after values are applied — partial rendering is not supported.
