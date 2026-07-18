# Delta: module-instance-processing (cli-kernel-adoption)

The CLI's `ProcessModuleInstance` rendering entrypoint (pkg/render) is deleted; the kernel's compile pipeline is the rendering entrypoint (0006 D9).

## REMOVED Requirements

### Requirement: ProcessModuleInstance is the public rendering entrypoint

**Reason**: `pkg/render` is deleted; kernel `Compile` is the pipeline.
**Migration**: `kernel-render` capability.

### Requirement: ProcessModuleInstance does not validate config

**Reason**: Moot — validation layering is the kernel's contract now.
**Migration**: Kernel `Validate`/`Compile` contract.

### Requirement: Finalized components are transient

**Reason**: Finalization semantics are the kernel's (`Finalize`).
**Migration**: Kernel compile contract.
