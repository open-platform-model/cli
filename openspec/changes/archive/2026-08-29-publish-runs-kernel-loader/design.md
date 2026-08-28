## Context

See proposal.md. `publish.Run` (`internal/publish/gates.go`) loads with `cue/load` via `loadArtifact`, then runs `conformIdentity`, `identityStates`, `resolveVersion` and the gate functions in dependency order, accumulating `Refusal`s on the `Plan`. `internal/publish/vet.go` `VetChecks` runs the local subset for `opm module vet`. The kernel loader is `github.com/open-platform-model/library/opm/helper/loader/file.LoadModulePackage(ctx, dir, LoadOptions{Registry})`, already imported by `internal/workflow/render`. Its sentinels are `ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`.

## Goals / Non-Goals

**Goals:** publish and vet refuse exactly what the kernel refuses, with the kernel's words.
**Non-Goals:** re-implementing any loader check in `cli`; a catalog-side equivalent; changing the identity tristate.

## Decisions

**D1. Call the loader; do not import the gate.** The gate is `internal` to library and single-sourced there. Calling `LoadModulePackage` inherits every future gate change with zero cli code. Alternative (export `shape.Gate` from library) rejected: a public symbol for one caller, and it would still miss what the loader does around the gate (instance count, build errors).

**D2. Placement: after `resolveVersion`, before the derivation gates; accumulate, do not short-circuit.** The gate runs once the identity tristate is known so it can apply D3's skip; the loader refusing does not make the remaining gates unevaluable, and the spec wants every evaluable gate in one pass. Short-circuit only where `Run` already does (no `cue.mod`, tree does not load as CUE). In `VetChecks` the gate runs after `gateDerivation`, at the end of the local sequence; vet has no further gates, so the position is equivalent. The plan prints a `kernel loader` row (`accepted` / `refused`) only when the gate ran.

**D3. One refusal per cause.** `identityStates` treats a defaulted field as concrete, so it does not refuse; only the loader gate does. Verified by reading `resolveVersion`: the defaulted case lands in `StateConcrete` and sets `TagSource += " (default)"`. The converse also holds: when any identity field is open or absent, the identity gates (or `--version`) already own that cause, and the loader would name the same missing value again; the kernel gate is skipped for those trees and the plan omits its row rather than showing a pass it never checked.

**D4. Refusal shape.**
```
Headline:    "the kernel would refuse to load this module"
Evidence:    [["loader", <err.Error()>]]
Consequence: "opm module build and the operator use this loader; a published tag would be unloadable everywhere"
Action:      by sentinel:
  ErrMissingRequiredField -> "Make the field a concrete literal in identity/identity.cue (see opm module vet)"
  ErrWrongKind            -> "Publish the artifact with its own command (opm catalog publish for a catalog)"
  ErrInvalidPackage       -> "Fix the package layout: one CUE package at the module root"
```
Numbered msg 12, the next free number in the gate numbering carried in the gate doc comments (`gates.go`, `catalog_gates.go`); there is no separate catalog listing in `refusal.go`.

**D5. Registry for the loader.** Pass `opts.Registry` through `LoadOptions{Registry}` so the loader resolves deps exactly as `cue/load` did in `loadArtifact` (same env, same GHCR mapping). No second resolution policy.

**D6. Vet.** If `VetChecks` calls `Run`'s local prefix, it inherits the gate; if it has its own sequence, insert the same call at the same position. Check during implementation; the spec requires vet to report it.

## Research & Decisions

### Why publish and the loader disagreed
**Context**: 20 modules passed publish, failed load.
**Explored**: 2026-08-28: publish `identityStates` reads `v.String()` (resolves defaults); loader `requireConcrete` reads `IsConcrete()` (false for a defaulted disjunction). Go probe against `cuelang.org/go v0.17.1` confirmed both.
**Options considered**:
1. Publish calls the loader (chosen): one source of truth, inherits future gates.
2. Publish re-implements `IsConcrete()` on `metadata.version`: duplicates the gate, drifts.
3. Loosen the loader to `Default()`-first: rejected in the library change; identity is a value, not a suggestion.
**Decision**: option 1.
**Rationale**: 0011's principle, and the least code.

## Data flow

```
opm module publish ./m
  loadArtifact (cue/load)  ──fail──> refusal (tree does not load), stop
  conformIdentity, identityStates, resolveVersion
  LoadModulePackage(./m)   ──fail──> refusal 12 (kernel), continue   (skipped if identity open/absent)
  gate*  ...
  plan prints; exit 2 if any refusal
```

## Error handling

Loader errors are wrapped sentinels; branch with `errors.Is`. A loader error that matches no sentinel (CUE build error the raw load did not surface) uses the generic action "fix the error above and re-run". Exit codes unchanged (0 GO, 2 refused, 3 registry unreachable); the loader gate never returns 3 because it resolves the same deps `loadArtifact` already resolved.

## Risks / Trade-offs

- [Second CUE build per publish] → CUE cache is shared; measured cost of `opm module build` on a fleet module is under a second.
- [Templates and the podinfo fixture fail the new gate until their sibling changes land] → they currently pass the loader (interpolation workaround), so no ordering constraint; the sibling changes only simplify them.
