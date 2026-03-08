## REMOVED Requirements

### Requirement: TransformerMatchPlan.Execute() method
**Reason**: The standalone `Execute(ctx, rel) ([]*Resource, []error)` method on `TransformerMatchPlan` is replaced by `pkg/engine/executeTransforms()`. The engine handles the full execution pipeline internally: pair iteration, component/context injection, output decoding.
**Migration**: Replace `matchPlan.Execute(ctx, rel)` with `engine.ModuleRenderer.Render(ctx, release)` which handles both matching and execution.
