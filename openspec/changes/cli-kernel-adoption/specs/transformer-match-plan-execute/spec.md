# Delta: transformer-match-plan-execute (cli-kernel-adoption)

The CLI's match-plan reproduction of CUE semantics is deleted; the kernel is the semantics (0006 D9). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Go matcher reproduces the current CUE match-plan semantics

**Reason**: There is no CLI-side matcher to keep in sync; the kernel's matcher is the single implementation (0006 D9).
**Migration**: Kernel `Match` via `kernel-render`.

### Requirement: MatchPlan provides structured diagnostics

**Reason**: The kernel's `MatchPlan`/`PlanResult` carries the structured diagnostics.
**Migration**: Surface kernel `PlanResult` in workflow output.
