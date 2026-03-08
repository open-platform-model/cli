## REMOVED Requirements

### Requirement: Transformer standalone package
**Reason**: The `internal/core/transformer/` package is eliminated. Transformer types, execution logic, match plan, and warnings all move into `pkg/engine/`. CUE-native matching means Go doesn't need standalone transformer types with RequiredResources/RequiredTraits maps.
**Migration**: Replace `transformer.TransformerMatchPlan` with `engine.MatchPlan`. Replace `transformer.Execute()` with `engine.ModuleRenderer.Render()`. Replace `transformer.CollectWarnings()` with `engine.MatchPlan.Warnings()`.

### Requirement: TransformerRequirements interface
**Reason**: The `TransformerRequirements` interface (`GetFQN()`, `GetRequiredLabels()`, `GetRequiredResources()`, `GetRequiredTraits()`) was used for Go-side matching diagnostics. With CUE-native matching, the `MatchResult` struct provides diagnostics directly (missing labels/resources/traits per transformer).
**Migration**: Replace `TransformerRequirements` interface usage with `engine.MatchResult` fields.

### Requirement: TransformerContext type
**Reason**: `TransformerContext` and its `ToMap()` method are absorbed into `pkg/engine/execute.go` as internal types (`moduleReleaseContextData`, `componentContextData`). They are not exported — the engine handles context injection internally.
**Migration**: No migration needed — context injection is internal to the engine.
