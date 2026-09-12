## REMOVED Requirements

### Requirement: ConfigError type for gate validation

**Reason**: The only producer of `ConfigError` was the CLI's copy of the kernel validator (`pkg/validate`), which is deleted; values validation now returns the library kernel's raw CUE error tree, and the grouped presentation reads positions from that tree directly.

**Migration**: Callers that matched `*ConfigError` with `errors.As` walk the CUE error tree instead (`cuelang.org/go/cue/errors.Errors`, or the CLI's `GroupedErrorsFromError`, which stays in `pkg/errors`). Callers that wanted the gate context and name frame the kernel error with `fmt.Errorf` at the call site.
