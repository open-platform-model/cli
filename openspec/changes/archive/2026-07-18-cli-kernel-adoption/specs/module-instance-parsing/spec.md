# Delta: module-instance-parsing (cli-kernel-adoption)

`ParseModuleInstance` is deleted; kernel `ProcessModuleInstance`/`SynthesizeInstance` prepare instances (0006 D9).

## REMOVED Requirements

### Requirement: ParseModuleInstance constructs a fully prepared Instance

**Reason**: Replaced by kernel `ProcessModuleInstance` (file path) and `SynthesizeInstance` (module path).
**Migration**: Kernel entry points via `kernel-render`.

### Requirement: ParseModuleInstance does not mutate inputs

**Reason**: Moot — the function is deleted; the kernel carries its own no-mutation contract.
**Migration**: None needed.
