# Delta: provider-match (cli-kernel-adoption)

Provider-driven matching is retired; the kernel matches against the materialized platform (enhancement 0006, "The CLI replaces its own render/match pipeline entirely with the `library` kernel" and "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2"). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Provider exposes a Match method

**Reason**: `pkg/provider` is deleted (enhancement 0006, "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2").
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
