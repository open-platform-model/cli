## REMOVED Requirements

### Requirement: Component and ComponentMetadata types
**Reason**: The `Component` Go type with its `Resources`, `Traits`, `Blueprints` maps is eliminated entirely. CUE-native matching via `#MatchPlan` inspects these fields in CUE space, not Go space. No Go code needs to iterate component fields for matching.
**Migration**: For display purposes (component names, labels in command output), derive information from the `MatchPlan` result or iterate the CUE `components` value directly.

### Requirement: ExtractComponents function
**Reason**: `ExtractComponents(v cue.Value) (map[string]*Component, error)` is eliminated. The engine works with CUE values directly — no extraction into Go structs needed.
**Migration**: Access components via `release.MatchComponents()` (CUE value) for matching or `release.ExecuteComponents()` (CUE value) for transform execution.
