# Delta: engine-rendering (cli-kernel-adoption)

The CLI-side module renderer is deleted; the kernel compiles/executes transformers (enhancement 0006, "The CLI replaces its own render/match pipeline entirely with the `library` kernel"). Capability retired wholesale — successor requirements live in `kernel-render`.

## REMOVED Requirements

### Requirement: Module renderer renders a Release into resources

**Reason**: `pkg/render`'s module renderer is deleted (enhancement 0006, "The CLI replaces its own render/match pipeline entirely with the `library` kernel").
**Migration**: Kernel `Compile`/`Finalize` via `kernel-render`.

### Requirement: Transform execution injects context and component

**Reason**: Transformer execution is kernel behavior now.
**Migration**: Kernel compile contract (library, enhancement 0001).

### Requirement: Transform execution functions accept Release

**Reason**: The CLI-side execution API is deleted.
**Migration**: Kernel compile contract.

### Requirement: No CLI logging framework dependency

**Reason**: Moot — the package it constrained is deleted; the kernel is logging-free by its own constitution.
**Migration**: None needed.

### Requirement: ModuleResult contract

**Reason**: Result contract replaced by kernel `CompileResult` adaptation.
**Migration**: `kernel-render` capability.
