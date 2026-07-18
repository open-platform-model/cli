# Delta: render-pipeline (cli-kernel-adoption)

The CLI's own render pipeline is deleted; renders go through the library kernel (0006 D9). Capability retired wholesale — successor requirements live in `kernel-render`.

## REMOVED Requirements

### Requirement: LoadedComponent carries annotations

**Reason**: `pkg/render`'s component model is deleted; the kernel's compile model carries component metadata (0006 D9).
**Migration**: Kernel `CompileResult`/`ComponentSummary`.

### Requirement: TransformerComponentMetadata carries annotations

**Reason**: Same — CLI-side transformer metadata model deleted.
**Migration**: Kernel transformer metadata via `MatchPlan`.

### Requirement: Legacy pipeline package

**Reason**: The pipeline package is deleted.
**Migration**: `kernel-render` capability.

### Requirement: Pipeline interface

**Reason**: The pipeline interface is deleted; the kernel's phase API (`Validate`/`Match`/`Plan`/`Compile`/`Finalize`) is the contract.
**Migration**: `kernel-render` capability.

### Requirement: RenderOptions struct

**Reason**: Options move to kernel inputs and `internal/workflow/render` options.
**Migration**: Kernel input structs (`synth.InstanceInput`, phase inputs).

### Requirement: Five-phase pipeline orchestration

**Reason**: Orchestration is the kernel's; the CLI no longer sequences its own phases.
**Migration**: `kernel-render` capability.

### Requirement: RenderResult with Unstructured resources

**Reason**: The result shape comes from kernel finalize output adapted in `internal/workflow/render`.
**Migration**: `kernel-render` capability; workflow `Result` remains the CLI-facing carrier.
