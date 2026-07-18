# Delta: pkg-types (cli-kernel-adoption)

The provider type and typed component accessors leave `pkg/`; `pkg/core.Resource` and the metadata types remain the CLI's exported surface.

## REMOVED Requirements

### Requirement: ModuleInstance has typed component accessors

**Reason**: The CLI `Instance` type is deleted with kernel adoption (0006 D9).
**Migration**: Kernel `module.Instance`.

### Requirement: Provider is a thin CUE wrapper

**Reason**: Providers are retired (0006 D39); `pkg/provider` is deleted.
**Migration**: `platform-resolution` capability.
