## REMOVED Requirements

### Requirement: Separate release building phase
**Reason**: `builder.Build()` with its FillPath chain, values validation, auto-secrets injection, and component extraction is eliminated. All of this is handled by the new loader: CUE evaluation naturally handles value unification and defaults, gates handle validation, `#AutoSecrets` handles secrets, and `finalizeValue()` handles constraint stripping.
**Migration**: Replace `builder.Build(ctx, mod, opts, valuesFiles)` with `loader.LoadReleasePackage()` + `loader.LoadModuleReleaseFromValue()`.
