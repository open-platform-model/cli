## REMOVED Requirements

### Requirement: Go-side component-to-transformer matching
**Reason**: Replaced by CUE-native `#MatchPlan` in `v1alpha1/core/matcher/matcher.cue`. The CUE matcher evaluates the full cartesian product of components x transformers with structured diagnostics. Go-side matching in `internal/core/provider/provider.go` is eliminated.
**Migration**: The engine calls `buildMatchPlan()` which fills `#MatchPlan.#provider` and `#MatchPlan.#components` and decodes the CUE result. See `engine-rendering` spec.
