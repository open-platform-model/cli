## REMOVED Requirements

### Requirement: Component extraction from CUE values
**Reason**: Superseded by CUE-native matching. The engine passes raw CUE component values to `#MatchPlan` and `#transform` without extracting them into Go structs first.
**Migration**: No direct replacement needed — component data stays in CUE throughout the pipeline.
