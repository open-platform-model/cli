# Delta: component-matching (cli-kernel-adoption)

CLI-side component-to-transformer matching is deleted; the kernel's matcher (enhancement 0001 model) is the single implementation (0006 D9). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Match components against transformer definitions

**Reason**: Matching is the kernel's (0006 D9); the CLI carries no second implementation.
**Migration**: Kernel `Match` via `kernel-render`.

### Requirement: Match details recorded for rejection reporting

**Reason**: Diagnostics come from the kernel's `MatchPlan`/`PlanResult` (per-component summaries, unmatched list, warnings).
**Migration**: Surface kernel `PlanResult` diagnostics in workflow output.

### Requirement: Optional labels, resources, and traits do not affect matching

**Reason**: Match semantics are the kernel contract (enhancement 0001), not a CLI requirement.
**Migration**: Library kernel spec.

### Requirement: Unhandled traits are tracked per match

**Reason**: Same — kernel `PlanResult.Warnings` carries unhandled-trait advisories.
**Migration**: Surface kernel warnings in workflow output.

### Requirement: Go-side component-to-transformer matching

**Reason**: The CLI's Go matcher is deleted.
**Migration**: Kernel `Match`.
