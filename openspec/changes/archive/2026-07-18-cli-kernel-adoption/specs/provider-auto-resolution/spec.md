# Delta: provider-auto-resolution (cli-kernel-adoption)

Provider auto-resolution is retired with the provider concept (0006 D39/D21). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Auto-select provider when single provider configured

**Reason**: No providers exist to auto-select; platform-source precedence replaces the selection problem (0006 D21).
**Migration**: `platform-resolution` precedence (`--platform` > cluster CR > local default).

### Requirement: Provider auto-resolution visible in verbose output

**Reason**: Replaced by platform-source provenance reporting on every command.
**Migration**: `platform-resolution` provenance requirement.
