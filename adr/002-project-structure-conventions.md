# ADR-002: Project Structure Conventions

## Status

Accepted

## Context

OPM modules need a predictable file layout so that tooling (CLI, linters, editors, CI) can locate definitions without per-project configuration. Module authors range from beginners prototyping a single service to enterprise teams managing multi-package deployments. A single rigid structure would either overwhelm beginners or constrain advanced users. The CLI needs to reliably find the module root directory, the module definition, and consumer-facing values.

## Decision

Mandatory files are required in every OPM module: `module.cue` (the module definition entry point), `values.cue` (consumer-facing defaults), and `cue.mod/module.cue` (CUE module metadata). Certain filenames are reserved for specific purposes — `components.cue`, `scopes.cue`, `policies.cue`, and `debug_values.cue` — and may only be used for their designated roles.

Three project templates are provided at different complexity levels: `simple` (single-file, for learning and prototypes), `standard` (separated concerns, the default for team projects), and `advanced` (multi-package, for enterprise deployments). The project root is identified by searching upward for the `cue.mod/` directory. Structure is enforced at validation time: a missing `module.cue` or `values.cue` produces exit code 2 (validation error).

A single template tier and flexible naming were both considered and rejected. A single tier cannot serve both beginners and advanced users without either overwhelming or constraining them. Flexible naming would prevent tooling from locating definitions reliably, requiring per-project configuration and undermining the convention-over-configuration goal.

## Consequences

Tooling can locate definitions by convention without configuration; `module.cue` is always the entry point and `values.cue` always provides defaults. The three tiers offer a natural progression path — authors start with `simple` and graduate to `standard` or `advanced` as complexity grows. Early validation (exit code 2) catches structural problems before rendering or deployment reaches a cluster.

Authors must follow naming conventions strictly, as deviation breaks module portability. Protected filenames reduce flexibility for authors who might want those names for unrelated purposes.
