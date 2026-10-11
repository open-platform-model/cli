# Delta: transformer-match-plan-execute (cli-kernel-adoption)

The CLI's match-plan reproduction of CUE semantics is deleted; the kernel is the semantics (enhancement 0006, "The CLI replaces its own render/match pipeline entirely with the `library` kernel"). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Go matcher reproduces the current CUE match-plan semantics

**Reason**: There is no CLI-side matcher to keep in sync; the kernel's matcher is the single implementation (enhancement 0006, "The CLI replaces its own render/match pipeline entirely with the `library` kernel").
**Migration**: Kernel `Match` via `kernel-render`.

### Requirement: MatchPlan provides structured diagnostics

**Reason**: The kernel's `MatchPlan`/`PlanResult` carries the structured diagnostics.
**Migration**: Surface kernel `PlanResult` in workflow output.
