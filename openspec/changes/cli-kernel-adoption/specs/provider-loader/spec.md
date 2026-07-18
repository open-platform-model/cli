# Delta: provider-loader (cli-kernel-adoption)

The provider concept is retired — core has no `#Provider`; platforms + catalog subscriptions replace it (0006 D39, enhancement 0001). Capability retired wholesale.

## REMOVED Requirements

### Requirement: LoadProvider returns a fully-populated core.Provider

**Reason**: `pkg/loader.LoadProvider` and `pkg/provider` are deleted; there are no providers to load (0006 D39).
**Migration**: Platform resolution + kernel materialization (`platform-resolution` capability).

### Requirement: Extract all provider metadata fields from CUE value

**Reason**: Provider metadata no longer exists.
**Migration**: Catalog/transformer metadata surfaces through the kernel's materialized platform.
