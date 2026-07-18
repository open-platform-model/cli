# Delta: provider-match (cli-kernel-adoption)

Provider-driven matching is retired; the kernel matches against the materialized platform (0006 D9/D39). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Provider exposes a Match method

**Reason**: `pkg/provider` is deleted (0006 D39).
**Migration**: Kernel `Match` against the materialized platform.

### Requirement: Matching algorithm evaluates required labels, resources, and traits

**Reason**: Match semantics are the kernel contract (enhancement 0001).
**Migration**: Library kernel spec.

### Requirement: TransformerMatchPlan carries match details for diagnostics

**Reason**: Replaced by kernel `MatchPlan`/`PlanResult`.
**Migration**: Surface kernel diagnostics in workflow output.

### Requirement: Provider.Match() method

**Reason**: The type and method are deleted.
**Migration**: Kernel `Match`.
