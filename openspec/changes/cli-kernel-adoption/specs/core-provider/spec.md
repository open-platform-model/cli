# Delta: core-provider (cli-kernel-adoption)

The CLI's `Provider` type is retired with the provider concept (0006 D39). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Provider type location and interface

**Reason**: `pkg/provider` (and the former `pkg/core` provider type) is deleted; core defines no `#Provider` (enhancement 0001, 0006 D39).
**Migration**: Platforms + catalog subscriptions (`platform-resolution`); transformers via the kernel's materialized platform.
