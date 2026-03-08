## REMOVED Requirements

### Requirement: Provider.Match() method
**Reason**: Go-side `Provider.Match(components map[string]*Component) *TransformerMatchPlan` is eliminated. Matching is now CUE-native via `#MatchPlan`. The `Provider` type becomes a thin CUE value wrapper.
**Migration**: The engine's `buildMatchPlan()` handles matching. See `engine-rendering` spec.
